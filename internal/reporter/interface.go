package reporter

import "go.carr.sh/litmus/internal/types"

// Reporter is the interface for reporting test results.
type Reporter interface {
	Report(report *types.RunReport) error
}
