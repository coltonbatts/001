package projector

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"orchschema/internal/schema"
)

func TestReplayDeterministic(t *testing.T) {
	events := sampleEvents(t)

	stateA, err := Replay(events)
	if err != nil {
		t.Fatalf("replay A failed: %v", err)
	}
	stateB, err := Replay(events)
	if err != nil {
		t.Fatalf("replay B failed: %v", err)
	}

	if !reflect.DeepEqual(stateA, stateB) {
		t.Fatalf("replay states differ")
	}
	if stateA.Status != schema.RunStatusCompleted {
		t.Fatalf("expected completed status, got %s", stateA.Status)
	}
	if len(stateA.Tasks) != 1 || len(stateA.Agents) != 1 || len(stateA.Artifacts) != 1 {
		t.Fatalf("unexpected counts: tasks=%d agents=%d artifacts=%d", len(stateA.Tasks), len(stateA.Agents), len(stateA.Artifacts))
	}
	if len(stateA.ActiveModelRequests) != 0 || len(stateA.ActiveToolCalls) != 0 {
		t.Fatalf("expected no active requests or tool calls")
	}
	if stateA.LastSeq != int64(len(events)) {
		t.Fatalf("last seq mismatch: got=%d want=%d", stateA.LastSeq, len(events))
	}
}

func TestProjectorSeqRegression(t *testing.T) {
	events := sampleEvents(t)
	p := New()

	if err := p.Apply(events[0]); err != nil {
		t.Fatalf("apply first event failed: %v", err)
	}

	bad := events[1]
	bad.Seq = 7
	bad.EventID = "evt_7"

	if err := p.Apply(bad); err == nil {
		t.Fatalf("expected seq regression error")
	}
}

func sampleEvents(t *testing.T) []schema.Event {
	t.Helper()
	runID := "run_test_001"
	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	artifact := schema.ArtifactRef{
		ArtifactID: "art_001",
		Kind:       schema.ArtifactKindText,
		Path:       "output/demo.txt",
		Digest:     schema.SHA256Hex([]byte("out\n")),
		Bytes:      4,
		MIME:       "text/plain",
	}

	actorSystem := schema.Actor{Kind: schema.ActorSystem, ID: "orch"}
	actorAgent := schema.Actor{Kind: schema.ActorAgent, ID: "agent_001"}
	actorModel := schema.Actor{Kind: schema.ActorModel, ID: "glm-4.7"}
	actorUser := schema.Actor{Kind: schema.ActorUser, ID: "user_local"}
	actorTool := schema.Actor{Kind: schema.ActorTool, ID: "shell"}

	return []schema.Event{
		mustEvent(t, runID, 1, base.Add(time.Second), schema.EventRunStarted, actorSystem, "", schema.RunStartedPayload{
			SchemaVersion: schema.CurrentSchemaVersion,
			CreatedBy:     "tester",
			CWD:           "/tmp",
			Host:          "localhost",
			Platform:      "test",
			ModelConfig: schema.ModelConfig{
				Provider:        schema.ModelProviderLocalOpenAICompat,
				BaseURL:         "http://127.0.0.1:11434/v1",
				Model:           "glm-4.7",
				MaxTokens:       512,
				Stream:          true,
				HeadersRedacted: map[string]string{"Authorization": schema.RedactedValue},
			},
		}),
		mustEvent(t, runID, 2, base.Add(2*time.Second), schema.EventAgentRegistered, actorSystem, "", schema.AgentRegisteredPayload{
			Agent: schema.AgentSpec{
				AgentID:       "agent_001",
				Name:          "executor",
				Role:          schema.AgentRoleExecutor,
				Instructions:  "execute",
				ToolPolicyRef: "default",
				Workspace:     schema.WorkspaceSpec{RootDir: "/tmp", SandboxMode: schema.SandboxModeWorktree, AllowPaths: []string{"/tmp"}},
				Limits:        schema.AgentLimits{MaxSteps: 10, MaxTokens: 4000, TimeoutMS: 60000},
			},
		}),
		mustEvent(t, runID, 3, base.Add(3*time.Second), schema.EventTaskPlanned, actorAgent, "", schema.TaskPlannedPayload{
			Task: schema.TaskSpec{TaskID: "task_001", Title: "t", Description: "d", Status: schema.TaskStatusPlanned, Priority: 1},
		}),
		mustEvent(t, runID, 4, base.Add(4*time.Second), schema.EventModelRequestStarted, actorAgent, "req_001", schema.ModelRequestStartedPayload{
			Request: schema.ModelRequest{
				RequestID:      "req_001",
				AgentID:        "agent_001",
				Purpose:        schema.ModelPurposeExecution,
				Messages:       []schema.Message{{Role: schema.MessageRoleUser, Content: "do x"}},
				ResponseFormat: schema.ResponseFormatText,
			},
		}),
		mustEvent(t, runID, 5, base.Add(5*time.Second), schema.EventModelToken, actorModel, "req_001", schema.ModelTokenPayload{RequestID: "req_001", Delta: "x"}),
		mustEvent(t, runID, 6, base.Add(6*time.Second), schema.EventModelResponseCompleted, actorModel, "req_001", schema.ModelResponseCompletedPayload{
			Response: schema.ModelResponse{
				RequestID:    "req_001",
				AgentID:      "agent_001",
				OutputText:   "ok",
				FinishReason: "stop",
				LatencyMS:    10,
				Usage:        schema.Usage{PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2},
			},
		}),
		mustEvent(t, runID, 7, base.Add(7*time.Second), schema.EventToolCallRequested, actorAgent, "tool_001", schema.ToolCallRequestedPayload{
			ToolCall: schema.ToolCall{
				ToolCallID:     "tool_001",
				AgentID:        "agent_001",
				ToolName:       "shell",
				ArgsJSON:       `{"cmd":"echo out"}`,
				CWD:            "/tmp",
				EnvRedacted:    map[string]string{"API_KEY": schema.RedactedValue},
				PolicyDecision: schema.ToolPolicyNeedsApproval,
				TimeoutMS:      1000,
			},
		}),
		mustEvent(t, runID, 8, base.Add(8*time.Second), schema.EventApprovalRequested, actorSystem, "appr_001", schema.ApprovalRequestedPayload{
			Approval: schema.ApprovalRequest{
				ApprovalID:    "appr_001",
				AgentID:       "agent_001",
				Reason:        "needs approval",
				Risk:          schema.ApprovalRiskMedium,
				ActionSummary: "run shell",
				ToolCallID:    "tool_001",
			},
		}),
		mustEvent(t, runID, 9, base.Add(9*time.Second), schema.EventApprovalGranted, actorUser, "appr_001", schema.ApprovalGrantedPayload{ApprovalID: "appr_001", GrantedBy: "user"}),
		mustEvent(t, runID, 10, base.Add(10*time.Second), schema.EventToolCallStarted, actorTool, "tool_001", schema.ToolCallStartedPayload{ToolCallID: "tool_001"}),
		mustEvent(t, runID, 11, base.Add(11*time.Second), schema.EventToolStdoutChunk, actorTool, "tool_001", schema.ToolStdoutChunkPayload{ToolCallID: "tool_001", Chunk: "out\n", StreamSeq: 1}),
		mustEvent(t, runID, 12, base.Add(12*time.Second), schema.EventToolCallCompleted, actorTool, "tool_001", schema.ToolCallCompletedPayload{
			ToolResult: schema.ToolResult{ToolCallID: "tool_001", ExitCode: 0, DurationMS: 5, ProducedArtifacts: []schema.ArtifactRef{artifact}},
		}),
		mustEvent(t, runID, 13, base.Add(13*time.Second), schema.EventArtifactWritten, actorSystem, "tool_001", schema.ArtifactWrittenPayload{Artifact: artifact, SourceEventID: "evt_12"}),
		mustEvent(t, runID, 14, base.Add(14*time.Second), schema.EventRunCompleted, actorSystem, "", schema.RunCompletedPayload{}),
	}
}

func mustEvent(t *testing.T, runID string, seq int64, ts time.Time, eventType schema.EventType, actor schema.Actor, correlationID string, payload any) schema.Event {
	t.Helper()
	evt, err := schema.NewEvent(runID, seq, ts, eventType, actor, correlationID, "", payload)
	if err != nil {
		t.Fatalf("new event failed: %v", err)
	}
	evt.EventID = fmt.Sprintf("evt_%d", seq)
	return evt
}
