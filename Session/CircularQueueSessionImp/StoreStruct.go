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
	c.lock.Lock()
	defer c.lock.Unlock()
	c.store[key] = val
}
func (c *CircularQueueStore) UpdateUsedTime(key string, timeStamp int64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	queueLen := len(c.queue)
	if queueLen == 0 {
		return
	}
	indexX := (c.CurrentIndex + queueLen - 1) % queueLen
	if indexes := c.callTimeMap[key]; len(indexes) != 0 && indexes[0] == indexX {
		c.queue[indexX].Time = timeStamp
		return
	}
	indexY := c.queue[indexX].add(key)
	c.queue[indexX].Time = timeStamp
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
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.store, key)
	delete(c.callTimeMap, key)
}
func (c *CircularQueueStore) GetItemInterFace(key string) Session.ItemInterFace {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.store[key]
}
func (c *CircularQueueStore) GetLastCallTime(key string) int64 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	indexes := c.callTimeMap[key]
	if len(indexes) != 0 && indexes[0] >= 0 && indexes[0] < len(c.queue) {
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
	if len(c.queue) == 0 {
		return
	}
	expKeys := c.queue[c.CurrentIndex]
	for _, key := range expKeys.Keys {
		indexes, exists := c.callTimeMap[key]
		if exists && len(indexes) != 0 && indexes[0] == c.CurrentIndex {
			delete(c.callTimeMap, key)
			delete(c.store, key)
		}
	}
	c.queue[c.CurrentIndex] = queueItem{
		Time: time.Now().Unix(),
		Keys: []string{},
	}
	c.CurrentIndex = (c.CurrentIndex + 1) % len(c.queue)
}

func (c *CircularQueueStore) ResetQueue(bucketCount int) {
	if bucketCount < 1 {
		bucketCount = 1
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	c.queue = make([]queueItem, bucketCount)
	c.CurrentIndex = 0
	c.callTimeMap = make(map[string][]int, len(c.store))
	lastBucket := bucketCount - 1
	c.queue[lastBucket].Time = time.Now().Unix()
	for key := range c.store {
		index := c.queue[lastBucket].add(key)
		c.callTimeMap[key] = []int{lastBucket, index}
	}
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
