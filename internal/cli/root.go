// Package cli provides the command-line interface for litmus.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"go.carr.sh/litmus/internal/buildinfo"
)

var rootCmd = &cobra.Command{
	Use:   "litmus",
	Short: "Specification testing for structured LLM outputs",
	Long: `Litmus is a CLI tool for testing structured LLM outputs against expected values.

It allows you to:
  - Define test cases with input strings and expected JSON outputs
  - Run tests against multiple LLM models via OpenRouter or Cloudflare AI Gateway
  - Compare accuracy, latency, throughput, and cost across models
  - Get detailed field-level diff reports for failures`,
}

// Execute runs the root command.
func Execute() {
	os.Exit(execute(rootCmd, os.Args[1:], os.Stderr))
}

// execute runs cmd with the given args and returns the process exit code,
// printing any non-test error to stderr.
func execute(cmd *cobra.Command, args []string, stderr io.Writer) int {
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		// ErrTestsFailed means tests ran but some failed - results already printed.
		if errors.Is(err, ErrTestsFailed) {
			return 1
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("litmus", buildinfo.String())
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
}
