package projector

import (
	"fmt"

	"orchschema/internal/schema"
)

type Projector struct {
	state *RunState
}

func New() *Projector {
	return &Projector{state: NewRunState()}
}

func (p *Projector) State() *RunState {
	return p.state
}

func (p *Projector) Apply(evt schema.Event) error {
	if err := schema.ValidateEvent(evt); err != nil {
		return err
	}
	if err := schema.ValidateSeqTransition(p.state.LastSeq, evt.Seq); err != nil {
		return err
	}

	switch evt.Type {
	case schema.EventRunStarted:
		var payload schema.RunStartedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		p.state.Header = payload
		p.state.HasHeader = true
		p.state.Status = schema.RunStatusRunning

	case schema.EventRunCompleted:
		p.state.Status = schema.RunStatusCompleted

	case schema.EventRunFailed:
		p.state.Status = schema.RunStatusFailed

	case schema.EventAgentRegistered:
		var payload schema.AgentRegisteredPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		p.state.Agents[payload.Agent.AgentID] = payload.Agent

	case schema.EventTaskPlanned:
		var payload schema.TaskPlannedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		p.state.Tasks[payload.Task.TaskID] = payload.Task

	case schema.EventTaskStarted:
		var payload schema.TaskStartedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		t := p.getTask(payload.TaskID)
		t.Status = schema.TaskStatusInProgress
		if payload.AgentID != "" {
			t.AgentID = payload.AgentID
		}
		p.state.Tasks[t.TaskID] = t

	case schema.EventTaskCompleted:
		var payload schema.TaskCompletedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		t := p.getTask(payload.TaskID)
		t.Status = schema.TaskStatusDone
		p.state.Tasks[t.TaskID] = t

	case schema.EventTaskFailed:
		var payload schema.TaskFailedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		t := p.getTask(payload.TaskID)
		t.Status = schema.TaskStatusFailed
		p.state.Tasks[t.TaskID] = t

	case schema.EventModelRequestStarted:
		var payload schema.ModelRequestStartedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		p.state.ActiveModelRequests[payload.Request.RequestID] = payload.Request

	case schema.EventModelResponseCompleted:
		var payload schema.ModelResponseCompletedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		delete(p.state.ActiveModelRequests, payload.Response.RequestID)

	case schema.EventModelResponseFailed:
		var payload schema.ModelResponseFailedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		delete(p.state.ActiveModelRequests, payload.RequestID)

	case schema.EventToolCallRequested:
		var payload schema.ToolCallRequestedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		p.state.ActiveToolCalls[payload.ToolCall.ToolCallID] = payload.ToolCall

	case schema.EventToolCallCompleted:
		var payload schema.ToolCallCompletedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		delete(p.state.ActiveToolCalls, payload.ToolResult.ToolCallID)
		for _, art := range payload.ToolResult.ProducedArtifacts {
			p.state.Artifacts[art.ArtifactID] = art
		}

	case schema.EventToolCallFailed:
		var payload schema.ToolCallFailedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		delete(p.state.ActiveToolCalls, payload.ToolCallID)

	case schema.EventApprovalRequested:
		var payload schema.ApprovalRequestedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		p.state.Approvals[payload.Approval.ApprovalID] = ApprovalState{
			Request: payload.Approval,
			Status:  schema.ApprovalStatusRequested,
		}

	case schema.EventApprovalGranted:
		var payload schema.ApprovalGrantedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		curr := p.state.Approvals[payload.ApprovalID]
		curr.Status = schema.ApprovalStatusGranted
		curr.DecidedBy = payload.GrantedBy
		curr.Note = payload.Note
		p.state.Approvals[payload.ApprovalID] = curr

	case schema.EventApprovalDenied:
		var payload schema.ApprovalDeniedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		curr := p.state.Approvals[payload.ApprovalID]
		curr.Status = schema.ApprovalStatusDenied
		curr.DecidedBy = payload.DeniedBy
		curr.Note = payload.Note
		p.state.Approvals[payload.ApprovalID] = curr

	case schema.EventArtifactWritten:
		var payload schema.ArtifactWrittenPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		p.state.Artifacts[payload.Artifact.ArtifactID] = payload.Artifact

	case schema.EventArtifactUpdated:
		var payload schema.ArtifactUpdatedPayload
		if err := evt.DecodePayload(&payload); err != nil {
			return decodeErr(evt.Type, err)
		}
		p.state.Artifacts[payload.Artifact.ArtifactID] = payload.Artifact

	case schema.EventModelToken, schema.EventToolCallStarted, schema.EventToolStdoutChunk,
		schema.EventToolStderrChunk, schema.EventNote, schema.EventWarning, schema.EventError:
		// No durable state transition for these events in v0.
	}

	p.state.LastSeq = evt.Seq
	return nil
}

func Replay(events []schema.Event) (*RunState, error) {
	p := New()
	for _, evt := range events {
		if err := p.Apply(evt); err != nil {
			return nil, err
		}
	}
	return p.State(), nil
}

func decodeErr(eventType schema.EventType, err error) error {
	return fmt.Errorf("decode payload for %s: %w", eventType, err)
}

func (p *Projector) getTask(taskID string) schema.TaskSpec {
	if task, ok := p.state.Tasks[taskID]; ok {
		return task
	}
	return schema.TaskSpec{
		TaskID: taskID,
		Status: schema.TaskStatusPlanned,
	}
}
