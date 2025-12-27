package reporter

import (
	"encoding/json"
	"fmt"
	"io"

	"go.carr.sh/litmus/internal/types"
)

// JSON outputs results as JSON.
type JSON struct {
	// w is the writer to output the report to.
	w io.Writer
}

// NewJSON creates a new JSON reporter.
func NewJSON(w io.Writer) *JSON {
	return &JSON{w: w}
}

// Report outputs the complete run report as JSON.
func (j *JSON) Report(report *types.RunReport) error {
	encoder := json.NewEncoder(j.w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("failed to encode JSON report: %w", err)
	}

	return nil
}
