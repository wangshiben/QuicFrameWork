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
}

func (m *CircularQueueSession) GetItem(key string) Session.ItemInterFace {
	return m.Base.GetItem(key)
}
func (m *CircularQueueSession) RemoveItem(key string) bool {
	return m.Base.RemoveItem(key)
}
func (m *CircularQueueSession) StoreSession(key any, val Session.ItemInterFace) bool {
	return m.Base.StoreSession(key, val)
}

// DestroySelf only Server exit called
func (m *CircularQueueSession) DestroySelf() bool {
	return m.Base.DestroySelf()
}
func (m *CircularQueueSession) GetExpireTime() time.Duration {
	return m.Base.GetExpireTime()
}
func (m *CircularQueueSession) SetExpireTime(exp time.Duration) {
	m.Base.SetExpireTime(exp)
	m.StoreItem.queue = make([]queueItem, exp/time.Minute)
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
	return m.Base.GetLastCallTime(key)
}

func (m *CircularQueueSession) CleanExpItem() {
	m.StoreItem.RemoveCurrentIndex()

}

// GetNextTimePicker wait x time to call CleanExpItem()
func (m *CircularQueueSession) GetNextTimePicker() time.Duration {
	return m.GetExpireTime() / time.Minute
}
func (m *CircularQueueSession) Close() error {
	err := m.Base.Close()
	if err != nil {
		return err
	}
	return m.StoreItem.Close()
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
	}
	store.queue = make([]queueItem, ServerSession.GetExpireTime()/time.Minute)
	return ServerSession
}
