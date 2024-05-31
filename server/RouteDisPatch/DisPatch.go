package RouteDisPatch

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
)

type Logger interface {
	Log(msg any)
	Error(msg any)
	Debug(msg any)
	Warn(msg any)
}

type ServerHandler struct {
	Routes *Route
	Log    *Logger
}

func InitHandler() *ServerHandler {
	route := InitRoute()
	server := &ServerHandler{Routes: route}
	return server
}

func (h *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := h.Routes.GetHttpHandler(r.URL.Path, r.Method)
	handler(w, r)
	defer func() {
		errors := recover()
		if errors != nil {
			switch errors.(type) {
			case error:
				w.Header().Set("Content-Type", "text/plain")  // 设置合适的Content-Type
				w.WriteHeader(http.StatusInternalServerError) //将报错内容写入响应体
				marshal, _ := json.Marshal(errorStruct{
					Code: http.StatusInternalServerError,
					Msg:  errors.(error).Error(),
				})
				w.Write(marshal)

				stack := make([]byte, 1024)
				length := runtime.Stack(stack, false)
				log.Printf("Recovered from panic: %v\nStack Trace:\n%s", errors, stack[:length])
			}
		}
	}()

}

type errorStruct struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
