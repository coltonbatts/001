package projector

import "orchschema/internal/schema"

type ApprovalState struct {
	Request   schema.ApprovalRequest
	Status    schema.ApprovalDecisionStatus
	DecidedBy string
	Note      string
}

type RunState struct {
	Header              schema.RunStartedPayload
	HasHeader           bool
	Agents              map[string]schema.AgentSpec
	Tasks               map[string]schema.TaskSpec
	ActiveModelRequests map[string]schema.ModelRequest
	ActiveToolCalls     map[string]schema.ToolCall
	Approvals           map[string]ApprovalState
	Artifacts           map[string]schema.ArtifactRef
	Status              schema.RunLifecycleStatus
	LastSeq             int64
}

func NewRunState() *RunState {
	return &RunState{
		Agents:              make(map[string]schema.AgentSpec),
		Tasks:               make(map[string]schema.TaskSpec),
		ActiveModelRequests: make(map[string]schema.ModelRequest),
		ActiveToolCalls:     make(map[string]schema.ToolCall),
		Approvals:           make(map[string]ApprovalState),
		Artifacts:           make(map[string]schema.ArtifactRef),
	}
}
