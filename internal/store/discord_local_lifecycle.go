package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// DiscordLocalVersion contains observed research metadata, never transport credentials.
type DiscordLocalVersion struct {
	BodyText       string `json:"body_text"`
	Metadata       string `json:"metadata"`
	SourceEditedAt string `json:"source_edited_at,omitempty"`
	ObservedAt     string `json:"observed_at"`
}

func (s *Store) insertDiscordLocalVersion(q querier, sourceID int64, messageID string, version DiscordLocalVersion) error {
	identity, err := json.Marshal([]string{version.BodyText, version.Metadata, version.SourceEditedAt})
	if err != nil {
		return err
	}
	hash := sha256.Sum256(identity)
	result, err := q.Exec(s.Rebind(`INSERT INTO discord_local_versions
 (source_id,source_message_id,version_hash,body_text,metadata,source_edited_at,observed_at)
 VALUES (?,?,?,?,?,?,?) ON CONFLICT (source_id,source_message_id,version_hash) DO NOTHING`),
		sourceID, messageID, hex.EncodeToString(hash[:]), version.BodyText, version.Metadata, version.SourceEditedAt, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return ErrDiscordLocalVersionReplayed
	}
	return err
}

// SDD-DLE-019: version and current message are committed in the same transaction.
func (s *Store) recordDiscordLocalVersion(q querier, data *MessagePersistData) error {
	msg := data.Message
	var kind string
	if err := q.QueryRow(s.Rebind(`SELECT source_type FROM sources WHERE id=?`), msg.SourceID).Scan(&kind); err != nil {
		return err
	}
	if kind != "discord_local" {
		return errors.New("observed history requires local Discord source")
	}
	var deleted int
	if err := q.QueryRow(s.Rebind(`SELECT COUNT(*) FROM discord_local_deletions WHERE source_id=? AND source_message_id=?`), msg.SourceID, msg.SourceMessageID).Scan(&deleted); err != nil {
		return err
	}
	if deleted != 0 {
		return ErrDiscordLocalDeleted
	}
	var count int
	if err := q.QueryRow(s.Rebind(`SELECT COUNT(*) FROM discord_local_versions WHERE source_id=? AND source_message_id=?`), msg.SourceID, msg.SourceMessageID).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		var body, metadata sql.NullString
		err := q.QueryRow(s.Rebind(`SELECT b.body_text,m.metadata FROM messages m LEFT JOIN message_bodies b ON b.message_id=m.id WHERE m.source_id=? AND m.source_message_id=?`), msg.SourceID, msg.SourceMessageID).Scan(&body, &metadata)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil {
			if err = s.insertDiscordLocalVersion(q, msg.SourceID, msg.SourceMessageID, DiscordLocalVersion{BodyText: body.String, Metadata: metadata.String}); err != nil && !errors.Is(err, ErrDiscordLocalVersionReplayed) {
				return err
			}
		}
	}
	version := *data.DiscordLocalVersion
	version.BodyText = data.BodyText.String
	return s.insertDiscordLocalVersion(q, msg.SourceID, msg.SourceMessageID, version)
}

// DiscordLocalVersions is an explicit local, read-only history seam.
func (s *Store) DiscordLocalVersions(sourceID int64, messageID string) ([]DiscordLocalVersion, error) {
	rows, err := s.db.Query(s.Rebind(`SELECT body_text,metadata,source_edited_at,observed_at FROM discord_local_versions WHERE source_id=? AND source_message_id=? ORDER BY observed_at,version_hash`), sourceID, messageID)
	if err != nil {
		return nil, fmt.Errorf("read local Discord history: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []DiscordLocalVersion{}
	for rows.Next() {
		var v DiscordLocalVersion
		if err := rows.Scan(&v.BodyText, &v.Metadata, &v.SourceEditedAt, &v.ObservedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ErrDiscordLocalVersionReplayed prevents known observation replay from rolling current state back.
var ErrDiscordLocalVersionReplayed = errors.New("local Discord version already observed")

// ErrDiscordLocalDeleted lets observation replay skip terminal provider identities.
var ErrDiscordLocalDeleted = errors.New("local Discord message is terminally deleted")

// MarkDiscordLocalDeleted atomically records provenance and removes all body representations.
func (s *Store) MarkDiscordLocalDeleted(ctx context.Context, sourceID int64, messageID string) error {
	return s.withTxContext(ctx, func(tx *loggedTx) error {
		q := boundQuerier{ctx: ctx, q: tx}
		var kind string
		if err := q.QueryRow(s.Rebind(`SELECT source_type FROM sources WHERE id=?`), sourceID).Scan(&kind); err != nil {
			return err
		}
		if kind != "discord_local" {
			return errors.New("terminal local delete requires local Discord source")
		}
		if _, err := q.Exec(s.Rebind(`INSERT INTO discord_local_deletions(source_id,source_message_id,deleted_at) VALUES(?,?,?) ON CONFLICT(source_id,source_message_id) DO NOTHING`), sourceID, messageID, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
		if _, err := q.Exec(s.Rebind(`DELETE FROM discord_local_versions WHERE source_id=? AND source_message_id=?`), sourceID, messageID); err != nil {
			return err
		}
		var id int64
		err := q.QueryRow(s.Rebind(`SELECT id FROM messages WHERE source_id=? AND source_message_id=?`), sourceID, messageID).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err = q.Exec(s.Rebind(`UPDATE messages SET deleted_from_source_at=COALESCE(deleted_from_source_at,?),subject=NULL,snippet=NULL WHERE id=?`), time.Now().UTC().Format(time.RFC3339Nano), id); err != nil {
			return err
		}
		for _, table := range []string{"message_bodies", "message_raw"} {
			if _, err = q.Exec(s.Rebind("DELETE FROM "+table+" WHERE message_id=?"), id); err != nil {
				return err
			}
		}
		if s.fts5Available {
			if err = s.dialect.InvalidateFTSForMessage(q, id); err != nil {
				return err
			}
		}
		return nil
	})
}

// DiscordLocalVersionsByIdentifier resolves only the local observation source.
func (s *Store) DiscordLocalVersionsByIdentifier(identifier, messageID string) ([]DiscordLocalVersion, error) {
	var sourceID int64
	err := s.db.QueryRow(s.Rebind("SELECT id FROM sources WHERE source_type='discord_local' AND identifier=?"), identifier).Scan(&sourceID)
	if errors.Is(err, sql.ErrNoRows) {
		return []DiscordLocalVersion{}, nil
	}
	if err != nil {
		return nil, err
	}
	return s.DiscordLocalVersions(sourceID, messageID)
}
