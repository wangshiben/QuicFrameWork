package RouteHand

import (
	"github.com/wangshiben/QuicFrameWork/RouteDisPatch"
	"github.com/wangshiben/QuicFrameWork/server"
	"net/http"
)

type QuickFrameWork[T interface{}] struct {
	Writer  http.ResponseWriter
	Request *http.Request
	Param   T
}

type Request[T any] func(work *QuickFrameWork[T])

func PathAutowired[T interface{}](server *server.Server, path, method string, request Request[T]) {
	server.Route.AddBodyParamHandler(path, method, new(T), func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		Param := r.Param.(*T)
		q := QuickFrameWork[T]{
			Writer:  w,
			Request: r.GetRequest(),
			Param:   *Param,
		}
		request(&q)
	})
}

func GetAutowired[T interface{}](server *server.Server, path string, request Request[T]) {
	PathAutowired(server, path, http.MethodGet, request)
}
func PostAutowired[T interface{}](server *server.Server, path string, request Request[T]) {
	PathAutowired(server, path, http.MethodPost, request)
}

func PutAutowired[T interface{}](server *server.Server, path string, request Request[T]) {
	PathAutowired(server, path, http.MethodPut, request)
}
func DeleteAutowired[T interface{}](server *server.Server, path string, request Request[T]) {
	PathAutowired(server, path, http.MethodDelete, request)
}

func PatchAutowired[T interface{}](server *server.Server, path string, request Request[T]) {
	PathAutowired(server, path, http.MethodPatch, request)
}
