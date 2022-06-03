package queue

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/utils/clock"
)

type waitFor[V any] struct {
	readyAt time.Time
	value   V
	index   int
}

// waitForPriorityQueue implements a priority queue for waitFor items.
//
// waitForPriorityQueue implements heap.Interface. The item occurring next in
// time (i.e., the item with the smallest readyAt) is at the root (index 0).
// Peek returns this minimum item at index 0. Pop returns the minimum item after
// it has been removed from the queue and placed at index Len()-1 by
// container/heap. Push adds an item at index Len(), and container/heap
// percolates it into the correct location.
type waitForPriorityQueue[V any] []*waitFor[V]

func (pq waitForPriorityQueue[V]) Len() int {
	return len(pq)
}
func (pq waitForPriorityQueue[V]) Less(i, j int) bool {
	return pq[i].readyAt.Before(pq[j].readyAt)
}
func (pq waitForPriorityQueue[V]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push adds an item to the queue. Push should not be called directly; instead,
// use `heap.Push`.
func (pq *waitForPriorityQueue[V]) Push(item *waitFor[V]) {
	n := len(*pq)
	item.index = n
	*pq = append(*pq, item)
}

// Pop removes an item from the queue. Pop should not be called directly;
// instead, use `heap.Pop`.
func (pq *waitForPriorityQueue[V]) Pop() *waitFor[V] {
	n := len(*pq)
	item := (*pq)[n-1]
	item.index = -1
	*pq = (*pq)[0:(n - 1)]
	return item
}

// Peek returns the item at the beginning of the queue, without removing the
// item or otherwise mutating the queue. It is safe to call directly.
func (pq waitForPriorityQueue[V]) Peek() (*waitFor[V], error) {
	if len(pq) == 0 {
		return nil, fmt.Errorf("queue is empty")

	}
	return pq[0], nil
}

type delayingQueue[V any] struct {
	mainQueue BlockQueue[V]

	lock sync.Locker

	// stopCh lets us signal a shutdown to the waiting loop
	stopCh chan struct{}
	// stopOnce guarantees we only signal shutdown a single time
	stopOnce sync.Once

	// heartbeat ensures we wait no more than maxWait before firing
	heartbeat clock.Ticker

	// waitingForAddCh is a buffered channel that feeds waitingForAdd
	waitingForAddCh chan *waitFor[V]

	waitQueue waitForPriorityQueue[V]
}

func NewDelayingQueue[V any](constraint HeapConstraint[V], opts ...Option) DelayingQueue[V] {
	cfg := &config{
		lock: &sync.Mutex{},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	queue := &delayingQueue[V]{
		mainQueue: newBlockQueue[V](),
	}
}

func (q *delayingQueue[V]) Add(value V) {
	q.mainQueue.Add(value)
}

func (q *delayingQueue[V]) Update(value V) error {
	_, ok := q.mainQueue.Get(value)
	if ok {
		return q.mainQueue.Update(value)
	}
}

func (q *delayingQueue[V]) Delete(value V) error {

}

func (q *delayingQueue[V]) Get(value V) (V, bool) {

}

func (q *delayingQueue[V]) Pop() (V, error) {

}

func (q *delayingQueue[V]) Peek() (V, error) {

}

func (q *delayingQueue[V]) List() []V {

}

func (q *delayingQueue[V]) Len() int {

}

func (q *delayingQueue[V]) BlockPop() (V, error) {

}

func (q *delayingQueue[V]) Shutdown() {

}

func (q *delayingQueue[V]) IsShutdown() bool {

}

func (q *delayingQueue[V]) AddAfter(value V, duration time.Duration) {

}
