package queue

import "sync"

type blockQueue struct {
	cond sync.Cond
}
