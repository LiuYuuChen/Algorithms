package heap

import (
	"fmt"
	"sync"

	cmap "github.com/orcaman/concurrent-map"
)

// heap is a producer/consumer queue that implements a heap data structure.
// It can be used to implement priority queues and similar data structures.
type currentHeap[VALUE any] struct {
	data *concurrentData[VALUE]
}

func (heap *currentHeap[VALUE]) Add(value VALUE) {
	key := heap.data.priority.FormStoreKey(value)
	if item, exist := heap.data.items.Get(key); exist {
		item.value = value
		heap.data.items.Set(key, item)
		Fix[VALUE](heap.data, item.index)
		return
	}
	Push[VALUE](heap.data, value)
}

// Delete removes an item.
func (heap *currentHeap[VALUE]) Delete(value VALUE) error {
	key := heap.data.priority.FormStoreKey(value)
	if item, ok := heap.data.items.Get(key); ok {
		_, err := Remove[VALUE](heap.data, item.index)
		return err
	}
	return fmt.Errorf("object not found")
}

// Peek returns the head of the heap without removing it.
func (heap *currentHeap[VALUE]) Peek() (VALUE, error) {
	return heap.data.Peek()
}

// Pop returns the head of the heap and removes it.
func (heap *currentHeap[VALUE]) Pop() (VALUE, error) {
	return Pop[VALUE](heap.data)
}

// Get returns the requested item, or sets exists=false.
func (heap *currentHeap[VALUE]) Get(value VALUE) (VALUE, bool) {
	key := heap.data.priority.FormStoreKey(value)
	val, ok := heap.data.items.Get(key)
	if !ok {
		var empty VALUE
		return empty, false
	}
	return val.value, ok
}

// List returns a list of all the items.
func (heap *currentHeap[VALUE]) List() []VALUE {
	list := make([]VALUE, 0, len(heap.data.items))
	for _, item := range heap.data.items.Items() {
		list = append(list, item.value)
	}
	return list
}

// Len returns the number of items in the heap.
func (heap *currentHeap[VALUE]) Len() int {
	return len(heap.data.queue)
}

type concurrentData[VALUE any] struct {
	lock     sync.Locker
	items    cmap.ConcurrentMap[*heapItem[VALUE]]
	queue    []string
	priority Constraint[string, VALUE]
}

func newConcurrentData[V any](lock sync.Locker, handler Constraint[string, V]) *concurrentData[V] {
	if lock == nil {
		lock = &sync.RWMutex{}
	}
	return &concurrentData[V]{
		lock:     lock,
		items:    cmap.New[*heapItem[V]](),
		queue:    make([]string, 0),
		priority: handler,
	}
}

func (h *concurrentData[V]) Less(i, j int) bool {
	h.lock.Lock()
	if len(h.queue) <= i || len(h.queue) <= j {
		h.lock.Unlock()
		return false
	}
	keyI, keyJ := h.queue[i], h.queue[j]
	h.lock.Unlock()

	itemI, ok := h.items.Get(keyI)
	if !ok {
		return false
	}
	itemJ, ok := h.items.Get(keyJ)
	if !ok {
		return false
	}

	return h.priority.Less(itemI.value, itemJ.value)
}

func (h *concurrentData[V]) Len() int {
	return len(h.queue)
}

func (h *concurrentData[V]) Swap(i, j int) {
	h.lock.Lock()
	if len(h.queue) <= i || len(h.queue) <= j {
		h.lock.Unlock()
		return
	}
	h.queue[i], h.queue[j] = h.queue[j], h.queue[i]
	item, _ := h.items.Get(h.queue[i])
	item.index = i
	item, _ = h.items.Get(h.queue[j])
	item.index = j
	h.lock.Unlock()
}

// Pop returns the head of the heap and removes it.
func (h *concurrentData[VALUE]) Pop() (VALUE, error) {
	if len(h.queue) == 0 {
		var empty VALUE
		return empty, fmt.Errorf("pop a empty heap")
	}
	h.lock.Lock()
	defer h.lock.Unlock()
	key := h.queue[len(h.queue)-1]
	h.queue = h.queue[0 : len(h.queue)-1]
	item, ok := h.items.Get(key)
	if !ok {
		var empty VALUE
		return empty, fmt.Errorf("pop a empty heap")
	}
	h.items.Remove(key)
	return item.value, nil
}

func (h *concurrentData[VALUE]) Push(value VALUE) {
	h.lock.Lock()
	n := len(h.queue)
	key := h.priority.FormStoreKey(value)
	h2 := heapItem[VALUE]{index: n, value: value}
	h.items.Set(key, &h2)
	h.queue = append(h.queue, key)
	h.lock.Unlock()
}

// Peek is supposed to be called by heap.Peek only.
func (h *concurrentData[VALUE]) Peek() (VALUE, error) {
	var empty VALUE
	if len(h.queue) > 0 {
		item, ok := h.items.Get(h.queue[0])
		if !ok {
			return empty, fmt.Errorf("can not find queue peek")
		}
		return item.value, nil
	}
	return empty, fmt.Errorf("peek a empty heap")
}
