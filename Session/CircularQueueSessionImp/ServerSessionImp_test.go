package CircularQueueSessionImp

import (
	"github.com/wangshiben/QuicFrameWork/Session"
	"github.com/wangshiben/QuicFrameWork/Session/defaultSessionImp"
	"net/http"
	"testing"
	"time"
)

// 创建一个简单的测试用Item
type TestItem struct {
	Data map[string]interface{}
}

func (t *TestItem) GetMemoPosition() Session.MemoryPosition {
	return Session.Memo
}

func (t *TestItem) Store(sessionKey string, sessionValue interface{}) error {
	t.Data[sessionKey] = sessionValue
	return nil
}

func (t *TestItem) GetStoreValue(sessionKey string) (interface{}, error) {
	return t.Data[sessionKey], nil
}

func (t *TestItem) RemoveStoreValue(sessionKey string) (bool, error) {
	delete(t.Data, sessionKey)
	return true, nil
}

func TestNewServerSession(t *testing.T) {
	session := NewServerSession()
	if session == nil {
		t.Error("NewServerSession returned nil")
	}
}

func TestStoreAndGetItem(t *testing.T) {
	session := NewServerSession()
	testItem := &TestItem{Data: make(map[string]interface{})}

	// 测试存储
	key := "testKey"
	if !session.StoreSession(key, testItem) {
		t.Error("Failed to store session item")
	}

	// 测试获取
	retrievedItem := session.GetItem(key)
	if retrievedItem == nil {
		t.Error("Failed to get stored item")
	}

	// 验证是否为同一个对象
	if retrievedItem != testItem {
		t.Error("Retrieved item is not the same as stored item")
	}
}

func TestRemoveItem(t *testing.T) {
	session := NewServerSession()
	testItem := &TestItem{Data: make(map[string]interface{})}

	key := "testKey"
	session.StoreSession(key, testItem)

	// 测试移除
	if !session.RemoveItem(key) {
		t.Error("Failed to remove item")
	}

	// 验证移除后无法获取
	if session.GetItem(key) != nil {
		t.Error("Item still exists after removal")
	}
}

func TestExpireTime(t *testing.T) {
	session := NewServerSession()
	defaultExpireTime := session.GetExpireTime()
	if defaultExpireTime != defaultSessionImp.DefaultExpTime {
		t.Errorf("Expected default expire time %v, got %v", defaultSessionImp.DefaultExpTime, defaultExpireTime)
	}

	// 测试设置过期时间
	newExpireTime := 30 * time.Minute
	session.SetExpireTime(newExpireTime)

	if session.GetExpireTime() != newExpireTime {
		t.Errorf("Expected expire time %v, got %v", newExpireTime, session.GetExpireTime())
	}
}

func TestGetKeyFromRequest(t *testing.T) {
	session := NewServerSession()

	// 创建一个测试请求
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	// 测试获取key
	key, exists := session.GetKeyFromRequest(req)
	if exists {
		t.Error("Expected no key to exist in empty request")
	}
	if key != "" {
		t.Error("Expected empty key for empty request")
	}
}

func TestCleanExpItem(t *testing.T) {
	session := NewServerSession()
	testItem := &TestItem{Data: make(map[string]interface{})}

	// 存储一个测试项
	key := "testKey"
	session.StoreSession(key, testItem)

	// 执行清理
	session.CleanExpItem()

	// 验证清理后的状态
	if session.GetItem(key) == nil {
		t.Error("Item was unexpectedly removed during cleanup")
	}
}

func TestExpiredItemCleanup(t *testing.T) {
	session := NewServerSession()
	testItem := &TestItem{Data: make(map[string]interface{})}

	// 设置一个较短的过期时间，方便测试
	shortExpireTime := 2 * time.Minute
	session.SetExpireTime(shortExpireTime)

	// 存储测试项
	key := "testKey"
	if !session.StoreSession(key, testItem) {
		t.Fatal("Failed to store test item")
	}

	// 验证初始状态
	if session.GetItem(key) == nil {
		t.Fatal("Item not found immediately after storage")
	}

	// 模拟时间流逝，执行清理
	// 由于CircularQueueSession使用分钟作为单位，我们需要等待足够的时间
	// 这里我们直接调用CleanExpItem来模拟清理
	for i := 0; i < int(shortExpireTime/time.Minute); i++ {
		//time.Sleep(shortExpireTime)
		session.CleanExpItem()
	}

	// 验证过期后是否被清理
	if session.GetItem(key) != nil {
		t.Error("Item still exists after expiration time")
	}
}
