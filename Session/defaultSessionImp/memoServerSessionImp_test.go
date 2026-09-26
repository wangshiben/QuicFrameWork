package defaultSessionImp

import (
	"strconv"
	"sync"
	"testing"
)

func TestServerSessionConcurrentAccess(t *testing.T) {
	session := NewServerSession()
	item := NewMemoItemInterFace()
	if !session.StoreSession("seed", item) {
		t.Fatal("failed to store seed session")
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				session.StoreSession(strconv.Itoa(worker*200+j), item)
			}
		}(i)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if session.GetItem("seed") == nil {
					t.Error("seed session disappeared")
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestMemorySessionConcurrentAccess(t *testing.T) {
	session := NewMemoItemInterFace()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = session.Store("key", worker+j)
			}
		}(i)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_, _ = session.GetStoreValue("key")
			}
		}()
	}
	wg.Wait()
}
