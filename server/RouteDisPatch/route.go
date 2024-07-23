package RouteDisPatch

import (
	"fmt"
	"net/http"
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
	Status               int    //状态码,用于匹配错误路径
	OriginPath           string //原始路径,用于参数注入
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

func pageError() HttpHandle {
	return func(w http.ResponseWriter, r *Request) {
		w.Header().Set("Content-Type", "text/plain")  // 设置合适的Content-Type
		w.WriteHeader(http.StatusInternalServerError) // 先设置状态码
		w.Write([]byte("500 page error"))
	}
}
func (r *Route) AddHttpHandler(path, HttpMethod string, handler HttpHandle) {
	path = formatPath(path)
	r.addHandler(path, HttpMethod, path, nil, "", handler)
}

func (r *Route) AddOriginHandler(path, HttpMethod string, paramPointer interface{}, defaultPosition string, handler HttpHandle) {
	path = formatPath(path)
	r.addHandler(path, HttpMethod, path, paramPointer, defaultPosition, handler)
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
	r.addHandler(path, HttpMethod, path, param, body, handler)
}
func (r *Route) AddHeaderParamHandler(path, HttpMethod string, param interface{}, handler HttpHandle) {
	path = formatPath(path)
	r.addHandler(path, HttpMethod, path, param, header, handler)
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
	if len(path) == 0 {
		handler := r.Handler
		if handler != nil {
			return r
		}
	}
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
	if len(routes) > 1 {
		for _, route := range r.NextRoute {
			if route.NextRoute == nil && route.method != HttpMethod {
				continue
			}
			switch route.path {
			case "*":
				return route.GetHandler(routes[1], HttpMethod)
			case "**":
				if route.Handler != nil {
					return route
				} else {
					continue
				}

			}
			//TODO:修改匹配模式
			//{name:2}->表示匹配从现在开始的往下两层路径,作为参数name的值
			//compile, err := regexp.Compile(route.path)
			//if err != nil {
			//	continue
			//}
			//match := compile.FindString(routes[1])
			//if len(match) != 0 {
			//	return route.GetHandler(routes[1], HttpMethod)
			//}
			_, forceStepCount, _ := getStrRegexpRes(route.path)
			if forceStepCount == 1 {
				handler := route.GetHandler(routes[1], HttpMethod)
				if handler.Status == http.StatusNotFound {
					continue
				} else {
					return handler
				}
			} else if forceStepCount > 0 {
				//TODO:匹配修改
				if len(routes) == 1 {
					continue
				}
				netRoutes := strings.SplitN(routes[1], "/", forceStepCount)
				if len(netRoutes)+1 == forceStepCount { //表明匹配到{xxx:num}结尾的Routes
					return route
				} else {
					handler := route.GetHandler(netRoutes[len(netRoutes)-1], HttpMethod)
					if handler.Status == http.StatusNotFound {
						continue
					} else {
						return handler
					}
				}

			} else {
				continue
			}
		}
	} else {
		for _, route := range r.NextRoute {
			if route.NextRoute == nil && route.method != HttpMethod {
				continue
			}
			switch route.path {
			case "*":
				if route.Handler != nil {
					return route
				} else {
					return pageNotFund()
				}
			case "**":
				if route.Handler != nil {
					return route
				} else {
					return pageNotFund()
				}
			}
			//TODO:修改匹配模式
			//{name:2}->表示匹配从现在开始的往下两层路径,作为参数name的值
			//compile, err := regexp.Compile(route.path)
			//if err != nil {
			//	continue
			//}
			//match := compile.FindString(routes[1])
			//if len(match) != 0 {
			//	return route.GetHandler(routes[1], HttpMethod)
			//}
			_, forceStepCount, _ := getStrRegexpRes(route.path)
			if forceStepCount == 1 {
				if route.Handler != nil {
					return route
				} else {
					continue
				}
			} else {
				continue
			}
		}
	}

	return pageNotFund()

}
func (r *Route) getMapKey(path, Method string) string {
	return fmt.Sprintf("%s?%s", path, Method)
}

func (r *Route) addHandler(path, HttpMethod, OriginPath string, paramPointer interface{}, defaultPosition string, handler HttpHandle) {
	//路径:  /a/b/c/d
	routes := strings.SplitN(path, "/", 2)
	if len(routes) == 1 { //最终的子路由
		r.NextRoute = append(r.NextRoute, &Route{
			path:                 routes[0],
			Handler:              handler,
			method:               HttpMethod,
			RequestParam:         paramPointer,
			DefaultParamPosition: defaultPosition,
			OriginPath:           OriginPath,
		})
		r.Index[r.getMapKey(routes[0], HttpMethod)] = len(r.NextRoute) - 1
	} else {
		nextIndex, exist := r.Index[routes[0]]
		if exist {
			r.NextRoute[nextIndex].addHandler(routes[1], HttpMethod, path, paramPointer, defaultPosition, handler)
		} else {
			r.NextRoute = append(r.NextRoute, &Route{path: routes[0], NextRoute: make([]*Route, 0), Index: make(map[string]int)})
			r.Index[routes[0]] = len(r.NextRoute) - 1
			r.NextRoute[len(r.NextRoute)-1].addHandler(routes[1], HttpMethod, path, paramPointer, defaultPosition, handler)
		}
	}
}
func InitRoute() *Route {
	r := &Route{path: "/", NextRoute: make([]*Route, 0), Index: make(map[string]int)}
	return r
}
