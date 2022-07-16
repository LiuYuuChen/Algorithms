package queue

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type waitFor[V any] struct {
	readyAt time.Time
	value   V
	index   int
}

func newWaitFor[V any](item V) *waitFor[V] {
	return &waitFor[V]{
		readyAt: time.Now(),
		value:   item,
	}
}

// maxWait keeps a max bound on the wait time. It's just insurance against weird things happening.
// Checking the queue every 10 seconds isn't expensive and we know that we'll never end up with an
// expired item sitting for more than 10 seconds.
const maxWait = 10 * time.Second

type waitConstraintConvertor[V any] struct {
	origin HeapConstraint[V]
}

func (convertor *waitConstraintConvertor[V]) FormStoreKey(item *waitFor[V]) string {
	return convertor.origin.FormStoreKey(item.value)
}

func (convertor *waitConstraintConvertor[V]) Less(itemI, itemJ *waitFor[V]) bool {
	return convertor.origin.Less(itemI.value, itemJ.value)
}

type delayingQueue[V any] struct {
	mainQueue *blockQueue[V]
	waitQueue *blockQueue[*waitFor[V]]
	heartbeat *time.Timer
	lock      sync.RWMutex
	// stopCh lets us signal a shutdown to the waiting loop
	stopCh chan struct{}
	// waitingForAddCh is a buffered channel that feeds waitingForAdd
	waitingForAddCh chan *waitFor[V]

	stopOnce sync.Once
	stop     bool
}

func NewDelayingQueue[V any](constraint HeapConstraint[V], opts ...Option) DelayingQueue[V] {
	cfg := &config{lock: &sync.Mutex{}}
	for _, opt := range opts {
		opt(cfg)
	}
	return newDelayingQueue(constraint, cfg)
}

func newDelayingQueue[V any](constraint HeapConstraint[V], cfg *config) *delayingQueue[V] {
	dQueue := &delayingQueue[V]{
		mainQueue: newBlockQueue[V](constraint, cfg),
		waitQueue: newBlockQueue[*waitFor[V]](&waitConstraintConvertor[V]{origin: constraint}, cfg),
		heartbeat: time.NewTimer(maxWait),

		waitingForAddCh: make(chan *waitFor[V], 1000),
		stopCh:          make(chan struct{}),
	}

	go dQueue.waitingLoop()
	return dQueue
}

func (q *delayingQueue[V]) AddAfter(item V, duration time.Duration) {
	if q.IsShutdown() {
		return
	}
	// immediately add things with no delay
	if duration <= 0 {
		q.mainQueue.Add(item)
		return
	}

	select {
	case _, ok := <-q.stopCh:
		if !ok {
			logrus.Errorf("unexpected closed")
		}
	case q.waitingForAddCh <- &waitFor[V]{value: item, readyAt: time.Now().Add(duration)}:
	}
}

func (q *delayingQueue[V]) Add(value V) {
	q.mainQueue.Add(value)
}

func (q *delayingQueue[V]) Update(obj V) error {
	_, ok := q.waitQueue.Get(newWaitFor[V](obj))
	if ok {
		return q.waitQueue.Update(newWaitFor[V](obj))
	}

	return q.mainQueue.Update(obj)
}

func (q *delayingQueue[V]) Refresh(obj V) error {
	item, ok := q.waitQueue.Get(newWaitFor[V](obj))
	if ok {
		item.value = obj
		return nil
	}
	return q.mainQueue.Update(obj)
}

// Delete object from both main queue and wait queue.
func (q *delayingQueue[V]) Delete(obj V) error {
	item, ok := q.mainQueue.Get(obj)
	if ok {
		return q.mainQueue.Delete(item)
	}

	queItem, ok := q.waitQueue.Get(newWaitFor[V](obj))
	if ok {
		return q.waitQueue.Delete(queItem)
	}

	return fmt.Errorf("can not find item: %v in delaying queue", obj)
}

func (q *delayingQueue[V]) List() []V {
	list := make([]V, 0, q.mainQueue.Len()+q.waitQueue.Len())
	list = append(list, q.mainQueue.List()...)
	for _, item := range q.waitQueue.List() {
		list = append(list, item.value)
	}
	return list
}

func (q *delayingQueue[V]) Get(obj V) (V, bool) {
	item, ok := q.mainQueue.Get(obj)
	if ok {
		return item, true
	}

	queItem, ok := q.waitQueue.Get(newWaitFor[V](obj))
	if ok {
		return queItem.value, true
	}

	return *new(V), false
}

func (q *delayingQueue[V]) Pop() (V, error) {
	return q.mainQueue.Pop()
}

func (q *delayingQueue[V]) Len() int {
	return q.mainQueue.Len() + q.waitQueue.Len()
}

// Shutdown stops the queue. After the queue drains, the returned shutdown bool
// on Get() will be true. This method may be invoked more than once.
func (q *delayingQueue[V]) Shutdown() {
	q.stopOnce.Do(func() {
		q.lock.Lock()
		q.stop = true
		q.lock.Unlock()

		q.mainQueue.Shutdown()
		q.waitQueue.Shutdown()
		close(q.stopCh)
		q.heartbeat.Stop()
	})
}

func (q *delayingQueue[V]) waitingLoop() {

	// Make a placeholder channel to use when there are no items in our list
	never := make(<-chan time.Time)

	// Make a timer that expires when the item at the head of the waiting queue is ready
	var nextReadyAtTimer *time.Timer

	for {
		if q.IsShutdown() {
			return
		}

		now := time.Now()

		// Add ready entries
		for q.waitQueue.Len() > 0 {
			entry, err := q.waitQueue.Peek()

			if err != nil {
				logrus.Errorf("waittingLoop error: %v", err)
				continue
			}

			if entry.readyAt.After(now) {
				break
			}

			item, err := q.waitQueue.Pop()
			if err != nil {
				logrus.Errorf("pop from wait queue with error %v", err)
				break
			}

			q.mainQueue.Add(item.value)
		}

		// Set up a wait for the first item's readyAt if one exists
		nextReadyAt := never
		if q.waitQueue.Len() > 0 {
			if nextReadyAtTimer != nil {
				nextReadyAtTimer.Stop()
			}

			entry, err := q.waitQueue.Peek()
			if err != nil {
				logrus.Errorf("delaying queue get wrong type item due to %v", err)
			}

			nextReadyAtTimer = time.NewTimer(entry.readyAt.Sub(now))
			nextReadyAt = nextReadyAtTimer.C
		}

		select {
		case <-q.stopCh:
			return

		case <-q.heartbeat.C:
			// continue the loop, which will add ready items

		case <-nextReadyAt:
			// continue the loop, which will add ready items

		case waitEntry := <-q.waitingForAddCh:
			q.receiveItems(waitEntry)
			q.drainChannel()
		}
	}
}

func (q *delayingQueue[V]) receiveItems(waitEntry *waitFor[V]) {
	if waitEntry.readyAt.After(time.Now()) {
		q.waitQueue.heap.Add(waitEntry)
	} else {
		q.mainQueue.Add(waitEntry.value)
	}
}

func (q *delayingQueue[V]) drainChannel() {
	drained := false
	for !drained {
		select {
		case waitEntry, ok := <-q.waitingForAddCh:
			if !ok {
				drained = true
				break
			}
			q.receiveItems(waitEntry)
		default:
			drained = true
		}
	}
}
func (q *delayingQueue[V]) IsShutdown() bool {
	q.lock.RLock()
	shutting := q.stop
	q.lock.RUnlock()
	return shutting
}

func (q *delayingQueue[V]) Peek() (V, error) {
	return q.mainQueue.Peek()
}
