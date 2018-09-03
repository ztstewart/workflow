package workflow

import (
	"context"
	"sync"
)

var empty struct{}

// A TaskFn is an individually executable unit of work. It is expected that
// when the context is closed, such as via a timetout or cancellation, that
// a TaskFun will cease execution and return immediately.
// Results, which can be nil, contains a map of task name to return result.
type TaskFn = func(ctx context.Context, results *sync.Map) (interface{}, error)

// A Task is a unit of work along with a name and set of dependencies.
type Task struct {
	name string
	fn   TaskFn
	deps map[string]struct{}
}

// NewTask constructs a Task with a name and set of dependencies.
func NewTask(name string, deps []string, fn TaskFn) Task {
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
// A Graph is safe to reuse for multiple executions.
type Graph struct {
	tasks map[string]Task
	// Map of task name to set of tasks that depend on it.
	// Map<String, Set<String>> in Java.
	taskToDependants map[string]map[string]struct{}
}

// NewGraph constructs a new task execution graph and validates the tasks.
// An error will be returned in the event that the dependency graph is not
// well-formed, such as when a dependency is not satisfied or a cycle is
// detected.
func NewGraph(
	tasks []Task,
) (Graph, error) {
	taskMap := make(map[string]Task, len(tasks))
	taskToDependants := make(map[string]map[string]struct{}, len(tasks))

	for _, task := range tasks {
		name := task.name
		taskMap[name] = task
		for depName := range task.deps {
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

		for dep := range g.taskToDependants[name] {
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
		return InvalidGraphError{}
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
	return newExecutionCtx(ctx, g).run()
}
