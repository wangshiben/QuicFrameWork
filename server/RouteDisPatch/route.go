package RouteDisPatch

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type Route struct {
	method    string     //请求方法
	path      string     //当前的路径
	Handler   HttpHandle //当前路径的处理函数
	Filter    HttpFilter //当前路径的过滤函数
	NextRoute []*Route   //下一个路径
	Index     map[string]int
}
type HttpFilter func(w http.ResponseWriter, r *http.Request, next HttpHandle) bool
type HttpHandle func(w http.ResponseWriter, r *http.Request)

func pageNotFund() HttpHandle {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain") // 设置合适的Content-Type
		w.WriteHeader(http.StatusNotFound)           // 先设置状态码
		if _, err := w.Write([]byte("404 page not found")); err != nil {
			// 这里处理写入错误，但注意，错误处理逻辑应考虑是否真的需要，因为写入w很少出错
			log.Printf("Error writing response: %v", err)
		}
		//fmt.Println("Page Not Found") // 日志或调试信息，不应影响HTTP响应
	}
}

func pageError() HttpHandle {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")  // 设置合适的Content-Type
		w.WriteHeader(http.StatusInternalServerError) // 先设置状态码
		w.Write([]byte("500 page error"))

	}
}
func (r *Route) AddHttpHandler(path, HttpMethod string, handler HttpHandle) {
	if path[0] == '/' {
		runes := []rune(path)
		path = string(runes[1:])
	}
	r.addHandler(path, HttpMethod, handler)
}
func (r *Route) GetHttpHandler(path, HttpMethod string) HttpHandle {
	if path[0] == '/' {
		runes := []rune(path)
		path = string(runes[1:])
	}
	return r.GetHandler(path, HttpMethod)
}

//func (r *Route) AddHttpFilter(path string, Filter HttpFilter) {
//
//}
//func (r *Route) addHttpFilter(path string, filter HttpFilter) {
//	routes := strings.SplitN(path, "/", 2)
//
//}

func (r *Route) GetHandler(path, HttpMethod string) HttpHandle {
	index, exist := 0, false
	routes := strings.SplitN(path, "/", 2)
	if len(routes) == 1 {
		index, exist = r.Index[r.getMapKey(routes[0], HttpMethod)]
	} else {
		index, exist = r.Index[routes[0]]
	}
	if len(routes) == 1 { //最终子路由
		if exist {
			return r.NextRoute[index].Handler
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
			return route.Handler
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

func (r *Route) addHandler(path, HttpMethod string, handler HttpHandle) {
	//路径:  /a/b/c/d
	routes := strings.SplitN(path, "/", 2)
	if len(routes) == 1 { //最终的子路由
		r.NextRoute = append(r.NextRoute, &Route{path: routes[0], Handler: handler, method: HttpMethod})
		r.Index[r.getMapKey(routes[0], HttpMethod)] = len(r.NextRoute) - 1
	} else {
		nextIndex, exist := r.Index[routes[0]]
		if exist {
			r.NextRoute[nextIndex].addHandler(routes[1], HttpMethod, handler)
		} else {
			r.NextRoute = append(r.NextRoute, &Route{path: routes[0], NextRoute: make([]*Route, 0), Index: make(map[string]int)})
			r.Index[routes[0]] = len(r.NextRoute) - 1
			r.NextRoute[len(r.NextRoute)-1].addHandler(routes[1], HttpMethod, handler)
		}
	}
}
func InitRoute() *Route {
	r := &Route{path: "/", NextRoute: make([]*Route, 0), Index: make(map[string]int)}
	return r
}
