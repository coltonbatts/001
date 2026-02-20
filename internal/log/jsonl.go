package runlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"orchschema/internal/schema"
)

const defaultScannerMaxBytes = 4 * 1024 * 1024

type Writer struct {
	mu      sync.Mutex
	w       *bufio.Writer
	runID   string
	nextSeq int64
	closed  bool
}

func NewWriter(w io.Writer, runID string, nextSeq int64) *Writer {
	if nextSeq <= 0 {
		nextSeq = 1
	}
	return &Writer{
		w:       bufio.NewWriter(w),
		runID:   runID,
		nextSeq: nextSeq,
	}
}

func (w *Writer) Append(evt *schema.Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return fmt.Errorf("writer is closed")
	}
	if evt == nil {
		return fmt.Errorf("event is nil")
	}

	if evt.RunID == "" {
		evt.RunID = w.runID
	}
	if w.runID != "" && evt.RunID != w.runID {
		return fmt.Errorf("run_id mismatch: writer=%s event=%s", w.runID, evt.RunID)
	}

	if evt.Seq == 0 {
		evt.Seq = w.nextSeq
	}
	if evt.Seq != w.nextSeq {
		return fmt.Errorf("unexpected seq: got=%d want=%d", evt.Seq, w.nextSeq)
	}

	if evt.EventID == "" {
		evt.EventID = fmt.Sprintf("evt_%d", evt.Seq)
	}
	if evt.TSUTC == "" {
		evt.TSUTC = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if len(evt.Payload) == 0 {
		evt.Payload = json.RawMessage(`{}`)
	}

	if err := schema.ValidateEvent(*evt); err != nil {
		return err
	}

	line, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if _, err := w.w.Write(line); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	if err := w.w.WriteByte('\n'); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}
	w.nextSeq++
	return nil
}

func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.Flush()
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	return w.w.Flush()
}

type Reader struct {
	scanner *bufio.Scanner
	line    int
}

func NewReader(r io.Reader) *Reader {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), defaultScannerMaxBytes)
	return &Reader{scanner: scanner}
}

func (r *Reader) Next() (schema.Event, error) {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return schema.Event{}, err
		}
		return schema.Event{}, io.EOF
	}
	r.line++
	var evt schema.Event
	if err := json.Unmarshal(r.scanner.Bytes(), &evt); err != nil {
		return schema.Event{}, fmt.Errorf("line %d: %w", r.line, err)
	}
	if err := schema.ValidateEvent(evt); err != nil {
		return schema.Event{}, fmt.Errorf("line %d: %w", r.line, err)
	}
	return evt, nil
}

func ReadAll(r io.Reader) ([]schema.Event, error) {
	reader := NewReader(r)
	var out []schema.Event
	for {
		evt, err := reader.Next()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, evt)
	}
}
