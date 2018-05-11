package workflow

import (
	"context"
	"sync"
)

var empty struct{}

// A TaskFn is an individually executable unit of work.
type TaskFn = func(context.Context) error

// A Task is a unit of work along with a name and dependencies
type Task struct {
	name string
	fn   TaskFn
	deps map[string]struct{}
}

// NewTask constructs a Task
func NewTask(name string, fn TaskFn, deps []string) Task {

	taskMap := make(map[string]struct{}, len(deps))

	for _, dep := range deps {
		taskMap[dep] = empty
	}

	return Task{
		name: name,
		fn:   fn,
		deps: taskMap,
	}
}

// A Graph is a representation of the dependencies of a workflow.
// It is capable of executing the dependencies of the workflow in
// a topological order.
type Graph struct {
	tasks map[string]Task
}

func NewGraph(
	tasks map[string]Task,
) {
	return Graph{
		tasks: tasks,
	}
}

// Run executes a workflow definition. It returns an error if any of the
// underlying tasks fail to execute, after waiting for any currently executing
// tasks to complete. The error returned is that returned by the first task.
func (g Graph) Run(ctx context.Context) error {

	var lockMap map[string]*sync.RWMutex
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
