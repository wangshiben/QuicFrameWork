package CircularQueueSessionImp

import (
	"github.com/wangshiben/QuicFrameWork/Session"
	"sync"
	"time"
)

type CircularQueueStore struct {
	// 存储结构
	store map[string]Session.ItemInterFace

	lock sync.RWMutex
	// 调用时队列
	queue        []queueItem
	CurrentIndex int
	callTimeMap  map[string][]int // index of queue(x,y) -> queue[x].get(y)
}

func (c *CircularQueueStore) StoreItemInterFace(key string, val Session.ItemInterFace) {
	c.store[key] = val

}
func (c *CircularQueueStore) UpdateUsedTime(key string, timeStamp int64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	queueLen := len(c.queue)
	indexX := (c.CurrentIndex + queueLen - 1) % queueLen
	indexY := c.queue[indexX].add(key)
	if len(c.callTimeMap[key]) != 0 {
		if c.callTimeMap[key][0] == indexX {
			return
		}
	}
	c.callTimeMap[key] = []int{indexX, indexY}
}
func (c *CircularQueueStore) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.store = nil
	c.callTimeMap = nil
	return nil
}

func (c *CircularQueueStore) RemoveItem(key string) {
	delete(c.store, key)
	delete(c.callTimeMap, key)
}
func (c *CircularQueueStore) GetItemInterFace(key string) Session.ItemInterFace {
	return c.store[key]
}
func (c *CircularQueueStore) GetLastCallTime(key string) int64 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	indexes := c.callTimeMap[key]
	if len(indexes) != 0 {
		return c.queue[indexes[0]].Time
	}
	return -1
}
func (c *CircularQueueStore) GetCallTimeMap() map[string]int64 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	mapResult := make(map[string]int64)
	for _, item := range c.queue {
		value := item.Time
		for _, key := range item.Keys {
			mapResult[key] = value
		}
	}

	return mapResult
}
func (c *CircularQueueStore) RemoveCurrentIndex() {
	c.lock.Lock()
	defer c.lock.Unlock()
	expKeys := c.queue[c.CurrentIndex]
	for _, key := range expKeys.Keys {
		delete(c.callTimeMap, key)
		delete(c.store, key)
	}
	c.queue[c.CurrentIndex] = queueItem{
		Time: time.Now().Unix(),
		Keys: []string{},
	}
	c.CurrentIndex = (c.CurrentIndex + 1) % len(c.queue)
}

type queueItem struct {
	Keys []string
	Time int64 // 调用时间
}

func (queue *queueItem) get(Index int) string {
	return queue.Keys[Index]
}
func (queue *queueItem) add(key string) int {
	queue.Keys = append(queue.Keys, key)
	return len(queue.Keys) - 1
}
