package defaultSessionImp

import (
	"errors"
	"github.com/wangshiben/QuicFrameWork/Session"
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

func TestStoreSessionWithinMemoryLimit(t *testing.T) {
	session := NewServerSession()
	item := NewMemoItemInterFace()
	if err := item.Store("payload", make([]byte, 256)); err != nil {
		t.Fatal(err)
	}

	if err := session.StoreSessionWithinLimit("large", item, 64); !errors.Is(err, Session.MaxMemo) {
		t.Fatalf("StoreSessionWithinLimit error = %v, want %v", err, Session.MaxMemo)
	}
	if got := session.GetItem("large"); got != nil {
		t.Fatal("oversized session was stored")
	}

	if err := session.StoreSessionWithinLimit("large", item, 4096); err != nil {
		t.Fatalf("session within limit was rejected: %v", err)
	}
	if usage := session.MemoryUsage(); usage <= 0 || usage > 4096 {
		t.Fatalf("unexpected memory usage: %d", usage)
	}
}

func TestMemorySessionUsageTracksReplacementAndRemoval(t *testing.T) {
	item := NewMemoItemInterFace().(*MemorySession)
	if err := item.Store("key", make([]byte, 128)); err != nil {
		t.Fatal(err)
	}
	firstUsage := item.MemoryUsage()
	if firstUsage <= 0 {
		t.Fatalf("memory usage = %d, want a positive value", firstUsage)
	}

	if err := item.Store("key", "small"); err != nil {
		t.Fatal(err)
	}
	if usage := item.MemoryUsage(); usage <= 0 || usage >= firstUsage {
		t.Fatalf("replacement usage = %d, want between 0 and %d", usage, firstUsage)
	}

	if _, err := item.RemoveStoreValue("key"); err != nil {
		t.Fatal(err)
	}
	if usage := item.MemoryUsage(); usage != 0 {
		t.Fatalf("memory usage after removal = %d, want 0", usage)
	}
}

func TestConcurrentStoresRespectMemoryLimit(t *testing.T) {
	const memoryLimit int64 = 4096
	session := NewServerSession()
	item := NewMemoItemInterFace()
	if err := item.Store("payload", make([]byte, 128)); err != nil {
		t.Fatal(err)
	}

	var waitGroup sync.WaitGroup
	for i := 0; i < 32; i++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			err := session.StoreSessionWithinLimit(strconv.Itoa(index), item, memoryLimit)
			if err != nil && !errors.Is(err, Session.MaxMemo) {
				t.Errorf("unexpected store error: %v", err)
			}
		}(i)
	}
	waitGroup.Wait()
	if usage := session.MemoryUsage(); usage > memoryLimit {
		t.Fatalf("memory usage = %d, exceeds limit %d", usage, memoryLimit)
	}
}
