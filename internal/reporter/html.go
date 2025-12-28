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
	w io.Writer
}

// NewHTML creates a new HTML reporter.
func NewHTML(w io.Writer) *HTML {
	return &HTML{w: w}
}

// templateData holds all data passed to the HTML template.
type templateData struct {
	Report      *types.RunReport
	GeneratedAt string
}

// Report outputs the complete run report as HTML.
func (h *HTML) Report(report *types.RunReport) error {
	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"json": func(v any) string {
			b, _ := json.MarshalIndent(v, "", "  ")
			return string(b)
		},
		"formatDuration": formatDurationHTML,
		"accuracyClass": func(acc float64) string {
			if acc >= 90 {
				return "success"
			} else if acc >= 70 {
				return "warning"
			}
			return "error"
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

func formatDurationHTML(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
