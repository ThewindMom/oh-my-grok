package hashline

import (
	"os"
	"strings"
)

// Enabled reports whether hashline guards are active (OMG_HASHLINE, default on).
func Enabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("OMG_HASHLINE"))) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}