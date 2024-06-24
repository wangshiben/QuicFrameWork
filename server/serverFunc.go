package server

import "github.com/wangshiben/QuicFrameWork/server/RouteDisPatch"

func (s *Server) AddHttpHandler(path, HttpMethod string, handler RouteDisPatch.HttpHandle) {
	s.Route.AddHttpHandler(path, HttpMethod, handler)
}

func (s *Server) AddFilter(path string, filter RouteDisPatch.HttpFilter) {
	s.Route.AddFilter(path, filter)
}
