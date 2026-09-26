package defaultSessionImp

import (
	"github.com/wangshiben/QuicFrameWork/Session"
	"sync"
)

type defaultStoreImp struct {
	store       map[string]Session.ItemInterFace
	callTimeMap map[string]int64
	lock        sync.RWMutex
}

func (d *defaultStoreImp) StoreItemInterFace(key string, val Session.ItemInterFace) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.store[key] = val
}
func (d *defaultStoreImp) UpdateUsedTime(key string, timeStamp int64) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.callTimeMap[key] = timeStamp
}

func (d *defaultStoreImp) RemoveItem(key string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	delete(d.store, key)
	delete(d.callTimeMap, key)
}
func (d *defaultStoreImp) GetItemInterFace(key string) Session.ItemInterFace {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.store[key]
}
func (d *defaultStoreImp) GetLastCallTime(key string) int64 {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.callTimeMap[key]
}
func (d *defaultStoreImp) GetCallTimeMap() map[string]int64 {
	d.lock.RLock()
	defer d.lock.RUnlock()

	result := make(map[string]int64, len(d.callTimeMap))
	for key, timestamp := range d.callTimeMap {
		result[key] = timestamp
	}
	return result
}
func (d *defaultStoreImp) Close() error {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.store = nil
	d.callTimeMap = nil
	return nil
}
