package defaultSessionImp

import (
	"github.com/wangshiben/QuicFrameWork/server/Session"
	"sync"
)

type MemoServerSession struct {
	memoMap map[string]Session.ItemInterFace
	lock    sync.Mutex
}

func (m *MemoServerSession) GetItem(key string) Session.ItemInterFace {
	return m.memoMap[key]
}
func (m *MemoServerSession) RemoveItem(key string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.memoMap, key)
	return true
}
func (m *MemoServerSession) StoreSession(key any, val Session.ItemInterFace) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch key.(type) {
	case string:
		m.memoMap[key.(string)] = val
		return true
	default:
		return false
	}
}
func (m *MemoServerSession) Destroy() bool {
	m.memoMap = nil
	return true
}
func NewMemoServerSession() *MemoServerSession {
	return &MemoServerSession{
		memoMap: make(map[string]Session.ItemInterFace),
		lock:    sync.Mutex{},
	}
}
