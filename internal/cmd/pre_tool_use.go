package cmd

import (
	"os"

	"github.com/mihazs/oh-my-grok/internal/hashline"
	"github.com/mihazs/oh-my-grok/internal/hookenv"
	"github.com/mihazs/oh-my-grok/internal/prometheus"
	"github.com/mihazs/oh-my-grok/internal/skillgate"
	"github.com/spf13/cobra"
)

func preToolUseCmd() *cobra.Command {
	return &cobra.Command{
		Use: "pre-tool-use",
		RunE: func(cmd *cobra.Command, args []string) error {
			ev, err := readEvent()
			if err != nil {
				return err
			}
			hookenv.ApplyEvent(ev)
			w := os.Stdout

			if reason := prometheus.DenyIfPlanMode(ev); reason != "" {
				denyPreTool(w, reason, "Prometheus plan mode: only .omg/**/*.md writes allowed.")
			}
			if reason := hashline.ValidatePreTool(ev); reason != "" {
				denyPreTool(w, reason, "Hashline: stale LINE#ID in edit; re-read the file.")
			}
			sid := sessionID(ev)
			if allow, reason := skillgate.PreTool(sid); allow {
				allowPreTool(w)
			} else {
				denyPreTool(w, reason, "Read at least one applicable skill before mutating files.")
			}
			return nil
		},
	}
}