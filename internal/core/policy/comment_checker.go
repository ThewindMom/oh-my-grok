// Package commentchecker detects AI-generated comments and returns whether
// they should be blocked based on the configured comment policy.
//
// AI-generated comments are patterns like:
//   - "Great function!"
//   - "This function does X"
//   - "TODO: implement this" (when added without implementation)
//   - Restating the code in English: "// Increment the counter"
//   - Praise comments: "// Beautiful implementation"
//
// This is an original implementation. It does not derive from any SUL-covered source.
package commentchecker

import (
	"regexp"
	"strings"
)

// Policy controls how comments are handled.
type Policy string

const (
	PolicyAllow Policy = "allow"
	PolicyWarn  Policy = "warn"
	PolicyDeny  Policy = "deny"
)

// Result represents the check result for a comment.
type Result struct {
	Line     int    `json:"line"`
	Comment  string `json:"comment"`
	Category string `json:"category"`
	Action   Policy `json:"action"`
	Reason   string `json:"reason"`
}

// aiCommentPatterns detects common AI-generated comment patterns.
var aiCommentPatterns = []struct {
	pattern  *regexp.Regexp
	category string
	reason   string
}{
	{
		regexp.MustCompile(`(?i)//\s*(great|awesome|beautiful|nice|perfect|excellent|wonderful|fantastic|amazing|brilliant|clever|elegant)\s`),
		"praise",
		"Praise comments add no value and signal AI generation",
	},
	{
		regexp.MustCompile(`(?i)//\s*(this|here)\s+(function|method|code|block|section)\s+\w+`),
		"restating",
		"Comment restates what the code does",
	},
	{
		regexp.MustCompile(`(?i)//\s*(this|the)\s+(function|method|class|variable|constant)\s+\w+`),
		"restating",
		"Comment restates the function's purpose",
	},
	{
		regexp.MustCompile(`(?i)//\s*(increment|decrement|add|remove|set|get|check|validate|update|delete|create|initialize|return|loop|iterate|process|handle|manage|calculate|compute|generate|build|construct|parse|format|convert|transform|apply|execute|run|start|stop)\s+(the|a|an)\s+\w+`),
		"restating",
		"Comment describes the action the code performs",
	},
	{
		regexp.MustCompile(`(?i)//\s*(note|note:|important|important:|warning|warning:|caution|caution:|remember|remember:|don't forget|do not forget)`),
		"filler",
		"Filler comment that doesn't add structural value",
	},
	{
		regexp.MustCompile(`(?i)//\s*(this\s+is|this\s+was|this\s+has\s+been)\s+(a\s+)?(great|good|nice|simple|clean|beautiful|elegant|clever)\s`),
		"praise",
		"Self-congratulatory comment",
	},
	{
		regexp.MustCompile(`(?i)//\s*(implemented|added|created|fixed|updated|modified|changed|refactored)\s+(by|for|to|from)\s`),
		"attribution",
		"AI attribution comment",
	},
	{
		regexp.MustCompile(`(?i)//\s*(ai|chatgpt|copilot|claude|gpt|generated|auto-generated|automatically\s+generated)`),
		"attribution",
		"AI generation attribution",
	},
	{
		regexp.MustCompile(`(?i)//\s*(this\s+is\s+a\s+(helper|utility|convenience|wrapper)\s+(function|method|class))`),
		"filler",
		"Filler comment describing the category",
	},
	{
		regexp.MustCompile(`(?i)//\s*(end\s+of|end\s+for|end\s+if|end\s+while|end\s+function|end\s+class|end\s+of\s+function|end\s+of\s+class|end\s+of\s+method|closing\s+bracket|close\s+bracket)`),
		"filler",
		"End-of-block comment adds no value",
	},
}

// CheckLine checks a single line for AI-generated comment patterns.
// lineNumber is 1-based. Returns nil if the line has no issues.
func CheckLine(lineNumber int, line string, policy Policy) *Result {
	// Extract comment from line
	comment := extractComment(line)
	if comment == "" {
		return nil
	}

	for _, p := range aiCommentPatterns {
		if p.pattern.MatchString(line) {
			return &Result{
				Line:     lineNumber,
				Comment:  comment,
				Category: p.category,
				Action:   policy,
				Reason:   p.reason,
			}
		}
	}

	return nil
}

// CheckContent checks all lines in a content string.
func CheckContent(content string, policy Policy) []Result {
	var results []Result
	if policy == PolicyAllow {
		return results
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if r := CheckLine(i+1, line, policy); r != nil {
			results = append(results, *r)
		}
	}
	return results
}

// ShouldDeny returns true if any result has a deny action.
func ShouldDeny(results []Result) bool {
	for _, r := range results {
		if r.Action == PolicyDeny {
			return true
		}
	}
	return false
}

// FormatResults returns a human-readable summary of check results.
func FormatResults(results []Result) string {
	if len(results) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("AI-generated comment patterns detected:\n")
	for _, r := range results {
		sb.WriteString("  line ")
		sb.WriteString(itoa(r.Line))
		sb.WriteString(" [")
		sb.WriteString(r.Category)
		sb.WriteString("]: ")
		sb.WriteString(r.Comment)
		sb.WriteString(" — ")
		sb.WriteString(r.Reason)
		sb.WriteString("\n")
	}
	return sb.String()
}

// extractComment extracts the comment portion from a line of code.
func extractComment(line string) string {
	// Line comment
	idx := strings.Index(line, "//")
	if idx >= 0 {
		return strings.TrimSpace(line[idx:])
	}
	// Block comment (single line)
	idx = strings.Index(line, "/*")
	if idx >= 0 {
		endIdx := strings.Index(line[idx:], "*/")
		if endIdx >= 0 {
			return strings.TrimSpace(line[idx : idx+endIdx+2])
		}
		return strings.TrimSpace(line[idx:])
	}
	// Hash comment (Python, Shell, etc.)
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "#!") {
		return trimmed
	}
	return ""
}

// itoa converts an int to string without importing strconv (to keep deps minimal).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}
