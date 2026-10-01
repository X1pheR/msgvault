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
	require := require.New(t)
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
	require.NoError(st.BackupDatabaseContext(t.Context(), dst))
	restored, err := store.OpenReadOnly(dst)
	require.NoError(err)
	defer restored.Close()
	var integrity string
	require.NoError(restored.DB().QueryRow("PRAGMA integrity_check").Scan(&integrity))
	require.Equal("ok", integrity)
	for _, query := range []string{
		"SELECT GROUP_CONCAT(name||':'||COALESCE(sql,''),char(10)) FROM (SELECT name,sql FROM sqlite_master WHERE name NOT LIKE 'sqlite_%' ORDER BY name)",
		"SELECT COUNT(*) FROM messages",
		"SELECT COUNT(*) FROM message_bodies",
		"SELECT COUNT(*) FROM message_raw",
		"SELECT COUNT(*) FROM discord_local_versions",
		"SELECT COUNT(*) FROM discord_local_deletions",
	} {
		var original, recovered string
		require.NoError(st.DB().QueryRow(query).Scan(&original))
		require.NoError(restored.DB().QueryRow(query).Scan(&recovered))
		require.Equal(original, recovered)
	}
	original, err := st.DiscordLocalVersionsByIdentifier("700", "900")
	require.NoError(err)
	recovered, err := restored.DiscordLocalVersionsByIdentifier("700", "900")
	require.NoError(err)
	require.Equal(original, recovered)
	require.Len(recovered, 2)
	tombstone, err := restored.DiscordLocalVersionsByIdentifier("700", "901")
	require.NoError(err)
	require.Empty(tombstone)
	miss, err := restored.DiscordLocalVersionsByIdentifier("700", "999")
	require.NoError(err)
	require.Empty(miss)
	var body string
	require.NoError(restored.DB().QueryRow("SELECT body_text FROM message_bodies").Scan(&body))
	require.Equal("latest recovery revision", body)
	_, err = restored.DB().Exec("DELETE FROM discord_local_versions")
	require.Error(err, "restored consumer must remain read-only")
}
