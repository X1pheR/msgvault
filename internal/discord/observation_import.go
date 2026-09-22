package discord

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	ObservationFormatVersion = 1
	sourceTypeDiscordLocal   = "discord_local"

	ObservationKindContainer = "container"
	ObservationKindMessage   = "message"
	ObservationKindDelete    = "delete"

	channelTypeDM      = 1
	channelTypeGroupDM = 3
)

// Observation is one locally acquired Discord fact.
//
// Container observations preserve metadata even for message-less parents such
// as forum channels. Message observations carry a complete message snapshot
// plus the container that owns it. Delete observations carry only an explicit
// provider message ID; absence is never interpreted as deletion.
type Observation struct {
	Version          int      `json:"version"`
	Kind             string   `json:"kind"`
	SourceType       string   `json:"source_type"`
	SourceIdentifier string   `json:"source_identifier"`
	Channel          *Channel `json:"channel,omitempty"`
	Message          *Message `json:"message,omitempty"`
	MessageID        string   `json:"message_id,omitempty"`
}

// ObservationImportOptions identifies one stable Discord source and one local
// JSONL observation stream. SourceIdentifier is deliberately acquisition
// neutral: the producer decides whether it represents a guild or an
// account-scoped local stream.
type ObservationImportOptions struct {
	SourceIdentifier  string
	SourceDisplayName string
	Reader            io.Reader
}

// ImportObservations imports locally acquired Discord observations without
// using imp.api or any Discord credential.
//
// The format deliberately does not model provider history cursors or infer
// completeness. Replaying a complete observation is safe because the shared
// msgvault source/message identity is upserted; only an explicit delete event
// creates an upstream tombstone.
func (imp *Importer) ImportObservations(
	ctx context.Context,
	opts ObservationImportOptions,
) (*ImportSummary, error) {
	if imp == nil || imp.store == nil {
		return nil, errors.New("discord observation importer store is required")
	}
	if strings.TrimSpace(opts.SourceIdentifier) == "" {
		return nil, errors.New("discord observation source identifier is required")
	}
	if opts.Reader == nil {
		return nil, errors.New("discord observation reader is required")
	}

	sourceScope, err := parseObservationSourceIdentifier(opts.SourceIdentifier)
	if err != nil {
		return nil, err
	}

	source, err := imp.store.GetOrCreateSource(sourceTypeDiscordLocal, opts.SourceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("get Discord observation source: %w", err)
	}
	if opts.SourceDisplayName != "" {
		if err := imp.store.UpdateSourceDisplayName(source.ID, opts.SourceDisplayName); err != nil {
			return nil, fmt.Errorf("update Discord observation source name: %w", err)
		}
	}

	summary := &ImportSummary{
		SourceID:            source.ID,
		processedMessageIDs: make(map[string]struct{}),
	}
	seenContainers := make(map[string]struct{})

	scanner := bufio.NewScanner(opts.Reader)
	scanner.Buffer(make([]byte, 64<<10), maxResponseBytes)

	line := 0
	for scanner.Scan() {
		line++
		if err := ctx.Err(); err != nil {
			return summary, err
		}

		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}

		var observation Observation
		if err := json.Unmarshal([]byte(raw), &observation); err != nil {
			return summary, fmt.Errorf("decode Discord observation line %d: %w", line, err)
		}
		if observation.Version != ObservationFormatVersion {
			return summary, fmt.Errorf(
				"Discord observation line %d: unsupported version %d",
				line,
				observation.Version,
			)
		}
		if err := validateObservationEnvelopeSource(opts.SourceIdentifier, observation); err != nil {
			return summary, fmt.Errorf("Discord observation line %d: %w", line, err)
		}

		switch observation.Kind {
		case ObservationKindContainer:
			if observation.Channel == nil {
				return summary, fmt.Errorf(
					"Discord observation line %d: container requires channel",
					line,
				)
			}
			if err := validateObservationChannelSource(sourceScope, observation.Channel); err != nil {
				return summary, fmt.Errorf("Discord observation line %d: %w", line, err)
			}
			if _, err := imp.ensureObservedConversation(source.ID, observation.Channel); err != nil {
				return summary, fmt.Errorf("Discord observation line %d: %w", line, err)
			}
			if _, known := seenContainers[observation.Channel.ID]; !known {
				seenContainers[observation.Channel.ID] = struct{}{}
				summary.ContainersProcessed++
			}

		case ObservationKindMessage:
			if observation.Channel == nil || observation.Message == nil {
				return summary, fmt.Errorf(
					"Discord observation line %d: message requires channel and message",
					line,
				)
			}
			if err := validateObservationChannelSource(sourceScope, observation.Channel); err != nil {
				return summary, fmt.Errorf("Discord observation line %d: %w", line, err)
			}
			if err := imp.importObservedMessage(
				ctx,
				source.ID,
				observation.Channel,
				observation.Message,
				summary,
			); err != nil {
				return summary, fmt.Errorf("Discord observation line %d: %w", line, err)
			}
			if _, known := seenContainers[observation.Channel.ID]; !known {
				seenContainers[observation.Channel.ID] = struct{}{}
				summary.ContainersProcessed++
			}

		case ObservationKindDelete:
			if observation.MessageID == "" {
				return summary, fmt.Errorf(
					"Discord observation line %d: delete requires message_id",
					line,
				)
			}
			if _, err := ParseSnowflake(observation.MessageID); err != nil {
				return summary, fmt.Errorf(
					"Discord observation line %d: invalid delete message_id: %w",
					line,
					err,
				)
			}
			if err := imp.store.MarkMessageDeleted(source.ID, observation.MessageID); err != nil {
				return summary, fmt.Errorf(
					"mark observed Discord message %s deleted: %w",
					observation.MessageID,
					err,
				)
			}

		default:
			return summary, fmt.Errorf(
				"Discord observation line %d: unsupported kind %q",
				line,
				observation.Kind,
			)
		}
	}
	if err := scanner.Err(); err != nil {
		return summary, fmt.Errorf("read Discord observations: %w", err)
	}

	if err := imp.resolveDeferredReplies(source.ID); err != nil {
		return summary, fmt.Errorf("resolve observed Discord replies: %w", err)
	}
	if err := imp.store.RecomputeConversationStats(source.ID); err != nil {
		return summary, fmt.Errorf("recompute observed Discord conversation stats: %w", err)
	}
	return summary, nil
}

func validateObservationEnvelopeSource(importIdentifier string, observation Observation) error {
	if strings.TrimSpace(observation.SourceType) == "" || strings.TrimSpace(observation.SourceIdentifier) == "" {
		return errors.New("Discord observation source identity is required")
	}
	if observation.SourceType != sourceTypeDiscordLocal {
		return fmt.Errorf(
			"Discord observation source_type %q is invalid; expected %q",
			observation.SourceType,
			sourceTypeDiscordLocal,
		)
	}
	if _, err := parseObservationSourceIdentifier(observation.SourceIdentifier); err != nil {
		return err
	}
	if observation.SourceIdentifier != importIdentifier {
		return fmt.Errorf(
			"Discord observation source %q does not match import source %q",
			observation.SourceIdentifier,
			importIdentifier,
		)
	}
	return nil
}

type observationSourceScope struct {
	kind string
	id   string
}

func parseObservationSourceIdentifier(identifier string) (observationSourceScope, error) {
	identifier = strings.TrimSpace(identifier)
	if strings.HasPrefix(identifier, "account:") {
		userID := strings.TrimPrefix(identifier, "account:")
		if _, err := ParseSnowflake(userID); err != nil {
			return observationSourceScope{}, fmt.Errorf(
				"invalid Discord local source identifier %q: account ID must be a Discord snowflake",
				identifier,
			)
		}
		return observationSourceScope{kind: "account", id: userID}, nil
	}

	if _, err := ParseSnowflake(identifier); err != nil {
		return observationSourceScope{}, fmt.Errorf(
			"invalid Discord local source identifier %q: expected guild snowflake or account:<user-snowflake>",
			identifier,
		)
	}
	return observationSourceScope{kind: "guild", id: identifier}, nil
}

func validateObservationChannelSource(scope observationSourceScope, channel *Channel) error {
	if channel == nil {
		return errors.New("observed Discord channel is required")
	}

	switch scope.kind {
	case "guild":
		if channel.GuildID == "" {
			return fmt.Errorf(
				"Discord local source %s is guild-scoped but channel %s has no guild ID",
				scope.id,
				channel.ID,
			)
		}
		if channel.GuildID != scope.id {
			return fmt.Errorf(
				"Discord local source %s does not match guild %s for channel %s",
				scope.id,
				channel.GuildID,
				channel.ID,
			)
		}
		if channel.Type == channelTypeDM || channel.Type == channelTypeGroupDM {
			return fmt.Errorf(
				"Discord local guild source %s cannot import DM channel %s",
				scope.id,
				channel.ID,
			)
		}
		return nil

	case "account":
		if channel.GuildID != "" {
			return fmt.Errorf(
				"Discord local account-scoped source account:%s cannot import guild %s channel %s",
				scope.id,
				channel.GuildID,
				channel.ID,
			)
		}
		if channel.Type != channelTypeDM && channel.Type != channelTypeGroupDM {
			return fmt.Errorf(
				"Discord local account-scoped source account:%s requires DM or group-DM channel, got type %d",
				scope.id,
				channel.Type,
			)
		}
		return nil

	default:
		return fmt.Errorf("unsupported Discord local source scope %q", scope.kind)
	}
}

func (imp *Importer) importObservedMessage(
	ctx context.Context,
	sourceID int64,
	channel *Channel,
	message *Message,
	summary *ImportSummary,
) error {
	if channel == nil || channel.ID == "" {
		return errors.New("observed Discord channel has an empty ID")
	}

	messageCopy := *message
	if messageCopy.ChannelID == "" {
		messageCopy.ChannelID = channel.ID
	}
	if messageCopy.GuildID != "" && channel.GuildID != "" && messageCopy.GuildID != channel.GuildID {
		return fmt.Errorf(
			"observed Discord message %s guild %s does not match channel guild %s",
			messageCopy.ID,
			messageCopy.GuildID,
			channel.GuildID,
		)
	}
	if messageCopy.GuildID == "" {
		messageCopy.GuildID = channel.GuildID
	}
	if err := validateDiscordMessage(channel.ID, messageCopy); err != nil {
		return err
	}

	conversationID, err := imp.ensureObservedConversation(sourceID, channel)
	if err != nil {
		return err
	}

	return imp.persistPage(
		ctx,
		sourceID,
		conversationID,
		[]Message{messageCopy},
		summary,
		nil,
	)
}

func (imp *Importer) ensureObservedConversation(
	sourceID int64,
	channel *Channel,
) (int64, error) {
	if channel == nil || channel.ID == "" {
		return 0, errors.New("observed Discord channel has an empty ID")
	}
	if _, err := ParseSnowflake(channel.ID); err != nil {
		return 0, fmt.Errorf("invalid observed Discord channel ID: %w", err)
	}

	mapped, err := mapConversation(channel)
	if err != nil {
		return 0, err
	}
	mapped.Conversation.ConversationType = observedConversationType(channel.Type)

	conversationID, err := imp.store.EnsureConversationWithType(
		sourceID,
		mapped.Conversation.SourceConversationID,
		mapped.Conversation.ConversationType,
		mapped.Conversation.Title,
	)
	if err != nil {
		return 0, fmt.Errorf("ensure observed Discord conversation: %w", err)
	}

	metadata := sql.NullString{
		String: string(mapped.Metadata),
		Valid:  len(mapped.Metadata) != 0,
	}
	if err := imp.store.SetConversationMetadata(conversationID, metadata); err != nil {
		return 0, fmt.Errorf("persist observed Discord conversation metadata: %w", err)
	}
	return conversationID, nil
}

func observedConversationType(channelType int) string {
	switch channelType {
	case channelTypeDM:
		return "direct_chat"
	case channelTypeGroupDM:
		return "group_chat"
	case channelTypeAnnouncementThread, channelTypePublicThread, channelTypePrivateThread:
		return "thread"
	default:
		return "channel"
	}
}
