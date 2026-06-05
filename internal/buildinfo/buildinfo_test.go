package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestResolve(t *testing.T) {
	withRevision := &debug.BuildInfo{
		Main:     debug.Module{Version: "v2.0.0"},
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "deadbeef"}},
	}

	cases := []struct {
		name        string
		version     string
		commit      string
		info        *debug.BuildInfo
		ok          bool
		wantVersion string
		wantCommit  string
	}{
		{
			name: "ldflags version is left untouched", version: "v1.2.3", commit: "abc1234",
			info: withRevision, ok: true,
			wantVersion: "v1.2.3", wantCommit: "abc1234",
		},
		{
			name: "no build info available", version: "dev", commit: "unknown",
			info: nil, ok: false,
			wantVersion: "dev", wantCommit: "unknown",
		},
		{
			name: "version and revision from build info", version: "dev", commit: "unknown",
			info: withRevision, ok: true,
			wantVersion: "v2.0.0", wantCommit: "deadbeef",
		},
		{
			name: "devel version is ignored", version: "dev", commit: "unknown",
			info:        &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			ok:          true,
			wantVersion: "dev", wantCommit: "unknown",
		},
		{
			name: "empty version is ignored", version: "dev", commit: "unknown",
			info:        &debug.BuildInfo{Main: debug.Module{Version: ""}},
			ok:          true,
			wantVersion: "dev", wantCommit: "unknown",
		},
		{
			name: "existing commit is not overwritten", version: "dev", commit: "mycommit",
			info: withRevision, ok: true,
			wantVersion: "v2.0.0", wantCommit: "mycommit",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotVersion, gotCommit := resolve(tc.version, tc.commit, tc.info, tc.ok)
			if gotVersion != tc.wantVersion || gotCommit != tc.wantCommit {
				t.Errorf("resolve(%q, %q) = (%q, %q), want (%q, %q)",
					tc.version, tc.commit, gotVersion, gotCommit, tc.wantVersion, tc.wantCommit)
			}
		})
	}
}

func TestString(t *testing.T) {
	origVersion, origCommit := Version, Commit
	t.Cleanup(func() { Version, Commit = origVersion, origCommit })

	cases := []struct {
		name    string
		version string
		commit  string
		want    string
	}{
		{name: "unknown commit", version: "v1.0.0", commit: "unknown", want: "v1.0.0"},
		{name: "short commit", version: "v1.0.0", commit: "abc12", want: "v1.0.0"},
		{name: "full commit", version: "v1.0.0", commit: "abcdef1234", want: "v1.0.0 (abcdef1)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			Version, Commit = tc.version, tc.commit
			if got := String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}
