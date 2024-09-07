package Session

import (
	"net/http"
	"time"
)

type ItemInterFace interface {
	GetMemoPosition() MemoryPosition
	Store(sessionKey string, sessionValue interface{}) error
	GetStoreValue(sessionKey string) (interface{}, error)
	RemoveStoreValue(sessionKey string) (bool, error)
}
type GenerateItemInterFace func() ItemInterFace
type ResponseSetSession func(w http.ResponseWriter, Key string)
type GenerateName func(initFunc GenerateItemInterFace) (string, ItemInterFace)

// ServerSession 类似于Java中的ApplicationContext中的SessionMap
type ServerSession interface {
	//GetMemoPosition() MemoryPosition
	GetItem(key string) ItemInterFace
	RemoveItem(key string) bool
	Destroy() bool
	StoreSession(key any, val ItemInterFace) bool
	GetExpireTime() time.Duration // 返回单个Session的过期时间
	// GetKeyFromRequest 从请求中获取SessionKey
	GetKeyFromRequest(r *http.Request) (key string, exist bool)
	SetKeyToResponse() ResponseSetSession
	GenerateName() GenerateName
}
