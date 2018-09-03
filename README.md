# Workflow

Workflow is a simple Go utility library that allows developers to execute tasks
that depend on other tasks in a clean, concurrent, and stress free manner.

## How do I use it?

Using your dependency management system of choice (glide, dep, go get) import the package.

From then on, it's relatively straightforward:

```
import (
    "context"

    "github.com/ztstewart/workflow"
)

func doSomething(ctx context.Context) {

    taskGraph, err := workflow.NewGraph(
        NewTask("taskOne", []string{"taskTwo"}, func(ctx context.Context, res Results) (interface{}, error) {
            // Do some useful work here...
            return nil, nil
        }),
        NewTask("taskTwo", nil, func(ctx context.Context, res Results) (interface{}, error) {
            // Do some useful work here...
            return nil, nil
        })
    )

    // Check for an error -- maybe we forgot to add "taskTwo"
    if err != nil {
        // Handle the error ....
    }

    if err := taskGraph.Run(ctx); err != nil {
        // Handle any errors from running the tasks
    }
}
```

# Why workflow?

Workflow is one of the few Go task libraries that supports Go's concept of context.

Using context allows developers to create complex task deadlines and cancellations to intelligently respond to cancelled requests,
timeouts, and so on. Secondly, keeping context in the signature allows us to signal cancellation to all worker tasks.

This means you can easily build a policy where if a task fails to execute, other tasks can be notified and clean up their own state
before early termination.