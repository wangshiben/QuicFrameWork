package RouteDisPatch

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"reflect"
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
func newReqParam(param interface{}) interface{} {
	// 获取输入接口的反射值
	val := reflect.ValueOf(param)

	// 检查是否为非空指针且指向一个结构体
	if val.Kind() == reflect.Ptr && !val.IsNil() && val.Elem().Kind() == reflect.Struct {
		// 获取指针指向的结构体的实际类型
		elemType := val.Elem().Type()

		// 创建目标类型的实例
		result := reflect.New(elemType).Elem()
		return result.Addr().Interface()
	}
	panic("you have send an invalid value")
	return nil
}
func (h *ServerHandler) httpHandler(w http.ResponseWriter, r *Request, route HttpHandle, FilterChain []HttpFilter) {
	next := &Next{
		chain:  FilterChain,
		handle: route,
		index:  0,
	}
	if len(FilterChain) != 0 {
		FilterChain[0](w, r, *next)
	} else {
		route(w, r)
	}
}

type Next struct {
	chain  []HttpFilter
	handle HttpHandle
	index  int
}

func (n *Next) Next(w http.ResponseWriter, r *Request) {
	n.index += 1
	if n.index >= len(n.chain) {
		n.handle(w, r)
	} else {
		n.chain[n.index](w, r, *n)
	}
}

func (h *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, filterChain := h.Routes.GetHttpHandler(r.URL.Path, r.Method)
	request := &Request{Request: r}

	if route.RequestParam == nil {
		h.httpHandler(w, request, route.Handler, filterChain)
	} else {
		all, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		data := newReqParam(route.RequestParam)
		err = json.Unmarshal(all, data)
		//if err != nil {
		//	return
		//}
		param := reflectBackToStructAsInterface(data, r, route.DefaultParamPosition, route.OriginPath)
		request.Param = param
		h.httpHandler(w, request, route.Handler, filterChain)

	}
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
