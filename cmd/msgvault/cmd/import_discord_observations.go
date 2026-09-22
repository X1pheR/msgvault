package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.kenn.io/msgvault/internal/discord"
)

type importDiscordObservationsOptions struct {
	sourceIdentifier string
	displayName      string
}

func newImportDiscordObservationsCmd(deps discordCommandDeps) *cobra.Command {
	opts := importDiscordObservationsOptions{}

	cmd := &cobra.Command{
		Use:   "import-discord-observations <jsonl-file>",
		Short: "Import locally acquired Discord observations",
		Long: `Import a local JSONL stream of externally acquired Discord observations.

The importer performs no Discord network requests and requires no Discord
credential. Each line must be a versioned container, complete message snapshot,
or explicit delete observation. Missing messages are never inferred deleted.

The file path is resolved on the msgvault daemon host when the CLI is connected
to a daemon. Use a path mounted into the daemon/container.

Examples:
  msgvault import-discord-observations /imports/discord/snapshot.jsonl \
    --source account:123456789012345678

  msgvault import-discord-observations /imports/discord/events.jsonl \
    --source 113456789012345678 --display-name "Example Guild"
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isDaemonCLISubprocess() {
				return runDaemonCLICommandHTTPFromCobra(cmd, args)
			}
			return runImportDiscordObservations(cmd, deps, args[0], opts)
		},
	}

	cmd.Flags().StringVar(
		&opts.sourceIdentifier,
		"source",
		"",
		"stable msgvault Discord source identifier (required)",
	)
	cmd.Flags().StringVar(
		&opts.displayName,
		"display-name",
		"",
		"optional display name for the source",
	)
	_ = cmd.MarkFlagRequired("source")
	return cmd
}

func runImportDiscordObservations(
	cmd *cobra.Command,
	deps discordCommandDeps,
	inputPath string,
	opts importDiscordObservationsOptions,
) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open Discord observation file: %w", err)
	}
	defer func() { _ = file.Close() }()

	st, cleanup, err := deps.openStore()
	if err != nil {
		return err
	}
	defer cleanup()

	summary, importErr := discord.NewImporter(st, nil).ImportObservations(
		cmd.Context(),
		discord.ObservationImportOptions{
			SourceIdentifier:  opts.sourceIdentifier,
			SourceDisplayName: opts.displayName,
			Reader:            file,
		},
	)

	var postErr error
	if summary != nil && summary.SourceID != 0 && deps.postSourceMigrations != nil {
		postErr = deps.postSourceMigrations(st)
		if postErr != nil {
			postErr = fmt.Errorf("post-source-create migrations: %w", postErr)
		}
	}

	var cacheErr error
	if summary != nil &&
		(summary.MessagesProcessed > 0 || summary.ContainersProcessed > 0) &&
		deps.rebuildCache != nil {
		cacheErr = deps.rebuildCache(deps.databaseDSN())
	}

	if summary != nil {
		_, _ = fmt.Fprintf(
			cmd.OutOrStdout(),
			"Discord observation import: source=%d containers=%d added=%d updated=%d processed=%d\n",
			summary.SourceID,
			summary.ContainersProcessed,
			summary.MessagesAdded,
			summary.MessagesUpdated,
			summary.MessagesProcessed,
		)
	}

	return errors.Join(importErr, postErr, cacheErr)
}

func init() {
	rootCmd.AddCommand(newImportDiscordObservationsCmd(defaultDiscordCommandDeps()))
}
