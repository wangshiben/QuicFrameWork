package defaultSessionImp

import (
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
	lock    sync.Mutex
	expTime time.Duration
}

func (m *BaseServerSession) GetItem(key string) Session.ItemInterFace {
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
	switch key.(type) {
	case string:
		m.store.StoreItemInterFace(key.(string), val)
		m.store.UpdateUsedTime(key.(string), time.Now().Unix())
		return true
	default:
		return false
	}
}
func (m *BaseServerSession) Close() error {
	err := m.store.Close()
	if err != nil {
		return err
	}
	return nil

}

// DestroySelf only Server exit called
func (m *BaseServerSession) DestroySelf() bool {
	m.store = nil
	return true
}
func (m *BaseServerSession) GetExpireTime() time.Duration {
	if m.expTime == 0 {
		m.lock.Lock()
		defer m.lock.Unlock()
		m.expTime = DefaultExpTime
	}
	return m.expTime
	//return DefaultExpTime
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
			Domain:   "localhost",
			Expires:  time.Now().Add(m.GetExpireTime()),
		}
		w.Header().Set("Set-Cookie", cookieIn.String())
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
	lastCallTime := m.store.GetLastCallTime(key)
	if lastCallTime == 0 {
		return -1
	}
	return lastCallTime
}

func (m *BaseServerSession) CleanExpItem() {
	expTime, now := m.expTime, time.Now().Unix()
	for key, val := range m.store.GetCallTimeMap() {
		if now-val > int64(expTime.Seconds()) {
			m.RemoveItem(key)
		}
	}
}
func (m *BaseServerSession) GetNextTimePicker() time.Duration {
	return m.GetExpireTime() / 2
}
func NewServerSession() *BaseServerSession {
	return &BaseServerSession{
		store: newDefaultStoreItem(),
		lock:  sync.Mutex{},
	}
}
func NewServerSessionWithStore(store Session.StoreStruct) Session.ServerSession {
	return &BaseServerSession{
		store:   store,
		lock:    sync.Mutex{},
		expTime: 0,
	}
}

func newDefaultStoreItem() Session.StoreStruct {
	return &defaultStoreImp{
		store:       make(map[string]Session.ItemInterFace),
		callTimeMap: make(map[string]int64),
	}
}
