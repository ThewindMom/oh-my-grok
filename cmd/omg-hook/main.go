package main

import (
	"os"

	"github.com/mihazs/oh-my-grok/internal/hookenv"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{Use: "omg-hook"}
	root.AddCommand(&cobra.Command{
		Use: "session-start",
		RunE: func(cmd *cobra.Command, args []string) error {
			ev, err := hookenv.ReadEvent(os.Stdin)
			if err != nil {
				return err
			}
			hookenv.ApplyEvent(ev)
			return nil
		},
	})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}