// Package cli implements the byteforge command-line tool: the same
// collections and environments the web UI builds can be run headlessly,
// which is what makes ByteForge usable as a CI gate rather than only an
// interactive client.
package cli

import "github.com/spf13/cobra"

// Execute builds and runs the root command, returning any error from the
// command that ran (cmd/byteforge's main.go turns that into an exit code).
func Execute() error {
	return newRootCommand().Execute()
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "byteforge",
		Short: "ByteForge is an API testing and debugging tool for developers",
		Long: "ByteForge is an API testing and debugging tool for developers.\n" +
			"It runs the same request collections you build in the desktop and web\n" +
			"UI from the command line, so an API test suite can run locally and in CI.",
		SilenceUsage: true,
	}

	root.AddCommand(newServeCommand())
	root.AddCommand(newRunCommand())
	root.AddCommand(newTestCommand())
	root.AddCommand(newExportCommand())
	return root
}
