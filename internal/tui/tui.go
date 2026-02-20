package tui

import (
	"fmt"
	"strings"

	"orchschema/internal/projector"
	"orchschema/internal/schema"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			Width(100).
			Align(lipgloss.Center)

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	labelStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	statusRunning = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("RUNNING")
	statusDone    = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render("COMPLETED")

	taskPlannedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	taskInProgressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	taskDoneStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	eventTickerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

type Model struct {
	tailer *Tailer
	proj   *projector.Projector

	width  int
	height int

	events    []schema.Event // Keep last N events for ticker
	llmStream string         // the current streaming token string
	streaming bool           // whether a model stream is actively happening

	ready bool
}

func NewModel(tailer *Tailer) Model {
	return Model{
		tailer: tailer,
		proj:   projector.New(),
		events: make([]schema.Event, 0, 50),
	}
}

func (m Model) Init() tea.Cmd {
	return pollTick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.tailer.Close()
			return m, tea.Quit
		}

	case TickMsg:
		// Read new lines
		tailCmds := m.tailer.ReadAvailable()
		cmds = append(cmds, tailCmds...)
		cmds = append(cmds, pollTick()) // schedule next poll

	case EventMsg:
		// Process event into projector
		m.proj.Apply(msg.Event)

		// Add to recent events ring buffer (max 10)
		m.events = append(m.events, msg.Event)
		if len(m.events) > 10 {
			m.events = m.events[1:]
		}

		// Handle streaming display logic
		switch msg.Event.Type {
		case schema.EventModelRequestStarted:
			m.streaming = true
			m.llmStream = ""
		case schema.EventModelToken:
			var payload schema.ModelTokenPayload
			if err := msg.Event.DecodePayload(&payload); err == nil {
				m.llmStream += payload.Delta
			}
		case schema.EventModelResponseCompleted, schema.EventModelResponseFailed:
			m.streaming = false
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing Dashboard...\n"
	}

	state := m.proj.State()
	runID := "Waiting for RunStarted event..."
	status := ""
	if state.HasHeader {
		runID = state.Header.CreatedBy + " / " + m.events[0].RunID // rough estimate if present
		if state.Status == schema.RunStatusRunning {
			status = statusRunning
		} else if state.Status == schema.RunStatusCompleted {
			status = statusDone
		}
	}

	header := headerStyle.Render(fmt.Sprintf("\n%s\n%s • Tasks: %d • Agents: %d\n",
		runID, status, len(state.Tasks), len(state.Agents)))

	// Left pane: Tasks
	tasksView := m.renderTasks(state)

	// Right pane: LLM Stream / Tools
	streamView := m.renderStream(state)

	// Sub-layout top: Tasks | Stream
	topHeight := m.height - 15
	if topHeight < 10 {
		topHeight = 10
	}

	leftWidth := m.width / 3
	rightWidth := m.width - leftWidth - 4

	leftBox := paneStyle.Width(leftWidth).Height(topHeight).Render("TASKS\n\n" + tasksView)
	rightBox := paneStyle.Width(rightWidth).Height(topHeight).Render("MODEL / TOOL STREAM\n\n" + streamView)

	topSection := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)

	// Bottom pane: Event Ticker
	tickerHeight := 6
	tickerView := m.renderTicker()
	bottomBox := paneStyle.Width(m.width - 2).Height(tickerHeight).Render("EVENT TICKER\n\n" + tickerView)

	return lipgloss.JoinVertical(lipgloss.Left, header, topSection, bottomBox)
}

func (m Model) renderTasks(state *projector.RunState) string {
	var out strings.Builder
	for _, t := range state.Tasks {
		prefix := "[ ]"
		style := taskPlannedStyle
		if t.Status == schema.TaskStatusInProgress {
			prefix = "[~]"
			style = taskInProgressStyle
		} else if t.Status == schema.TaskStatusDone {
			prefix = "[x]"
			style = taskDoneStyle
		}

		out.WriteString(style.Render(fmt.Sprintf("%s %s\n    %s\n\n", prefix, t.Title, t.Description)))
	}
	if len(state.Tasks) == 0 {
		out.WriteString("No tasks planned yet.")
	}
	return out.String()
}

func (m Model) renderStream(state *projector.RunState) string {
	var out strings.Builder

	if len(state.ActiveToolCalls) > 0 {
		out.WriteString(labelStyle.Render("Active Tool Calls:\n"))
		for _, tc := range state.ActiveToolCalls {
			out.WriteString(fmt.Sprintf("- %s: %s\n", tc.ToolName, tc.ArgsJSON))
		}
		out.WriteString("\n")
	}

	if len(state.Approvals) > 0 {
		var active bool
		for _, a := range state.Approvals {
			if a.Status == schema.ApprovalStatusRequested {
				if !active {
					out.WriteString(labelStyle.Render("AWAITING APPROVAL\n"))
					active = true
				}
				out.WriteString(fmt.Sprintf("Risk: %s | %s\n", a.Request.Risk, a.Request.ActionSummary))
			}
		}
		if active {
			out.WriteString("\n")
		}
	}

	if m.streaming {
		out.WriteString(labelStyle.Render("Thinking...\n"))
		// naive wrapping
		out.WriteString(m.llmStream)
	} else {
		out.WriteString("Idle. Waiting for next action.")
	}

	return out.String()
}

func (m Model) renderTicker() string {
	var out strings.Builder
	for _, e := range m.events {
		t, _ := e.Timestamp()
		out.WriteString(eventTickerStyle.Render(fmt.Sprintf("[%s] %-25s %s\n",
			t.Format("15:04:05.000"),
			e.Type,
			e.Actor.ID)))
	}
	return out.String()
}
