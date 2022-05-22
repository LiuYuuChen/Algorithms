package heap

import (
	"fmt"
	"sort"
)

type Interface[VALUE any] interface {
	sort.Interface
	Push(x VALUE)
	Pop() (VALUE, error)
}

type Heap[VALUE any] interface {
	Add(value VALUE)
	Delete(value VALUE) error
	Peek() (VALUE, error)
	Pop() (VALUE, error)
	Get(value VALUE) (VALUE, bool)
	List() []VALUE
	Len() int
}

func BuildHeap[VALUE any](heap Interface[VALUE]) {

	n := heap.Len()

	for i := n/2 - 1; i >= 0; i-- {
		heapifyDown(heap, i, n)
	}

}

func Push[VALUE any](heap Interface[VALUE], x VALUE) {
	heap.Push(x)
	heapifyUp(heap, heap.Len()-1)
}

func Pop[VALUE any](heap Interface[VALUE]) (VALUE, error) {
	n := heap.Len() - 1

	if n < 0 {
		var empty VALUE
		return empty, fmt.Errorf("pop a empty heap")
	}

	heap.Swap(0, n)
	heapifyDown(heap, 0, n)
	return heap.Pop()
}

func Remove[VALUE any](heap Interface[VALUE], i int) (VALUE, error) {
	n := heap.Len() - 1

	if n < 0 {
		var empty VALUE
		return empty, fmt.Errorf("remove a empty heap")
	}

	if n != i {
		heap.Swap(i, n)
		if !heapifyDown(heap, i, n) {
			heapifyUp(heap, i)
		}
	}
	return heap.Pop()
}

func Fix[VALUE any](h Interface[VALUE], i int) {
	if !heapifyDown(h, i, h.Len()) {
		heapifyUp(h, i)
	}
}

func heapifyUp[VALUE any](heap Interface[VALUE], i int) {
	for {
		parent := (i - 1) / 2
		if parent == i || !heap.Less(i, parent) {
			break
		}

		heap.Swap(parent, i)
		i = parent
	}
}

func heapifyDown[VALUE any](h Interface[VALUE], i0, n int) bool {
	i := i0
	for {
		left := 2*i + 1
		if left >= n || left < 0 { // left < 0 after int overflow
			break
		}

		minimum := left // left child
		if right := left + 1; right < n && h.Less(right, left) {
			minimum = right // = 2*i + 2  // right child
		}

		if !h.Less(minimum, i) {
			break
		}

		h.Swap(i, minimum)
		i = minimum
	}
	return i > i0
}
