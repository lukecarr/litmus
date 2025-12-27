// Package buildinfo contains build-time information set via ldflags.
package buildinfo

// These variables are set at build time via ldflags.
// Example: go build -ldflags "-X go.carr.sh/litmus/internal/buildinfo.Version=v1.0.0 -X go.carr.sh/litmus/internal/buildinfo.Commit=abc123"
var (
	// Version is the semantic version of the build.
	Version = "dev"

	// Commit is the git commit hash of the build.
	Commit = "unknown"
)

// String returns a formatted version string.
func String() string {
	if Commit == "unknown" || len(Commit) < 7 {
		return Version
	}
	return Version + " (" + Commit[:7] + ")"
}

