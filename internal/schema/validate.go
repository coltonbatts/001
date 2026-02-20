package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var knownEventTypes = map[EventType]struct{}{
	EventRunStarted:             {},
	EventRunCompleted:           {},
	EventRunFailed:              {},
	EventAgentRegistered:        {},
	EventTaskPlanned:            {},
	EventTaskStarted:            {},
	EventTaskCompleted:          {},
	EventTaskFailed:             {},
	EventModelRequestStarted:    {},
	EventModelToken:             {},
	EventModelResponseCompleted: {},
	EventModelResponseFailed:    {},
	EventToolCallRequested:      {},
	EventToolCallStarted:        {},
	EventToolStdoutChunk:        {},
	EventToolStderrChunk:        {},
	EventToolCallCompleted:      {},
	EventToolCallFailed:         {},
	EventApprovalRequested:      {},
	EventApprovalGranted:        {},
	EventApprovalDenied:         {},
	EventArtifactWritten:        {},
	EventArtifactUpdated:        {},
	EventNote:                   {},
	EventWarning:                {},
	EventError:                  {},
}

func IsKnownEventType(t EventType) bool {
	_, ok := knownEventTypes[t]
	return ok
}

func ValidateEvent(e Event) error {
	if strings.TrimSpace(e.EventID) == "" {
		return fmt.Errorf("event_id is required")
	}
	if strings.TrimSpace(e.RunID) == "" {
		return fmt.Errorf("run_id is required")
	}
	if e.Seq <= 0 {
		return fmt.Errorf("seq must be > 0")
	}
	if strings.TrimSpace(e.TSUTC) == "" {
		return fmt.Errorf("ts_utc is required")
	}
	if _, err := time.Parse(time.RFC3339Nano, e.TSUTC); err != nil {
		return fmt.Errorf("ts_utc must be RFC3339Nano: %w", err)
	}
	if !IsKnownEventType(e.Type) {
		return fmt.Errorf("unknown event type: %s", e.Type)
	}
	if err := validateActor(e.Actor); err != nil {
		return err
	}
	if len(e.Payload) == 0 {
		return fmt.Errorf("payload is required")
	}
	if err := ValidatePayloadForType(e.Type, e.Payload); err != nil {
		return fmt.Errorf("payload validation failed for %s: %w", e.Type, err)
	}
	return nil
}

func ValidateSeqTransition(prevSeq, nextSeq int64) error {
	if nextSeq != prevSeq+1 {
		return fmt.Errorf("seq must increase by 1: prev=%d next=%d", prevSeq, nextSeq)
	}
	return nil
}

func ValidateEventSequence(prev, next Event) error {
	if prev.RunID != next.RunID {
		return fmt.Errorf("run_id mismatch in sequence: %s vs %s", prev.RunID, next.RunID)
	}
	return ValidateSeqTransition(prev.Seq, next.Seq)
}

func ValidatePayloadForType(eventType EventType, payload json.RawMessage) error {
	switch eventType {
	case EventRunStarted:
		var p RunStartedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		return validateRunStartedPayload(p)
	case EventRunCompleted:
		var p RunCompletedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		return nil
	case EventRunFailed:
		var p RunFailedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if strings.TrimSpace(p.Error) == "" {
			return fmt.Errorf("error is required")
		}
		return nil
	case EventAgentRegistered:
		var p AgentRegisteredPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		return validateAgentSpec(p.Agent)
	case EventTaskPlanned:
		var p TaskPlannedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		return validateTaskSpec(p.Task)
	case EventTaskStarted:
		var p TaskStartedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if strings.TrimSpace(p.TaskID) == "" {
			return fmt.Errorf("task_id is required")
		}
		return nil
	case EventTaskCompleted:
		var p TaskCompletedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if strings.TrimSpace(p.TaskID) == "" {
			return fmt.Errorf("task_id is required")
		}
		return nil
	case EventTaskFailed:
		var p TaskFailedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if strings.TrimSpace(p.TaskID) == "" {
			return fmt.Errorf("task_id is required")
		}
		if strings.TrimSpace(p.Error) == "" {
			return fmt.Errorf("error is required")
		}
		return nil
	case EventModelRequestStarted:
		var p ModelRequestStartedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		return validateModelRequest(p.Request)
	case EventModelToken:
		var p ModelTokenPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if strings.TrimSpace(p.RequestID) == "" {
			return fmt.Errorf("request_id is required")
		}
		return nil
	case EventModelResponseCompleted:
		var p ModelResponseCompletedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		return validateModelResponse(p.Response)
	case EventModelResponseFailed:
		var p ModelResponseFailedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if strings.TrimSpace(p.RequestID) == "" {
			return fmt.Errorf("request_id is required")
		}
		if strings.TrimSpace(p.Error) == "" {
			return fmt.Errorf("error is required")
		}
		return nil
	case EventToolCallRequested:
		var p ToolCallRequestedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		return validateToolCall(p.ToolCall)
	case EventToolCallStarted:
		var p ToolCallStartedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if strings.TrimSpace(p.ToolCallID) == "" {
			return fmt.Errorf("tool_call_id is required")
		}
		return nil
	case EventToolStdoutChunk:
		var p ToolStdoutChunkPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if strings.TrimSpace(p.ToolCallID) == "" {
			return fmt.Errorf("tool_call_id is required")
		}
		if p.StreamSeq <= 0 {
			return fmt.Errorf("stream_seq must be > 0")
		}
		return nil
	case EventToolStderrChunk:
		var p ToolStderrChunkPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if strings.TrimSpace(p.ToolCallID) == "" {
			return fmt.Errorf("tool_call_id is required")
		}
		if p.StreamSeq <= 0 {
			return fmt.Errorf("stream_seq must be > 0")
		}
		return nil
	case EventToolCallCompleted:
		var p ToolCallCompletedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		return validateToolResult(p.ToolResult)
	case EventToolCallFailed:
		var p ToolCallFailedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if strings.TrimSpace(p.ToolCallID) == "" {
			return fmt.Errorf("tool_call_id is required")
		}
		if strings.TrimSpace(p.Error) == "" {
			return fmt.Errorf("error is required")
		}
		return nil
	case EventApprovalRequested:
		var p ApprovalRequestedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		return validateApprovalRequest(p.Approval)
	case EventApprovalGranted:
		var p ApprovalGrantedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if strings.TrimSpace(p.ApprovalID) == "" {
			return fmt.Errorf("approval_id is required")
		}
		if p.GrantedBy != "user" && p.GrantedBy != "system" {
			return fmt.Errorf("granted_by must be user or system")
		}
		return nil
	case EventApprovalDenied:
		var p ApprovalDeniedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if strings.TrimSpace(p.ApprovalID) == "" {
			return fmt.Errorf("approval_id is required")
		}
		if strings.TrimSpace(p.DeniedBy) == "" {
			return fmt.Errorf("denied_by is required")
		}
		return nil
	case EventArtifactWritten:
		var p ArtifactWrittenPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if err := validateArtifactRef(p.Artifact); err != nil {
			return err
		}
		if strings.TrimSpace(p.SourceEventID) == "" {
			return fmt.Errorf("source_event_id is required")
		}
		return nil
	case EventArtifactUpdated:
		var p ArtifactUpdatedPayload
		if err := strictUnmarshal(payload, &p); err != nil {
			return err
		}
		if err := validateArtifactRef(p.Artifact); err != nil {
			return err
		}
		if strings.TrimSpace(p.SourceEventID) == "" {
			return fmt.Errorf("source_event_id is required")
		}
		return nil
	case EventNote:
		var p NotePayload
		return strictUnmarshal(payload, &p)
	case EventWarning:
		var p WarningPayload
		return strictUnmarshal(payload, &p)
	case EventError:
		var p ErrorPayload
		return strictUnmarshal(payload, &p)
	default:
		return fmt.Errorf("unknown event type: %s", eventType)
	}
}

func validateActor(a Actor) error {
	if strings.TrimSpace(a.ID) == "" {
		return fmt.Errorf("actor.id is required")
	}
	switch a.Kind {
	case ActorSystem, ActorAgent, ActorUser, ActorTool, ActorModel:
		return nil
	default:
		return fmt.Errorf("actor.kind is invalid: %s", a.Kind)
	}
}

func validateRunStartedPayload(p RunStartedPayload) error {
	if strings.TrimSpace(p.SchemaVersion) == "" {
		return fmt.Errorf("schema_version is required")
	}
	if strings.TrimSpace(p.CreatedBy) == "" {
		return fmt.Errorf("created_by is required")
	}
	if err := ValidateSafePath(p.CWD); err != nil {
		return fmt.Errorf("cwd: %w", err)
	}
	if strings.TrimSpace(p.Host) == "" {
		return fmt.Errorf("host is required")
	}
	if strings.TrimSpace(p.Platform) == "" {
		return fmt.Errorf("platform is required")
	}
	if err := validateModelConfig(p.ModelConfig); err != nil {
		return fmt.Errorf("model_config: %w", err)
	}
	return nil
}

func validateAgentSpec(a AgentSpec) error {
	if strings.TrimSpace(a.AgentID) == "" {
		return fmt.Errorf("agent_id is required")
	}
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("name is required")
	}
	switch a.Role {
	case AgentRolePlanner, AgentRoleExecutor, AgentRoleVerifier, AgentRoleCustom:
	default:
		return fmt.Errorf("role is invalid: %s", a.Role)
	}
	if strings.TrimSpace(a.ToolPolicyRef) == "" {
		return fmt.Errorf("tool_policy_ref is required")
	}
	if err := ValidateSafePath(a.Workspace.RootDir); err != nil {
		return fmt.Errorf("workspace.root_dir: %w", err)
	}
	switch a.Workspace.SandboxMode {
	case SandboxModeNone, SandboxModeWorktree, SandboxModeDirCopy, SandboxModeContainer:
	default:
		return fmt.Errorf("workspace.sandbox_mode is invalid: %s", a.Workspace.SandboxMode)
	}
	for _, p := range a.Workspace.AllowPaths {
		if err := ValidateSafePath(p); err != nil {
			return fmt.Errorf("workspace.allow_paths: %w", err)
		}
	}
	for _, p := range a.Workspace.DenyPaths {
		if err := ValidateSafePath(p); err != nil {
			return fmt.Errorf("workspace.deny_paths: %w", err)
		}
	}
	if a.Limits.MaxSteps <= 0 {
		return fmt.Errorf("limits.max_steps must be > 0")
	}
	if a.Limits.MaxTokens <= 0 {
		return fmt.Errorf("limits.max_tokens must be > 0")
	}
	if a.Limits.TimeoutMS <= 0 {
		return fmt.Errorf("limits.timeout_ms must be > 0")
	}
	if a.ModelOverrides != nil {
		if err := validateModelConfig(*a.ModelOverrides); err != nil {
			return fmt.Errorf("model_overrides: %w", err)
		}
	}
	return nil
}

func validateTaskSpec(t TaskSpec) error {
	if strings.TrimSpace(t.TaskID) == "" {
		return fmt.Errorf("task_id is required")
	}
	if strings.TrimSpace(t.Title) == "" {
		return fmt.Errorf("title is required")
	}
	switch t.Status {
	case TaskStatusPlanned, TaskStatusInProgress, TaskStatusDone, TaskStatusFailed, TaskStatusBlocked:
	default:
		return fmt.Errorf("task.status is invalid: %s", t.Status)
	}
	for _, dep := range t.DependsOn {
		if dep == t.TaskID {
			return fmt.Errorf("task cannot depend on itself")
		}
	}
	return nil
}

func validateModelConfig(c ModelConfig) error {
	switch c.Provider {
	case ModelProviderLocalOpenAICompat, ModelProviderOther:
	default:
		return fmt.Errorf("provider is invalid: %s", c.Provider)
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("base_url is required")
	}
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("model is required")
	}
	if c.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be > 0")
	}
	if err := ValidateRedactedMap(c.HeadersRedacted, "headers_redacted"); err != nil {
		return err
	}
	return nil
}

func validateModelRequest(r ModelRequest) error {
	if strings.TrimSpace(r.RequestID) == "" {
		return fmt.Errorf("request_id is required")
	}
	if strings.TrimSpace(r.AgentID) == "" {
		return fmt.Errorf("agent_id is required")
	}
	switch r.Purpose {
	case ModelPurposePlanning, ModelPurposeExecution, ModelPurposeVerification, ModelPurposeChat:
	default:
		return fmt.Errorf("purpose is invalid: %s", r.Purpose)
	}
	if len(r.Messages) == 0 {
		return fmt.Errorf("messages are required")
	}
	for i, m := range r.Messages {
		switch m.Role {
		case MessageRoleSystem, MessageRoleUser, MessageRoleAssistant, MessageRoleTool:
		default:
			return fmt.Errorf("messages[%d].role is invalid: %s", i, m.Role)
		}
	}
	switch r.ResponseFormat {
	case ResponseFormatText, ResponseFormatJSON:
	default:
		return fmt.Errorf("response_format is invalid: %s", r.ResponseFormat)
	}
	return nil
}

func validateModelResponse(r ModelResponse) error {
	if strings.TrimSpace(r.RequestID) == "" {
		return fmt.Errorf("request_id is required")
	}
	if strings.TrimSpace(r.AgentID) == "" {
		return fmt.Errorf("agent_id is required")
	}
	if strings.TrimSpace(r.FinishReason) == "" {
		return fmt.Errorf("finish_reason is required")
	}
	if r.LatencyMS < 0 {
		return fmt.Errorf("latency_ms must be >= 0")
	}
	if r.Usage.PromptTokens < 0 || r.Usage.CompletionTokens < 0 || r.Usage.TotalTokens < 0 {
		return fmt.Errorf("usage token counts must be >= 0")
	}
	for i, tc := range r.ToolCalls {
		if strings.TrimSpace(tc.ToolName) == "" {
			return fmt.Errorf("tool_calls[%d].tool_name is required", i)
		}
		if strings.TrimSpace(tc.ArgsJSON) == "" {
			return fmt.Errorf("tool_calls[%d].args_json is required", i)
		}
		switch tc.SuggestedPolicy {
		case SuggestedPolicySafe, SuggestedPolicyNeedsApproval:
		default:
			return fmt.Errorf("tool_calls[%d].suggested_policy is invalid: %s", i, tc.SuggestedPolicy)
		}
	}
	return nil
}

func validateToolCall(c ToolCall) error {
	if strings.TrimSpace(c.ToolCallID) == "" {
		return fmt.Errorf("tool_call_id is required")
	}
	if strings.TrimSpace(c.AgentID) == "" {
		return fmt.Errorf("agent_id is required")
	}
	if strings.TrimSpace(c.ToolName) == "" {
		return fmt.Errorf("tool_name is required")
	}
	if strings.TrimSpace(c.ArgsJSON) == "" {
		return fmt.Errorf("args_json is required")
	}
	if !json.Valid([]byte(c.ArgsJSON)) {
		return fmt.Errorf("args_json must contain valid JSON")
	}
	if err := ValidateSafePath(c.CWD); err != nil {
		return fmt.Errorf("cwd: %w", err)
	}
	if err := ValidateRedactedMap(c.EnvRedacted, "env_redacted"); err != nil {
		return err
	}
	switch c.PolicyDecision {
	case ToolPolicyAutoAllow, ToolPolicyNeedsApproval, ToolPolicyDenied:
	default:
		return fmt.Errorf("policy_decision is invalid: %s", c.PolicyDecision)
	}
	if c.TimeoutMS <= 0 {
		return fmt.Errorf("timeout_ms must be > 0")
	}
	return nil
}

func validateToolResult(r ToolResult) error {
	if strings.TrimSpace(r.ToolCallID) == "" {
		return fmt.Errorf("tool_call_id is required")
	}
	if r.DurationMS < 0 {
		return fmt.Errorf("duration_ms must be >= 0")
	}
	for i := range r.ProducedArtifacts {
		if err := validateArtifactRef(r.ProducedArtifacts[i]); err != nil {
			return fmt.Errorf("produced_artifacts[%d]: %w", i, err)
		}
	}
	return nil
}

func validateApprovalRequest(a ApprovalRequest) error {
	if strings.TrimSpace(a.ApprovalID) == "" {
		return fmt.Errorf("approval_id is required")
	}
	if strings.TrimSpace(a.AgentID) == "" {
		return fmt.Errorf("agent_id is required")
	}
	if strings.TrimSpace(a.Reason) == "" {
		return fmt.Errorf("reason is required")
	}
	switch a.Risk {
	case ApprovalRiskLow, ApprovalRiskMedium, ApprovalRiskHigh:
	default:
		return fmt.Errorf("risk is invalid: %s", a.Risk)
	}
	if strings.TrimSpace(a.ActionSummary) == "" {
		return fmt.Errorf("action_summary is required")
	}
	for _, fileOp := range a.FileOps {
		if err := ValidateSafePath(fileOp); err != nil {
			return fmt.Errorf("file_ops: %w", err)
		}
	}
	if strings.TrimSpace(a.ExpiresTSUTC) != "" {
		if _, err := time.Parse(time.RFC3339Nano, a.ExpiresTSUTC); err != nil {
			return fmt.Errorf("expires_ts_utc must be RFC3339Nano: %w", err)
		}
	}
	return nil
}

func validateArtifactRef(a ArtifactRef) error {
	if strings.TrimSpace(a.ArtifactID) == "" {
		return fmt.Errorf("artifact_id is required")
	}
	switch a.Kind {
	case ArtifactKindFile, ArtifactKindDiff, ArtifactKindImage, ArtifactKindVideo, ArtifactKindJSON, ArtifactKindText, ArtifactKindOther:
	default:
		return fmt.Errorf("kind is invalid: %s", a.Kind)
	}
	if err := ValidateSafePath(a.Path); err != nil {
		return fmt.Errorf("path: %w", err)
	}
	if a.Bytes < 0 {
		return fmt.Errorf("bytes must be >= 0")
	}
	if a.Digest != "" {
		if len(a.Digest) != 64 {
			return fmt.Errorf("digest must be a sha256 hex string")
		}
		for _, r := range a.Digest {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				return fmt.Errorf("digest must be lowercase sha256 hex")
			}
		}
	}
	return nil
}

func ValidateRedactedMap(m map[string]string, field string) error {
	for k, v := range m {
		if strings.TrimSpace(k) == "" {
			return fmt.Errorf("%s has empty key", field)
		}
		if v != RedactedValue {
			return fmt.Errorf("%s value for key %q must be %q", field, k, RedactedValue)
		}
	}
	return nil
}

func ValidateSafePath(p string) error {
	if strings.TrimSpace(p) == "" {
		return fmt.Errorf("path is required")
	}
	if strings.ContainsRune(p, 0) {
		return fmt.Errorf("path contains NUL byte")
	}
	clean := filepath.Clean(p)
	if clean == "." {
		return fmt.Errorf("path resolves to current directory")
	}
	return nil
}

func strictUnmarshal(payload []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}
