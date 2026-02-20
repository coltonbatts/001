package schema

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEventJSONRoundTrip(t *testing.T) {
	evt, err := NewEvent(
		"run_test_001",
		1,
		time.Date(2026, time.January, 1, 0, 0, 1, 0, time.UTC),
		EventRunStarted,
		Actor{Kind: ActorSystem, ID: "orch"},
		"",
		"",
		RunStartedPayload{
			SchemaVersion: CurrentSchemaVersion,
			CreatedBy:     "tester",
			CWD:           "/tmp",
			Host:          "localhost",
			Platform:      "test",
			ModelConfig: ModelConfig{
				Provider:        ModelProviderLocalOpenAICompat,
				BaseURL:         "http://127.0.0.1:11434/v1",
				Model:           "glm-4.7",
				MaxTokens:       128,
				Stream:          true,
				HeadersRedacted: map[string]string{"Authorization": RedactedValue},
			},
		},
	)
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}

	b, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got Event
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if got.EventID != evt.EventID || got.RunID != evt.RunID || got.Seq != evt.Seq || got.Type != evt.Type {
		t.Fatalf("event fields changed after round-trip: got=%+v want=%+v", got, evt)
	}

	var payload RunStartedPayload
	if err := got.DecodePayload(&payload); err != nil {
		t.Fatalf("decode payload failed: %v", err)
	}
	if payload.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("schema_version mismatch: %q", payload.SchemaVersion)
	}
	if payload.ModelConfig.HeadersRedacted["Authorization"] != RedactedValue {
		t.Fatalf("redacted header mismatch")
	}
}

func TestValidateEventFailures(t *testing.T) {
	valid := mustValidRunStartedEvent(t)

	missingRunID := valid
	missingRunID.RunID = ""
	if err := ValidateEvent(missingRunID); err == nil || !strings.Contains(err.Error(), "run_id") {
		t.Fatalf("expected run_id validation error, got: %v", err)
	}

	invalidPayload := valid
	if err := invalidPayload.SetPayload(AgentRegisteredPayload{Agent: AgentSpec{}}); err != nil {
		t.Fatalf("set payload failed: %v", err)
	}
	invalidPayload.Type = EventAgentRegistered
	if err := ValidateEvent(invalidPayload); err == nil {
		t.Fatalf("expected payload validation error")
	}
}

func TestValidateSeqTransition(t *testing.T) {
	if err := ValidateSeqTransition(1, 2); err != nil {
		t.Fatalf("expected valid transition, got: %v", err)
	}
	if err := ValidateSeqTransition(2, 2); err == nil {
		t.Fatalf("expected invalid transition error")
	}
}

func mustValidRunStartedEvent(t *testing.T) Event {
	t.Helper()
	evt, err := NewEvent(
		"run_test_001",
		1,
		time.Date(2026, time.January, 1, 0, 0, 1, 0, time.UTC),
		EventRunStarted,
		Actor{Kind: ActorSystem, ID: "orch"},
		"",
		"",
		RunStartedPayload{
			SchemaVersion: CurrentSchemaVersion,
			CreatedBy:     "tester",
			CWD:           "/tmp",
			Host:          "localhost",
			Platform:      "test",
			ModelConfig: ModelConfig{
				Provider:        ModelProviderLocalOpenAICompat,
				BaseURL:         "http://127.0.0.1:11434/v1",
				Model:           "glm-4.7",
				MaxTokens:       128,
				Stream:          true,
				HeadersRedacted: map[string]string{"Authorization": RedactedValue},
			},
		},
	)
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}
	return evt
}
