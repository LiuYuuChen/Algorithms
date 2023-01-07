package queue

import (
	"time"

	"github.com/LiuYuuChen/algorithms/heap"
)

type Queue[V any] interface {
	Add(value V)
	Update(value V) error
	Delete(value V) error
	Get(value V) (V, bool)
	Pop() (V, error)
	Peek() (V, error)
	List() []V
	Len() int
}

type HeapConstraint[VALUE any] interface {
	heap.Constraint[string, VALUE]
}

type BlockQueue[V any] interface {
	Queue[V]
	Shutdown()
	IsShutdown() bool
}

type DelayingQueue[V any] interface {
	BlockQueue[V]
	AddAfter(value V, duration time.Duration)
	Refresh(obj V) error
}
