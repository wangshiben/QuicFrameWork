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
	lock    sync.Mutex
	ExpTime time.Duration
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
func NewMemoServerSession() *MemoServerSession {
	return &MemoServerSession{
		memoMap: make(map[string]Session.ItemInterFace),
		lock:    sync.Mutex{},
	}
}
