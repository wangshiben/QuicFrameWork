package server

import (
	"context"
	"github.com/wangshiben/QuicFrameWork/RouteDisPatch"
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
