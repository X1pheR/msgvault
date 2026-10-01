package discord

import (
	"github.com/stretchr/testify/require"
	"go.kenn.io/msgvault/internal/store"
	"go.kenn.io/msgvault/internal/testutil"
	"path/filepath"
	"testing"
)

// SDD-DLE-022: existing archive backup and passive read-only restore use no provider.
func TestSDDDLE022PassiveArchiveRecovery(t *testing.T) {
	st := testutil.NewSQLiteTestStore(t)
	importer := NewImporter(st, nil)
	channel := Channel{ID: "800", GuildID: "700", Type: channelTypeGuildText, Name: "synthetic"}
	for _, body := range []string{"first recovery revision", "latest recovery revision"} {
		message := observedMessage("900", channel.ID, channel.GuildID, body)
		importObservationsForTest(t, importer, "700", observationJSONL(t, "700", Observation{Version: 1, Kind: ObservationKindMessage, Channel: &channel, Message: &message}))
	}
	deleted := observedMessage("901", channel.ID, channel.GuildID, "deleted recovery canary")
	importObservationsForTest(t, importer, "700", observationJSONL(t, "700", Observation{Version: 1, Kind: ObservationKindMessage, Channel: &channel, Message: &deleted}))
	importObservationsForTest(t, importer, "700", observationJSONL(t, "700", Observation{Version: 1, Kind: ObservationKindDelete, MessageID: "901"}))
	dst := filepath.Join(t.TempDir(), "archive.db")
	require.NoError(t, st.BackupDatabaseContext(t.Context(), dst))
	restored, err := store.OpenReadOnly(dst)
	require.NoError(t, err)
	defer restored.Close()
	var integrity string
	require.NoError(t, restored.DB().QueryRow("PRAGMA integrity_check").Scan(&integrity))
	require.Equal(t, "ok", integrity)
	for _, query := range []string{
		"SELECT GROUP_CONCAT(name||':'||COALESCE(sql,''),char(10)) FROM (SELECT name,sql FROM sqlite_master WHERE name NOT LIKE 'sqlite_%' ORDER BY name)",
		"SELECT COUNT(*) FROM messages",
		"SELECT COUNT(*) FROM message_bodies",
		"SELECT COUNT(*) FROM message_raw",
		"SELECT COUNT(*) FROM discord_local_versions",
		"SELECT COUNT(*) FROM discord_local_deletions",
	} {
		var original, recovered string
		require.NoError(t, st.DB().QueryRow(query).Scan(&original))
		require.NoError(t, restored.DB().QueryRow(query).Scan(&recovered))
		require.Equal(t, original, recovered)
	}
	original, err := st.DiscordLocalVersionsByIdentifier("700", "900")
	require.NoError(t, err)
	recovered, err := restored.DiscordLocalVersionsByIdentifier("700", "900")
	require.NoError(t, err)
	require.Equal(t, original, recovered)
	require.Len(t, recovered, 2)
	tombstone, err := restored.DiscordLocalVersionsByIdentifier("700", "901")
	require.NoError(t, err)
	require.Empty(t, tombstone)
	miss, err := restored.DiscordLocalVersionsByIdentifier("700", "999")
	require.NoError(t, err)
	require.Empty(t, miss)
	var body string
	require.NoError(t, restored.DB().QueryRow("SELECT body_text FROM message_bodies").Scan(&body))
	require.Equal(t, "latest recovery revision", body)
	_, err = restored.DB().Exec("DELETE FROM discord_local_versions")
	require.Error(t, err, "restored consumer must remain read-only")
}
