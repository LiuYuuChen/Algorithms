package heap

import (
	"fmt"
)

type PriorityHandler[KEY comparable, VALUE any] interface {
	FormStoreKey(VALUE) KEY
	Less(VALUE, VALUE) bool
}

type heapItem[VALUE any] struct {
	index int
	value VALUE
}

type data[KEY comparable, VALUE any] struct {
	items    map[KEY]*heapItem[VALUE]
	queue    []KEY
	priority PriorityHandler[KEY, VALUE]
}

func newHeap[KEY comparable, VALUE any](priority PriorityHandler[KEY, VALUE]) *data[KEY, VALUE] {
	return &data[KEY, VALUE]{
		items:    make(map[KEY]*heapItem[VALUE]),
		priority: priority,
	}
}

func (h *data[_, _]) Less(i, j int) bool {
	if len(h.queue) < i || len(h.queue) < j {
		return false
	}

	return h.priority.Less(h.items[h.queue[i]].value, h.items[h.queue[j]].value)
}

func (h *data[_, _]) Len() int {
	return len(h.queue)
}

func (h *data[_, _]) Swap(i, j int) {
	h.queue[i], h.queue[j] = h.queue[j], h.queue[i]
	item := h.items[h.queue[i]]
	item.index = i
	item = h.items[h.queue[j]]
	item.index = j
}

// Pop returns the head of the heap and removes it.
func (h *data[_, VALUE]) Pop() (VALUE, error) {
	key := h.queue[len(h.queue)-1]
	h.queue = h.queue[0 : len(h.queue)-1]
	item, ok := h.items[key]
	if !ok {
		var empty VALUE
		return empty, fmt.Errorf("pop a empty heap")
	}
	delete(h.items, key)
	return item.value, nil
}

func (h *data[_, VALUE]) Push(value VALUE) {
	n := len(h.queue)
	key := h.priority.FormStoreKey(value)
	h2 := heapItem[VALUE]{index: n, value: value}
	h.items[key] = &h2
	h.queue = append(h.queue, key)
}

// Peek is supposed to be called by heap.Peek only.
func (h *data[_, VALUE]) Peek() (VALUE, error) {
	if len(h.queue) > 0 {
		return h.items[h.queue[0]].value, nil
	}
	var empty VALUE
	return empty, fmt.Errorf("peek a empty heap")
}

// Heap is a producer/consumer queue that implements a heap data structure.
// It can be used to implement priority queues and similar data structures.
type Heap[KEY comparable, VALUE any] struct {
	data *data[KEY, VALUE]
}

func (heap *Heap[KEY, VALUE]) Add(value VALUE) {
	key := heap.data.priority.FormStoreKey(value)
	if _, exist := heap.data.items[key]; exist {
		heap.data.items[key].value = value
		Fix[VALUE](heap.data, heap.data.items[key].index)
		return
	}
	Push[VALUE](heap.data, value)
}

// Delete removes an item.
func (heap *Heap[KEY, VALUE]) Delete(value VALUE) error {
	key := heap.data.priority.FormStoreKey(value)
	if item, ok := heap.data.items[key]; ok {
		_, err := Remove[VALUE](heap.data, item.index)
		return err
	}
	return fmt.Errorf("object not found")
}

// Peek returns the head of the heap without removing it.
func (heap *Heap[KEY, VALUE]) Peek() (VALUE, error) {
	return heap.data.Peek()
}

// Pop returns the head of the heap and removes it.
func (heap *Heap[KEY, VALUE]) Pop() (VALUE, error) {
	return Pop[VALUE](heap.data)
}

// Get returns the requested item, or sets exists=false.
func (heap *Heap[KEY, VALUE]) Get(value VALUE) (VALUE, bool) {
	key := heap.data.priority.FormStoreKey(value)
	val, ok := heap.data.items[key]
	if !ok {
		var empty VALUE
		return empty, false
	}
	return val.value, ok
}

// List returns a list of all the items.
func (heap *Heap[KEY, VALUE]) List() []VALUE {
	list := make([]VALUE, 0, len(heap.data.items))
	for _, item := range heap.data.items {
		list = append(list, item.value)
	}
	return list
}

// Len returns the number of items in the heap.
func (heap *Heap[KEY, VALUE]) Len() int {
	return len(heap.data.queue)
}

// New returns a Heap which can be used to queue up items to process.
func New[KEY comparable, VALUE any](priority PriorityHandler[KEY, VALUE]) *Heap[KEY, VALUE] {
	return &Heap[KEY, VALUE]{
		data: newHeap[KEY, VALUE](priority),
	}
}
