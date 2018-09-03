package workflow

import (
	"context"
	"sync"

	"github.com/ztstewart/workflow/internal/atomic"
)

// executionCtx holds everything needed to execute a Graph.
// The Graph type can be thought of as stateless, whereas the
// executionCtx type can be thought of as mutable. This allows the Graph
// to be executed multiple times without needing any external cleanup, so long
// as the caller's tasks are idempotent.
type executionCtx struct {
	wg            sync.WaitGroup
	g             Graph
	taskToNumdeps map[string]*atomic.Int32
	err           error
	ctx           context.Context
	cancel        context.CancelFunc
	errCounter    *atomic.Int32 // There are no atomic errors, sadly
}

func newExecutionCtx(ctx context.Context, g Graph) *executionCtx {
	iCtx, cancel := context.WithCancel(ctx)

	return &executionCtx{
		taskToNumdeps: make(map[string]*atomic.Int32, len(g.tasks)),
		g:             g,
		ctx:           iCtx,
		cancel:        cancel,
		errCounter:    atomic.NewInt32(0),
	}
}

func (ec *executionCtx) run() error {
	for _, t := range ec.g.tasks {
		ec.taskToNumdeps[t.name] = atomic.NewInt32(int32(len(t.deps)))
	}

	for _, t := range ec.g.tasks {
		// When a task has no dependencies, it is free to be run.
		if ec.taskToNumdeps[t.name].Load() == 0 {
			ec.enqueueTask(t)
		}
	}

	ec.wg.Wait()
	ec.cancel()

	return ec.err
}

func (ec *executionCtx) hasEncounteredErr() bool {
	return ec.errCounter.Load() != 0
}

func (ec *executionCtx) markFailure(err error) {
	// Return only the first error encountered
	if !ec.hasEncounteredErr() {
		ec.err = err
	}

	ec.cancel()
}

func (ec *executionCtx) enqueueTask(t Task) {
	ec.wg.Add(1)
	go ec.runTask(t)
}

func (ec *executionCtx) runTask(t Task) {
	defer ec.wg.Done()

	// Do not execute if we have encountered an error.
	if ec.hasEncounteredErr() {
		return
	}

	if err := t.fn(ec.ctx); err != nil {
		ec.markFailure(err)
		// Do not queue up additional tasks after encountering an error
		return
	}

	for dep := range ec.g.taskToDependants[t.name] {
		if ec.taskToNumdeps[dep].Add(-1) == int32(0) {
			ec.enqueueTask(ec.g.tasks[dep])
		}
	}
}
