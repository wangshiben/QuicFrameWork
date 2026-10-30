package server

import (
	"context"
	"errors"
	"github.com/wangshiben/QuicFrameWork/RouteDisPatch"
	"github.com/wangshiben/QuicFrameWork/Session"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPServerProvidesSessionContext(t *testing.T) {
	s := NewHttpServer(":0")
	s.AddHttpHandler("/session", http.MethodGet, func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		if _, err := r.GetSession(); err != nil {
			t.Errorf("GetSession returned an error: %v", err)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})

	recorder := httptest.NewRecorder()
	s.Server.Handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/session", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "ok" {
		t.Fatalf("unexpected response: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	if cookies := recorder.Result().Cookies(); len(cookies) != 1 || cookies[0].Name != "quickSession" {
		t.Fatalf("session cookie was not set: %#v", cookies)
	}
}

func TestHTTPServerPreservesRequestCancellation(t *testing.T) {
	s := NewHttpServer(":0")
	s.AddHttpHandler("/cancel", http.MethodGet, func(_ http.ResponseWriter, r *RouteDisPatch.Request) {
		if err := r.GetRequest().Context().Err(); err != context.Canceled {
			t.Errorf("context error = %v, want context.Canceled", err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/cancel", nil).WithContext(ctx)
	s.Server.Handler.ServeHTTP(httptest.NewRecorder(), req)
}

func TestHTTPServerCloseIsSafeAndIdempotent(t *testing.T) {
	s := NewHttpServer(":0")
	s.ScheduledTask()

	if err := s.Close(); err != nil {
		t.Fatalf("first Close returned an error: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second Close returned an error: %v", err)
	}

	select {
	case <-s.stopCh:
	default:
		t.Fatal("server stop channel was not closed")
	}
}

func TestHTTPServerUsesDefensiveTimeouts(t *testing.T) {
	s := NewHttpServer(":0")
	if s.Server.ReadHeaderTimeout != DefaultReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %v, want %v", s.Server.ReadHeaderTimeout, DefaultReadHeaderTimeout)
	}
	if s.Server.ReadTimeout != DefaultReadTimeout {
		t.Fatalf("ReadTimeout = %v, want %v", s.Server.ReadTimeout, DefaultReadTimeout)
	}
	if s.Server.IdleTimeout != DefaultIdleTimeout {
		t.Fatalf("IdleTimeout = %v, want %v", s.Server.IdleTimeout, DefaultIdleTimeout)
	}
}

func TestHTTPServerEnforcesSessionMemoryLimit(t *testing.T) {
	s := NewHttpServer(":0")
	s.SetMaxSessionMemoryBytes(1)
	var sessionError error
	s.AddHttpHandler("/session-limit", http.MethodGet, func(_ http.ResponseWriter, request *RouteDisPatch.Request) {
		_, sessionError = request.GetSession()
	})

	recorder := httptest.NewRecorder()
	s.Server.Handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/session-limit", nil))
	if !errors.Is(sessionError, Session.MaxMemo) {
		t.Fatalf("GetSession error = %v, want %v", sessionError, Session.MaxMemo)
	}
	if cookies := recorder.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("session cookie was set after rejection: %#v", cookies)
	}
}
