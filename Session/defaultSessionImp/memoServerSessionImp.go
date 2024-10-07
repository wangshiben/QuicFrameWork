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
	store   storeStruct
	lock    sync.Mutex
	expTime time.Duration
}
type storeStruct interface {
	StoreItemInterFace(key string, val Session.ItemInterFace)
	UpdateUsedTime(key string, timeStamp int64)
	RemoveItem(key string)
	GetItemInterFace(key string) Session.ItemInterFace
	GetLastCallTime(key string) int64
	GetCallTimeMap() map[string]int64
}
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
func NewMemoServerSession() *BaseServerSession {
	return &BaseServerSession{
		store: newDefaultStoreItem(),
		lock:  sync.Mutex{},
	}
}
func newDefaultStoreItem() storeStruct {
	return &defaultStoreImp{
		store:       make(map[string]Session.ItemInterFace, 0),
		callTimeMap: make(map[string]int64, 0),
	}
}
