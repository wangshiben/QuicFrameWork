package RouteDisPatch

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type ParamLocation struct {
}

func (p *ParamLocation) Body() string {
	return "Body"
}
func (p *ParamLocation) RequestParam() string {
	return "Param"
}

type Route struct {
	method               string       //请求方法
	path                 string       //当前的路径
	Handler              HttpHandle   //当前路径的处理函数
	Filter               []HttpFilter //当前路径的过滤函数
	NextRoute            []*Route     //下一个路径
	RequestParam         interface{}  //接收参数类型
	DefaultParamPosition string       //默认接收参数位置
	Index                map[string]int
}

type GetHttpParam[T any] func(r *http.Request) T
type HttpFilter func(w http.ResponseWriter, r *Request, next Next)
type HttpHandle func(w http.ResponseWriter, r *Request)
type NextFunc func(w http.ResponseWriter, r *http.Request, next []HttpFilter, nextFunc NextFunc)

type Request struct {
	*http.Request
	Param interface{}
}

func NewRequest(r *http.Request) *Request {
	return &Request{
		Request: r,
	}
}

func pageNotFund() *Route {
	rou := Route{
		Handler: func(w http.ResponseWriter, r *Request) {
			w.Header().Set("Content-Type", "text/plain") // 设置合适的Content-Type
			w.WriteHeader(http.StatusNotFound)           // 先设置状态码
			if _, err := w.Write([]byte("404 page not found")); err != nil {
				log.Printf("Error writing response: %v", err)
			}
			//fmt.Println("Page Not Found") // 日志或调试信息，不应影响HTTP响应
		},
	}
	return &rou
}

func pageError() HttpHandle {
	return func(w http.ResponseWriter, r *Request) {
		w.Header().Set("Content-Type", "text/plain")  // 设置合适的Content-Type
		w.WriteHeader(http.StatusInternalServerError) // 先设置状态码
		w.Write([]byte("500 page error"))
	}
}
func (r *Route) AddHttpHandler(path, HttpMethod string, handler HttpHandle) {
	path = formatPath(path)
	r.addHandler(path, HttpMethod, nil, "", handler)
}

func (r *Route) AddOriginHandler(path, HttpMethod string, paramPointer interface{}, defaultPosition string, handler HttpHandle) {
	path = formatPath(path)
	r.addHandler(path, HttpMethod, paramPointer, defaultPosition, handler)
}
func formatPath(path string) string {
	if path[0] == '/' {
		runes := []rune(path)
		path = string(runes[1:])
	}
	return path
}
func (r *Route) AddBodyParamHandler(path, HttpMethod string, param interface{}, handler HttpHandle) {
	path = formatPath(path)
	r.addHandler(path, HttpMethod, param, body, handler)
}
func (r *Route) AddHeaderParamHandler(path, HttpMethod string, param interface{}, handler HttpHandle) {
	path = formatPath(path)
	r.addHandler(path, HttpMethod, param, header, handler)
}

func (r *Route) GetHttpHandler(path, HttpMethod string) (*Route, []HttpFilter) {
	if path[0] == '/' {
		runes := []rune(path)
		path = string(runes[1:])
	}
	FilterChain := make([]HttpFilter, 0)
	return r.GetHandler(path, HttpMethod), r.getFilter(path, FilterChain)
}

//func (r *Route) AddHttpFilter(path string, Filter HttpFilter) {
//
//}
//func (r *Route) addHttpFilter(path string, filter HttpFilter) {
//	routes := strings.SplitN(path, "/", 2)
//
//}

func (r *Route) GetHandler(path, HttpMethod string) *Route {
	index, exist := 0, false
	routes := strings.SplitN(path, "/", 2)
	if len(routes) == 1 {
		index, exist = r.Index[r.getMapKey(routes[0], HttpMethod)]
	} else {
		index, exist = r.Index[routes[0]]
	}
	if len(routes) == 1 { //最终子路由
		if exist {
			return r.NextRoute[index]
		}
	} else { //其他子路由
		if exist {
			return r.NextRoute[index].GetHandler(routes[1], HttpMethod)
		}
	}
	//进行正则匹配
	for _, route := range r.NextRoute {
		if route.method != HttpMethod {
			continue
		}
		switch route.path {
		case "*":
			return route.GetHandler(routes[1], HttpMethod)
		case "**":
			return route
		}
		compile, err := regexp.Compile(route.path)
		if err != nil {
			continue
		}
		match := compile.FindString(routes[1])
		if len(match) != 0 {
			return route.GetHandler(routes[1], HttpMethod)
		}
	}
	return pageNotFund()

}
func (r *Route) getMapKey(path, Method string) string {
	return fmt.Sprintf("%s?%s", path, Method)
}

func (r *Route) addHandler(path, HttpMethod string, paramPointer interface{}, defaultPosition string, handler HttpHandle) {
	//路径:  /a/b/c/d
	routes := strings.SplitN(path, "/", 2)
	if len(routes) == 1 { //最终的子路由
		r.NextRoute = append(r.NextRoute, &Route{
			path:                 routes[0],
			Handler:              handler,
			method:               HttpMethod,
			RequestParam:         paramPointer,
			DefaultParamPosition: defaultPosition,
		})
		r.Index[r.getMapKey(routes[0], HttpMethod)] = len(r.NextRoute) - 1
	} else {
		nextIndex, exist := r.Index[routes[0]]
		if exist {
			r.NextRoute[nextIndex].addHandler(routes[1], HttpMethod, paramPointer, defaultPosition, handler)
		} else {
			r.NextRoute = append(r.NextRoute, &Route{path: routes[0], NextRoute: make([]*Route, 0), Index: make(map[string]int)})
			r.Index[routes[0]] = len(r.NextRoute) - 1
			r.NextRoute[len(r.NextRoute)-1].addHandler(routes[1], HttpMethod, paramPointer, defaultPosition, handler)
		}
	}
}
func InitRoute() *Route {
	r := &Route{path: "/", NextRoute: make([]*Route, 0), Index: make(map[string]int)}
	return r
}
