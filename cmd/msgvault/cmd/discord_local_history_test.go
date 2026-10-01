package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.kenn.io/msgvault/internal/config"
	"go.kenn.io/msgvault/internal/discord"
	"go.kenn.io/msgvault/internal/store"
)

func TestSDDDLE019ReadOnlyHistoryCommandExists(t *testing.T) {
	assert := assert.New(t)
	found := false
	for _, command := range rootCmd.Commands() {
		if command.Name() == "discord-message-history" {
			found = true
		}
	}
	assert.True(found)
}

func TestSDDDLE019ReadOnlyHistoryConsumer(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "msgvault.db"))
	require.NoError(err)
	require.NoError(st.InitSchema())
	message := discord.Message{ID: "900", ChannelID: "800", GuildID: "700", Content: "synthetic observed version",
		Author: discord.User{ID: "600", Username: "synthetic"}, Timestamp: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}
	channel := discord.Channel{ID: "800", GuildID: "700", Type: 0, Name: "synthetic"}
	observation := discord.Observation{Version: 1, Kind: discord.ObservationKindMessage, SourceType: "discord_local", SourceIdentifier: "700", Channel: &channel, Message: &message}
	var input bytes.Buffer
	require.NoError(json.NewEncoder(&input).Encode(observation))
	_, err = discord.NewImporter(st, nil).ImportObservations(t.Context(), discord.ObservationImportOptions{SourceIdentifier: "700", Reader: &input})
	require.NoError(err)
	require.NoError(st.Close())
	originalConfig := cfg
	cfg = &config.Config{Data: config.DataConfig{DataDir: dir}}
	t.Cleanup(func() { cfg = originalConfig })
	for _, tc := range []struct {
		source, message string
		count           int
	}{
		{"700", "900", 1}, {"700", "901", 0}, {"701", "900", 0},
	} {
		command := newDiscordLocalHistoryCmd()
		var output bytes.Buffer
		command.SetOut(&output)
		require.NoError(command.RunE(command, []string{tc.source, tc.message}))
		var versions []store.DiscordLocalVersion
		require.NoError(json.Unmarshal(output.Bytes(), &versions))
		require.Len(versions, tc.count)
	}
	// A true read-only consumer leaves no sync run or history mutation.
	check, err := store.OpenReadOnly(filepath.Join(dir, "msgvault.db"))
	require.NoError(err)
	defer func() { require.NoError(check.Close()) }()
	var count int
	require.NoError(check.DB().QueryRow("SELECT COUNT(*) FROM discord_local_versions").Scan(&count))
	assert.Equal(1, count)
	_, err = check.DB().Exec("DELETE FROM discord_local_versions")
	require.Error(err)
}
