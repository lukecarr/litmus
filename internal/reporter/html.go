package reporter

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"time"

	"go.carr.sh/litmus/internal/types"
)

//go:embed template.html
var htmlTemplate string

// HTML outputs results as a self-contained HTML file.
type HTML struct {
	// w is the writer to output the report to.
	w io.Writer
}

// NewHTML creates a new HTML reporter.
func NewHTML(w io.Writer) *HTML {
	return &HTML{w: w}
}

// templateData holds all data passed to the HTML template.
type templateData struct {
	Report *types.RunReport
	// GeneratedAt is the time the report was generated, which may differ from
	// Report.Timestamp (when the tests were run) if reports are generated later.
	GeneratedAt string
}

// Report outputs the complete run report as HTML.
func (h *HTML) Report(report *types.RunReport) error {
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"json": func(v any) string {
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return fmt.Sprintf("error marshaling JSON: %v", err)
			}
			return string(b)
		},
		"formatDuration": formatDuration,
		"accuracyClass": func(acc float64) string {
			if acc >= 90 {
				return "success"
			} else if acc >= 70 {
				return "warning"
			}
			return "error"
		},
		"add": func(a, b int) int {
			return a + b
		},
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	data := templateData{
		Report:      report,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	if err := tmpl.Execute(h.w, data); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return nil
}
