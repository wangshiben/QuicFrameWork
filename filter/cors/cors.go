package cors

import (
	"github.com/wangshiben/QuicFrameWork/RouteDisPatch"
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig 用于存储CORS相关的配置
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	ExposeHeaders    []string
	MaxAge           int
}

// DefaultCORSConfig 提供默认的CORS配置
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS 中间件函数
func CORS(config CORSConfig) RouteDisPatch.HttpFilter {
	return func(w http.ResponseWriter, r *RouteDisPatch.Request, next RouteDisPatch.Next) {
		// 设置CORS响应头
		w.Header().Set("Access-Control-Allow-Origin", strings.Join(config.AllowOrigins, ","))
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ","))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ","))
		if config.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if len(config.ExposeHeaders) != 0 {
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ","))
		}
		w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))

		// 如果是预检请求，直接返回204
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 处理其他请求
		next.Next(w, r)
	}
}
