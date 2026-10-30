package RouteDisPatch

import (
	"github.com/wangshiben/QuicFrameWork/Connections"
	"net/http"
	"testing"
)

func TestRouteRegistrationRejectsEmptyPath(t *testing.T) {
	type params struct{}
	route := InitRoute()
	httpHandler := func(http.ResponseWriter, *Request) {}
	sseHandler := func(*Connections.SSEConnection) {}

	tests := []struct {
		name     string
		register func()
	}{
		{name: "HTTP handler", register: func() { route.AddHttpHandler("", http.MethodGet, httpHandler) }},
		{name: "origin handler", register: func() { route.AddOriginHandler("", http.MethodGet, &params{}, "", httpHandler) }},
		{name: "body handler", register: func() { route.AddBodyParamHandler("", http.MethodPost, &params{}, httpHandler) }},
		{name: "header handler", register: func() { route.AddHeaderParamHandler("", http.MethodGet, &params{}, httpHandler) }},
		{name: "SSE handler", register: func() { route.AddSSEHandler("", http.MethodGet, sseHandler) }},
		{name: "SSE handler with request", register: func() { route.AddSSEHandlerWithReq("", http.MethodGet, &params{}, sseHandler) }},
		{name: "filter", register: func() { route.AddFilter("", func(http.ResponseWriter, *Request, Next) {}) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertRoutePathPanic(t, test.register)
		})
	}
}

func TestRouteRegistrationRejectsWhitespacePath(t *testing.T) {
	assertRoutePathPanic(t, func() {
		InitRoute().AddHttpHandler("  \t  ", http.MethodGet, func(http.ResponseWriter, *Request) {})
	})
}

func TestRootRouteRemainsValid(t *testing.T) {
	route := InitRoute()
	route.AddHttpHandler("/", http.MethodGet, func(http.ResponseWriter, *Request) {})

	matched, _ := route.GetHttpHandler("/", http.MethodGet)
	if matched.Handler == nil || matched.Status == http.StatusNotFound {
		t.Fatal("root route was not registered")
	}
}

func assertRoutePathPanic(t *testing.T, register func()) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != ErrorInvalidRoutePath {
			t.Fatalf("panic = %v, want %q", recovered, ErrorInvalidRoutePath)
		}
	}()
	register()
}
