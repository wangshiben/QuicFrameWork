package server

import (
	"context"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// QUICProtocol handles all streams and datagrams of a connection negotiated
// with ALPN. Implementations must support concurrent calls to ServeQUICConn.
type QUICProtocol interface {
	ALPN() string

	// ServeQUICConn owns the connection until it returns. The endpoint then
	// closes the connection. ctx is canceled on peer disconnect or shutdown;
	// handlers should stop accepting work and wait for their workers to finish.
	// A returned *quic.ApplicationError supplies the fallback close code.
	ServeQUICConn(ctx context.Context, conn *quic.Conn) error

	// Shutdown starts protocol-specific draining (for example, HTTP/3 GOAWAY).
	// It runs concurrently with connection handlers. ctx is canceled when the
	// endpoint is forcibly closed. Implementations must honor cancellation.
	Shutdown(ctx context.Context) error
}

type QUICProtocolHandlerFunc func(context.Context, *quic.Conn) error

type quicProtocolFunc struct {
	alpn    string
	handler QUICProtocolHandlerFunc
}

// QUICProtocolFunc adapts a connection handler to QUICProtocol. A nil handler
// returns nil and is rejected at registration. Graceful draining is driven by
// the handler context; no additional shutdown hook is needed.
func QUICProtocolFunc(alpn string, handler QUICProtocolHandlerFunc) QUICProtocol {
	if handler == nil {
		return nil
	}
	return &quicProtocolFunc{alpn: alpn, handler: handler}
}

func (p *quicProtocolFunc) ALPN() string { return p.alpn }

func (p *quicProtocolFunc) ServeQUICConn(ctx context.Context, conn *quic.Conn) error {
	return p.handler(ctx, conn)
}

func (*quicProtocolFunc) Shutdown(context.Context) error { return nil }

type http3Protocol struct{ server *http3.Server }

// NewHTTP3Protocol adapts an HTTP/3 server to the shared QUIC endpoint. The
// endpoint configures TLS and QUIC; configure datagram support on both the
// endpoint and HTTP/3 server if needed. Do not also serve this server elsewhere.
func NewHTTP3Protocol(s *http3.Server) QUICProtocol {
	if s == nil {
		return nil
	}
	return &http3Protocol{server: s}
}

func (*http3Protocol) ALPN() string { return http3.NextProtoH3 }

func (p *http3Protocol) ServeQUICConn(_ context.Context, conn *quic.Conn) error {
	return p.server.ServeQUICConn(conn)
}

func (p *http3Protocol) Shutdown(ctx context.Context) error {
	return p.server.Shutdown(ctx)
}
