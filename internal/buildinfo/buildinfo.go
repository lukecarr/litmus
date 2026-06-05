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
	info, ok := debug.ReadBuildInfo()
	Version, Commit = resolve(Version, Commit, info, ok)
}

// resolve derives version and commit from embedded build info when ldflags
// weren't set (e.g., when using `go install module@version`). Values already
// provided via ldflags are left untouched.
func resolve(version, commit string, info *debug.BuildInfo, ok bool) (string, string) {
	if version != "dev" || !ok {
		return version, commit
	}

	// When installed via `go install module@version`, Main.Version holds the
	// version (e.g., "v1.0.0" or "v1.0.0-0.20210101000000-abcdef123456").
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}

	// Try to get the VCS revision from build settings.
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && commit == "unknown" {
			commit = setting.Value
			break
		}
	}

	return version, commit
}

// String returns a formatted version string.
func String() string {
	if Commit == "unknown" || len(Commit) < 7 {
		return Version
	}
	return Version + " (" + Commit[:7] + ")"
}
