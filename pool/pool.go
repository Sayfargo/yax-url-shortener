// Package pool provides a type-safe generic wrapper around sync.Pool
// for reusing objects that implement the Resettable interface.
package pool

import (
	"reflect"
	"sync"
)

// Resettable is the interface that wraps the Reset method.
//
// Reset restores the object to its initial zero state before reuse.
type Resettable interface {
	Reset()
}

// Pool is a generic object pool that stores and reuses values of type T.
// It automatically resets objects via Reset before returning them to the pool.
type Pool[T Resettable] struct {
	pool sync.Pool
}

// New creates and returns a new initialized Pool for type T using the provided factory function.
func New[T Resettable](new func() T) *Pool[T] {
	var fn func() any
	if new != nil {
		fn = func() any {
			return new()
		}
	}
	return &Pool[T]{
		pool: sync.Pool{
			New: fn,
		},
	}
}

// Get retrieves an object of type T from the pool, or creates a new one if the pool is empty.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put resets the state of x and returns it to the pool for future reuse.
// Nil values and nil pointers are ignored.
func (p *Pool[T]) Put(x T) {
	t := reflect.ValueOf(x)
	if !t.IsValid() || (t.Kind() == reflect.Pointer && t.IsNil()) {
		return
	}
	x.Reset()
	p.pool.Put(x)
}
