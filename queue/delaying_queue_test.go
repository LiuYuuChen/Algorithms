package queue

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func TestDelayingQueue_MainQueueBasicFunction(t *testing.T) {
	cfg := &config{lock: &sync.RWMutex{}}
	queue := newDelayingQueue[*testItem](&testConstraint{}, cfg)
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

			item.key = "Item_100"
			queue.Add(item)
			ret, ok := queue.Get(item)
			convey.So(ok, convey.ShouldBeTrue)

			convey.So(ret.key == item.key, convey.ShouldBeTrue)

			err := queue.Delete(item)
			convey.So(err == nil, convey.ShouldBeTrue)
			convey.So(queue.Len() == testItemNum, convey.ShouldBeTrue)

			item.key = "Item_0"
			item.value = testItemNum
			err = queue.Update(item)
			convey.So(err == nil, convey.ShouldBeTrue)
			ret, _ = queue.Get(item)
			convey.So(ret.value == testItemNum, convey.ShouldBeTrue)
			item.value = 0
			err = queue.Update(item)
			convey.So(err == nil, convey.ShouldBeTrue)
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

	})
}

func TestDelayingQueue_DelayingQueueFunctions(t *testing.T) {
	cfg := &config{lock: &sync.RWMutex{}}
	queue := newDelayingQueue[*testItem](&testConstraint{}, cfg)
	testItems := make([]*testItem, testItemNum)
	for i := range testItems {
		item := &testItem{
			key:   fmt.Sprintf("Item_%d", i),
			value: i,
		}
		testItems[i] = item
	}

	convey.Convey("test delay functions", t, func() {
		convey.Convey("test AddAfter", func() {
			interval := 100 * time.Millisecond
			go func() {
				for _, item := range testItems {
					queue.AddAfter(item, interval)
					time.Sleep(interval)
				}
			}()
			time.Sleep(50 * time.Millisecond)
			for i := 0; i < 10; i++ {
				convey.So(queue.Len() == i+1, convey.ShouldBeTrue)
				time.Sleep(interval)
			}

			newItem := &testItem{
				key: "Item_11",
			}

			queue.AddAfter(newItem, -time.Second)
			convey.So(queue.Len() == 11, convey.ShouldBeTrue)
		})

		convey.Convey("test waitQueue relate functions", func() {
			newItem := &testItem{
				key: "Item_12",
			}
			queue.AddAfter(newItem, time.Minute)
			time.Sleep(10 * time.Millisecond)

			updateItem := &testItem{
				key:   "Item_12",
				value: 10,
			}
			err := queue.Update(updateItem)
			convey.So(err == nil, convey.ShouldBeTrue)

			getItem, ok := queue.Get(newItem)
			convey.So(ok, convey.ShouldBeTrue)
			convey.So(getItem.key == newItem.key, convey.ShouldBeTrue)
			convey.So(getItem.value == updateItem.value, convey.ShouldBeTrue)

			convey.So(queue.Len() == 12, convey.ShouldBeTrue)

			err = queue.Delete(newItem)
			convey.So(err == nil, convey.ShouldBeTrue)
			_, ok = queue.Get(newItem)
			convey.So(!ok, convey.ShouldBeTrue)

			convey.So(queue.Len() == 11, convey.ShouldBeTrue)
		})

		convey.Convey("shutdown", func() {
			queue.Shutdown()
			newItem := &testItem{
				key: "Item_12",
			}

			queue.AddAfter(newItem, time.Second)
			convey.So(queue.Len() == 11, convey.ShouldBeTrue)

		})
	})
}
