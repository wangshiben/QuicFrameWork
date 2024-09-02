package defaultSessionImp

import (
	"github.com/wangshiben/QuicFrameWork/server/Session"
	"sync"
)

type MemorySession struct {
	MemoryPosition Session.MemoryPosition
	sessionMap     map[string]interface{}
	lock           sync.Mutex
}

func (m *MemorySession) GetMemoPosition() Session.MemoryPosition {
	return Session.Memo
}
func (m *MemorySession) Store(key string, value interface{}) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.sessionMap[key] = value
	return nil
}
func (m *MemorySession) GetStoreValue(Key string) (interface{}, error) {
	return m.sessionMap[Key], nil
}
func (m *MemorySession) RemoveStoreValue(Key string) (bool, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.sessionMap, Key)
	return true, nil
}
func NewMemoItemInterFace() Session.ItemInterFace {
	resp := MemorySession{
		MemoryPosition: Session.Memo,
		sessionMap:     make(map[string]interface{}),
		lock:           sync.Mutex{},
	}
	return &resp
}
