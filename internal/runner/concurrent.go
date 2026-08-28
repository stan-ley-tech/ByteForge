package runner

import (
	"context"
	"sync"

	"github.com/stan-ley-tech/ByteForge/internal/collections"
	"github.com/stan-ley-tech/ByteForge/internal/environments"
)

// RunConcurrent executes a batch of independent requests (no chaining
// between them — each gets its own empty variable scope) in parallel,
// bounded by concurrency in-flight at once. This is the tool for smoke
// testing a set of unrelated endpoints quickly, as opposed to Run, which
// executes a collection sequentially so later steps can depend on earlier
// ones.
//
// Results are returned in the same order as reqs regardless of completion
// order. Cancelling ctx stops in-flight requests and prevents new ones from
// starting; already-completed results are still returned.
func (rn *Runner) RunConcurrent(ctx context.Context, reqs []collections.Request, env *environments.Environment, opts Options, concurrency int) []StepResult {
	if concurrency < 1 {
		concurrency = 1
	}

	results := make([]StepResult, len(reqs))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, req := range reqs {
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(i int, req collections.Request) {
			defer wg.Done()
			defer func() { <-sem }()

			step, _ := rn.runStep(ctx, req, env, make(map[string]string), opts)
			results[i] = step
		}(i, req)
	}

	wg.Wait()
	return results
}
