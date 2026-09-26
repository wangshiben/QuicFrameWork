package CircularQueueSessionImp

import (
	"github.com/wangshiben/QuicFrameWork/Session"
	"github.com/wangshiben/QuicFrameWork/Session/defaultSessionImp"
	"net/http"
	"sync"
	"time"
)

type CircularQueueSession struct {
	Base      *defaultSessionImp.BaseServerSession
	StoreItem *CircularQueueStore
	lock      sync.Mutex
}

func (m *CircularQueueSession) GetItem(key string) Session.ItemInterFace {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.Base.GetItem(key)
}
func (m *CircularQueueSession) RemoveItem(key string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.Base.RemoveItem(key)
}
func (m *CircularQueueSession) StoreSession(key any, val Session.ItemInterFace) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.Base.StoreSession(key, val)
}

// DestroySelf only Server exit called
func (m *CircularQueueSession) DestroySelf() bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.Base.DestroySelf()
}
func (m *CircularQueueSession) GetExpireTime() time.Duration {
	return m.Base.GetExpireTime()
}
func (m *CircularQueueSession) SetExpireTime(exp time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if exp <= 0 {
		exp = time.Minute
	}
	m.Base.SetExpireTime(exp)
	bucketCount := int((exp + time.Minute - 1) / time.Minute)
	m.StoreItem.ResetQueue(bucketCount)
}
func (m *CircularQueueSession) GetKeyFromRequest(req *http.Request) (string, bool) {
	return m.Base.GetKeyFromRequest(req)
}
func (m *CircularQueueSession) SetKeyToResponse() Session.ResponseSetSession {
	return m.Base.SetKeyToResponse()
}
func (m *CircularQueueSession) GenerateName() Session.GenerateName {
	return m.Base.GenerateName()
}

// GetLastCallTime 上次调用的时间戳
func (m *CircularQueueSession) GetLastCallTime(key string) int64 {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.Base.GetLastCallTime(key)
}

func (m *CircularQueueSession) CleanExpItem() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.StoreItem.RemoveCurrentIndex()

}

// GetNextTimePicker wait x time to call CleanExpItem()
func (m *CircularQueueSession) GetNextTimePicker() time.Duration {
	return time.Minute
}
func (m *CircularQueueSession) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.Base.Close()
}

func NewServerSession() *CircularQueueSession {
	store := &CircularQueueStore{
		store:        make(map[string]Session.ItemInterFace),
		lock:         sync.RWMutex{},
		queue:        nil,
		CurrentIndex: 0,
		callTimeMap:  make(map[string][]int),
	}
	BaseStore := defaultSessionImp.NewServerSessionWithStore(store).(*defaultSessionImp.BaseServerSession)
	ServerSession := &CircularQueueSession{
		Base:      BaseStore,
		StoreItem: store,
		lock:      sync.Mutex{},
	}
	store.ResetQueue(int(ServerSession.GetExpireTime() / time.Minute))
	return ServerSession
}
