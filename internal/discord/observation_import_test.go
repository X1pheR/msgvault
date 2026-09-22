package discord

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.kenn.io/msgvault/internal/testutil"
)

func observationJSONL(t *testing.T, sourceIdentifier string, observations ...Observation) string {
	t.Helper()

	var out strings.Builder
	for _, observation := range observations {
		observation.SourceType = sourceTypeDiscordLocal
		observation.SourceIdentifier = sourceIdentifier
		raw, err := json.Marshal(observation)
		require.NoError(t, err)
		out.Write(raw)
		out.WriteByte('\n')
	}
	return out.String()
}

func observedMessage(id, channelID, guildID, content string) Message {
	timestamp, _ := TimestampFromSnowflake(id)
	return Message{
		ID:        id,
		ChannelID: channelID,
		GuildID:   guildID,
		Content:   content,
		Timestamp: timestamp,
		Type:      0,
		Author:    User{ID: "author-" + id, Username: "author-" + id},
	}
}

func importObservationsForTest(t *testing.T, importer *Importer, sourceIdentifier, input string) *ImportSummary {
	t.Helper()

	summary, err := importer.ImportObservations(t.Context(), ObservationImportOptions{
		SourceIdentifier:  sourceIdentifier,
		SourceDisplayName: "Local Discord observations",
		Reader:            strings.NewReader(input),
	})
	require.NoError(t, err)
	return summary
}

func TestSDDDLE001LocalObservationSourceTypeAndMessageType(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	st := testutil.NewSQLiteTestStore(t)
	channel := Channel{ID: "800", GuildID: "700", Type: channelTypeGuildText, Name: "general"}
	message := observedMessage("900", channel.ID, channel.GuildID, "source isolation")

	_, err := NewImporter(st, nil).ImportObservations(t.Context(), ObservationImportOptions{
		SourceIdentifier: "700",
		Reader: observationReader(t, "700",
			Observation{Version: 1, Kind: ObservationKindMessage, Channel: &channel, Message: &message},
		),
	})
	require.NoError(err)

	var sourceType, messageType string
	require.NoError(st.DB().QueryRow(st.Rebind("SELECT s.source_type, m.message_type FROM messages m JOIN sources s ON s.id = m.source_id WHERE s.identifier = ? AND m.source_message_id = ?"), "700", "900").Scan(&sourceType, &messageType))
	assert.Equal("discord_local", sourceType)
	assert.Equal("discord", messageType)
}

func TestSDDDLE002003SourceIdentifierAndScopeValidation(t *testing.T) {
	st := testutil.NewSQLiteTestStore(t)

	guildChannel := Channel{ID: "800", GuildID: "700", Type: channelTypeGuildText, Name: "general"}
	guildMessage := observedMessage("900", guildChannel.ID, guildChannel.GuildID, "guild")
	dmChannel := Channel{ID: "801", Type: 1, Name: "dm"}
	dmMessage := observedMessage("901", dmChannel.ID, "", "dm")

	tests := []struct {
		name       string
		source     string
		channel    Channel
		message    Message
		wantErrSub string
	}{
		{name: "guild valid", source: "700", channel: guildChannel, message: guildMessage},
		{name: "guild mismatch", source: "701", channel: guildChannel, message: guildMessage, wantErrSub: "does not match guild"},
		{name: "account DM valid", source: "account:999", channel: dmChannel, message: dmMessage},
		{name: "account cannot import guild", source: "account:999", channel: guildChannel, message: guildMessage, wantErrSub: "account-scoped"},
		{name: "invalid source", source: "local:legacy", channel: dmChannel, message: dmMessage, wantErrSub: "invalid Discord local source identifier"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewImporter(st, nil).ImportObservations(t.Context(), ObservationImportOptions{
				SourceIdentifier: tc.source,
				Reader: observationReader(t, tc.source,
					Observation{Version: 1, Kind: ObservationKindMessage, Channel: &tc.channel, Message: &tc.message},
				),
			})
			if tc.wantErrSub == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErrSub)
		})
	}
}

func observationReader(t *testing.T, sourceIdentifier string, observations ...Observation) *strings.Reader {
	t.Helper()
	return strings.NewReader(observationJSONL(t, sourceIdentifier, observations...))
}

func TestImportObservationsArchivesLocalDiscordFactsWithoutProviderAPI(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	st := testutil.NewSQLiteTestStore(t)
	importer := NewImporter(st, nil)

	guild := Channel{ID: "300", GuildID: "200", Type: channelTypeGuildText, Name: "general"}
	thread := Channel{ID: "301", GuildID: "200", ParentID: guild.ID, Type: channelTypePublicThread, Name: "topic"}
	forum := Channel{ID: "302", GuildID: "200", Type: channelTypeGuildForum, Name: "ideas"}
	forumPost := Channel{ID: "303", GuildID: "200", ParentID: forum.ID, Type: channelTypePublicThread, Name: "proposal"}

	guildMessage := observedMessage("501", guild.ID, guild.GuildID, "guild alpha token")
	threadMessage := observedMessage("502", thread.ID, thread.GuildID, "thread needle token")
	threadMessage.Attachments = []Attachment{{
		ID:          "900",
		Filename:    "notes.txt",
		ContentType: "text/plain",
		Size:        42,
		URL:         "https://cdn.discordapp.com/attachments/301/900/notes.txt",
	}}
	forumMessage := observedMessage("506", forumPost.ID, forumPost.GuildID, "forum indigo token")

	initial := observationJSONL(
		t, "200",
		Observation{Version: 1, Kind: ObservationKindMessage, Channel: &guild, Message: &guildMessage},
		Observation{Version: 1, Kind: ObservationKindMessage, Channel: &thread, Message: &threadMessage},
		Observation{Version: 1, Kind: ObservationKindContainer, Channel: &forum},
		Observation{Version: 1, Kind: ObservationKindMessage, Channel: &forumPost, Message: &forumMessage},
	)

	first := importObservationsForTest(t, importer, "200", initial)
	assert.Equal(int64(3), first.MessagesAdded)
	assert.Equal(int64(4), first.ContainersProcessed)
	assert.Zero(first.SyncRunID, "local observations must not share bot-history checkpoint state")

	second := importObservationsForTest(t, NewImporter(st, nil), "200", initial)
	assert.Equal(int64(0), second.MessagesAdded)
	assert.Equal(int64(3), second.MessagesUpdated)

	source, err := st.GetOrCreateSource(sourceTypeDiscordLocal, "200")
	require.NoError(err)

	var messageCount int
	require.NoError(st.DB().QueryRow(
		st.Rebind("SELECT COUNT(*) FROM messages WHERE source_id = ?"),
		source.ID,
	).Scan(&messageCount))
	assert.Equal(3, messageCount)

	wantConversationTypes := map[string]string{
		guild.ID:     "channel",
		thread.ID:    "thread",
		forum.ID:     "channel",
		forumPost.ID: "thread",
	}
	for sourceConversationID, wantType := range wantConversationTypes {
		var got string
		require.NoError(st.DB().QueryRow(st.Rebind(
			"SELECT conversation_type FROM conversations WHERE source_id = ? AND source_conversation_id = ?",
		), source.ID, sourceConversationID).Scan(&got))
		assert.Equal(wantType, got, sourceConversationID)
	}

	var forumPostMetadata string
	require.NoError(st.DB().QueryRow(st.Rebind(
		"SELECT metadata FROM conversations WHERE source_id = ? AND source_conversation_id = ?",
	), source.ID, forumPost.ID).Scan(&forumPostMetadata))
	assert.Contains(forumPostMetadata, `"parent_channel_id":"302"`)

	results, total, err := st.SearchMessages("needle", 0, 10)
	require.NoError(err)
	assert.Equal(int64(1), total)
	if assert.Len(results, 1) {
		assert.Equal(threadMessage.ID, results[0].SourceMessageID)
	}

	var attachmentCount int
	require.NoError(st.DB().QueryRow(st.Rebind(`
		SELECT COUNT(*)
		FROM attachments a
		JOIN messages m ON m.id = a.message_id
		WHERE m.source_id = ? AND m.source_message_id = ?
		  AND a.source_attachment_id = ?
	`), source.ID, threadMessage.ID, "discord:900").Scan(&attachmentCount))
	assert.Equal(1, attachmentCount)

	guildMessage.Content = "guild cobalt-updated token"
	edited := time.Now().UTC()
	guildMessage.EditedTimestamp = &edited
	update := observationJSONL(
		t, "200",
		Observation{Version: 1, Kind: ObservationKindMessage, Channel: &guild, Message: &guildMessage},
	)
	importObservationsForTest(t, NewImporter(st, nil), "200", update)

	var missingStillActive int
	require.NoError(st.DB().QueryRow(st.Rebind(`
		SELECT COUNT(*) FROM messages
		WHERE source_id = ? AND source_message_id = ? AND deleted_from_source_at IS NULL
	`), source.ID, threadMessage.ID).Scan(&missingStillActive))
	assert.Equal(1, missingStillActive)

	results, total, err = st.SearchMessages("cobalt-updated", 0, 10)
	require.NoError(err)
	assert.Equal(int64(1), total)
	assert.Len(results, 1)

	var editedStored bool
	require.NoError(st.DB().QueryRow(st.Rebind(`
		SELECT is_edited FROM messages WHERE source_id = ? AND source_message_id = ?
	`), source.ID, guildMessage.ID).Scan(&editedStored))
	assert.True(editedStored)

	deletion := observationJSONL(
		t, "200",
		Observation{Version: 1, Kind: ObservationKindDelete, MessageID: guildMessage.ID},
	)
	importObservationsForTest(t, NewImporter(st, nil), "200", deletion)

	var deleted bool
	require.NoError(st.DB().QueryRow(st.Rebind(`
		SELECT deleted_from_source_at IS NOT NULL
		FROM messages WHERE source_id = ? AND source_message_id = ?
	`), source.ID, guildMessage.ID).Scan(&deleted))
	assert.True(deleted)

	_, total, err = st.SearchMessages("cobalt-updated", 0, 10)
	require.NoError(err)
	assert.Zero(total)

	assert.Nil(importer.api, "observation import must not have a Discord provider client")
}

func TestSDDDLE005AccountScopedDMAndGroupDMConversationTypes(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	st := testutil.NewSQLiteTestStore(t)
	dm := Channel{ID: "400", Type: channelTypeDM, Name: "Alice"}
	groupDM := Channel{ID: "500", Type: channelTypeGroupDM, Name: "Alice, Bob"}

	dmRoot := observedMessage("503", dm.ID, "", "direct violet token")
	dmReply := observedMessage("504", dm.ID, "", "reply orange token")
	dmReply.MessageReference = &MessageReference{MessageID: dmRoot.ID, ChannelID: dm.ID}
	groupMessage := observedMessage("505", groupDM.ID, "", "group emerald token")

	input := observationJSONL(
		t, "account:999",
		Observation{Version: 1, Kind: ObservationKindMessage, Channel: &dm, Message: &dmRoot},
		Observation{Version: 1, Kind: ObservationKindMessage, Channel: &dm, Message: &dmReply},
		Observation{Version: 1, Kind: ObservationKindMessage, Channel: &groupDM, Message: &groupMessage},
	)
	importObservationsForTest(t, NewImporter(st, nil), "account:999", input)

	source, err := st.GetOrCreateSource(sourceTypeDiscordLocal, "account:999")
	require.NoError(err)

	for sourceConversationID, wantType := range map[string]string{
		dm.ID:      "direct_chat",
		groupDM.ID: "group_chat",
	} {
		var got string
		require.NoError(st.DB().QueryRow(st.Rebind(
			"SELECT conversation_type FROM conversations WHERE source_id = ? AND source_conversation_id = ?",
		), source.ID, sourceConversationID).Scan(&got))
		assert.Equal(wantType, got)
	}

	var replyTarget string
	require.NoError(st.DB().QueryRow(st.Rebind(`
		SELECT target.source_message_id
		FROM messages reply
		JOIN messages target ON target.id = reply.reply_to_message_id
		WHERE reply.source_id = ? AND reply.source_message_id = ?
	`), source.ID, dmReply.ID).Scan(&replyTarget))
	assert.Equal(dmRoot.ID, replyTarget)
}

type failAfterChunkReader struct {
	chunk []byte
	done  bool
}

func (r *failAfterChunkReader) Read(p []byte) (int, error) {
	if !r.done {
		r.done = true
		return copy(p, r.chunk), nil
	}
	return 0, errors.New("synthetic interrupted observation stream")
}

func TestImportObservationsInterruptedReplayConverges(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	st := testutil.NewSQLiteTestStore(t)
	channel := Channel{ID: "600", GuildID: "701", Type: channelTypeGuildText, Name: "replay"}
	first := observedMessage("601", channel.ID, channel.GuildID, "first replay token")
	second := observedMessage("602", channel.ID, channel.GuildID, "second replay token")
	firstLine := observationJSONL(
		t, "701",
		Observation{Version: 1, Kind: ObservationKindMessage, Channel: &channel, Message: &first},
	)
	full := firstLine + observationJSONL(
		t, "701",
		Observation{Version: 1, Kind: ObservationKindMessage, Channel: &channel, Message: &second},
	)

	_, err := NewImporter(st, nil).ImportObservations(t.Context(), ObservationImportOptions{
		SourceIdentifier: "701",
		Reader:           &failAfterChunkReader{chunk: []byte(firstLine)},
	})
	require.ErrorContains(err, "synthetic interrupted observation stream")

	source, err := st.GetOrCreateSource(sourceTypeDiscordLocal, "701")
	require.NoError(err)

	var partialCount int
	require.NoError(st.DB().QueryRow(
		st.Rebind("SELECT COUNT(*) FROM messages WHERE source_id = ?"),
		source.ID,
	).Scan(&partialCount))
	assert.Equal(1, partialCount)

	_, err = NewImporter(st, nil).ImportObservations(t.Context(), ObservationImportOptions{
		SourceIdentifier: "701",
		Reader:           strings.NewReader(full),
	})
	require.NoError(err)

	var finalCount int
	require.NoError(st.DB().QueryRow(
		st.Rebind("SELECT COUNT(*) FROM messages WHERE source_id = ?"),
		source.ID,
	).Scan(&finalCount))
	assert.Equal(2, finalCount)

	hits, total, err := st.SearchMessages("replay", 0, 10)
	require.NoError(err)
	assert.Equal(int64(2), total)
	assert.Len(hits, 2)
}

func TestImportObservationsRejectsMalformedInput(t *testing.T) {
	st := testutil.NewSQLiteTestStore(t)
	importer := NewImporter(st, nil)

	for name, input := range map[string]string{
		"version": `{"version":2,"kind":"delete","source_type":"discord_local","source_identifier":"account:999","message_id":"1"}` + "\n",
		"kind":    `{"version":1,"kind":"unknown","source_type":"discord_local","source_identifier":"account:999"}` + "\n",
		"delete":  `{"version":1,"kind":"delete","source_type":"discord_local","source_identifier":"account:999"}` + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := importer.ImportObservations(t.Context(), ObservationImportOptions{
				SourceIdentifier: "account:999",
				Reader:           strings.NewReader(input),
			})
			require.Error(t, err)
		})
	}
}

var _ io.Reader = (*failAfterChunkReader)(nil)

func TestSDDDLE003004RejectsMissingOrMismatchedEmbeddedSourceIdentity(t *testing.T) {
	st := testutil.NewSQLiteTestStore(t)

	tests := []struct {
		name       string
		input      string
		wantErrSub string
	}{
		{
			name:       "missing source identity",
			input:      `{"version":1,"kind":"delete","message_id":"900"}` + "\n",
			wantErrSub: "source identity is required",
		},
		{
			name:       "native source type forbidden",
			input:      `{"version":1,"kind":"delete","source_type":"discord","source_identifier":"700","message_id":"900"}` + "\n",
			wantErrSub: "source_type",
		},
		{
			name:       "source identifier mismatch",
			input:      `{"version":1,"kind":"delete","source_type":"discord_local","source_identifier":"701","message_id":"900"}` + "\n",
			wantErrSub: "does not match import source",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewImporter(st, nil).ImportObservations(t.Context(), ObservationImportOptions{
				SourceIdentifier: "700",
				Reader:           strings.NewReader(tc.input),
			})
			require.ErrorContains(t, err, tc.wantErrSub)
		})
	}
}
