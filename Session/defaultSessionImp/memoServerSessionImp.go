package defaultSessionImp

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/wangshiben/QuicFrameWork/Session"
	"net/http"
	"sync"
	"time"
)

const DefaultExpTime = time.Minute * 30
const quicSessionName = "quickSession"

type BaseServerSession struct {
	store   Session.StoreStruct
	lock    sync.RWMutex
	expTime time.Duration
}

func (m *BaseServerSession) GetItem(key string) Session.ItemInterFace {
	m.lock.Lock()
	defer m.lock.Unlock()
	res := m.store.GetItemInterFace(key)
	if res != nil {
		m.store.UpdateUsedTime(key, time.Now().Unix())
	}
	return res
}
func (m *BaseServerSession) RemoveItem(key string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.store.RemoveItem(key)
	return true
}
func (m *BaseServerSession) StoreSession(key any, val Session.ItemInterFace) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	keyString, ok := key.(string)
	if !ok {
		return false
	}
	m.store.StoreItemInterFace(keyString, val)
	m.store.UpdateUsedTime(keyString, time.Now().Unix())
	return true
}
func (m *BaseServerSession) StoreSessionWithinLimit(key any, val Session.ItemInterFace, maxBytes int64) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	keyString, ok := key.(string)
	if !ok {
		return fmt.Errorf("session key must be a string")
	}
	if maxBytes > 0 {
		prospectiveUsage := m.memoryUsageLocked() + estimatedSessionEntrySize(keyString, val)
		if prospectiveUsage > maxBytes {
			return Session.MaxMemo
		}
	}
	m.store.StoreItemInterFace(keyString, val)
	m.store.UpdateUsedTime(keyString, time.Now().Unix())
	return nil
}
func (m *BaseServerSession) MemoryUsage() int64 {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.memoryUsageLocked()
}
func (m *BaseServerSession) memoryUsageLocked() int64 {
	if reporter, ok := m.store.(Session.MemoryUsageReporter); ok {
		return reporter.MemoryUsage()
	}
	return 0
}
func estimatedSessionEntrySize(key string, val Session.ItemInterFace) int64 {
	return 128 + int64(len(key))*3 + Session.MemoryUsageOf(val)
}
func (m *BaseServerSession) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	err := m.store.Close()
	if err != nil {
		return err
	}
	return nil

}

// DestroySelf only Server exit called
func (m *BaseServerSession) DestroySelf() bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.store = nil
	return true
}
func (m *BaseServerSession) GetExpireTime() time.Duration {
	m.lock.RLock()
	expTime := m.expTime
	m.lock.RUnlock()
	if expTime != 0 {
		return expTime
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	if m.expTime == 0 {
		m.expTime = DefaultExpTime
	}
	return m.expTime
}
func (m *BaseServerSession) SetExpireTime(exp time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.expTime = exp
}
func (m *BaseServerSession) GetKeyFromRequest(req *http.Request) (string, bool) {
	cookie, err := req.Cookie(quicSessionName)
	if err != nil || cookie == nil {
		return "", false
	}
	return cookie.Value, true
}
func (m *BaseServerSession) SetKeyToResponse() Session.ResponseSetSession {
	return func(w http.ResponseWriter, Key string) {
		cookieIn := &http.Cookie{
			Name:     quicSessionName,
			Value:    Key,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
			MaxAge:   0,
			Domain:   "",
			Expires:  time.Now().Add(m.GetExpireTime()),
		}
		http.SetCookie(w, cookieIn)
	}
}
func (m *BaseServerSession) GenerateName() Session.GenerateName {
	return func(initFunc Session.GenerateItemInterFace) (string, Session.ItemInterFace) {
		uid := uuid.New()
		return uid.String(), initFunc()
	}
}

// GetLastCallTime 上次调用的时间戳
func (m *BaseServerSession) GetLastCallTime(key string) int64 {
	m.lock.RLock()
	defer m.lock.RUnlock()
	lastCallTime := m.store.GetLastCallTime(key)
	if lastCallTime == 0 {
		return -1
	}
	return lastCallTime
}

func (m *BaseServerSession) CleanExpItem() {
	expTime, now := m.GetExpireTime(), time.Now().Unix()
	m.lock.Lock()
	defer m.lock.Unlock()
	for key, val := range m.store.GetCallTimeMap() {
		if now-val > int64(expTime.Seconds()) {
			m.store.RemoveItem(key)
		}
	}
}
func (m *BaseServerSession) GetNextTimePicker() time.Duration {
	return m.GetExpireTime() / 2
}
func NewServerSession() *BaseServerSession {
	return &BaseServerSession{
		store: newDefaultStoreItem(),
		lock:  sync.RWMutex{},
	}
}
func NewServerSessionWithStore(store Session.StoreStruct) Session.ServerSession {
	return &BaseServerSession{
		store:   store,
		lock:    sync.RWMutex{},
		expTime: 0,
	}
}

func newDefaultStoreItem() Session.StoreStruct {
	return &defaultStoreImp{
		store:       make(map[string]Session.ItemInterFace),
		callTimeMap: make(map[string]int64),
	}
}
