package defaultSessionImp

import "github.com/wangshiben/QuicFrameWork/Session"

type defaultStoreImp struct {
	store       map[string]Session.ItemInterFace
	callTimeMap map[string]int64
}

func (d *defaultStoreImp) StoreItemInterFace(key string, val Session.ItemInterFace) {
	d.store[key] = val
}
func (d *defaultStoreImp) UpdateUsedTime(key string, timeStamp int64) {
	d.callTimeMap[key] = timeStamp
}

func (d *defaultStoreImp) RemoveItem(key string) {
	delete(d.store, key)
	delete(d.callTimeMap, key)
}
func (d *defaultStoreImp) GetItemInterFace(key string) Session.ItemInterFace {
	return d.store[key]
}
func (d *defaultStoreImp) GetLastCallTime(key string) int64 {
	return d.callTimeMap[key]
}
func (d *defaultStoreImp) GetCallTimeMap() map[string]int64 {
	return d.callTimeMap
}
