package cmd

import (
	"github.com/mihazs/oh-my-grok/internal/hookenv"
	"github.com/spf13/cobra"
)

func sessionStartCmd() *cobra.Command {
	return &cobra.Command{
		Use: "session-start",
		RunE: func(cmd *cobra.Command, args []string) error {
			ev, err := readEvent()
			if err != nil {
				return err
			}
			hookenv.ApplyEvent(ev)
			return nil
		},
	}
}