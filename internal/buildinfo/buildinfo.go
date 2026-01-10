// Package buildinfo contains build-time information set via ldflags.
package buildinfo

import "runtime/debug"

// These variables are set at build time via ldflags.
// Example: go build -ldflags "-X go.carr.sh/litmus/internal/buildinfo.Version=v1.0.0 -X go.carr.sh/litmus/internal/buildinfo.Commit=abc123"
var (
	// Version is the semantic version of the build.
	Version = "dev"

	// Commit is the git commit hash of the build.
	Commit = "unknown"
)

func init() {
	// If ldflags weren't set (e.g., when using `go install module@version`),
	// try to get version info from the embedded build info.
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			// When installed via `go install module@version`, the Main.Version
			// will be the version (e.g., "v1.0.0" or "v1.0.0-0.20210101000000-abcdef123456").
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				Version = info.Main.Version
			}

			// Try to get the VCS revision from build settings.
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" && Commit == "unknown" {
					Commit = setting.Value
				}
			}
		}
	}
}

// String returns a formatted version string.
func String() string {
	if Commit == "unknown" || len(Commit) < 7 {
		return Version
	}
	return Version + " (" + Commit[:7] + ")"
}
