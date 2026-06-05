package reporter

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
)

// failingWriter returns an error on every write.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestWriteJobSummaryOpenError(t *testing.T) {
	// A path whose parent directory does not exist cannot be opened for append.
	missing := filepath.Join(t.TempDir(), "no-such-dir", "summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", missing)

	if err := NewGitHub(&bytes.Buffer{}).Report(sampleReport()); err == nil {
		t.Fatal("expected an error when GITHUB_STEP_SUMMARY cannot be opened, got nil")
	}
}

func TestWriteSummaryTableWriteError(t *testing.T) {
	if err := writeSummaryTable(failingWriter{}, sampleReport()); err == nil {
		t.Fatal("expected an error from a failing writer, got nil")
	}
}
