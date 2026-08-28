package cli

import (
	"fmt"
	"io"

	"github.com/stan-ley-tech/ByteForge/internal/runner"
)

// printReport renders a run the way a developer reads it in a terminal:
// one line per request, one line per assertion under it, then a summary
// line CI logs can grep for.
func printReport(w io.Writer, report *runner.Report) {
	for _, step := range report.Steps {
		fmt.Fprintf(w, "\n%s %s", step.Method, step.URL)
		if step.Error != "" {
			fmt.Fprintf(w, "\n  ✗ %s\n", step.Error)
			continue
		}
		fmt.Fprintf(w, " -> %d (%s)\n", step.Status, step.Duration)

		for _, a := range step.Assertions {
			mark := "✓"
			if !a.Passed {
				mark = "✗"
			}
			fmt.Fprintf(w, "  %s %s\n", mark, a.Message)
		}
	}

	fmt.Fprintf(w, "\n%d/%d PASSED", report.Passed, report.Passed+report.Failed)
	if report.Failed > 0 {
		fmt.Fprintf(w, " (%d failed)", report.Failed)
	}
	fmt.Fprintln(w)
}
