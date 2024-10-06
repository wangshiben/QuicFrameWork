package Session

import (
	"net/http"
	"time"
)

// SesionItem
type ItemInterFace interface {
	GetMemoPosition() MemoryPosition
	Store(sessionKey string, sessionValue interface{}) error
	GetStoreValue(sessionKey string) (interface{}, error)
	RemoveStoreValue(sessionKey string) (bool, error)
}

// 生成SessionItem的函数
type GenerateItemInterFace func() ItemInterFace
type ResponseSetSession func(w http.ResponseWriter, Key string)
type GenerateName func(initFunc GenerateItemInterFace) (string, ItemInterFace)

// ServerSession 类似于Java中的ApplicationContext中的SessionMap
type ServerSession interface {
	//GetMemoPosition() MemoryPosition
	GetItem(key string) ItemInterFace
	RemoveItem(key string) bool                   //从session中移除某个元素
	DestroySelf() bool                            // 生命周期结束调用
	StoreSession(key any, val ItemInterFace) bool //生成后存储单个Session
	GetExpireTime() time.Duration                 // 返回单个Session的过期时间
	// GetKeyFromRequest 从请求中获取SessionKey
	GetKeyFromRequest(r *http.Request) (key string, exist bool)
	SetKeyToResponse() ResponseSetSession // 在首次response中设置sessionKey,可自定义key的存在形式，类似于java中的JSESSION Cookie
	GenerateName() GenerateName           //生成 map[SessionItem]string中的string
	CleanExpItem()                        // 清理过期的Session
}
