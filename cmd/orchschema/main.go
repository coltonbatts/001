package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	runlog "orchschema/internal/log"
	"orchschema/internal/projector"
	"orchschema/internal/schema"
)

const demoPath = "/tmp/orchschema_demo_run.jsonl"

func main() {
	runID := "run_demo_000001"
	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	f, err := os.Create(demoPath)
	if err != nil {
		log.Fatalf("create demo log: %v", err)
	}

	writer := runlog.NewWriter(f, runID, 1)
	events := demoEvents(runID, base)
	for i := range events {
		if err := writer.Append(&events[i]); err != nil {
			_ = f.Close()
			log.Fatalf("append event %d: %v", i+1, err)
		}
	}
	if err := writer.Close(); err != nil {
		_ = f.Close()
		log.Fatalf("flush demo log: %v", err)
	}
	if err := f.Close(); err != nil {
		log.Fatalf("close demo log: %v", err)
	}

	rf, err := os.Open(demoPath)
	if err != nil {
		log.Fatalf("open demo log for replay: %v", err)
	}
	defer rf.Close()

	reader := runlog.NewReader(rf)
	proj := projector.New()
	for {
		evt, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("read event: %v", err)
		}
		if err := proj.Apply(evt); err != nil {
			log.Fatalf("apply event %s: %v", evt.EventID, err)
		}
	}

	state := proj.State()
	fmt.Printf("run_id=%s tasks=%d agents=%d artifacts=%d final_status=%s\n",
		runID,
		len(state.Tasks),
		len(state.Agents),
		len(state.Artifacts),
		state.Status,
	)
	fmt.Printf("log=%s\n", demoPath)
}

func demoEvents(runID string, base time.Time) []schema.Event {
	modelReqID := "req_001"
	toolCallID := "tool_001"
	approvalID := "appr_001"

	stdout := "hello from tool\n"
	artifact := schema.ArtifactRef{
		ArtifactID: "art_001",
		Kind:       schema.ArtifactKindText,
		Path:       "output/demo.txt",
		Digest:     schema.SHA256Hex([]byte(stdout)),
		Bytes:      int64(len(stdout)),
		MIME:       "text/plain",
		Label:      "tool output",
	}

	actorSystem := schema.Actor{Kind: schema.ActorSystem, ID: "orch"}
	actorAgent := schema.Actor{Kind: schema.ActorAgent, ID: "agent_001"}
	actorModel := schema.Actor{Kind: schema.ActorModel, ID: "glm-4.7"}
	actorUser := schema.Actor{Kind: schema.ActorUser, ID: "user_local"}
	actorTool := schema.Actor{Kind: schema.ActorTool, ID: "shell"}

	return []schema.Event{
		mustEvent(runID, 1, base.Add(1*time.Second), schema.EventRunStarted, actorSystem, "", schema.RunStartedPayload{
			SchemaVersion: schema.CurrentSchemaVersion,
			CreatedBy:     "orchschema-demo",
			CWD:           "/tmp",
			Host:          "localhost",
			Platform:      "local",
			ToolPolicies: schema.ToolPolicies{
				AllowedTools:       []string{"shell", "read_file", "write_file"},
				RequireApprovalFor: []string{"write_file", "git_commit", "shell"},
				ShellAllowlist:     []string{"ls", "cat", "echo"},
			},
			ModelConfig: schema.ModelConfig{
				Provider:        schema.ModelProviderLocalOpenAICompat,
				BaseURL:         "http://127.0.0.1:11434/v1",
				Model:           "glm-4.7",
				Temperature:     0.2,
				TopP:            1.0,
				MaxTokens:       1024,
				Stream:          true,
				JSONMode:        false,
				HeadersRedacted: map[string]string{"Authorization": schema.RedactedValue},
			},
			Tags: map[string]string{"env": "demo"},
		}),
		mustEvent(runID, 2, base.Add(2*time.Second), schema.EventAgentRegistered, actorSystem, "", schema.AgentRegisteredPayload{
			Agent: schema.AgentSpec{
				AgentID:       "agent_001",
				Name:          "executor",
				Role:          schema.AgentRoleExecutor,
				Instructions:  "Execute planned steps with safety gates.",
				ToolPolicyRef: "default",
				Workspace: schema.WorkspaceSpec{
					RootDir:     "/tmp",
					SandboxMode: schema.SandboxModeWorktree,
					AllowPaths:  []string{"/tmp"},
				},
				Limits: schema.AgentLimits{MaxSteps: 20, MaxTokens: 8000, TimeoutMS: 120000},
			},
		}),
		mustEvent(runID, 3, base.Add(3*time.Second), schema.EventTaskPlanned, actorAgent, "", schema.TaskPlannedPayload{
			Task: schema.TaskSpec{
				TaskID:      "task_001",
				Title:       "Inspect workspace",
				Description: "List current files and summarize",
				Status:      schema.TaskStatusPlanned,
				Checklist:   []string{"List files", "Summarize"},
				Acceptance:  []string{"Produces one artifact"},
				Priority:    1,
				Metadata:    map[string]string{"epic": "demo"},
			},
		}),
		mustEvent(runID, 4, base.Add(4*time.Second), schema.EventModelRequestStarted, actorAgent, modelReqID, schema.ModelRequestStartedPayload{
			Request: schema.ModelRequest{
				RequestID: modelReqID,
				AgentID:   "agent_001",
				Purpose:   schema.ModelPurposeExecution,
				Messages: []schema.Message{
					{Role: schema.MessageRoleSystem, Content: "You are a local orchestrator agent."},
					{Role: schema.MessageRoleUser, Content: "Plan the next shell step safely."},
				},
				Tools:          []schema.ToolSchemaRef{{ToolName: "shell", Summary: "Run allowlisted shell commands"}},
				ResponseFormat: schema.ResponseFormatText,
				Metadata:       map[string]string{"trace": "demo"},
			},
		}),
		mustEvent(runID, 5, base.Add(5*time.Second), schema.EventModelToken, actorModel, modelReqID, schema.ModelTokenPayload{
			RequestID: modelReqID,
			Delta:     "I will run an allowlisted command.",
		}),
		mustEvent(runID, 6, base.Add(6*time.Second), schema.EventModelResponseCompleted, actorModel, modelReqID, schema.ModelResponseCompletedPayload{
			Response: schema.ModelResponse{
				RequestID:    modelReqID,
				AgentID:      "agent_001",
				OutputText:   "Request approval then run echo.",
				FinishReason: "stop",
				LatencyMS:    120,
				Usage:        schema.Usage{PromptTokens: 30, CompletionTokens: 10, TotalTokens: 40},
			},
		}),
		mustEvent(runID, 7, base.Add(7*time.Second), schema.EventToolCallRequested, actorAgent, toolCallID, schema.ToolCallRequestedPayload{
			ToolCall: schema.ToolCall{
				ToolCallID:     toolCallID,
				AgentID:        "agent_001",
				ToolName:       "shell",
				ArgsJSON:       `{"cmd":"echo hello from tool"}`,
				CWD:            "/tmp",
				EnvRedacted:    map[string]string{"API_KEY": schema.RedactedValue},
				PolicyDecision: schema.ToolPolicyNeedsApproval,
				AllowlistMatch: "echo",
				TimeoutMS:      30000,
			},
		}),
		mustEvent(runID, 8, base.Add(8*time.Second), schema.EventApprovalRequested, actorSystem, approvalID, schema.ApprovalRequestedPayload{
			Approval: schema.ApprovalRequest{
				ApprovalID:    approvalID,
				AgentID:       "agent_001",
				Reason:        "shell command requires gate",
				Risk:          schema.ApprovalRiskMedium,
				ActionSummary: "Run: echo hello from tool",
				ToolCallID:    toolCallID,
				FileOps:       []string{"/tmp"},
			},
		}),
		mustEvent(runID, 9, base.Add(9*time.Second), schema.EventApprovalGranted, actorUser, approvalID, schema.ApprovalGrantedPayload{
			ApprovalID: approvalID,
			GrantedBy:  "user",
			Note:       "Approved for demo",
		}),
		mustEvent(runID, 10, base.Add(10*time.Second), schema.EventToolCallStarted, actorTool, toolCallID, schema.ToolCallStartedPayload{
			ToolCallID: toolCallID,
		}),
		mustEvent(runID, 11, base.Add(11*time.Second), schema.EventToolStdoutChunk, actorTool, toolCallID, schema.ToolStdoutChunkPayload{
			ToolCallID: toolCallID,
			Chunk:      stdout,
			StreamSeq:  1,
		}),
		mustEvent(runID, 12, base.Add(12*time.Second), schema.EventToolCallCompleted, actorTool, toolCallID, schema.ToolCallCompletedPayload{
			ToolResult: schema.ToolResult{
				ToolCallID:        toolCallID,
				ExitCode:          0,
				StdoutTruncated:   schema.TruncateForSummary(stdout, schema.DefaultToolSummaryMaxBytes),
				DurationMS:        25,
				ProducedArtifacts: []schema.ArtifactRef{artifact},
			},
		}),
		mustEvent(runID, 13, base.Add(13*time.Second), schema.EventArtifactWritten, actorSystem, toolCallID, schema.ArtifactWrittenPayload{
			Artifact:      artifact,
			SourceEventID: "evt_12",
			Note:          "Captured tool output",
		}),
		mustEvent(runID, 14, base.Add(14*time.Second), schema.EventRunCompleted, actorSystem, "", schema.RunCompletedPayload{
			Reason: "demo complete",
		}),
	}
}

func mustEvent(
	runID string,
	seq int64,
	ts time.Time,
	eventType schema.EventType,
	actor schema.Actor,
	correlationID string,
	payload any,
) schema.Event {
	e, err := schema.NewEvent(runID, seq, ts, eventType, actor, correlationID, "", payload)
	if err != nil {
		panic(err)
	}
	e.EventID = fmt.Sprintf("evt_%d", seq)
	return e
}
