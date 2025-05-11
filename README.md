# QuickFrameWork

A framework compatible with Http1-1 and Http2, supporting Http1-1, Http2, and Http3 requests.

Default self-signed certificate (ESDA) support.

Quick Start: [Documentation](https://quicframeworkdoc.github.io/)

Language: English | [中文](README_zh.md)

## Advantages:
1. Faster processing in high-concurrency scenarios
> For test results, please refer to the test documentation:
>
> [Test Documentation](test.md)

2. Automatic content injection based on request structure, supporting custom request locations, default values, and parameter renaming

## Currently Supported:

1. Path supports regular expression matching, * matching, and ** matching
2. JSON output for request error handling
3. Support for custom signature certificates
4. Automatic content injection based on request structure, supporting custom request locations, default values, and parameter renaming

## Quick Start

1. Import:

```bash
go get github.com/wangshiben/QuicFrameWork
```

2. Usage

```go
func main() {
    // Trusted certificate
    newServer := server.NewServer("cert.pem", "cert.key", ":4445")
    // Or: newServer := server.NewServer("", "", ":4445") to use self-signed certificate
    newServer.AddHttpHandler("/bck/**", http.MethodGet, func (w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Welcome to http3 page")
        fmt.Println(r.Proto)
    })
    newServer.AddHttpHandler("/bck/**", http.MethodPost, func (w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Welcome to http3 POST page")
        fmt.Println(r.Proto)
    })
    newServer.StartServer()
}
```

3. Usage Reference

See main.go for examples

## TODO

> v0.1.0 TODO

1. [x] Interceptor registration
2. [ ] Path matching optimization
3. [x] Authentication design and Session management
4. [x] Regular expression matching optimization

> v0.2.0 TODO
1. [ ] SSE protocol support
2. [ ] WebSocket protocol support
3. [ ] StreamHttp protocol support 