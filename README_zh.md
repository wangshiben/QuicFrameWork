# QuickFrameWork

兼容Http1-1和Http2的框架，支持Http1-1,Http2,Http3的请求

默认可自签名证书(ESDA)

快速上手: [参考文档](https://quicframeworkdoc.github.io/)

## 优点:
1. 对于高并发场景下处理更快
> 测试内容请移步测试文档:
>
> [测试文档](test.md)

2. 根据request结构体自动注入内容,支持自定义request位置和默认值以及参数重命名

## 目前支持:

1. 路径支持正则匹配以及 * 匹配和 ** 匹配
2. 请求报错捕捉JSON输出
3. 支持自定义签名证书
4. 根据request结构体自动注入内容,支持自定义request位置和默认值以及参数重命名
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
    newServer.AddHttpHandler("/bck/**", http.MethodGet, func (w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "欢迎访问http3页面")
    fmt.Println(r.Proto)
    })
    newServer.AddHttpHandler("/bck/**", http.MethodPost, func (w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "欢迎访问http3 POST页面")
    fmt.Println(r.Proto)
    })
    newServer.StartServer()
}
```

3. 使用参考

main.go中内容

## TODO

> v0.1.0 TODO

1. [x] 拦截器注册
2. [ ] 优化路径匹配
3. [x] 鉴权设计以及Session管理
4. [x] 优化正则匹配
> v0.2.0 TODO
1. [ ] 支持SSE协议
2. [ ] 支持WebSocket协议
3. [ ] 支持StreamHttp协议
