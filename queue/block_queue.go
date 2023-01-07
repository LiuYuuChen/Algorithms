package queue

import (
	"fmt"
	"sync"

	"github.com/LiuYuuChen/algorithms/heap"
)

type blockQueue[V any] struct {
	cond *sync.Cond
	heap heap.Heap[V]

	globalCnt uint64
	stopping  bool
	stopped   bool
}

func NewBlockQueue[V any](constraint HeapConstraint[V]) BlockQueue[V] {
	return newBlockQueue[V](constraint)
}

func newBlockQueue[V any](constraint HeapConstraint[V]) *blockQueue[V] {
	return &blockQueue[V]{
		cond: sync.NewCond(&sync.RWMutex{}),
		heap: heap.NewConcurrent[V](constraint),
	}
}

func (que *blockQueue[V]) Add(value V) {
	que.cond.L.Lock()
	que.heap.Add(value)
	que.cond.L.Unlock()
	que.cond.Broadcast()
}

func (que *blockQueue[V]) Update(value V) error {
	que.cond.L.Lock()
	defer que.cond.Broadcast()
	defer que.cond.L.Unlock()
	if que.stopping {
		return fmt.Errorf("can not update an item to a closing queue")
	}

	_, ok := que.heap.Get(value)
	if !ok {
		return fmt.Errorf("can not update an item not in queue")
	}

	que.heap.Add(value)
	return nil
}

func (que *blockQueue[V]) Delete(value V) error {
	err := que.heap.Delete(value)
	if err != nil {
		return err
	}
	que.cond.Broadcast()
	return nil
}

func (que *blockQueue[V]) Get(value V) (V, bool) {
	v, ok := que.heap.Get(value)
	que.cond.Broadcast()
	return v, ok
}

func (que *blockQueue[V]) List() []V {
	list := que.heap.List()
	que.cond.Broadcast()
	return list
}

func (que *blockQueue[V]) Pop() (V, error) {
	return que.BlockPop()
}

func (que *blockQueue[V]) BlockPop() (V, error) {
	que.cond.L.Lock()
	defer que.cond.L.Unlock()
BlockLoop:
	for que.heap.Len() == 0 && !que.stopping {
		que.cond.Wait()
	}

	if que.stopped {
		return *new(V), fmt.Errorf("pop a closed queue")
	}

	if que.stopping {
		que.stopped = true
	}

	item, err := que.heap.Pop()
	if err != nil {
		goto BlockLoop
	}

	return item, nil
}

func (que *blockQueue[V]) Len() int {
	return que.heap.Len()
}

func (que *blockQueue[V]) Shutdown() {
	que.cond.L.Lock()
	que.stopping = true
	que.cond.L.Unlock()
	que.cond.Broadcast()
}

func (que *blockQueue[V]) IsShutdown() bool {
	que.cond.L.Lock()
	stopping := que.stopping
	que.cond.L.Unlock()
	return stopping
}

func (que *blockQueue[V]) Peek() (V, error) {
	v, err := que.heap.Peek()
	if err != nil {
		return *new(V), err
	}
	que.cond.Broadcast()
	return v, nil
}
