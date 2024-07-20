package RouteDisPatch

import (
	"log"
	"net/http"
)

func pageNotFund() *Route {
	notfound := Route{
		Handler: func(w http.ResponseWriter, r *Request) {
			w.Header().Set("Content-Type", "text/plain") // 设置合适的Content-Type
			w.WriteHeader(http.StatusNotFound)           // 先设置状态码
			if _, err := w.Write([]byte("404 page not found")); err != nil {
				log.Printf("Error writing response: %v", err)
			}
			//fmt.Println("Page Not Found") // 日志或调试信息，不应影响HTTP响应
		},
		Status: http.StatusNotFound,
	}
	return &notfound
}
