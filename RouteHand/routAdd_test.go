package RouteHand

import (
	"github.com/wangshiben/QuicFrameWork/server"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostAutowiredInjectsBody(t *testing.T) {
	type params struct {
		Name string `json:"name"`
	}

	s := server.NewHttpServer(":0")
	var got params
	PostAutowired[params](s, "/users", func(work *QuickFrameWork[params]) {
		got = work.Param
	})
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"alice"}`))
	recorder := httptest.NewRecorder()
	s.Server.Handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got.Name != "alice" {
		t.Fatalf("Name = %q, want alice", got.Name)
	}
}
