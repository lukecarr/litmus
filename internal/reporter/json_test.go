package reporter

import (
	"bytes"
	"encoding/json"
	"testing"

	"go.carr.sh/litmus/internal/types"
)

func TestJSONReport(t *testing.T) {
	var buf bytes.Buffer
	if err := NewJSON(&buf).Report(sampleReport()); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	var got types.RunReport
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}

	if got.TestFile != "example/tests.json" {
		t.Errorf("TestFile = %q, want example/tests.json", got.TestFile)
	}
	if len(got.Models) != 1 || got.Models[0].Model != "openai/gpt-4o" {
		t.Fatalf("Models = %+v, want one openai/gpt-4o model", got.Models)
	}
	if got.Models[0].Metrics.Passed != 1 {
		t.Errorf("Passed = %d, want 1", got.Models[0].Metrics.Passed)
	}
}

func TestJSONReportWriteError(t *testing.T) {
	if err := NewJSON(failingWriter{}).Report(sampleReport()); err == nil {
		t.Fatal("expected an error from a failing writer, got nil")
	}
}
