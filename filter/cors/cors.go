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
		request := r.GetRequest()
		origin := request.Header.Get("Origin")
		requestedMethod := request.Header.Get("Access-Control-Request-Method")
		isPreflight := request.Method == http.MethodOptions && requestedMethod != ""

		if origin == "" {
			next.Next(w, r)
			return
		}

		allowedOrigin, allowed := matchOrigin(config, origin)
		if !allowed {
			if isPreflight {
				http.Error(w, "CORS origin denied", http.StatusForbidden)
				return
			}
			next.Next(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		if allowedOrigin != "*" {
			addVary(w.Header(), "Origin")
		}
		if config.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if len(config.ExposeHeaders) != 0 {
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ","))
		}
		if isPreflight {
			if !containsFold(config.AllowMethods, requestedMethod) {
				http.Error(w, "CORS method denied", http.StatusForbidden)
				return
			}
			requestedHeaders := splitHeaderValues(request.Header.Get("Access-Control-Request-Headers"))
			if !allHeadersAllowed(config.AllowHeaders, requestedHeaders) {
				http.Error(w, "CORS headers denied", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ","))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ","))
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
			addVary(w.Header(), "Access-Control-Request-Method")
			addVary(w.Header(), "Access-Control-Request-Headers")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.Next(w, r)
	}
}

func matchOrigin(config CORSConfig, origin string) (string, bool) {
	for _, allowedOrigin := range config.AllowOrigins {
		allowedOrigin = strings.TrimSpace(allowedOrigin)
		if allowedOrigin == "*" {
			if config.AllowCredentials {
				return origin, true
			}
			return "*", true
		}
		if allowedOrigin == origin {
			return origin, true
		}
	}
	return "", false
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "*" || strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func splitHeaderValues(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func allHeadersAllowed(allowed, requested []string) bool {
	for _, requestedHeader := range requested {
		if !containsFold(allowed, requestedHeader) {
			return false
		}
	}
	return true
}

func addVary(header http.Header, value string) {
	for _, existing := range splitHeaderValues(strings.Join(header.Values("Vary"), ",")) {
		if strings.EqualFold(existing, value) {
			return
		}
	}
	header.Add("Vary", value)
}
