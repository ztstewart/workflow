package workflow

import "sync"

// Results holds a reference to the intermediate results of a workflow
// execution. It used to task the result of one function into its
// dependencies.
type Results struct {
	resMap *sync.Map
}

// Load retrieves the result for a particular task.
func (r Results) Load(taskName string) (val interface{}, ok bool) {
	return r.resMap.Load(taskName)
}

func (r Results) store(taskName string, result interface{}) {
	r.resMap.Store(taskName, result)
}
