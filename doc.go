// Package workflow is akin to the popular JavaScript library `async`'s `auto`
// function. That is to say, it runs a series of named tasks in parallel when
// it can and serially if there are interdependencies between tasks.
//
// For example, if we have a task A with no dependencies, and a task B that
// depends on A, workflow would run A before running B.
//
// For a more complex scenario, imagine task A with no dependencies, task B
// with no dependencies, and task C that depends on both A and B. In this case
// A and B would be run in parallel while execution of C would wait until both
// A and B have completed.
//
// Let's see this in code:
//
//	 import (
//	 	"context"
//
//	 	"github.com/ztstewart/workflow"
//	 )
//
//	 func doSomething(ctx context.Context) {
//	 	taskGraph, err := workflow.NewGraph(
//	 		NewTask("taskOne", []string{"taskTwo"}, func(ctx context.Context, res Results) (interface{}, error) {
//	 			// Do some useful work here...
//	 			return nil, nil
//	 		}),
//	 		NewTask("taskTwo", nil, func(ctx context.Context, res Results) (interface{}, error) {
//	 			// Do some useful work here...
//	 			return nil, nil
//	 		})
//	 	)
//
//	 	// Check for an error constructing the graph
//	 	if err != nil {
//	 		// Handle the error ....
//	 	}
//
//	 	if err := taskGraph.Run(ctx); err != nil {
//	 		// Handle any errors from running the tasks
//	 	}
//	 }
//
// In this example, `taskOne` will run before `taskTwo` does, assuming
// that `taskOne` does not return an error. `taskOne` will also have the
// result of `taskTwo` included in the `Results` parameter.
// If "taskTwo" returns an error, `taskGraph.Run()` will return an error and
// `taskOne` will not execute.
//
// Currently, 2^32 - 1 tasks are supported per workflow.
package workflow
