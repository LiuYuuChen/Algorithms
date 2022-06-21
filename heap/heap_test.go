package heap

import (
	"sync"
	"testing"
)

type testHeapObject struct {
	name string
	val  int
}

func mkHeapObj(name string, val int) testHeapObject {
	return testHeapObject{name: name, val: val}
}

type priorityHandler struct {
}

func (p *priorityHandler) FormStoreKey(value testHeapObject) string {
	return value.name
}

func (p *priorityHandler) Less(key1, key2 testHeapObject) bool {
	return key1.val < key2.val
}

func Test_HeapFunction(t *testing.T) {
	handler := priorityHandler{}
	h := New[string, testHeapObject](&handler)
	const amount = 500
	var i int

	for i = amount; i > 0; i-- {
		h.Add(mkHeapObj(string([]rune{'a', rune(i)}), i))
	}

	// Make sure that the numbers are popped in ascending order.
	prevNum := 0
	for i := 0; i < amount; i++ {
		obj, err := h.Pop()
		num := obj.val
		// All the items must be sorted.
		if err != nil || prevNum > num {
			t.Errorf("got %v out of order, last was %v", obj, prevNum)
		}
		prevNum = num
	}
}

// Tests heap.Add and ensures that heap invariant is preserved after adding items.
func TestHeap_Add(t *testing.T) {
	handler := priorityHandler{}
	h := New[string, testHeapObject](&handler)
	h.Add(mkHeapObj("foo", 10))
	h.Add(mkHeapObj("bar", 1))
	h.Add(mkHeapObj("baz", 11))
	h.Add(mkHeapObj("zab", 30))
	h.Add(mkHeapObj("foo", 13)) // This updates "foo".

	item, err := h.Pop()
	if e, a := 1, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
	item, err = h.Pop()
	if e, a := 11, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
	_ = h.Delete(mkHeapObj("baz", 11)) // Nothing is deleted.
	h.Add(mkHeapObj("foo", 14))        // foo is updated.
	item, err = h.Pop()
	if e, a := 14, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
	item, err = h.Pop()
	if e, a := 30, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
}

// TestHeap_Delete tests heap.Delete and ensures that heap invariant is
// preserved after deleting items.
func TestHeap_Delete(t *testing.T) {
	handler := priorityHandler{}
	h := newHeap[string, testHeapObject](&handler)
	h.Add(mkHeapObj("foo", 10))
	h.Add(mkHeapObj("bar", 1))
	h.Add(mkHeapObj("bal", 31))
	h.Add(mkHeapObj("baz", 11))

	// Delete head. Delete should work with "key" and doesn't care about the value.
	if err := h.Delete(mkHeapObj("bar", 200)); err != nil {
		t.Fatalf("Failed to delete head.")
	}
	item, err := h.Pop()
	if e, a := 10, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
	h.Add(mkHeapObj("zab", 30))
	h.Add(mkHeapObj("faz", 30))
	length := h.data.Len()
	// Delete non-existing item.
	if err = h.Delete(mkHeapObj("non-existent", 10)); err == nil || length != h.data.Len() {
		t.Fatalf("Didn't expect any item removal")
	}
	// Delete tail.
	if err = h.Delete(mkHeapObj("bal", 31)); err != nil {
		t.Fatalf("Failed to delete tail.")
	}
	// Delete one of the items with value 30.
	if err = h.Delete(mkHeapObj("zab", 30)); err != nil {
		t.Fatalf("Failed to delete item.")
	}
	item, err = h.Pop()
	if e, a := 11, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
	item, err = h.Pop()
	if e, a := 30, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
	if h.data.Len() != 0 {
		t.Fatalf("expected an empty heap.")
	}
}

// TestHeap_Get tests heap.Get.
func TestHeap_Get(t *testing.T) {
	handler := priorityHandler{}
	h := New[string, testHeapObject](&handler)
	h.Add(mkHeapObj("foo", 10))
	h.Add(mkHeapObj("bar", 1))
	h.Add(mkHeapObj("bal", 31))
	h.Add(mkHeapObj("baz", 11))

	// Get works with the key.
	obj, exists := h.Get(mkHeapObj("baz", 0))
	if exists == false || obj.val != 11 {
		t.Fatalf("unexpected error in getting element")
	}
	// Get non-existing object.
	_, exists = h.Get(mkHeapObj("non-existing", 0))
	if exists == true {
		t.Fatalf("didn't expect to get any object")
	}
}

// TestHeap_List tests heap.List function.
func TestHeap_List(t *testing.T) {
	handler := priorityHandler{}
	h := New[string, testHeapObject](&handler)
	list := h.List()
	if len(list) != 0 {
		t.Errorf("expected an empty list")
	}

	items := map[string]int{
		"foo": 10,
		"bar": 1,
		"bal": 30,
		"baz": 11,
		"faz": 30,
	}
	for k, v := range items {
		h.Add(mkHeapObj(k, v))
	}
	list = h.List()
	if len(list) != len(items) {
		t.Errorf("expected %d items, got %d", len(items), len(list))
	}
	for _, heapObj := range list {
		v, ok := items[heapObj.name]
		if !ok || v != heapObj.val {
			t.Errorf("unexpected item in the list: %v", heapObj)
		}
	}
}

func Test_ConcurrentHeapFunction(t *testing.T) {
	handler := priorityHandler{}
	h := NewConcurrent[testHeapObject](&handler)
	const amount = 500
	var i int

	for i = amount; i > 0; i-- {
		h.Add(mkHeapObj(string([]rune{'a', rune(i)}), i))
	}

	// Make sure that the numbers are popped in ascending order.
	prevNum := 0
	for i := 0; i < amount; i++ {
		obj, err := h.Pop()
		num := obj.val
		// All the items must be sorted.
		if err != nil || prevNum > num {
			t.Errorf("got %v out of order, last was %v", obj, prevNum)
		}
		prevNum = num
	}
}

// Tests heap.Add and ensures that heap invariant is preserved after adding items.
func TestConcurrentHeap_Add(t *testing.T) {
	handler := priorityHandler{}
	h := NewConcurrent[testHeapObject](&handler)
	h.Add(mkHeapObj("foo", 10))
	h.Add(mkHeapObj("bar", 1))
	h.Add(mkHeapObj("baz", 11))
	h.Add(mkHeapObj("zab", 30))
	h.Add(mkHeapObj("foo", 13)) // This updates "foo".

	item, err := h.Pop()
	if e, a := 1, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
	item, err = h.Pop()
	if e, a := 11, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
	_ = h.Delete(mkHeapObj("baz", 11)) // Nothing is deleted.
	h.Add(mkHeapObj("foo", 14))        // foo is updated.
	item, err = h.Pop()
	if e, a := 14, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
	item, err = h.Pop()
	if e, a := 30, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
}

// TestHeap_Delete tests heap.Delete and ensures that heap invariant is
// preserved after deleting items.
func TestConcurrentHeap_Delete(t *testing.T) {
	cfg := options{lock: &sync.RWMutex{}}
	handler := priorityHandler{}
	h := newConcurrent[testHeapObject](&handler, &cfg)
	h.Add(mkHeapObj("foo", 10))
	h.Add(mkHeapObj("bar", 1))
	h.Add(mkHeapObj("bal", 31))
	h.Add(mkHeapObj("baz", 11))

	// Delete head. Delete should work with "key" and doesn't care about the value.
	if err := h.Delete(mkHeapObj("bar", 200)); err != nil {
		t.Fatalf("Failed to delete head.")
	}
	item, err := h.Pop()
	if e, a := 10, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
	h.Add(mkHeapObj("zab", 30))
	h.Add(mkHeapObj("faz", 30))
	length := h.data.Len()
	// Delete non-existing item.
	if err = h.Delete(mkHeapObj("non-existent", 10)); err == nil || length != h.data.Len() {
		t.Fatalf("Didn't expect any item removal")
	}
	// Delete tail.
	if err = h.Delete(mkHeapObj("bal", 31)); err != nil {
		t.Fatalf("Failed to delete tail.")
	}
	// Delete one of the items with value 30.
	if err = h.Delete(mkHeapObj("zab", 30)); err != nil {
		t.Fatalf("Failed to delete item.")
	}
	item, err = h.Pop()
	if e, a := 11, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
	item, err = h.Pop()
	if e, a := 30, item.val; err != nil || a != e {
		t.Fatalf("expected %d, got %d", e, a)
	}
	if h.data.Len() != 0 {
		t.Fatalf("expected an empty heap.")
	}
}

// TestHeap_Get tests heap.Get.
func TestConcurrentHeap_Get(t *testing.T) {
	handler := priorityHandler{}
	h := NewConcurrent[testHeapObject](&handler)
	h.Add(mkHeapObj("foo", 10))
	h.Add(mkHeapObj("bar", 1))
	h.Add(mkHeapObj("bal", 31))
	h.Add(mkHeapObj("baz", 11))

	// Get works with the key.
	obj, exists := h.Get(mkHeapObj("baz", 0))
	if exists == false || obj.val != 11 {
		t.Fatalf("unexpected error in getting element")
	}
	// Get non-existing object.
	_, exists = h.Get(mkHeapObj("non-existing", 0))
	if exists == true {
		t.Fatalf("didn't expect to get any object")
	}
}

// TestHeap_List tests heap.List function.
func TestConcurrentHeap_List(t *testing.T) {
	handler := priorityHandler{}
	h := NewConcurrent[testHeapObject](&handler)
	list := h.List()
	if len(list) != 0 {
		t.Errorf("expected an empty list")
	}

	items := map[string]int{
		"foo": 10,
		"bar": 1,
		"bal": 30,
		"baz": 11,
		"faz": 30,
	}
	for k, v := range items {
		h.Add(mkHeapObj(k, v))
	}
	list = h.List()
	if len(list) != len(items) {
		t.Errorf("expected %d items, got %d", len(items), len(list))
	}
	for _, heapObj := range list {
		v, ok := items[heapObj.name]
		if !ok || v != heapObj.val {
			t.Errorf("unexpected item in the list: %v", heapObj)
		}
	}
}
