package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stan-ley-tech/ByteForge/internal/collections"
)

// newExportCommand validates a collection file and re-serializes it as
// normalized, pretty-printed JSON — the same codec path used everywhere
// else in ByteForge, so a file that passes `export` is guaranteed to load
// cleanly in the UI, the API, and `run`/`test`.
func newExportCommand() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "export <collection.json>",
		Short: "Validate a collection and write it back out as normalized JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return exportCollection(cmd, args[0], output)
		},
	}
	cmd.Flags().StringVar(&output, "output", "", "write to this path instead of stdout")
	return cmd
}

func exportCollection(cmd *cobra.Command, path, output string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open collection: %w", err)
	}
	defer f.Close()

	coll, err := collections.Decode(f)
	if err != nil {
		return fmt.Errorf("load collection: %w", err)
	}

	w := cmd.OutOrStdout()
	if output != "" {
		out, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer out.Close()
		w = out
	}

	return collections.Encode(w, coll)
}
