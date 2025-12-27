// Package cli provides the command-line interface for litmus.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "litmus",
	Short: "Specification testing for structured LLM outputs",
	Long: `Litmus is a CLI tool for testing structured LLM outputs against expected values.

It allows you to:
  - Define test cases with input strings and expected JSON outputs
  - Run tests against multiple LLM models via OpenRouter
  - Compare accuracy, latency, throughput, and cost across models
  - Get detailed field-level diff reports for failures`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// ErrTestsFailed means tests ran but some failed - results already printed
		if errors.Is(err, ErrTestsFailed) {
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
}
