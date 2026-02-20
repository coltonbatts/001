package tui

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"time"

	"orchschema/internal/schema"

	tea "github.com/charmbracelet/bubbletea"
)

type EventMsg struct {
	Event schema.Event
}

type TailErrorMsg struct {
	Err error
}

// TickMsg used to frequently check for new lines
type TickMsg time.Time

func pollTick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

type Tailer struct {
	file   *os.File
	reader *bufio.Reader
}

func NewTailer(path string) (*Tailer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// Start at the beginning
	return &Tailer{
		file:   f,
		reader: bufio.NewReader(f),
	}, nil
}

// ReadAvailable reads all available complete lines and returns them as tea.Cmds
func (t *Tailer) ReadAvailable() []tea.Cmd {
	var cmds []tea.Cmd
	for {
		line, err := t.reader.ReadBytes('\n')
		if len(line) > 0 {
			// We have a full line, even if EOF was hit right after
			var evt schema.Event
			if parseErr := json.Unmarshal(line, &evt); parseErr == nil {
				// We don't strictly care about validation errors for the live dashboard,
				// as long as we can parse the JSON into the event structure.
				cmds = append(cmds, func(e schema.Event) tea.Cmd {
					return func() tea.Msg { return EventMsg{Event: e} }
				}(evt))
			}
		}

		if err != nil {
			if err == io.EOF {
				// No more complete lines available right now
				break
			}
			// Let the top layer handle real errors if needed, but for tailing we often just retry
			break
		}
	}
	return cmds
}

func (t *Tailer) Close() {
	if t.file != nil {
		t.file.Close()
	}
}
