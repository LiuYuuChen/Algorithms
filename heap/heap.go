package heap

import (
	"fmt"
	cmap "github.com/orcaman/concurrent-map"
	"sync"
)

type options struct {
	lock sync.Locker
}

type Option func(opts *options)

func WithLock(lock sync.Locker) Option {
	return func(opts *options) {
		opts.lock = lock
	}
}

type heapItem[VALUE any] struct {
	index int
	value VALUE
}

type data[KEY comparable, VALUE any] struct {
	items    map[KEY]*heapItem[VALUE]
	queue    []KEY
	priority Constraint[KEY, VALUE]
}

func newData[KEY comparable, VALUE any](priority Constraint[KEY, VALUE]) *data[KEY, VALUE] {
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

// heap is a producer/consumer queue that implements a heap data structure.
// It can be used to implement priority queues and similar data structures.
type heap[KEY comparable, VALUE any] struct {
	data *data[KEY, VALUE]
}

func (heap *heap[KEY, VALUE]) Add(value VALUE) {
	key := heap.data.priority.FormStoreKey(value)
	if _, exist := heap.data.items[key]; exist {
		heap.data.items[key].value = value
		Fix[VALUE](heap.data, heap.data.items[key].index)
		return
	}
	Push[VALUE](heap.data, value)
}

// Delete removes an item.
func (heap *heap[KEY, VALUE]) Delete(value VALUE) error {
	key := heap.data.priority.FormStoreKey(value)
	if item, ok := heap.data.items[key]; ok {
		_, err := Remove[VALUE](heap.data, item.index)
		return err
	}
	return fmt.Errorf("object not found")
}

// Peek returns the head of the heap without removing it.
func (heap *heap[KEY, VALUE]) Peek() (VALUE, error) {
	return heap.data.Peek()
}

// Pop returns the head of the heap and removes it.
func (heap *heap[KEY, VALUE]) Pop() (VALUE, error) {
	return Pop[VALUE](heap.data)
}

// Get returns the requested item, or sets exists=false.
func (heap *heap[KEY, VALUE]) Get(value VALUE) (VALUE, bool) {
	key := heap.data.priority.FormStoreKey(value)
	val, ok := heap.data.items[key]
	if !ok {
		var empty VALUE
		return empty, false
	}
	return val.value, ok
}

// List returns a list of all the items.
func (heap *heap[KEY, VALUE]) List() []VALUE {
	list := make([]VALUE, 0, len(heap.data.items))
	for _, item := range heap.data.items {
		list = append(list, item.value)
	}
	return list
}

// Len returns the number of items in the heap.
func (heap *heap[KEY, VALUE]) Len() int {
	return len(heap.data.queue)
}

// New returns a heap which can be used to queue up items to process.
func New[KEY comparable, VALUE any](priority Constraint[KEY, VALUE]) Heap[VALUE] {
	return newHeap[KEY, VALUE](priority)
}

func newHeap[KEY comparable, VALUE any](priority Constraint[KEY, VALUE]) *heap[KEY, VALUE] {
	return &heap[KEY, VALUE]{
		data: newData[KEY, VALUE](priority),
	}
}

func NewConcurrent[VALUE any](priority Constraint[string, VALUE], opts ...Option) Heap[VALUE] {
	cfg := options{lock: &sync.RWMutex{}}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &currentHeap[VALUE]{
		data: &concurrentData[VALUE]{
			lock:     cfg.lock,
			priority: priority,
			items:    cmap.New[*heapItem[VALUE]](),
		},
	}
}

func newConcurrent[VALUE any](priority Constraint[string, VALUE], cfg *options) *currentHeap[VALUE] {
	return &currentHeap[VALUE]{
		data: &concurrentData[VALUE]{
			lock:     cfg.lock,
			priority: priority,
			items:    cmap.New[*heapItem[VALUE]](),
		},
	}
}
