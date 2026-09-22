package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportDiscordObservationsRoutesThroughDaemonCLIRunner(t *testing.T) {
	server, requests := newDaemonCLIRunnerTestServer(t, func(req daemonCLIRunTestRequest) {
		assert.Equal(t, []string{
			"import-discord-observations",
			"--display-name=Local Discord",
			"--source=account:999",
			"/imports/discord/events.jsonl",
		}, req.Args)
	}, `{"type":"complete"}`)
	configureRemoteDaemonForTest(t, server.URL)
	t.Setenv(daemonCLISubprocessEnv, "")

	cmd := newImportDiscordObservationsCmd(discordCommandDeps{})
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{
		"/imports/discord/events.jsonl",
		"--source", "account:999",
		"--display-name", "Local Discord",
	})
	require.NoError(t, cmd.Execute())
	assert.Equal(t, 1, int(requests.Load()))
}

func TestSDDDLE001NativeDiscordResolverExcludesLocalObservationSources(t *testing.T) {
	st := newDiscordCLIStore(t)
	_, err := st.GetOrCreateSource("discord_local", "700")
	require.NoError(t, err)

	_, err = resolveDiscordSources(st, "")
	require.ErrorContains(t, err, "no Discord guilds are registered")
}
