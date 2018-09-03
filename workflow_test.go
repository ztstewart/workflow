package workflow

import (
	"context"
	"errors"
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
		NewTask("taskName", []string{"someOtherTask"}, func(ctx context.Context) error {
			// Do some useful work here...
			taskOneRan = true
			return nil
		}),
		NewTask("someOtherTask", nil, func(ctx context.Context) error {
			// Do some useful work here...
			taskTwoRan = true
			return nil
		}),
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

func TestWorkflowReturnsError(t *testing.T) {

	retErr := errors.New("bad error")
	taskOneRan := false
	taskTwoRan := false
	taskGraph, err := NewGraph([]Task{
		NewTask("taskName", []string{"someOtherTask"}, func(ctx context.Context) error {
			// Do some useful work here...
			taskOneRan = true
			return nil
		}),
		NewTask("someOtherTask", nil, func(ctx context.Context) error {
			// Do some useful work here...
			taskTwoRan = true
			return retErr
		}),
	})

	if err != nil {
		t.Fatal("failed to initialize the graph")
	}

	err = taskGraph.Run(context.Background())

	if err != retErr {
		t.Fatal("failed to return an error")
	}

	if taskOneRan {
		t.Fatal("should not have started taskOne")
	}

	if !taskTwoRan {
		t.Fatal("failed to run taskTwo")
	}

}
