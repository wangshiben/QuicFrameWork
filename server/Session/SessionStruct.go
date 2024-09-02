package Session

type ItemInterFace interface {
	GetMemoPosition() MemoryPosition
	Store(sessionKey string, sessionValue interface{}) error
	GetStoreValue(sessionKey string) (interface{}, error)
	RemoveStoreValue(sessionKey string) (bool, error)
}
type GenerateItemInterFace func() ItemInterFace

type ServerSession interface {
	//GetMemoPosition() MemoryPosition
	GetItem(key string) ItemInterFace
	RemoveItem(key string) bool
	Destroy() bool
	StoreSession(key any, val ItemInterFace) bool
}
