# Shared QUIC endpoint and ALPN protocols

TCP serves HTTP/1.1 and HTTP/2. A single UDP socket and `quic.Transport` accept
QUIC connections for both HTTP/3 and developer-defined protocols. TLS negotiates
one ALPN per connection; the endpoint dispatches the entire connection to that
protocol's handler. Stream framing, messages, authentication and sessions belong
to each protocol implementation.

```text
TCP -> http.Server -> HTTP/1.1, HTTP/2
UDP -> quic.Transport -> quic.Listener -> ALPN registry
                                        |- h3 -> http3.Server.ServeQUICConn
                                        |- echo/1 -> developer handler
                                        `- any other registered ALPN -> developer handler
```

## Registering a protocol

Existing constructors, startup methods, HTTP handlers, filters, SSE handlers and
Session interfaces retain their signatures. Existing applications can continue
using `NewServer(...)` and `StartServer()`. HTTP/3 is registered automatically as
`h3`; adding native protocols is opt-in:

```go
s := server.NewServer("cert.pem", "cert.key", ":4445")
err := s.RegisterQUICProtocol(server.QUICProtocolFunc("chat/1",
    func(ctx context.Context, conn *quic.Conn) error {
        // AcceptStream(ctx), AcceptUniStream(ctx), ReceiveDatagram(ctx), etc.
        // Own the connection until all application workers have finished.
        return serveChat(ctx, conn)
    }))
if err != nil {
    log.Fatal(err)
}
s.StartServer()
```

For protocol-specific draining, implement the `server.QUICProtocol` interface:

```go
type QUICProtocol interface {
    ALPN() string
    ServeQUICConn(context.Context, *quic.Conn) error
    Shutdown(context.Context) error
}
```

- ALPN names contain 1–255 **bytes**. Empty names, duplicate names, nil handlers
  and registrations after startup are rejected. `h3` cannot be overwritten.
- Registration order defines server preference when clients offer multiple
  ALPNs. Unsupported ALPNs fail during the TLS handshake.
- One QUIC connection uses one application protocol. HTTP/3 frames and custom
  protocol frames cannot share that connection without a protocol defining it.
- The endpoint clones TLS configuration and sets ALPNs from the registry,
  including configurations returned by `GetConfigForClient`. TCP ALPNs remain
  separate. Configure `s.QUIC.TLSConfig` before serving to customize QUIC TLS.
- Configure `s.QUIC.QUICConfig` before serving for incoming stream limits, flow
  control, idle timeouts and `EnableDatagrams`. Settings apply to the whole
  endpoint. Native datagrams are opt-in; enabling QUIC datagrams alone does not
  enable HTTP/3 datagram framing in the HTTP/3 adapter.
- Acceptance waits for the TLS handshake; this endpoint does not accept 0-RTT
  application data. Custom protocols do not pass through HTTP filters, cookie
  Session resolution, HTTP status codes or the HTTP response writer.

## Lifetime and ownership

`ServeQUICConn` is called concurrently for different connections. Its context is
canceled when the peer disconnects or shutdown starts. Stop accepting new work
on cancellation, drain active work, and join protocol-owned goroutines before
returning. The endpoint closes the connection when the handler returns, so
returning immediately after starting asynchronous work will abort that work.
Stream deadlines, stream reset handling and payload limits belong to the handler.

The endpoint catches panics in the connection handler and shutdown hook. A
handler's own child goroutines must handle their own panics. The fallback close
code is 0 for normal completion/shutdown and 1 for a handler error/panic; return
`*quic.ApplicationError` to supply the protocol's own code and message, or close
the connection explicitly. The endpoint does not invent an application wire
format or reserve `quick/1`.

`Shutdown(ctx)` stops acceptance, cancels native handler contexts, calls every
protocol's shutdown hook concurrently, and waits for handlers. The HTTP/3 hook
sends GOAWAY through the existing `http3.Server`. The socket stays alive while
protocols drain. If `ctx` expires, tracked connections are forcibly closed and
Shutdown returns the context error. Hooks must honor their context, which is
canceled on forced closure. `Close()` closes connections immediately without
waiting for application handlers that ignore cancellation. Shutdown/Close are
safe to repeat. An endpoint cannot be restarted.

New `Server.ListenAndServe(ctx)` binds TCP and UDP to the same numeric port,
including when using `:0`. `s.QUIC.Addr()` reports the actual UDP address, which
also supplies HTTP `Alt-Svc`. Its context requests immediate closure; call
`s.Shutdown` with a separate timeout for graceful draining. The existing
`StartServer()` and `StartHttpSerer()` retain automatic interrupt/SIGTERM
handling. The new context-based APIs leave process signal handling to the caller.

For a QUIC-only service, use `NewQUICEndpoint(tlsConfig, quicConfig)`, register
protocols with `RegisterProtocol`, then call `ListenAndServe(ctx, addr)`. To add
HTTP/3 explicitly, register `NewHTTP3Protocol(http3Server)` with this endpoint.
The adapter's TLS/listener settings do not configure the endpoint.

`QUICEndpoint.Serve(ctx, packetConn)` and the existing `Server.ServePacket`
accept caller-owned `net.PacketConn` implementations, including wrappers. The
caller must keep the socket open until Shutdown finishes and then close it.
`ListenAndServe` owns and closes its socket. Only one QUIC transport may read a
socket; do not also pass it to `http3.Server.Serve` or another transport.

## Run the demonstration

The example includes a server and a verifying Go client. Create a localhost
certificate with Subject Alternative Names (the existing legacy certificate
generator is separate from this example):

```sh
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
  -keyout cert.key -out cert.pem -days 1 -subj '/CN=localhost' \
  -addext 'subjectAltName=DNS:localhost,IP:127.0.0.1'

go run ./examples/quic-alpn -mode server -addr localhost:4445
```

In another terminal, connect to the **same UDP port** with each ALPN:

```sh
go run ./examples/quic-alpn -mode client -addr localhost:4445 -alpn h3
go run ./examples/quic-alpn -mode client -addr localhost:4445 -alpn echo/1
go run ./examples/quic-alpn -mode client -addr localhost:4445 -alpn upper/1
```

Expected output:

```text
ALPN=h3 HTTP=HTTP/3.0 response="hello over HTTP/3.0\n"
ALPN=echo/1 response="hello QUIC"
ALPN=upper/1 response="HELLO QUIC"
```

The same TCP port serves `https://localhost:4445/hello`. The example's custom
protocols use a single request per bidirectional stream, FIN as the request
boundary, a 1 MiB body limit and a 10-second stream deadline. These are example
application choices; the framework makes none of them mandatory.

## Verification

`go test ./...`, `go test -race ./...` and `go vet ./...` cover the existing API
and new implementation. The new tests use trusted ephemeral certificates and
real loopback UDP/TCP sockets, not mocked QUIC connections:

- HTTP/1.1, HTTP/2 and HTTP/3, Session-cookie reuse and correct Alt-Svc.
- HTTP/3 plus two custom ALPNs on one socket; SNI and protocol priority.
- Rejected registration and unsupported ALPN negotiation.
- Concurrent bidirectional streams, reset isolation and stream limits.
- Unidirectional streams and opt-in QUIC datagrams.
- Native and HTTP/3 draining, forced deadline closure and context cancellation.
- Handler panic/application errors, repeated Close, caller socket ownership.
- Legacy StartServer/ServePacket use and cleanup after startup failure.
