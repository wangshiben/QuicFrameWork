# QuicFrameWork

兼容Http1-1和Http2的框架，支持Http1-1,Http2,Http3的请求

默认可自签名证书(ESDA)

## 目前支持:

1. 路径支持正则匹配以及 * 匹配和 ** 匹配
2. 请求报错捕捉JSON输出
3. 支持自定义签名证书

## 快速开始

1. 引入:

```bash
go get github.com/wangshiben/QuicFrameWork
```

2. 使用

```go
func main() {
//可信的证书      
newServer := server.NewServer("cert.pem", "cert.key", ":4445")
// 或: newServer := server.NewServer("", "", ":4445")使用自签名证书
newServer.Route.AddHttpHandler("/bck/**", http.MethodGet, func (w http.ResponseWriter, r *http.Request) {
fmt.Fprintf(w, "欢迎访问http3页面")
fmt.Println(r.Proto)
})
newServer.Route.AddHttpHandler("/bck/**", http.MethodPost, func (w http.ResponseWriter, r *http.Request) {
fmt.Fprintf(w, "欢迎访问http3 POST页面")
fmt.Println(r.Proto)
})
newServer.StartServer()
}
```

## TODO

1. 拦截器注册
2. 优化路径匹配
3. 鉴权设计以及Session管理