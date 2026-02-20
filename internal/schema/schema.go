package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	CurrentSchemaVersion          = "0.1.0"
	RedactedValue                 = "***"
	DefaultToolSummaryMaxBytes    = 4096
	DefaultToolStreamChunkMaxByte = 8192
)

type RunLifecycleStatus string

const (
	RunStatusRunning   RunLifecycleStatus = "running"
	RunStatusCompleted RunLifecycleStatus = "completed"
	RunStatusFailed    RunLifecycleStatus = "failed"
)

type EventType string

const (
	EventRunStarted             EventType = "run_started"
	EventRunCompleted           EventType = "run_completed"
	EventRunFailed              EventType = "run_failed"
	EventAgentRegistered        EventType = "agent_registered"
	EventTaskPlanned            EventType = "task_planned"
	EventTaskStarted            EventType = "task_started"
	EventTaskCompleted          EventType = "task_completed"
	EventTaskFailed             EventType = "task_failed"
	EventModelRequestStarted    EventType = "model_request_started"
	EventModelToken             EventType = "model_token"
	EventModelResponseCompleted EventType = "model_response_completed"
	EventModelResponseFailed    EventType = "model_response_failed"
	EventToolCallRequested      EventType = "tool_call_requested"
	EventToolCallStarted        EventType = "tool_call_started"
	EventToolStdoutChunk        EventType = "tool_stdout_chunk"
	EventToolStderrChunk        EventType = "tool_stderr_chunk"
	EventToolCallCompleted      EventType = "tool_call_completed"
	EventToolCallFailed         EventType = "tool_call_failed"
	EventApprovalRequested      EventType = "approval_requested"
	EventApprovalGranted        EventType = "approval_granted"
	EventApprovalDenied         EventType = "approval_denied"
	EventArtifactWritten        EventType = "artifact_written"
	EventArtifactUpdated        EventType = "artifact_updated"
	EventNote                   EventType = "note"
	EventWarning                EventType = "warning"
	EventError                  EventType = "error"
)

type ActorKind string

const (
	ActorSystem ActorKind = "system"
	ActorAgent  ActorKind = "agent"
	ActorUser   ActorKind = "user"
	ActorTool   ActorKind = "tool"
	ActorModel  ActorKind = "model"
)

type Actor struct {
	Kind ActorKind `json:"kind"`
	ID   string    `json:"id"`
}

type AgentRole string

const (
	AgentRolePlanner  AgentRole = "planner"
	AgentRoleExecutor AgentRole = "executor"
	AgentRoleVerifier AgentRole = "verifier"
	AgentRoleCustom   AgentRole = "custom"
)

type SandboxMode string

const (
	SandboxModeNone      SandboxMode = "none"
	SandboxModeWorktree  SandboxMode = "worktree"
	SandboxModeDirCopy   SandboxMode = "dircopy"
	SandboxModeContainer SandboxMode = "container"
)

type TaskStatus string

const (
	TaskStatusPlanned    TaskStatus = "planned"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusBlocked    TaskStatus = "blocked"
)

type ModelProvider string

const (
	ModelProviderLocalOpenAICompat ModelProvider = "local_openai_compat"
	ModelProviderOther             ModelProvider = "other"
)

type ModelPurpose string

const (
	ModelPurposePlanning     ModelPurpose = "planning"
	ModelPurposeExecution    ModelPurpose = "execution"
	ModelPurposeVerification ModelPurpose = "verification"
	ModelPurposeChat         ModelPurpose = "chat"
)

type ResponseFormat string

const (
	ResponseFormatText ResponseFormat = "text"
	ResponseFormatJSON ResponseFormat = "json"
)

type SuggestedPolicy string

const (
	SuggestedPolicySafe          SuggestedPolicy = "safe"
	SuggestedPolicyNeedsApproval SuggestedPolicy = "needs_approval"
)

type ToolPolicyDecision string

const (
	ToolPolicyAutoAllow     ToolPolicyDecision = "auto_allow"
	ToolPolicyNeedsApproval ToolPolicyDecision = "needs_approval"
	ToolPolicyDenied        ToolPolicyDecision = "denied"
)

type ApprovalRisk string

const (
	ApprovalRiskLow    ApprovalRisk = "low"
	ApprovalRiskMedium ApprovalRisk = "medium"
	ApprovalRiskHigh   ApprovalRisk = "high"
)

type ApprovalDecisionStatus string

const (
	ApprovalStatusRequested ApprovalDecisionStatus = "requested"
	ApprovalStatusGranted   ApprovalDecisionStatus = "granted"
	ApprovalStatusDenied    ApprovalDecisionStatus = "denied"
)

type ArtifactKind string

const (
	ArtifactKindFile  ArtifactKind = "file"
	ArtifactKindDiff  ArtifactKind = "diff"
	ArtifactKindImage ArtifactKind = "image"
	ArtifactKindVideo ArtifactKind = "video"
	ArtifactKindJSON  ArtifactKind = "json"
	ArtifactKindText  ArtifactKind = "text"
	ArtifactKindOther ArtifactKind = "other"
)

type ToolPolicies struct {
	AllowedTools       []string `json:"allowed_tools,omitempty"`
	RequireApprovalFor []string `json:"require_approval_for,omitempty"`
	DeniedTools        []string `json:"denied_tools,omitempty"`
	ShellAllowlist     []string `json:"shell_allowlist,omitempty"`
}

type RunStartedPayload struct {
	SchemaVersion string            `json:"schema_version"`
	CreatedBy     string            `json:"created_by"`
	CWD           string            `json:"cwd"`
	Host          string            `json:"host"`
	Platform      string            `json:"platform"`
	ToolPolicies  ToolPolicies      `json:"tool_policies"`
	ModelConfig   ModelConfig       `json:"model_config"`
	Tags          map[string]string `json:"tags,omitempty"`
}

type RunCompletedPayload struct {
	Reason string `json:"reason,omitempty"`
}

type RunFailedPayload struct {
	Error string `json:"error"`
}

type AgentRegisteredPayload struct {
	Agent AgentSpec `json:"agent"`
}

type AgentSpec struct {
	AgentID        string        `json:"agent_id"`
	Name           string        `json:"name"`
	Role           AgentRole     `json:"role"`
	Instructions   string        `json:"instructions"`
	ToolPolicyRef  string        `json:"tool_policy_ref"`
	ModelOverrides *ModelConfig  `json:"model_overrides,omitempty"`
	Workspace      WorkspaceSpec `json:"workspace"`
	Limits         AgentLimits   `json:"limits"`
}

type WorkspaceSpec struct {
	RootDir        string      `json:"root_dir"`
	SandboxMode    SandboxMode `json:"sandbox_mode"`
	WorktreeBranch string      `json:"worktree_branch,omitempty"`
	AllowPaths     []string    `json:"allow_paths,omitempty"`
	DenyPaths      []string    `json:"deny_paths,omitempty"`
}

type AgentLimits struct {
	MaxSteps  int   `json:"max_steps"`
	MaxTokens int   `json:"max_tokens"`
	TimeoutMS int64 `json:"timeout_ms"`
}

type TaskPlannedPayload struct {
	Task TaskSpec `json:"task"`
}

type TaskStartedPayload struct {
	TaskID  string `json:"task_id"`
	AgentID string `json:"agent_id,omitempty"`
}

type TaskCompletedPayload struct {
	TaskID string `json:"task_id"`
	Note   string `json:"note,omitempty"`
}

type TaskFailedPayload struct {
	TaskID string `json:"task_id"`
	Error  string `json:"error"`
}

type TaskSpec struct {
	TaskID      string            `json:"task_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      TaskStatus        `json:"status"`
	DependsOn   []string          `json:"depends_on,omitempty"`
	AgentID     string            `json:"agent_id,omitempty"`
	Checklist   []string          `json:"checklist,omitempty"`
	Acceptance  []string          `json:"acceptance,omitempty"`
	Priority    int               `json:"priority"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type ModelRequestStartedPayload struct {
	Request ModelRequest `json:"request"`
}

type ModelTokenPayload struct {
	RequestID string `json:"request_id"`
	Delta     string `json:"delta"`
}

type ModelResponseCompletedPayload struct {
	Response ModelResponse `json:"response"`
}

type ModelResponseFailedPayload struct {
	RequestID string `json:"request_id"`
	Error     string `json:"error"`
	LatencyMS int64  `json:"latency_ms,omitempty"`
}

type ModelConfig struct {
	Provider        ModelProvider     `json:"provider"`
	BaseURL         string            `json:"base_url"`
	Model           string            `json:"model"`
	Temperature     float64           `json:"temperature"`
	TopP            float64           `json:"top_p"`
	MaxTokens       int               `json:"max_tokens"`
	Seed            *int64            `json:"seed,omitempty"`
	Stream          bool              `json:"stream"`
	JSONMode        bool              `json:"json_mode"`
	Stop            []string          `json:"stop,omitempty"`
	HeadersRedacted map[string]string `json:"headers_redacted,omitempty"`
}

type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
)

type Message struct {
	Role       MessageRole `json:"role"`
	Content    string      `json:"content"`
	Name       string      `json:"name,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

type ToolSchemaRef struct {
	ToolName string `json:"tool_name"`
	Summary  string `json:"summary,omitempty"`
}

type ModelRequest struct {
	RequestID      string            `json:"request_id"`
	AgentID        string            `json:"agent_id"`
	Purpose        ModelPurpose      `json:"purpose"`
	Messages       []Message         `json:"messages"`
	Tools          []ToolSchemaRef   `json:"tools,omitempty"`
	ResponseFormat ResponseFormat    `json:"response_format"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type ModelResponse struct {
	RequestID    string          `json:"request_id"`
	AgentID      string          `json:"agent_id"`
	OutputText   string          `json:"output_text,omitempty"`
	OutputJSON   string          `json:"output_json,omitempty"`
	ToolCalls    []ToolCallDraft `json:"tool_calls,omitempty"`
	Usage        Usage           `json:"usage"`
	FinishReason string          `json:"finish_reason"`
	LatencyMS    int64           `json:"latency_ms"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ToolCallDraft struct {
	ToolName        string          `json:"tool_name"`
	ArgsJSON        string          `json:"args_json"`
	SuggestedPolicy SuggestedPolicy `json:"suggested_policy"`
}

type ToolCallRequestedPayload struct {
	ToolCall ToolCall `json:"tool_call"`
}

type ToolCallStartedPayload struct {
	ToolCallID string `json:"tool_call_id"`
}

type ToolStdoutChunkPayload struct {
	ToolCallID string `json:"tool_call_id"`
	Chunk      string `json:"chunk"`
	StreamSeq  int64  `json:"stream_seq"`
}

type ToolStderrChunkPayload struct {
	ToolCallID string `json:"tool_call_id"`
	Chunk      string `json:"chunk"`
	StreamSeq  int64  `json:"stream_seq"`
}

type ToolCallCompletedPayload struct {
	ToolResult ToolResult `json:"tool_result"`
}

type ToolCallFailedPayload struct {
	ToolCallID string `json:"tool_call_id"`
	Error      string `json:"error"`
	DurationMS int64  `json:"duration_ms,omitempty"`
}

type ToolCall struct {
	ToolCallID     string             `json:"tool_call_id"`
	AgentID        string             `json:"agent_id"`
	ToolName       string             `json:"tool_name"`
	ArgsJSON       string             `json:"args_json"`
	CWD            string             `json:"cwd"`
	EnvRedacted    map[string]string  `json:"env_redacted,omitempty"`
	PolicyDecision ToolPolicyDecision `json:"policy_decision"`
	AllowlistMatch string             `json:"allowlist_match,omitempty"`
	TimeoutMS      int64              `json:"timeout_ms"`
}

type ToolResult struct {
	ToolCallID        string        `json:"tool_call_id"`
	ExitCode          int           `json:"exit_code"`
	StdoutTruncated   string        `json:"stdout_truncated,omitempty"`
	StderrTruncated   string        `json:"stderr_truncated,omitempty"`
	DurationMS        int64         `json:"duration_ms"`
	ProducedArtifacts []ArtifactRef `json:"produced_artifacts,omitempty"`
	Error             string        `json:"error,omitempty"`
}

type ApprovalRequestedPayload struct {
	Approval ApprovalRequest `json:"approval"`
}

type ApprovalGrantedPayload struct {
	ApprovalID string `json:"approval_id"`
	GrantedBy  string `json:"granted_by"`
	Note       string `json:"note,omitempty"`
}

type ApprovalDeniedPayload struct {
	ApprovalID string `json:"approval_id"`
	DeniedBy   string `json:"denied_by"`
	Note       string `json:"note,omitempty"`
}

type ApprovalRequest struct {
	ApprovalID    string       `json:"approval_id"`
	AgentID       string       `json:"agent_id"`
	Reason        string       `json:"reason"`
	Risk          ApprovalRisk `json:"risk"`
	ActionSummary string       `json:"action_summary"`
	ToolCallID    string       `json:"tool_call_id,omitempty"`
	FileOps       []string     `json:"file_ops,omitempty"`
	ExpiresTSUTC  string       `json:"expires_ts_utc,omitempty"`
}

type ArtifactRef struct {
	ArtifactID string       `json:"artifact_id"`
	Kind       ArtifactKind `json:"kind"`
	Path       string       `json:"path"`
	Digest     string       `json:"digest,omitempty"`
	Bytes      int64        `json:"bytes"`
	MIME       string       `json:"mime,omitempty"`
	Label      string       `json:"label,omitempty"`
}

type ArtifactWrittenPayload struct {
	Artifact      ArtifactRef `json:"artifact"`
	SourceEventID string      `json:"source_event_id"`
	Note          string      `json:"note,omitempty"`
}

type ArtifactUpdatedPayload struct {
	Artifact      ArtifactRef `json:"artifact"`
	SourceEventID string      `json:"source_event_id"`
	Note          string      `json:"note,omitempty"`
}

type NotePayload struct {
	Message string `json:"message"`
}

type WarningPayload struct {
	Message string `json:"message"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

type Event struct {
	EventID       string          `json:"event_id"`
	RunID         string          `json:"run_id"`
	Seq           int64           `json:"seq"`
	TSUTC         string          `json:"ts_utc"`
	Type          EventType       `json:"type"`
	Actor         Actor           `json:"actor"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	ParentEventID string          `json:"parent_event_id,omitempty"`
	Payload       json.RawMessage `json:"payload"`
}

func NewEvent(
	runID string,
	seq int64,
	ts time.Time,
	eventType EventType,
	actor Actor,
	correlationID string,
	parentEventID string,
	payload any,
) (Event, error) {
	e := Event{
		EventID:       fmt.Sprintf("evt_%d", seq),
		RunID:         runID,
		Seq:           seq,
		TSUTC:         ts.UTC().Format(time.RFC3339Nano),
		Type:          eventType,
		Actor:         actor,
		CorrelationID: correlationID,
		ParentEventID: parentEventID,
	}
	if err := e.SetPayload(payload); err != nil {
		return Event{}, err
	}
	return e, nil
}

func (e *Event) SetPayload(payload any) error {
	if payload == nil {
		e.Payload = json.RawMessage(`{}`)
		return nil
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	e.Payload = b
	return nil
}

func (e Event) DecodePayload(dst any) error {
	if len(e.Payload) == 0 {
		return fmt.Errorf("event payload is empty")
	}
	return json.Unmarshal(e.Payload, dst)
}

func (e Event) Timestamp() (time.Time, error) {
	return time.Parse(time.RFC3339Nano, e.TSUTC)
}

func RedactMapValues(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for k := range values {
		out[k] = RedactedValue
	}
	return out
}

func TruncateForSummary(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes] + "...(truncated)"
}

func SHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

type DeterministicIDGen struct {
	prefix string
	next   int64
}

func NewDeterministicIDGen(prefix string, start int64) *DeterministicIDGen {
	if strings.TrimSpace(prefix) == "" {
		prefix = "id"
	}
	if start <= 0 {
		start = 1
	}
	return &DeterministicIDGen{prefix: prefix, next: start}
}

func (g *DeterministicIDGen) Next() string {
	id := fmt.Sprintf("%s_%06d", g.prefix, g.next)
	g.next++
	return id
}

func StableRunID(seed string) string {
	norm := normalizeID(seed)
	if norm == "" {
		norm = "demo"
	}
	return "run_" + norm
}

func normalizeID(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	return out
}
