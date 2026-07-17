// Package delegate provides smart subagent delegation with retry guidance
// and model selection logic.
//
// This is an original implementation. It does not derive from any SUL-covered source.
package delegate

import (
	"fmt"
	"strings"
)

// RetryStrategy defines how to handle subagent failures.
type RetryStrategy string

const (
	RetryNone    RetryStrategy = "none"
	RetryOnce    RetryStrategy = "once"
	RetryTwice   RetryStrategy = "twice"
	RetryEscalate RetryStrategy = "escalate"
)

// DelegationAdvice provides guidance for delegating a task to a subagent.
type DelegationAdvice struct {
	AgentType       string        `json:"agentType"`
	CapabilityMode  string        `json:"capabilityMode"`
	RetryStrategy   RetryStrategy `json:"retryStrategy"`
	ModelGuidance   string        `json:"modelGuidance,omitempty"`
	Isolation       string        `json:"isolation,omitempty"`
	Reason          string        `json:"reason"`
}

// ClassifyTask determines the best delegation strategy for a task.
func ClassifyTask(taskDescription string) DelegationAdvice {
	desc := strings.ToLower(taskDescription)

	// Review tasks (check before implementation since "review changes" contains "change")
	if containsAny(desc, "review", "check", "verify", "audit", "inspect") {
		return DelegationAdvice{
			AgentType:      "momus",
			CapabilityMode: "read-only",
			RetryStrategy:  RetryNone,
			Reason:         "Review task — delegate to Momus with read-only access",
		}
	}

	// Implementation tasks
	if containsAny(desc, "implement", "build", "create", "add", "fix", "refactor", "modify", "update", "change") {
		return DelegationAdvice{
			AgentType:      "hephaestus",
			CapabilityMode: "read-write",
			RetryStrategy:  RetryOnce,
			Reason:         "Implementation task — delegate to Hephaestus with read-write access",
		}
	}

	// Planning tasks
	if containsAny(desc, "plan", "design", "architect", "strategy", "approach") {
		return DelegationAdvice{
			AgentType:      "prometheus",
			CapabilityMode: "read-only",
			RetryStrategy:  RetryNone,
			Reason:         "Planning task — delegate to Prometheus with read-only access",
		}
	}

	// Research tasks
	if containsAny(desc, "research", "find", "search", "investigate", "explore", "look up") {
		return DelegationAdvice{
			AgentType:      "explore",
			CapabilityMode: "read-only",
			RetryStrategy:  RetryOnce,
			Reason:         "Research task — delegate to Explore with read-only access",
		}
	}

	// Debugging tasks
	if containsAny(desc, "debug", "diagnose", "troubleshoot", "investigate bug", "root cause") {
		return DelegationAdvice{
			AgentType:      "oracle",
			CapabilityMode: "read-only",
			RetryStrategy:  RetryTwice,
			Reason:         "Debugging task — delegate to Oracle with read-only access and retry",
		}
	}

	// Gap analysis
	if containsAny(desc, "gap analysis", "missing requirements", "assumptions", "migration issues") {
		return DelegationAdvice{
			AgentType:      "metis",
			CapabilityMode: "read-only",
			RetryStrategy:  RetryNone,
			Reason:         "Gap analysis task — delegate to Metis",
		}
	}

	// External research
	if containsAny(desc, "documentation", "api reference", "library version", "upstream") {
		return DelegationAdvice{
			AgentType:      "librarian",
			CapabilityMode: "read-only",
			RetryStrategy:  RetryOnce,
			Reason:         "External research task — delegate to Librarian",
		}
	}

	// Default: general purpose
	return DelegationAdvice{
		AgentType:      "general-purpose",
		CapabilityMode: "all",
		RetryStrategy:  RetryOnce,
		Reason:         "General task — delegate to general-purpose agent",
	}
}

// RetryGuidance provides advice on whether to retry a failed subagent.
type RetryGuidance struct {
	ShouldRetry bool   `json:"shouldRetry"`
	Strategy    RetryStrategy `json:"strategy"`
	Attempt     int    `json:"attempt"`
	MaxAttempts int    `json:"maxAttempts"`
	Reason      string `json:"reason"`
}

// ShouldRetry determines whether a failed subagent should be retried.
func ShouldRetry(strategy RetryStrategy, attempt int, failureReason string) RetryGuidance {
	maxAttempts := 0
	switch strategy {
	case RetryNone:
		maxAttempts = 0
	case RetryOnce:
		maxAttempts = 1
	case RetryTwice:
		maxAttempts = 2
	case RetryEscalate:
		maxAttempts = 3
	}

	if attempt >= maxAttempts {
		return RetryGuidance{
			ShouldRetry: false,
			Strategy:    strategy,
			Attempt:     attempt,
			MaxAttempts: maxAttempts,
			Reason:      fmt.Sprintf("max retries (%d) reached for strategy %s: %s", maxAttempts, strategy, failureReason),
		}
	}

	// Don't retry on certain failures
	lowerReason := strings.ToLower(failureReason)
	if containsAny(lowerReason, "permission denied", "unauthorized", "forbidden", "not found") {
		return RetryGuidance{
			ShouldRetry: false,
			Strategy:    strategy,
			Attempt:     attempt,
			MaxAttempts: maxAttempts,
			Reason:      fmt.Sprintf("non-retryable failure: %s", failureReason),
		}
	}

	return RetryGuidance{
		ShouldRetry: true,
		Strategy:    strategy,
		Attempt:     attempt + 1,
		MaxAttempts: maxAttempts,
		Reason:      fmt.Sprintf("retrying (attempt %d/%d): %s", attempt+1, maxAttempts, failureReason),
	}
}

// ModelGuidance provides guidance on model selection without hard-coding model names.
type ModelGuidance struct {
	Recommendation string `json:"recommendation"`
	Reason         string `json:"reason"`
}

// GetModelGuidance returns model selection guidance for a task type.
func GetModelGuidance(agentType string) ModelGuidance {
	switch agentType {
	case "hephaestus":
		return ModelGuidance{
			Recommendation: "inherit parent model or use a coding-capable model",
			Reason:         "Implementation requires strong code generation",
		}
	case "prometheus":
		return ModelGuidance{
			Recommendation: "inherit parent model or use a reasoning-capable model",
			Reason:         "Planning requires strategic thinking",
		}
	case "momus":
		return ModelGuidance{
			Recommendation: "inherit parent model or use a reasoning-capable model",
			Reason:         "Review requires critical analysis",
		}
	case "oracle":
		return ModelGuidance{
			Recommendation: "use the strongest available reasoning model",
			Reason:         "Architecture and debugging require deep reasoning",
		}
	default:
		return ModelGuidance{
			Recommendation: "inherit parent model",
			Reason:         "Default: inherit the parent session's model",
		}
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
