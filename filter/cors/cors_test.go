package cors

import (
	"github.com/wangshiben/QuicFrameWork/RouteDisPatch"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 测试默认CORS配置
func TestDefaultCORS(t *testing.T) {
	// 创建默认的CORS配置
	corsConfig := DefaultCORSConfig()
	corsHandler := CORS(corsConfig)

	// 创建一个预检请求（OPTIONS）
	req, _ := http.NewRequest("OPTIONS", "/", nil)
	DisReq := RouteDisPatch.NewRequest(req, nil)
	DisReq.Request.Header.Set("Origin", "http://example.com")
	DisReq.Request.Header.Set("Access-Control-Request-Method", "GET")

	// 创建一个响应记录器
	rr := httptest.NewRecorder()
	dNext := RouteDisPatch.Next{}

	// 执行CORS过滤器
	corsHandler(rr, DisReq, dNext)

	// 检查状态码
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNoContent)
	}

	// 检查CORS响应头
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Access-Control-Allow-Origin header not set correctly: got %v want %v",
			rr.Header().Get("Access-Control-Allow-Origin"), "*")
	}

	if !strings.Contains(rr.Header().Get("Access-Control-Allow-Methods"), "OPTIONS") {
		t.Errorf("Access-Control-Allow-Methods header not set correctly: got %v",
			rr.Header().Get("Access-Control-Allow-Methods"))
	}

	if !strings.Contains(rr.Header().Get("Access-Control-Allow-Headers"), "Origin") {
		t.Errorf("Access-Control-Allow-Headers header not set correctly: got %v",
			rr.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestCORSWithActualRequest(t *testing.T) {
	handler := RouteDisPatch.InitHandler()
	handler.Routes.AddHttpHandler("/resource", http.MethodGet, func(w http.ResponseWriter, _ *RouteDisPatch.Request) {
		_, _ = w.Write([]byte("Hello, World!"))
	})
	handler.Routes.AddFilter("/resource", CORS(DefaultCORSConfig()))

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "http://example.com")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", origin, "*")
	}
	if body := recorder.Body.String(); body != "Hello, World!" {
		t.Fatalf("body = %q, want %q", body, "Hello, World!")
	}
}

func TestCustomCORSPreflight(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"https://app.example.com"},
		AllowMethods:     []string{http.MethodGet, http.MethodOptions},
		AllowHeaders:     []string{"Authorization"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Request-ID"},
		MaxAge:           600,
	}
	handler := RouteDisPatch.InitHandler()
	handler.Routes.AddHttpHandler("/resource", http.MethodGet, func(http.ResponseWriter, *RouteDisPatch.Request) {
		t.Fatal("preflight request reached the route handler")
	})
	handler.Routes.AddFilter("/resource", CORS(config))

	req := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	checks := map[string]string{
		"Access-Control-Allow-Origin":      "https://app.example.com",
		"Access-Control-Allow-Methods":     "GET,OPTIONS",
		"Access-Control-Allow-Headers":     "Authorization",
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Expose-Headers":    "X-Request-ID",
		"Access-Control-Max-Age":           "600",
	}
	for header, want := range checks {
		if got := recorder.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestCORSMultipleOriginsReturnsOnlyRequestOrigin(t *testing.T) {
	config := CORSConfig{
		AllowOrigins: []string{"https://first.example.com", "https://second.example.com"},
		AllowMethods: []string{http.MethodGet},
	}
	handler := RouteDisPatch.InitHandler()
	handler.Routes.AddHttpHandler("/resource", http.MethodGet, func(w http.ResponseWriter, _ *RouteDisPatch.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	handler.Routes.AddFilter("/resource", CORS(config))

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "https://second.example.com")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://second.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := strings.Join(recorder.Header().Values("Vary"), ","); !strings.Contains(got, "Origin") {
		t.Fatalf("Vary = %q, want Origin", got)
	}
}

func TestCORSCredentialsWithWildcardEchoesOrigin(t *testing.T) {
	config := DefaultCORSConfig()
	config.AllowCredentials = true
	handler := RouteDisPatch.InitHandler()
	handler.Routes.AddHttpHandler("/resource", http.MethodGet, func(w http.ResponseWriter, _ *RouteDisPatch.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	handler.Routes.AddFilter("/resource", CORS(config))

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "https://app.example.com")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q", got)
	}
}

func TestCORSRejectsInvalidPreflight(t *testing.T) {
	config := CORSConfig{
		AllowOrigins: []string{"https://app.example.com"},
		AllowMethods: []string{http.MethodGet},
		AllowHeaders: []string{"Authorization"},
	}
	tests := []struct {
		name    string
		origin  string
		method  string
		headers string
	}{
		{name: "origin", origin: "https://denied.example.com", method: http.MethodGet},
		{name: "method", origin: "https://app.example.com", method: http.MethodDelete},
		{name: "headers", origin: "https://app.example.com", method: http.MethodGet, headers: "X-Denied"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := RouteDisPatch.InitHandler()
			handler.Routes.AddHttpHandler("/resource", http.MethodGet, func(http.ResponseWriter, *RouteDisPatch.Request) {
				t.Fatal("rejected preflight reached handler")
			})
			handler.Routes.AddFilter("/resource", CORS(config))

			req := httptest.NewRequest(http.MethodOptions, "/resource", nil)
			req.Header.Set("Origin", test.origin)
			req.Header.Set("Access-Control-Request-Method", test.method)
			if test.headers != "" {
				req.Header.Set("Access-Control-Request-Headers", test.headers)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
			}
		})
	}
}
