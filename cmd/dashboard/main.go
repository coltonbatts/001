package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"orchschema/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var logPath string
	flag.StringVar(&logPath, "log", "/tmp/orchschema_demo_run.jsonl", "path to the jsonl event log file")
	flag.Parse()

	tailer, err := tui.NewTailer(logPath)
	if err != nil {
		fmt.Printf("Error opening log file %s:\n%v\n\nIf the file does not exist, run the demo driver first:\n  go run ./cmd/orchschema\n", logPath, err)
		os.Exit(1)
	}
	defer tailer.Close()

	p := tea.NewProgram(tui.NewModel(tailer), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI Error: %v", err)
	}
}
