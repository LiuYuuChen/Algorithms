package queue

import "time"

type queueItem[V any] struct {
	time   time.Time
	value  V
	number uint64
}
