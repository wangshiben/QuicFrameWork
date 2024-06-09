package main

import (
	"fmt"
	"github.com/wangshiben/QuicFrameWork/server"
	"github.com/wangshiben/QuicFrameWork/server/RouteDisPatch"
	"net/http"
	"reflect"
)

type TestStruct struct {
	Name         string
	RequestParam string `quickLoc:"param"`
	Header       string `quickLoc:"header"`
	Age          int    `quickLoc:"param"`
}

func main() {
	//可信的证书
	newServer := server.NewServer("", "", ":4445")
	// 或: newServer := server.NewServer("", "", ":4445")使用自签名证书
	// /bck/
	newServer.Route.AddHttpHandler("/bck/**", http.MethodGet, func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		//param := r.Param
		fmt.Fprintf(w, "欢迎访问http3页面")
		fmt.Println(r.Proto)
	})
	newServer.Route.AddHttpHandler("/bck/**", http.MethodPost, func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		value := reflect.ValueOf(r.Param)
		value.Type()
		fmt.Fprintf(w, "欢迎访问http3 POST页面")
		fmt.Println(r.Proto)
	})
	newServer.Route.AddBodyParamHandler("/temp/**", http.MethodPost, &TestStruct{}, func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		testStruct := r.Param.(*TestStruct)
		fmt.Println(*testStruct)
	})
	newServer.StartServer()
}
