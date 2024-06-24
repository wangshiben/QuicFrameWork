package RouteHand

import (
	"github.com/wangshiben/QuicFrameWork/server"
	"github.com/wangshiben/QuicFrameWork/server/RouteDisPatch"
	"net/http"
)

type QuickFrameWork[T interface{}] struct {
	Writer  http.ResponseWriter
	Request *http.Request
	Param   *T
}

type Request[T any] func(work *QuickFrameWork[T])

func GetAutowired[T interface{}](server *server.Server, path string, request Request[T]) {
	server.Route.AddHeaderParamHandler(path, http.MethodGet, new(T), func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		Param := r.Param.(*T)
		q := QuickFrameWork[T]{
			Writer:  w,
			Request: r.Request,
			Param:   Param,
		}
		request(&q)
	})
}
func PostAutowired[T interface{}](server *server.Server, path string, request Request[T]) {
	server.Route.AddHeaderParamHandler(path, http.MethodPost, new(T), func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		Param := r.Param.(*T)
		q := QuickFrameWork[T]{
			Writer:  w,
			Request: r.Request,
			Param:   Param,
		}
		request(&q)
	})
}

func PutAutowired[T interface{}](server *server.Server, path string, request Request[T]) {
	server.Route.AddHeaderParamHandler(path, http.MethodPut, new(T), func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		Param := r.Param.(*T)
		q := QuickFrameWork[T]{
			Writer:  w,
			Request: r.Request,
			Param:   Param,
		}
		request(&q)
	})
}
func DeleteAutowired[T interface{}](server *server.Server, path string, request Request[T]) {
	server.Route.AddHeaderParamHandler(path, http.MethodDelete, new(T), func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		Param := r.Param.(*T)
		q := QuickFrameWork[T]{
			Writer:  w,
			Request: r.Request,
			Param:   Param,
		}
		request(&q)
	})
}

func PatchAutowired[T interface{}](server *server.Server, path string, request Request[T]) {
	server.Route.AddHeaderParamHandler(path, http.MethodPatch, new(T), func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		Param := r.Param.(*T)
		q := QuickFrameWork[T]{
			Writer:  w,
			Request: r.Request,
			Param:   Param,
		}
		request(&q)
	})
}
