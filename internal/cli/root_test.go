package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newStubCmd(runErr error) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "stub",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE:          func(*cobra.Command, []string) error { return runErr },
	}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	return cmd
}

func TestExecute(t *testing.T) {
	t.Run("success returns 0", func(t *testing.T) {
		var stderr bytes.Buffer
		if code := execute(newStubCmd(nil), nil, &stderr); code != 0 {
			t.Errorf("code = %d, want 0", code)
		}
		if stderr.Len() != 0 {
			t.Errorf("stderr = %q, want empty", stderr.String())
		}
	})

	t.Run("test failure returns 1 without printing", func(t *testing.T) {
		var stderr bytes.Buffer
		if code := execute(newStubCmd(ErrTestsFailed), nil, &stderr); code != 1 {
			t.Errorf("code = %d, want 1", code)
		}
		if stderr.Len() != 0 {
			t.Errorf("stderr = %q, want empty for ErrTestsFailed", stderr.String())
		}
	})

	t.Run("generic error returns 1 and prints", func(t *testing.T) {
		var stderr bytes.Buffer
		if code := execute(newStubCmd(errors.New("boom")), nil, &stderr); code != 1 {
			t.Errorf("code = %d, want 1", code)
		}
		if !strings.Contains(stderr.String(), "boom") {
			t.Errorf("stderr = %q, want it to contain boom", stderr.String())
		}
	})
}

func TestVersionCommand(t *testing.T) {
	var out bytes.Buffer
	versionCmd.SetOut(&out)
	versionCmd.Run(versionCmd, nil)
}
