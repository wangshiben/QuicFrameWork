package Connections

import (
	"errors"
	"net"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestSSEConnectionCloseIsConcurrentAndNonBlocking(t *testing.T) {
	connection, closed, err := NewSSEConnection(httptest.NewRecorder(), nil)
	if err != nil {
		t.Fatal(err)
	}

	var waitGroup sync.WaitGroup
	for i := 0; i < 16; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if err := connection.Close(); err != nil {
				t.Errorf("Close returned an error: %v", err)
			}
		}()
	}
	waitGroup.Wait()

	if message := <-closed; message != Close {
		t.Fatalf("close message = %q, want %q", message, Close)
	}
	select {
	case message := <-closed:
		t.Fatalf("received duplicate close message %q", message)
	default:
	}

	if _, err := connection.Write([]byte("data: after close\n\n")); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Write after Close error = %v, want %v", err, net.ErrClosed)
	}
}

func TestSSEConnectionWriteReturnsOriginalLength(t *testing.T) {
	recorder := httptest.NewRecorder()
	connection, _, err := NewSSEConnection(recorder, nil)
	if err != nil {
		t.Fatal(err)
	}

	message := []byte("hello")
	written, err := connection.Write(message)
	if err != nil {
		t.Fatal(err)
	}
	if written != len(message) {
		t.Fatalf("Write returned %d, want %d", written, len(message))
	}
	if got, want := recorder.Body.String(), "data: hello\n\n"; got != want {
		t.Fatalf("response body = %q, want %q", got, want)
	}
}
