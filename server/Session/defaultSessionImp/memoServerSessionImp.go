package defaultSessionImp

import (
	"github.com/google/uuid"
	"github.com/wangshiben/QuicFrameWork/server/Session"
	"net/http"
	"sync"
	"time"
)

const DefaultExpTime = time.Minute * 30
const quicSessionName = "quickSession"

type MemoServerSession struct {
	memoMap map[string]Session.ItemInterFace
	timeMap map[string]int64
	lock    sync.Mutex
	ExpTime time.Duration
}

func (m *MemoServerSession) GetItem(key string) Session.ItemInterFace {
	res := m.memoMap[key]
	if res != nil {
		m.timeMap[key] = time.Now().Unix()
	}
	return res
}
func (m *MemoServerSession) RemoveItem(key string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.memoMap, key)
	delete(m.timeMap, key)
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
func (m *MemoServerSession) DestroySelf() bool {
	m.memoMap = nil
	return true
}
func (m *MemoServerSession) GetExpireTime() time.Duration {
	if m.ExpTime == 0 {
		m.lock.Lock()
		defer m.lock.Unlock()
		m.ExpTime = DefaultExpTime
	}
	return m.ExpTime
	//return DefaultExpTime
}
func (m *MemoServerSession) SetExpireTime(exp time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.ExpTime = exp
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
	if m.timeMap[key] == 0 {
		return -1
	}
	return m.timeMap[key]
}

func (m *MemoServerSession) CleanExpItem() {
	expTime, now := m.ExpTime, time.Now().Unix()
	for key, val := range m.timeMap {
		if now-val > int64(expTime.Seconds()) {
			m.RemoveItem(key)
		}
	}
}
func NewMemoServerSession() *MemoServerSession {
	return &MemoServerSession{
		memoMap: make(map[string]Session.ItemInterFace),
		timeMap: make(map[string]int64),
		lock:    sync.Mutex{},
	}
}
