package server

import (
	"github.com/wangshiben/QuicFrameWork/server/RouteDisPatch"
	"github.com/wangshiben/QuicFrameWork/server/filter/cors"
)

func (s *Server) AddHttpHandler(path, HttpMethod string, handler RouteDisPatch.HttpHandle) {
	s.Route.AddHttpHandler(path, HttpMethod, handler)
}

func (s *Server) AddFilter(path string, filter RouteDisPatch.HttpFilter) {
	s.Route.AddFilter(path, filter)
}

func (s *Server) CORS(path string, cconf ...cors.CORSConfig) {
	for _, v := range cconf {
		s.AddFilter(path, cors.CORS(v))
	}

	if len(cconf) == 0 {
		s.AddFilter(path, cors.CORS(cors.DefaultCORSConfig()))
	}
}

func (s *Server) AddBodyParamHandler(path, HttpMethod string, param interface{}, handler RouteDisPatch.HttpHandle) {
	s.Route.AddBodyParamHandler(path, HttpMethod, param, handler)
}

func (s *Server) AddHeaderParamHandler(path, HttpMethod string, param interface{}, handler RouteDisPatch.HttpHandle) {
	s.Route.AddHeaderParamHandler(path, HttpMethod, param, handler)
}
