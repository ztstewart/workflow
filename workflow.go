package workflow

import (
	"context"
	"errors"
	"sync"

	"github.com/ztstewart/workflow/internal/atomic"
)

var empty struct{}

// A TaskFn is an individually executable unit of work. It is expected that
// when the context is closed, such as via a timetout or cancellation, that
// a TaskFun will cease execution and return immediately.
type TaskFn = func(ctx context.Context) error

// A Task is a unit of work along with a name and set of dependencies.
type Task struct {
	name string
	fn   TaskFn
	deps map[string]struct{}
}

// NewTask constructs a Task with a name and set of dependencies.
func NewTask(name string, fn TaskFn, deps ...string) Task {
	depSet := make(map[string]struct{}, len(deps))

	for _, dep := range deps {
		depSet[dep] = empty
	}

	return Task{
		name: name,
		fn:   fn,
		deps: depSet,
	}
}

// A Graph is a representation of the dependencies of a workflow.
// It is capable of executing the dependencies of the workflow in
// a topological order.
//
// An error will be returned in the event that the graph is not well-formed,
// such as when a dependency is not satisfied or a cycle is detected.
type Graph struct {
	tasks map[string]Task
	// Map of task name to set of tasks that depend on it.
	// Map<String, Set<String>> in Java.
	taskToDependants map[string]map[string]struct{}
}

func NewGraph(
	tasks []Task,
) (Graph, error) {

	taskMap := make(map[string]Task, len(tasks))
	taskToDependants := make(map[string]map[string]struct{}, len(tasks))

	for _, task := range tasks {
		name := task.name
		taskMap[name] = task
		for depName, _ := range task.deps {
			depSet, ok := taskToDependants[depName]

			if !ok {
				depSet = make(map[string]struct{})
			}

			depSet[name] = empty
			taskToDependants[depName] = depSet
		}
	}

	g := Graph{
		tasks:            taskMap,
		taskToDependants: taskToDependants,
	}

	err := g.isWellFormed()

	return g, err
}

// isWellFormed checks to ensure that there are no cycles or missing
// dependencies.
func (g Graph) isWellFormed() error {
	tasksWithNoDeps := make([]string, 0, len(g.tasks))
	taskToNumDeps := make(map[string]int32, len(g.tasks))

	// Count the number of dependencies and mark as ready to execute if no
	// deps are present.
	for name, task := range g.tasks {
		numDeps := len(task.deps)

		if numDeps == 0 {
			tasksWithNoDeps = append(tasksWithNoDeps, name)
		}

		taskToNumDeps[name] = int32(numDeps)
	}

	// For every task with no dependencies, increment the number of jobs that
	// we have visited. Then repeat for any child tasks that are now runnable.
	// See https://en.wikipedia.org/wiki/Topological_sorting for more details.
	visitedJobs := 0

	for i := 0; i < len(tasksWithNoDeps); i++ {
		name := tasksWithNoDeps[i]
		visitedJobs++

		for dep, _ := range g.taskToDependants[name] {
			taskToNumDeps[dep]--

			// If a job has no dependencies unfulfilled, we can visit it.
			if taskToNumDeps[dep] == 0 {
				tasksWithNoDeps = append(tasksWithNoDeps, dep)
			}
		}
	}

	// If we have failed to visit all jobs, or we have visited some more than
	// once somehow, we either are missing a dependency or we have a cycle.
	// Either way, we can't execute the graph.
	if visitedJobs != len(g.tasks) {
		return errors.New("dependency graph is unsolvable; check for cycles or missing dependencies")
	}

	return nil
}

// Run executes a workflow definition. It returns an error if any of the
// underlying tasks fail to execute, after waiting for any currently executing
// tasks to complete. The error returned is that returned by the first task.
// Additionally, if the `ctx.Done()` channel is written to prior to
// executing all tasks, that will also count as a failure, and
// context.Error() will be the return value of this function.
//
// Tasks are run concurrently when it is possible to do so.
func (g Graph) Run(ctx context.Context) error {

	ec := newExecutionCtx(ctx, g)
	return ec.run()
}

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
	readyTasks    chan func()
	errCounter    *atomic.Int32 // There are no atomic errors, sadly
}

func newExecutionCtx(ctx context.Context, g Graph) *executionCtx {
	iCtx, cancel := context.WithCancel(ctx)

	return &executionCtx{
		taskToNumdeps: make(map[string]*atomic.Int32, len(g.tasks)),
		readyTasks:    make(chan func(), len(g.tasks)),
		g:             g,
		ctx:           iCtx,
		cancel:        cancel,
		errCounter:    atomic.NewInt32(0),
	}
}

func (ec *executionCtx) run() error {
	for _, t := range ec.g.tasks {
		ec.taskToNumdeps[t.name] = atomic.NewInt32(int32(len(t.deps)))

		// When a task has no dependencies, it is free to be run.
		if ec.taskToNumdeps[t.name].Load() == 0 {
			ec.enqueueTask(t)
		}
	}

	go func() {
		ec.wg.Wait()
		close(ec.readyTasks)
	}()

	for task := range ec.readyTasks {
		go task()
	}

	ec.cancel()

	return ec.err
}

func (ec *executionCtx) encounteredErr() bool {
	return ec.errCounter.Load() != 0
}

func (ec *executionCtx) markFailure(err error) {
	if !ec.encounteredErr() {
		ec.err = err
	}

	ec.cancel()
}

func (ec *executionCtx) enqueueTask(t Task) {
	ec.wg.Add(1)
	ec.readyTasks <- func() { ec.runTask(t) }
}

func (ec *executionCtx) runTask(t Task) {
	defer ec.wg.Done()

	// Do not execute if we have encountered an error.
	if ec.encounteredErr() {
		return
	}

	if err := t.fn(ec.ctx); err != nil {
		ec.markFailure(err)
		// Do not queue up additional tasks after encountering an error
		return
	}

	for dep, _ := range ec.g.taskToDependants[t.name] {
		if ec.taskToNumdeps[dep].Add(-1) == int32(0) {
			ec.enqueueTask(ec.g.tasks[dep])
		}
	}
}
