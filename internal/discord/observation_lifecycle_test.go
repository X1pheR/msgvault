package discord

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.kenn.io/msgvault/internal/testutil"
	"testing"
)

func TestSDDDLE019ObservedHistoryReplay(t *testing.T) {
	st := testutil.NewSQLiteTestStore(t)
	importer := NewImporter(st, nil)
	channel := Channel{ID: "800", GuildID: "700", Type: channelTypeGuildText, Name: "synthetic"}
	for _, body := range []string{"first revision", "edited revision", "edited revision"} {
		message := observedMessage("900", channel.ID, channel.GuildID, body)
		importObservationsForTest(t, importer, "700", observationJSONL(t, "700", Observation{Version: 1, Kind: ObservationKindMessage, Channel: &channel, Message: &message}))
	}
	var exists int
	require.NoError(t, st.DB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='discord_local_versions'").Scan(&exists))
	require.Equal(t, 1, exists, "SDD-DLE-019 observed edit history is missing")
	var count int
	require.NoError(t, st.DB().QueryRow("SELECT COUNT(*) FROM discord_local_versions WHERE source_message_id='900'").Scan(&count))
	assert.Equal(t, 2, count, "unchanged replay must not duplicate versions")
	var body string
	require.NoError(t, st.DB().QueryRow("SELECT body_text FROM discord_local_versions WHERE source_message_id='900' ORDER BY observed_at, rowid LIMIT 1").Scan(&body))
	assert.Equal(t, "first revision", body)
}

func TestSDDDLE020DeleteScrubsBodyAndTerminalReplay(t *testing.T) {
	st := testutil.NewSQLiteTestStore(t)
	importer := NewImporter(st, nil)
	channel := Channel{ID: "800", GuildID: "700", Type: channelTypeGuildText, Name: "synthetic"}
	message := observedMessage("900", channel.ID, channel.GuildID, "deleted private canary")
	input := observationJSONL(t, "700", Observation{Version: 1, Kind: ObservationKindMessage, Channel: &channel, Message: &message})
	importObservationsForTest(t, importer, "700", input)
	importObservationsForTest(t, importer, "700", observationJSONL(t, "700", Observation{Version: 1, Kind: ObservationKindDelete, MessageID: "900"}))
	var bodyCount, rawCount, historyCount int
	require.NoError(t, st.DB().QueryRow("SELECT COUNT(*) FROM message_bodies WHERE COALESCE(body_text,'')<>''").Scan(&bodyCount))
	assert.Zero(t, bodyCount, "deleted body must not be retained")
	require.NoError(t, st.DB().QueryRow("SELECT COUNT(*) FROM message_raw").Scan(&rawCount))
	assert.Zero(t, rawCount, "raw body must be removed")
	require.NoError(t, st.DB().QueryRow("SELECT COUNT(*) FROM discord_local_versions").Scan(&historyCount))
	assert.Zero(t, historyCount, "prior bodies must be removed")
	importObservationsForTest(t, importer, "700", input)
	var resurrected int
	require.NoError(t, st.DB().QueryRow("SELECT COUNT(*) FROM messages WHERE deleted_from_source_at IS NULL").Scan(&resurrected))
	assert.Zero(t, resurrected, "stale replay must not resurrect")
}

func TestSDDDLE020UnknownDeleteAndSourceIsolation(t *testing.T) {
	st := testutil.NewSQLiteTestStore(t)
	importer := NewImporter(st, nil)
	deletion := observationJSONL(t, "700", Observation{Version: 1, Kind: ObservationKindDelete, MessageID: "900"})
	importObservationsForTest(t, importer, "700", deletion)
	importObservationsForTest(t, importer, "700", deletion)
	var count int
	require.NoError(t, st.DB().QueryRow("SELECT COUNT(*) FROM discord_local_deletions").Scan(&count))
	assert.Equal(t, 1, count)
	channel := Channel{ID: "800", GuildID: "700", Type: channelTypeGuildText, Name: "synthetic"}
	message := observedMessage("900", channel.ID, channel.GuildID, "stale")
	importObservationsForTest(t, importer, "700", observationJSONL(t, "700", Observation{Version: 1, Kind: ObservationKindMessage, Channel: &channel, Message: &message}))
	channel.GuildID = "701"
	message.GuildID = "701"
	importObservationsForTest(t, importer, "701", observationJSONL(t, "701", Observation{Version: 1, Kind: ObservationKindMessage, Channel: &channel, Message: &message}))
	require.NoError(t, st.DB().QueryRow("SELECT COUNT(*) FROM messages").Scan(&count))
	assert.Equal(t, 1, count, "another source must remain independent")
}

func TestSDDDLE020ScrubFailureRollsBack(t *testing.T) {
	st := testutil.NewSQLiteTestStore(t)
	importer := NewImporter(st, nil)
	channel := Channel{ID: "800", GuildID: "700", Type: channelTypeGuildText, Name: "synthetic"}
	message := observedMessage("900", channel.ID, channel.GuildID, "preserve on failed scrub")
	importObservationsForTest(t, importer, "700", observationJSONL(t, "700", Observation{Version: 1, Kind: ObservationKindMessage, Channel: &channel, Message: &message}))
	_, err := st.DB().Exec("CREATE TRIGGER fail_delete BEFORE DELETE ON message_raw BEGIN SELECT RAISE(ABORT,'synthetic_failure'); END")
	require.NoError(t, err)
	_, err = importer.ImportObservations(t.Context(), ObservationImportOptions{SourceIdentifier: "700", Reader: observationReader(t, "700", Observation{Version: 1, Kind: ObservationKindDelete, MessageID: "900"})})
	require.Error(t, err)
	var count int
	require.NoError(t, st.DB().QueryRow("SELECT COUNT(*) FROM discord_local_deletions").Scan(&count))
	assert.Zero(t, count)
	require.NoError(t, st.DB().QueryRow("SELECT COUNT(*) FROM messages WHERE deleted_from_source_at IS NULL").Scan(&count))
	assert.Equal(t, 1, count)
	require.NoError(t, st.DB().QueryRow("SELECT COUNT(*) FROM discord_local_versions").Scan(&count))
	assert.Equal(t, 1, count)
	var body string
	require.NoError(t, st.DB().QueryRow("SELECT body_text FROM message_bodies").Scan(&body))
	assert.Equal(t, message.Content, body)
}

func TestSDDDLE019ReadOnlyHistoryAndMiss(t *testing.T) {
	st := testutil.NewSQLiteTestStore(t)
	channel := Channel{ID: "800", GuildID: "700", Type: channelTypeGuildText, Name: "synthetic"}
	message := observedMessage("900", channel.ID, channel.GuildID, "readable version")
	summary := importObservationsForTest(t, NewImporter(st, nil), "700", observationJSONL(t, "700", Observation{Version: 1, Kind: ObservationKindMessage, Channel: &channel, Message: &message}))
	versions, err := st.DiscordLocalVersions(summary.SourceID, "900")
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, message.Content, versions[0].BodyText)
	versions, err = st.DiscordLocalVersions(summary.SourceID, "901")
	require.NoError(t, err)
	assert.Empty(t, versions)
}

func TestSDDDLE017019OldVersionReplayKeepsCurrent(t *testing.T) {
	st := testutil.NewSQLiteTestStore(t)
	channel := Channel{ID: "800", GuildID: "700", Type: channelTypeGuildText, Name: "synthetic"}
	importer := NewImporter(st, nil)
	first := observedMessage("900", channel.ID, channel.GuildID, "first")
	latest := first
	latest.Content = "latest"
	for _, message := range []Message{first, latest, first} {
		m := message
		importObservationsForTest(t, importer, "700", observationJSONL(t, "700", Observation{Version: 1, Kind: ObservationKindMessage, Channel: &channel, Message: &m}))
	}
	var body string
	require.NoError(t, st.DB().QueryRow("SELECT body_text FROM message_bodies").Scan(&body))
	assert.Equal(t, "latest", body, "old observed replay must not roll back current content")
}
