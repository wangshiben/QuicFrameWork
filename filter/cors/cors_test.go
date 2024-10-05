package cors

import (
	"github.com/wangshiben/QuicFrameWork/server/RouteDisPatch"
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
	DisReq := RouteDisPatch.NewRequest(req)
	DisReq.Header.Set("Origin", "http://example.com")
	DisReq.Header.Set("Access-Control-Request-Method", "GET")

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

//// 测试带有实际请求的CORS配置
//func TestCORSWithActualRequest(t *testing.T) {
//	// 创建默认的CORS配置
//	corsConfig := DefaultCORSConfig()
//	corsHandler := CORS(corsConfig)
//
//	// 创建一个预检请求（OPTIONS）
//	req, _ := http.NewRequest("OPTIONS", "/", nil)
//	DisReq := RouteDisPatch.NewRequest(req)
//	DisReq.Header.Set("Origin", "http://example.com")
//	DisReq.Header.Set("Access-Control-Request-Method", "GET")
//
//	// 创建一个响应记录器
//	rr := httptest.NewRecorder()
//	// todo 无法创建手动添加了HttpHandle的Next结构体
//	dNext := RouteDisPatch.Next{}
//
//	// 执行CORS过滤器
//	corsHandler(rr, DisReq, dNext)
//
//	// 检查状态码
//	if status := rr.Code; status != http.StatusOK {
//		t.Errorf("handler returned wrong status code: got %v want %v",
//			status, http.StatusOK)
//	}
//
//	// 检查CORS响应头
//	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
//		t.Errorf("Access-Control-Allow-Origin header not set correctly: got %v want %v",
//			rr.Header().Get("Access-Control-Allow-Origin"), "*")
//	}
//
//	// 检查响应内容
//	if body := rr.Body.String(); body != "Hello, World!" {
//		t.Errorf("handler returned wrong body: got %v want %v",
//			body, "Hello, World!")
//	}
//}
