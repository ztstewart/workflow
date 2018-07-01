package workflow

import (
	"context"
	"errors"
	"sync"
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
func NewTask(name string, fn TaskFn, deps []string) Task {

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
	tasks            map[string]Task
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
		tasks: taskMap,
		taskToDependants: taskToDependants,
	}

	err := g.isWellFormed()

	return g, err
}

func (g Graph) isWellFormed() error {
	noDeps := make([]string, 0, len(g.tasks))
	taskToNumDeps := make(map[string]int32, len(g.tasks))

	for name, task := range g.tasks {
		numDeps := len(task.deps)

		if numDeps == 0 {
			noDeps = append(noDeps, name)
		}

		taskToNumDeps[name] = int32(numDeps)
	}

	visitedJobs := 0

	for i := 0; i < len(noDeps); i++ {
		name := noDeps[i]
		visitedJobs++

		for dep, _ := range g.taskToDependants[name] {
			taskToNumDeps[dep]--

			if taskToNumDeps[dep] == 0 {
				noDeps = append(noDeps, dep)
			}
		}
	}

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

	lockMap := make(map[string]*sync.RWMutex, len(g.tasks))
	var wg sync.WaitGroup
	wg.Add(len(g.tasks))
	var retErr error
	gCtx, cancel := context.WithCancel(ctx)

	tasks := make([]func(), 0, len(g.tasks))

	for name, task := range g.tasks {

		lock := sync.RWMutex{}
		lock.Lock()
		lockMap[name] = &lock

		tasks = append(tasks, func() {
			defer wg.Done()
			defer lock.Unlock()

			for depName, _ := range g.tasks[name].deps {
				lockMap[depName].RLock()
				lockMap[depName].RUnlock()
			}

			if err := task.fn(gCtx); err != nil {
				retErr = err
				cancel()
			}
		})
	}

	for _, task := range tasks {
		go task()
	}

	wg.Wait()

	cancel()
	return retErr
}
