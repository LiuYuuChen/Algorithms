package queue

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

const testItemNum = 10

type testItem struct {
	key   string
	value int
}

type testConstraint struct {
}

func (t *testConstraint) FormStoreKey(item *testItem) string {
	return item.key
}
func (t *testConstraint) Less(left, right *testItem) bool {
	return left.value < right.value
}

func Test_BasicBlockQueueFunction(t *testing.T) {
	cfg := &config{lock: &sync.RWMutex{}}
	queue := newBlockQueue[*testItem](&testConstraint{}, cfg)
	testItems := make([]*testItem, testItemNum)
	for i := range testItems {
		item := &testItem{
			key:   fmt.Sprintf("Item_%d", i),
			value: i,
		}
		testItems[i] = item
	}

	//itemMap := formTestItemMap(formTestKey, testItems)

	convey.Convey("test basic block queue functions", t, func() {
		convey.Convey("test Add", func() {
			for _, item := range testItems {
				queue.Add(item)
			}
		})

		convey.Convey("test get, peek, delete and update", func() {
			item := &testItem{
				key:   fmt.Sprintf("Item_%d", 1),
				value: 1,
			}
			peek, err := queue.Peek()
			convey.So(err == nil, convey.ShouldBeTrue)
			queueKey := peek.key
			item.key = "Item_0"
			convey.So(queueKey == item.key, convey.ShouldBeTrue)

			item.key = "Item_100"
			queue.Add(item)
			convey.So(err == nil, convey.ShouldBeTrue)
			ret, ok := queue.Get(item)
			convey.So(ok, convey.ShouldBeTrue)
			convey.So(ret.key == item.key, convey.ShouldBeTrue)

			err = queue.Delete(item)
			convey.So(err == nil, convey.ShouldBeTrue)
			convey.So(queue.Len() == testItemNum, convey.ShouldBeTrue)

			item.key = "Item_0"
			item.value = testItemNum
			err = queue.Update(item)
			convey.So(err == nil, convey.ShouldBeTrue)
			ret, _ = queue.Get(item)
			convey.So(ret.value == testItemNum, convey.ShouldBeTrue)
			item.value = 0
			_ = queue.Update(item)
		})

		convey.Convey("test Pop", func() {
			for _, value := range testItems {
				popItem, err := queue.Pop()
				convey.So(err == nil, convey.ShouldBeTrue)
				convey.So(value.key, convey.ShouldEqual, popItem.key)
			}

			convey.So(queue.Len() == 0, convey.ShouldBeTrue)
		})

		convey.Convey("test Pop wait", func() {
			go func() {
				newItem := &testItem{
					key: "Item_11",
				}

				time.Sleep(time.Second)
				queue.Add(newItem)
			}()

			popItem, err := queue.Pop()
			convey.So(err == nil, convey.ShouldBeTrue)
			convey.So(popItem.key, convey.ShouldEqual, "Item_11")
		})

		convey.Convey("test shutdown", func() {
			newItem := &testItem{
				key: "Item_11",
			}

			queue.Add(newItem)
			go func() {
				time.Sleep(100 * time.Millisecond)
				queue.Shutdown()
			}()
			_, err := queue.Pop()
			convey.So(err == nil, convey.ShouldBeTrue)

			_, err = queue.Pop()
			convey.So(err != nil, convey.ShouldBeTrue)
		})
	})
}

func Test_BlockQueueConcurrent(t *testing.T) {
	cfg := &config{lock: &sync.RWMutex{}}
	queue := newBlockQueue[*testItem](&testConstraint{}, cfg)
	testItems := make([]*testItem, testItemNum)
	for i := range testItems {
		item := &testItem{
			key:   fmt.Sprintf("Item_%d", i),
			value: i,
		}
		testItems[i] = item
	}

	convey.Convey("test block queue concurrent", t, func() {
		for _, item := range testItems {
			queue.Add(item)
		}

		const patchSize = 100

		addProcessor := func(patch int) {
			item := &testItem{
				key:   fmt.Sprintf("Item_%d", 1000),
				value: 1000,
			}
			scope := [2]int{1, 5}
			for i := 0; i < patch; i++ {
				randSleep := time.Duration(randInt(scope)) * time.Millisecond
				queue.Add(item)
				time.Sleep(randSleep)
			}
		}

		delProcessor := func(patch int) {
			item := &testItem{
				key:   fmt.Sprintf("Item_%d", 1000),
				value: 1000,
			}
			scope := [2]int{1, 5}
			for i := 0; i < patch; i++ {
				randSleep := time.Duration(randInt(scope)) * time.Millisecond
				_ = queue.Delete(item)
				time.Sleep(randSleep)
			}
		}

		concurrent := func(patch int) {
			item := &testItem{
				key:   fmt.Sprintf("Item_%d", 1000),
				value: 1000,
			}

			scope := [2]int{1, 5}
			for i := 0; i < patch; i++ {
				randSleep := time.Duration(randInt(scope)) * time.Millisecond
				go queue.Delete(item)
				go queue.Add(item)
				time.Sleep(randSleep)
			}
		}

		convey.Convey("concurrent Add and delete", func() {
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				addProcessor(patchSize)
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				delProcessor(patchSize)
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				concurrent(patchSize)
				wg.Done()
			}()
			wg.Wait()
		})
	})
}

func randInt(scope [2]int) int {
	rand.Seed(time.Now().UnixNano())
	if scope[0] > scope[1] {
		scope[0], scope[1] = scope[1], scope[0]
	}
	return scope[0] + rand.Intn(scope[1]-scope[0])
}
