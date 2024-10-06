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

type MemoServerSession struct {
	memoMap storeStruct[Session.ItemInterFace]
	timeMap storeStruct[int64]
	lock    sync.Mutex
	expTime time.Duration
}
type storeStruct[T interface{}] interface {
	StoreItem(key string, val T)
	GetItem(Key string) T
	DeleteItem(key string)
	GetRange() map[string]T // only used in CleanExpItem
}
type defaultStoreImp[T interface{}] struct {
	store map[string]T
}

func (d *defaultStoreImp[T]) StoreItem(key string, val T) {
	d.store[key] = val
}
func (d *defaultStoreImp[T]) GetItem(Key string) T {
	return d.store[Key]
}
func (d *defaultStoreImp[T]) DeleteItem(Key string) {
	delete(d.store, Key)
}
func (d *defaultStoreImp[T]) GetRange() map[string]T {
	return d.store
}

func (m *MemoServerSession) GetItem(key string) Session.ItemInterFace {
	res := m.memoMap.GetItem(key)
	if res != nil {
		m.timeMap.StoreItem(key, time.Now().Unix())
	}
	return res
}
func (m *MemoServerSession) RemoveItem(key string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.memoMap.DeleteItem(key)
	m.timeMap.DeleteItem(key)
	return true
}
func (m *MemoServerSession) StoreSession(key any, val Session.ItemInterFace) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	switch key.(type) {
	case string:
		m.memoMap.StoreItem(key.(string), val)
		return true
	default:
		return false
	}
}
func (m *MemoServerSession) DestroySelf() bool {
	m.memoMap = nil
	return true
}
func (m *MemoServerSession) GetExpireTime() time.Duration {
	if m.expTime == 0 {
		m.lock.Lock()
		defer m.lock.Unlock()
		m.expTime = DefaultExpTime
	}
	return m.expTime
	//return DefaultExpTime
}
func (m *MemoServerSession) SetExpireTime(exp time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.expTime = exp
}
func (m *MemoServerSession) GetKeyFromRequest(req *http.Request) (string, bool) {
	cookie, err := req.Cookie(quicSessionName)
	if err != nil || cookie == nil {
		return "", false
	}
	return cookie.Value, true
}
func (m *MemoServerSession) SetKeyToResponse() Session.ResponseSetSession {
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
func (m *MemoServerSession) GenerateName() Session.GenerateName {
	return func(initFunc Session.GenerateItemInterFace) (string, Session.ItemInterFace) {
		uid := uuid.New()
		return uid.String(), initFunc()
	}
}

// GetLastCallTime 上次调用的时间戳
func (m *MemoServerSession) GetLastCallTime(key string) int64 {
	if m.timeMap.GetItem(key) == 0 {
		return -1
	}
	return m.timeMap.GetItem(key)
}

func (m *MemoServerSession) CleanExpItem() {
	expTime, now := m.expTime, time.Now().Unix()
	for key, val := range m.timeMap.GetRange() {
		if now-val > int64(expTime.Seconds()) {
			m.RemoveItem(key)
		}
	}
}
func NewMemoServerSession() *MemoServerSession {
	return &MemoServerSession{
		memoMap: newDefaultStoreItem[Session.ItemInterFace](),
		timeMap: newDefaultStoreItem[int64](),
		lock:    sync.Mutex{},
	}
}
func newDefaultStoreItem[T interface{}]() storeStruct[T] {
	return &defaultStoreImp[T]{store: make(map[string]T, 0)}
}
