package workflow

import (
	"context"
	"errors"
	"testing"
)

func TestWorkflowNoJobs(t *testing.T) {
	graph, err := NewGraph()

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
	taskThreeRan := false
	taskOneRetVal := "someReturnValue"
	taskGraph, err := NewGraph(
		NewTask("taskOne", []string{"taskTwo"}, func(ctx context.Context, res Results) (interface{}, error) {
			taskOneRan = true
			return taskOneRetVal, nil
		}),
		NewTask("taskTwo", nil, func(ctx context.Context, res Results) (interface{}, error) {
			taskTwoRan = true
			return nil, nil
		}),
		NewTask("taskThree", []string{"taskOne"}, func(ctx context.Context, res Results) (interface{}, error) {
			taskThreeRan = true

			taskOneRes, ok := res.Load("taskOne")
			if !ok {
				t.Fatal("failed to stored value in result map")
			}

			resAsString, ok := taskOneRes.(string)
			if !ok {
				t.Fatal("failed to cast result to a string")
			}

			if resAsString != taskOneRetVal {
				t.Fatal("incorrect value returned")
			}

			return nil, nil
		}),
	)

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

	if !taskThreeRan {
		t.Fatal("failed to run taskThree")
	}
}

func TestWorkflowReturnsError(t *testing.T) {
	retErr := errors.New("bad error")
	taskOneRan := false
	taskTwoRan := false
	taskGraph, err := NewGraph(
		NewTask("taskName", []string{"someOtherTask"}, func(ctx context.Context, res Results) (interface{}, error) {
			taskOneRan = true
			return nil, nil
		}),
		NewTask("someOtherTask", nil, func(ctx context.Context, res Results) (interface{}, error) {
			taskTwoRan = true
			return nil, retErr
		}),
	)

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
