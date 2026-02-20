package runlog

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"time"

	"orchschema/internal/schema"
)

func TestWriterReaderRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf, "run_test_001", 1)

	evt1 := schema.Event{
		Type:  schema.EventRunStarted,
		Actor: schema.Actor{Kind: schema.ActorSystem, ID: "orch"},
		TSUTC: time.Date(2026, time.January, 1, 0, 0, 1, 0, time.UTC).Format(time.RFC3339Nano),
	}
	if err := evt1.SetPayload(schema.RunStartedPayload{
		SchemaVersion: schema.CurrentSchemaVersion,
		CreatedBy:     "tester",
		CWD:           "/tmp",
		Host:          "localhost",
		Platform:      "test",
		ModelConfig: schema.ModelConfig{
			Provider:        schema.ModelProviderLocalOpenAICompat,
			BaseURL:         "http://127.0.0.1:11434/v1",
			Model:           "glm-4.7",
			MaxTokens:       128,
			Stream:          true,
			HeadersRedacted: map[string]string{"Authorization": schema.RedactedValue},
		},
	}); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := writer.Append(&evt1); err != nil {
		t.Fatalf("append evt1: %v", err)
	}

	evt2 := schema.Event{
		Type:  schema.EventRunCompleted,
		Actor: schema.Actor{Kind: schema.ActorSystem, ID: "orch"},
		TSUTC: time.Date(2026, time.January, 1, 0, 0, 2, 0, time.UTC).Format(time.RFC3339Nano),
	}
	if err := evt2.SetPayload(schema.RunCompletedPayload{Reason: "done"}); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := writer.Append(&evt2); err != nil {
		t.Fatalf("append evt2: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	reader := NewReader(bytes.NewReader(buf.Bytes()))
	var got []schema.Event
	for {
		evt, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reader next: %v", err)
		}
		got = append(got, evt)
	}

	want := []schema.Event{evt1, evt2}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("events mismatch\nwant=%+v\ngot=%+v", want, got)
	}
}
