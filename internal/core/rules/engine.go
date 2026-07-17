// Package rules implements a rule discovery and injection engine.
//
// It discovers rule files (AGENTS.md, .grok/rules/*.md, .agents/rules/*.md,
// .claude/rules/*.md) from the workspace root to the target file, parses
// their frontmatter, matches them against the target file path, and formats
// them for injection into the prompt context.
//
// Rules support frontmatter with:
//   - description: human-readable summary
//   - globs: glob patterns to match against the target file
//   - alwaysApply: inject regardless of target file
//
// This is an original Go implementation. It does not derive from any
// SUL-covered source. The architecture is inspired by the documented behavior
// of rule engines but implemented independently.
package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// MaxRuleChars is the maximum characters per rule body.
const MaxRuleChars = 8192

// MaxResultChars is the maximum total characters for all injected rules.
const MaxResultChars = 32768

// RuleSource identifies where a rule was discovered.
type RuleSource string

const (
	SourceAgentsMD       RuleSource = "AGENTS.md"
	SourceGroRules       RuleSource = ".grok/rules"
	SourceAgentsRules    RuleSource = ".agents/rules"
	SourceClaudeRules    RuleSource = ".claude/rules"
	SourceCursorRules    RuleSource = ".cursor/rules"
	SourceGithubInstr    RuleSource = ".github/instructions"
	SourceCopilotInstr   RuleSource = ".github/copilot-instructions.md"
	SourcePluginBundled  RuleSource = "plugin-bundled"
)

// Frontmatter represents parsed YAML frontmatter from a rule file.
type Frontmatter struct {
	Description   string   `json:"description,omitempty"`
	Globs         []string `json:"globs,omitempty"`
	AlwaysApply   bool     `json:"alwaysApply,omitempty"`
}

// ParsedRule is a rule file that has been parsed.
type ParsedRule struct {
	Path         string      `json:"path"`
	RealPath     string      `json:"realPath"`
	Source       RuleSource  `json:"source"`
	Distance     int         `json:"distance"`
	IsGlobal     bool        `json:"isGlobal"`
	RelativePath string      `json:"relativePath"`
	Frontmatter  Frontmatter `json:"frontmatter"`
	Body         string      `json:"body"`
	ContentHash  string      `json:"contentHash"`
}

// LoadedRule is a parsed rule that has been matched against a target.
type LoadedRule struct {
	ParsedRule
	MatchReason string `json:"matchReason"`
}

// Diagnostic reports an issue during rule loading.
type Diagnostic struct {
	Severity string `json:"severity"` // "warning" or "error"
	Source   string `json:"source"`
	Message  string `json:"message"`
}

// LoadResult contains the rules and diagnostics from a load operation.
type LoadResult struct {
	Rules       []LoadedRule `json:"rules"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Engine is the rule discovery and matching engine.
type Engine struct {
	maxRuleChars   int
	maxResultChars int
}

// NewEngine creates a new rules engine with the given limits.
func NewEngine() *Engine {
	return &Engine{
		maxRuleChars:   MaxRuleChars,
		maxResultChars: MaxResultChars,
	}
}

// fmRegex matches YAML frontmatter delimiters.
var fmRegex = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n(.*)$`)

// DiscoverAndLoad finds all rule files from workspace root to targetPath,
// parses them, matches them against the target, and returns the loaded rules.
func (e *Engine) DiscoverAndLoad(workspaceRoot, targetPath string) LoadResult {
	result := LoadResult{}

	if workspaceRoot == "" {
		return result
	}

	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "error", Source: "engine", Message: err.Error(),
		})
		return result
	}
	absRoot = filepath.Clean(absRoot)

	absTarget := absRoot
	if targetPath != "" {
		absTarget, err = filepath.Abs(targetPath)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "error", Source: "engine", Message: err.Error(),
			})
			return result
		}
		absTarget = filepath.Clean(absTarget)
	}

	// Verify containment
	if !strings.HasPrefix(absTarget+string(filepath.Separator), absRoot+string(filepath.Separator)) && absTarget != absRoot {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "error", Source: "engine",
			Message:  fmt.Sprintf("target %s escapes workspace %s", targetPath, workspaceRoot),
		})
		return result
	}

	// Discover rule files
	candidates := discoverCandidates(absRoot, absTarget)

	// Parse and match
	var rules []LoadedRule
	seen := map[string]bool{} // dedup by content hash

	for _, c := range candidates {
		parsed, err := parseRuleFile(c)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: "warning", Source: string(c.source), Message: err.Error(),
			})
			continue
		}

		// Truncate body if needed
		if len(parsed.Body) > e.maxRuleChars {
			parsed.Body = truncateAtLine(parsed.Body, e.maxRuleChars) + "\n... (truncated)\n"
		}

		// Compute content hash
		parsed.ContentHash = hashContent(parsed.Body)

		// Deduplicate
		if seen[parsed.ContentHash] {
			continue
		}
		seen[parsed.ContentHash] = true

		// Match against target
		relTarget, _ := filepath.Rel(absRoot, absTarget)
		matchReason := matchRule(parsed, relTarget)
		if matchReason == "no-match" {
			continue
		}

		rules = append(rules, LoadedRule{
			ParsedRule:  parsed,
			MatchReason: matchReason,
		})
	}

	// Sort by distance (nearest first)
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Distance < rules[j].Distance
	})

	result.Rules = rules
	return result
}

// Format formats loaded rules into a single injection string.
func (e *Engine) Format(rules []LoadedRule) string {
	if len(rules) == 0 {
		return ""
	}

	var sb strings.Builder
	totalChars := 0

	for _, r := range rules {
		if totalChars >= e.maxResultChars {
			break
		}

		remaining := e.maxResultChars - totalChars
		body := r.Body
		if len(body) > remaining {
			body = truncateAtLine(body, remaining) + "\n... (truncated)\n"
		}

		sb.WriteString(fmt.Sprintf("=== %s (%s) ===\n", r.RelativePath, r.MatchReason))
		sb.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		totalChars += len(body)
	}

	return sb.String()
}

// LoadAndFormat is a convenience method that discovers, loads, and formats.
func (e *Engine) LoadAndFormat(workspaceRoot, targetPath string) (string, []Diagnostic) {
	result := e.DiscoverAndLoad(workspaceRoot, targetPath)
	return e.Format(result.Rules), result.Diagnostics
}

// candidate represents a discovered rule file before parsing.
type candidate struct {
	path         string
	realPath     string
	source       RuleSource
	distance     int
	isGlobal     bool
	relativePath string
}

// discoverCandidates walks from targetPath up to workspaceRoot, collecting
// rule files from various sources.
func discoverCandidates(absRoot, absTarget string) []candidate {
	var candidates []candidate
	dir := absTarget
	distance := 0

	for {
		// AGENTS.md
		if addCandidate(&candidates, filepath.Join(dir, "AGENTS.md"), SourceAgentsMD, distance, absRoot) {
		}
		// .grok/AGENTS.md
		if addCandidate(&candidates, filepath.Join(dir, ".grok", "AGENTS.md"), SourceAgentsMD, distance, absRoot) {
		}
		// .agents/AGENTS.md
		if addCandidate(&candidates, filepath.Join(dir, ".agents", "AGENTS.md"), SourceAgentsMD, distance, absRoot) {
		}
		// .grok/rules/*.md
		addDirCandidates(&candidates, filepath.Join(dir, ".grok", "rules"), SourceGroRules, distance, absRoot)
		// .agents/rules/*.md
		addDirCandidates(&candidates, filepath.Join(dir, ".agents", "rules"), SourceAgentsRules, distance, absRoot)
		// .claude/rules/*.md
		addDirCandidates(&candidates, filepath.Join(dir, ".claude", "rules"), SourceClaudeRules, distance, absRoot)
		// .cursor/rules/*.md
		addDirCandidates(&candidates, filepath.Join(dir, ".cursor", "rules"), SourceCursorRules, distance, absRoot)
		// .github/copilot-instructions.md
		addCandidate(&candidates, filepath.Join(dir, ".github", "copilot-instructions.md"), SourceCopilotInstr, distance, absRoot)

		if dir == absRoot {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
		distance++
	}

	return candidates
}

func addCandidate(candidates *[]candidate, path string, source RuleSource, distance int, absRoot string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	realPath, _ := filepath.EvalSymlinks(path)
	if realPath == "" {
		realPath = path
	}
	relPath, _ := filepath.Rel(absRoot, path)
	*candidates = append(*candidates, candidate{
		path:         path,
		realPath:     realPath,
		source:       source,
		distance:     distance,
		isGlobal:     false,
		relativePath: relPath,
	})
	return true
}

func addDirCandidates(candidates *[]candidate, dir string, source RuleSource, distance int, absRoot string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		addCandidate(candidates, path, source, distance, absRoot)
	}
}

// parseRuleFile reads and parses a rule file's frontmatter and body.
func parseRuleFile(c candidate) (ParsedRule, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return ParsedRule{}, err
	}

	content := string(data)
	parsed := ParsedRule{
		Path:         c.path,
		RealPath:     c.realPath,
		Source:       c.source,
		Distance:     c.distance,
		IsGlobal:     c.isGlobal,
		RelativePath: c.relativePath,
	}

	matches := fmRegex.FindStringSubmatch(content)
	if matches == nil {
		// No frontmatter — treat entire file as body with alwaysApply
		parsed.Body = strings.TrimSpace(content)
		parsed.Frontmatter = Frontmatter{AlwaysApply: true}
		return parsed, nil
	}

	fmText := matches[1]
	body := matches[2]
	parsed.Body = strings.TrimSpace(body)
	parsed.Frontmatter = parseFrontmatter(fmText)

	return parsed, nil
}

// parseFrontmatter parses simple YAML frontmatter.
func parseFrontmatter(fm string) Frontmatter {
	result := Frontmatter{}
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, "\"'")

		switch key {
		case "description":
			result.Description = val
		case "globs", "paths", "applyTo":
			// Parse as list or comma-separated
			val = strings.Trim(val, "[]")
			for _, g := range strings.Split(val, ",") {
				g = strings.TrimSpace(g)
				g = strings.Trim(g, "\"'")
				if g != "" {
					result.Globs = append(result.Globs, g)
				}
			}
		case "alwaysApply", "always_apply":
			result.AlwaysApply = val == "true" || val == "True"
		}
	}
	return result
}

// matchRule determines if a rule applies to the target file.
func matchRule(rule ParsedRule, relTarget string) string {
	if rule.Frontmatter.AlwaysApply {
		return "alwaysApply"
	}

	if len(rule.Frontmatter.Globs) == 0 {
		// No globs and not alwaysApply — match if in same directory or above
		return "scope"
	}

	for _, pattern := range rule.Frontmatter.Globs {
		if matchGlob(pattern, relTarget) {
			return "glob:" + pattern
		}
	}

	return "no-match"
}

// matchGlob matches a glob pattern against a path.
func matchGlob(pattern, path string) bool {
	// Simple glob matching: * matches any chars except /, ** matches everything
	regexPattern := globToRegex(pattern)
	matched, err := regexp.MatchString(regexPattern, path)
	if err != nil {
		return false
	}
	return matched
}

// globToRegex converts a glob pattern to a regex string.
// `*` matches any characters except `/`, `**` matches everything including `/`.
func globToRegex(glob string) string {
	var sb strings.Builder
	sb.WriteString("^")
	runes := []rune(glob)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch c {
		case '*':
			if i+1 < len(runes) && runes[i+1] == '*' {
				// ** matches everything
				sb.WriteString(".*")
				i++ // skip the second *
			} else {
				// * matches any chars except /
				sb.WriteString("[^/]*")
			}
		case '?':
			sb.WriteString("[^/]")
		case '.', '+', '(', ')', '{', '}', '|', '^', '$':
			sb.WriteString("\\")
			sb.WriteRune(c)
		default:
			sb.WriteRune(c)
		}
	}
	sb.WriteString("$")
	return sb.String()
}

// truncateAtLine truncates text at a line boundary near the limit.
func truncateAtLine(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	cutoff := maxChars
	for cutoff > 0 && text[cutoff-1] != '\n' {
		cutoff--
	}
	if cutoff == 0 {
		cutoff = maxChars
	}
	return text[:cutoff]
}

// hashContent computes a short hash of content for deduplication.
func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])[:16]
}
