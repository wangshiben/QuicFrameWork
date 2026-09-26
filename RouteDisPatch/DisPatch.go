package RouteDisPatch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/wangshiben/QuicFrameWork/Writer"
	"io"
	"log"
	"net/http"
	"reflect"
	"runtime/debug"
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
	writer := Writer.NewWriter(w)
	defer func() {
		_, err := writer.FinishWrite()
		if err != nil {
			log.Printf("failed to finish response: %v", err)
		}
	}()
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		if recovered == http.ErrAbortHandler {
			panic(recovered)
		}

		writer.Reset()
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Del("Content-Length")
		writer.WriteHeader(http.StatusInternalServerError)
		message := fmt.Sprint(recovered)
		if err, ok := recovered.(error); ok {
			message = err.Error()
		}
		marshal, _ := json.Marshal(errorStruct{
			Code: http.StatusInternalServerError,
			Msg:  message,
		})
		_, _ = writer.Write(marshal)
		log.Printf("Recovered from panic: %v\nStack Trace:\n%s", recovered, debug.Stack())
	}()

	route, filterChain := h.Routes.GetHttpHandler(r.URL.Path, r.Method)
	request := NewRequest(r, writer)
	if route.RequestParam == nil {
		h.httpHandler(writer, request, route.Handler, filterChain)
	} else {
		all, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		data := newReqParam(route.RequestParam)
		if len(bytes.TrimSpace(all)) != 0 {
			err = json.Unmarshal(all, data)
			if err != nil {
				http.Error(writer, "invalid JSON body", http.StatusBadRequest)
				return
			}
		}
		param := reflectBackToStructAsInterface(data, r, route.DefaultParamPosition, route.OriginPath)
		request.Param = param
		h.httpHandler(writer, request, route.Handler, filterChain)
	}
}

type errorStruct struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
