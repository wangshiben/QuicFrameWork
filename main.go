package main

import (
	"fmt"
	"github.com/wangshiben/QuicFrameWork/server"
	"net/http"
)

func main() {
	//可信的证书
	newServer := server.NewServer("cert.pem", "cert.key", ":4445")
	// 或: newServer := server.NewServer("", "", ":4445")使用自签名证书
	newServer.Route.AddHttpHandler("/bck/**", http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "欢迎访问http3页面")
		fmt.Println(r.Proto)
	})
	newServer.Route.AddHttpHandler("/bck/**", http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "欢迎访问http3 POST页面")
		fmt.Println(r.Proto)
	})
	newServer.StartServer()
}
