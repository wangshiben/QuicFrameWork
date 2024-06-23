package RouteHand

import (
	"github.com/wangshiben/QuicFrameWork/server/RouteDisPatch"
	"net/http"
)

type QuickFrameWork[T interface{}] struct {
	Writer  http.ResponseWriter
	Request *http.Request
	Param   *T
}

type Request[T any] func(work *QuickFrameWork[T])

func Get[T interface{}](ServerRoute *RouteDisPatch.Route, path string, request Request[T]) {
	ServerRoute.AddHeaderParamHandler(path, http.MethodGet, new(T), func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		Param := r.Param.(*T)
		q := QuickFrameWork[T]{
			Writer:  w,
			Request: r.Request,
			Param:   Param,
		}
		request(&q)
	})
}
