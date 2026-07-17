package delegate

import (
	"testing"
)

func TestClassifyTaskImplementation(t *testing.T) {
	advice := ClassifyTask("implement a new hash function")
	if advice.AgentType != "hephaestus" {
		t.Errorf("agentType = %s, want hephaestus", advice.AgentType)
	}
	if advice.CapabilityMode != "read-write" {
		t.Errorf("capabilityMode = %s, want read-write", advice.CapabilityMode)
	}
}

func TestClassifyTaskPlanning(t *testing.T) {
	advice := ClassifyTask("plan the architecture for the new system")
	if advice.AgentType != "prometheus" {
		t.Errorf("agentType = %s, want prometheus", advice.AgentType)
	}
}

func TestClassifyTaskReview(t *testing.T) {
	advice := ClassifyTask("review the code changes")
	if advice.AgentType != "momus" {
		t.Errorf("agentType = %s, want momus", advice.AgentType)
	}
}

func TestClassifyTaskResearch(t *testing.T) {
	advice := ClassifyTask("search for all callers of this function")
	if advice.AgentType != "explore" {
		t.Errorf("agentType = %s, want explore", advice.AgentType)
	}
}

func TestClassifyTaskDebugging(t *testing.T) {
	advice := ClassifyTask("debug the race condition in the state package")
	if advice.AgentType != "oracle" {
		t.Errorf("agentType = %s, want oracle", advice.AgentType)
	}
}

func TestClassifyTaskDefault(t *testing.T) {
	advice := ClassifyTask("do something unspecified")
	if advice.AgentType != "general-purpose" {
		t.Errorf("agentType = %s, want general-purpose", advice.AgentType)
	}
}

func TestShouldRetryNone(t *testing.T) {
	g := ShouldRetry(RetryNone, 0, "timeout")
	if g.ShouldRetry {
		t.Error("should not retry with RetryNone")
	}
}

func TestShouldRetryOnce(t *testing.T) {
	g := ShouldRetry(RetryOnce, 0, "timeout")
	if !g.ShouldRetry {
		t.Error("should retry on first attempt with RetryOnce")
	}

	g = ShouldRetry(RetryOnce, 1, "timeout")
	if g.ShouldRetry {
		t.Error("should not retry after max attempts with RetryOnce")
	}
}

func TestShouldRetryNonRetryable(t *testing.T) {
	g := ShouldRetry(RetryTwice, 0, "permission denied")
	if g.ShouldRetry {
		t.Error("should not retry on permission denied")
	}
}

func TestShouldRetryNotFound(t *testing.T) {
	g := ShouldRetry(RetryTwice, 0, "file not found")
	if g.ShouldRetry {
		t.Error("should not retry on not found")
	}
}

func TestGetModelGuidance(t *testing.T) {
	tests := []struct {
		agentType string
		want      string
	}{
		{"hephaestus", "inherit parent model or use a coding-capable model"},
		{"prometheus", "inherit parent model or use a reasoning-capable model"},
		{"oracle", "use the strongest available reasoning model"},
		{"unknown", "inherit parent model"},
	}

	for _, tt := range tests {
		g := GetModelGuidance(tt.agentType)
		if g.Recommendation != tt.want {
			t.Errorf("agentType %s: recommendation = %q, want %q", tt.agentType, g.Recommendation, tt.want)
		}
	}
}

func TestRetryEscalate(t *testing.T) {
	g := ShouldRetry(RetryEscalate, 0, "timeout")
	if !g.ShouldRetry {
		t.Error("should retry on first attempt with RetryEscalate")
	}
	if g.MaxAttempts != 3 {
		t.Errorf("maxAttempts = %d, want 3", g.MaxAttempts)
	}

	g = ShouldRetry(RetryEscalate, 3, "timeout")
	if g.ShouldRetry {
		t.Error("should not retry after 3 attempts")
	}
}
