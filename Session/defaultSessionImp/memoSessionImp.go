package defaultSessionImp

import (
	"github.com/wangshiben/QuicFrameWork/Session"
	"sync"
)

const memorySessionEntryOverhead int64 = 32

type MemorySession struct {
	MemoryPosition Session.MemoryPosition
	sessionMap     map[string]interface{}
	entrySizes     map[string]int64
	memoryUsage    int64
	lock           sync.RWMutex
}

func (m *MemorySession) GetMemoPosition() Session.MemoryPosition {
	return Session.Memo
}
func (m *MemorySession) Store(key string, value interface{}) error {
	entrySize := int64(len(key)) + estimateStoredValueSize(value) + memorySessionEntryOverhead
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.sessionMap == nil {
		m.sessionMap = make(map[string]interface{})
	}
	if m.entrySizes == nil {
		m.entrySizes = make(map[string]int64)
	}
	m.memoryUsage -= m.entrySizes[key]
	m.sessionMap[key] = value
	m.entrySizes[key] = entrySize
	m.memoryUsage += entrySize
	return nil
}
func (m *MemorySession) GetStoreValue(Key string) (interface{}, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.sessionMap[Key], nil
}
func (m *MemorySession) RemoveStoreValue(Key string) (bool, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.sessionMap, Key)
	m.memoryUsage -= m.entrySizes[Key]
	delete(m.entrySizes, Key)
	return true, nil
}
func (m *MemorySession) MemoryUsage() int64 {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.memoryUsage
}
func NewMemoItemInterFace() Session.ItemInterFace {
	resp := MemorySession{
		MemoryPosition: Session.Memo,
		sessionMap:     make(map[string]interface{}),
		entrySizes:     make(map[string]int64),
		lock:           sync.RWMutex{},
	}
	return &resp
}

func estimateStoredValueSize(value any) (result int64) {
	return Session.MemoryUsageOf(value)
}
