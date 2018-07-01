package workflow

import (
	"context"
	"testing"
)

func TestWorkflowNoJobs(t *testing.T) {
	graph, err := NewGraph(nil)

	if err != nil {
		t.Fatal("failed to initialize the graph")
	}

	err = graph.Run(context.Background())

	if err != nil {
		t.Fatal("failed to run the graph")
	}
}

func TestWorkflowSimpleCase(t *testing.T) {

	taskOneRan := false
	taskTwoRan := false
	taskGraph, err := NewGraph([]Task{
        NewTask("taskName", func(ctx context.Context) error {
			// Do some useful work here...
			taskOneRan = true
            return nil
        }, []string{"someOtherTask"}),
        NewTask("someOtherTask", func(ctx context.Context) error {
			// Do some useful work here...
			taskTwoRan = true
            return nil
        }, nil),
	})
	
	if err != nil {
		t.Fatal("failed to initialize the graph")
	}

	err = taskGraph.Run(context.Background())

	if err != nil {
		t.Fatal("failed to run the graph")
	}

	if !taskOneRan {
		t.Fatal("failed to run taskOne")
	}

	if !taskTwoRan {
		t.Fatal("failed to run taskTwo")
	}

}
