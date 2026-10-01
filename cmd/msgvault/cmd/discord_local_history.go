package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"go.kenn.io/msgvault/internal/store"
)

// newDiscordLocalHistoryCmd opens only local read-only archive state.
func newDiscordLocalHistoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discord-message-history <source-identifier> <message-id>",
		Short: "Read locally observed Discord message versions",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := store.OpenReadOnly(cfg.DatabaseDSN())
			if err != nil {
				return fmt.Errorf("open local archive: %w", err)
			}
			defer st.Close()
			versions, err := st.DiscordLocalVersionsByIdentifier(args[0], args[1])
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(versions)
		},
	}
}

func init() { rootCmd.AddCommand(newDiscordLocalHistoryCmd()) }
