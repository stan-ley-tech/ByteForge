package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/stan-ley-tech/ByteForge/internal/collections"
	"github.com/stan-ley-tech/ByteForge/internal/httpclient"
	"github.com/stan-ley-tech/ByteForge/internal/runner"
)

type runFlags struct {
	envPath       string
	vars          []string
	timeout       time.Duration
	stopOnFailure bool
}

func addRunFlags(cmd *cobra.Command, flags *runFlags) {
	cmd.Flags().StringVar(&flags.envPath, "env", "", "path to an environment JSON file")
	cmd.Flags().StringArrayVar(&flags.vars, "var", nil, "set or override a variable as KEY=VALUE (repeatable)")
	cmd.Flags().DurationVar(&flags.timeout, "timeout", 30*time.Second, "per-request timeout")
	cmd.Flags().BoolVar(&flags.stopOnFailure, "stop-on-failure", false, "stop the chain at the first failed request")
}

// newRunCommand runs a collection and always exits 0 (assuming the
// collection itself loaded and executed), for exploring results locally
// without a failure aborting a shell script.
func newRunCommand() *cobra.Command {
	var flags runFlags
	cmd := &cobra.Command{
		Use:   "run <collection.json>",
		Short: "Run a collection and print the results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := executeCollection(cmd, args[0], flags)
			return err
		},
	}
	addRunFlags(cmd, &flags)
	return cmd
}

// newTestCommand runs a collection and exits 1 if any assertion failed,
// which is what makes `byteforge test` usable as a CI gate.
func newTestCommand() *cobra.Command {
	var flags runFlags
	cmd := &cobra.Command{
		Use:   "test <collection.json>",
		Short: "Run a collection as a CI gate (fails the build on any failed assertion)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := executeCollection(cmd, args[0], flags)
			if err != nil {
				return err
			}
			if !report.AllPassed() {
				return fmt.Errorf("%d of %d assertions failed", report.Failed, report.Passed+report.Failed)
			}
			return nil
		},
	}
	addRunFlags(cmd, &flags)
	return cmd
}

func executeCollection(cmd *cobra.Command, path string, flags runFlags) (*runner.Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open collection: %w", err)
	}
	defer f.Close()

	coll, err := collections.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("load collection: %w", err)
	}

	env, err := loadEnvironment(flags.envPath, flags.vars)
	if err != nil {
		return nil, err
	}

	rn := runner.New(httpclient.New(httpclient.DefaultConfig()))

	report, runErr := rn.Run(context.Background(), coll, env, runner.Options{
		StopOnFailure:  flags.stopOnFailure,
		RequestTimeout: flags.timeout,
	})
	if report != nil {
		printReport(cmd.OutOrStdout(), report)
	}
	if runErr != nil {
		return report, fmt.Errorf("run aborted: %w", runErr)
	}
	return report, nil
}
