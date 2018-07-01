package atomic

import (
	"sync/atomic"
)

// Int32 is a wrapper around an int32, providing atomic operations.
// All operations provided are atomic.
type Int32 struct{ 
	val int32 
}

// NewInt32 instantiates an Int32.
func NewInt32(i int32) *Int32 {
	return &Int32{i}
}

// Add atomically adds to the internal int32, returning the new value.
func (i *Int32) Add(n int32) int32 {
	return atomic.AddInt32(&i.val, n)
}

// Load atomically retrieves the value of the wrapped int32.
func (i *Int32) Load() int32 {
	return atomic.LoadInt32(&i.val)
}
