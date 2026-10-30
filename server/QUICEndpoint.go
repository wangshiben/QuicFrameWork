package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"reflect"
	"runtime/debug"
	"slices"
	"sync"

	"github.com/quic-go/quic-go"
)

// QUICEndpoint owns one QUIC transport and dispatches accepted connections by
// negotiated ALPN. Configure it and register protocols before serving. An
// endpoint can be served only once, and cannot be restarted after shutdown.
// The zero value is usable after setting TLSConfig and registering protocols.
type QUICEndpoint struct {
	TLSConfig  *tls.Config
	QUICConfig *quic.Config
	Logger     *slog.Logger

	mu            sync.Mutex
	protocols     map[string]QUICProtocol
	alpns         []string
	started       bool
	closing       bool
	listener      *quic.Listener
	transport     *quic.Transport
	addr          net.Addr
	acceptCancel  context.CancelFunc
	connections   map[*quic.Conn]struct{}
	workers       sync.WaitGroup
	drainCtx      context.Context
	drainCancel   context.CancelFunc
	hardCtx       context.Context
	hardCancel    context.CancelFunc
	stopOnce      sync.Once
	transportOnce sync.Once
	done          chan struct{}
	shutdownErr   error
	transportErr  error
}

func NewQUICEndpoint(tlsConfig *tls.Config, quicConfig *quic.Config) *QUICEndpoint {
	e := &QUICEndpoint{}
	// net/http initializes HTTP/2 by modifying its TLS config during Serve.
	// Separate copies prevent that initialization from racing QUIC startup.
	if tlsConfig != nil {
		e.TLSConfig = tlsConfig.Clone()
	}
	if quicConfig != nil {
		e.QUICConfig = quicConfig.Clone()
	}
	e.initLocked()
	return e
}

func (e *QUICEndpoint) initLocked() {
	if e.protocols == nil {
		e.protocols = make(map[string]QUICProtocol)
		e.connections = make(map[*quic.Conn]struct{})
		e.drainCtx, e.drainCancel = context.WithCancel(context.Background())
		e.hardCtx, e.hardCancel = context.WithCancel(context.Background())
		e.done = make(chan struct{})
	}
}

// RegisterProtocol registers an ALPN in server preference order. Empty,
// oversized, duplicate and nil registrations are rejected. Registration is
// frozen when Serve starts. ALPN identifiers are byte strings, not HTTP headers.
func (e *QUICEndpoint) RegisterProtocol(protocol QUICProtocol) error {
	if protocol == nil || isNilProtocol(protocol) {
		return errors.New("QUIC protocol must not be nil")
	}
	alpn := protocol.ALPN()
	if len(alpn) == 0 || len(alpn) > 255 {
		return errors.New("QUIC ALPN must contain 1 to 255 bytes")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.initLocked()
	if e.started || e.closing {
		return errors.New("QUIC protocol registration is closed")
	}
	if _, exists := e.protocols[alpn]; exists {
		return fmt.Errorf("QUIC ALPN %q is already registered", alpn)
	}
	e.protocols[alpn] = protocol
	e.alpns = append(e.alpns, alpn)
	return nil
}

func isNilProtocol(p QUICProtocol) bool {
	v := reflect.ValueOf(p)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// ALPNs returns a copy of the registered protocols in preference order.
func (e *QUICEndpoint) ALPNs() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.alpns)
}

// Addr returns the actual bound UDP address, or nil before successful startup.
func (e *QUICEndpoint) Addr() net.Addr {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.addr
}

// ListenAndServe opens and owns a UDP socket. It waits for shutdown, including
// protocol draining, before closing that socket. Canceling ctx forces closure.
func (e *QUICEndpoint) ListenAndServe(ctx context.Context, addr string) error {
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	defer pc.Close()
	err = e.Serve(ctx, pc)
	// Serve stops accepting before graceful draining completes. Keep UDP open
	// so active connections can exchange their remaining protocol messages.
	if err == nil {
		err = e.Shutdown(ctx)
		if errors.Is(err, context.Canceled) && ctx.Err() == nil {
			err = nil // Explicit Close is a normal termination of the accept loop.
		}
	}
	return err
}

// Serve accepts QUIC connections on pc and blocks until acceptance stops.
// The caller retains ownership of pc and must keep it open until Shutdown
// finishes. It must not read/write pc or give it to another QUIC transport.
// Unknown ALPNs are rejected by TLS; cancellation of ctx forces closure.
func (e *QUICEndpoint) Serve(ctx context.Context, pc net.PacketConn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if pc == nil {
		return errors.New("QUIC packet connection must not be nil")
	}
	e.mu.Lock()
	e.initLocked()
	if e.closing {
		e.mu.Unlock()
		return http.ErrServerClosed
	}
	if e.started {
		e.mu.Unlock()
		return errors.New("QUIC endpoint has already been started")
	}
	if e.TLSConfig == nil || len(e.alpns) == 0 {
		e.mu.Unlock()
		return errors.New("QUIC endpoint requires TLS configuration and at least one protocol")
	}
	tlsConfig := endpointTLSConfig(e.TLSConfig, e.alpns)
	var config *quic.Config
	if e.QUICConfig != nil {
		config = e.QUICConfig.Clone()
	}
	e.started = true
	e.transport = &quic.Transport{Conn: pc}
	ln, err := e.transport.Listen(tlsConfig, config)
	if err != nil {
		e.mu.Unlock()
		_ = e.Close()
		return fmt.Errorf("listen for QUIC: %w", err)
	}
	e.listener = ln
	e.addr = ln.Addr()
	acceptCtx, cancel := context.WithCancel(ctx)
	e.acceptCancel = cancel
	e.mu.Unlock()
	defer cancel()
	stop := context.AfterFunc(ctx, func() { _ = e.Close() })
	defer stop()

	for {
		conn, err := ln.Accept(acceptCtx)
		if err != nil {
			if ctx.Err() != nil {
				_ = e.Close()
				return ctx.Err()
			}
			e.mu.Lock()
			closing := e.closing
			e.mu.Unlock()
			if closing {
				return nil
			}
			_ = e.Close()
			return fmt.Errorf("accept QUIC connection: %w", err)
		}
		e.mu.Lock()
		protocol := e.protocols[conn.ConnectionState().TLS.NegotiatedProtocol]
		if e.closing || protocol == nil {
			e.mu.Unlock()
			_ = conn.CloseWithError(0, "QUIC endpoint is closing")
			continue
		}
		e.connections[conn] = struct{}{}
		e.workers.Add(1)
		e.mu.Unlock()
		go e.serveConnection(protocol, conn)
	}
}

// endpointTLSConfig also constrains configurations returned by SNI callbacks,
// so a callback cannot accidentally remove registered ALPNs or advertise h2.
func endpointTLSConfig(base *tls.Config, alpns []string) *tls.Config {
	config := base.Clone()
	config.MinVersion = tls.VersionTLS13
	config.NextProtos = slices.Clone(alpns)
	if getConfig := base.GetConfigForClient; getConfig != nil {
		config.GetConfigForClient = func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			selected, err := getConfig(hello)
			if err != nil || selected == nil {
				return selected, err
			}
			return endpointTLSConfig(selected, alpns), nil
		}
	}
	return config
}

func (e *QUICEndpoint) serveConnection(protocol QUICProtocol, conn *quic.Conn) {
	ctx, cancel := context.WithCancel(conn.Context())
	stop := context.AfterFunc(e.drainCtx, cancel)
	var err error
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("QUIC protocol handler panic: %v", recovered)
			e.logger().Error("QUIC protocol handler panic", "alpn", conn.ConnectionState().TLS.NegotiatedProtocol, "panic", recovered, "stack", string(debug.Stack()))
		}
		code, message := quic.ApplicationErrorCode(0), "protocol handler completed"
		if err != nil && conn.Context().Err() == nil {
			code, message = 1, "protocol handler failed"
			var appErr *quic.ApplicationError
			if errors.As(err, &appErr) {
				code, message = appErr.ErrorCode, appErr.ErrorMessage
			}
			e.logger().Debug("QUIC protocol handler returned", "alpn", conn.ConnectionState().TLS.NegotiatedProtocol, "error", err)
		}
		_ = conn.CloseWithError(code, message)
		stop()
		cancel()
		e.mu.Lock()
		delete(e.connections, conn)
		e.mu.Unlock()
		e.workers.Done()
	}()
	err = protocol.ServeQUICConn(ctx, conn)
}

func (e *QUICEndpoint) logger() *slog.Logger {
	if e.Logger != nil {
		return e.Logger
	}
	return slog.Default()
}

func (e *QUICEndpoint) stopAccepting() {
	e.stopOnce.Do(func() {
		e.mu.Lock()
		e.initLocked()
		e.closing = true
		e.drainCancel()
		if e.acceptCancel != nil {
			e.acceptCancel()
		}
		ln := e.listener
		protocols := make([]QUICProtocol, 0, len(e.alpns))
		for _, alpn := range e.alpns {
			protocols = append(protocols, e.protocols[alpn])
		}
		e.mu.Unlock()
		if ln != nil {
			_ = ln.Close()
		}
		go e.drain(protocols)
	})
}

func (e *QUICEndpoint) drain(protocols []QUICProtocol) {
	results := make(chan error, len(protocols))
	for _, protocol := range protocols {
		go func() {
			var err error
			defer func() {
				if recovered := recover(); recovered != nil {
					err = fmt.Errorf("QUIC protocol shutdown panic: %v", recovered)
				}
				results <- err
			}()
			err = protocol.Shutdown(e.hardCtx)
		}()
	}
	var errs []error
	for range protocols {
		if err := <-results; err != nil {
			errs = append(errs, err)
		}
	}
	e.workers.Wait()
	if err := e.closeTransport(); err != nil {
		errs = append(errs, err)
	}
	e.mu.Lock()
	e.shutdownErr = errors.Join(errs...)
	close(e.done)
	e.mu.Unlock()
}

// Shutdown stops accepting, cancels handler contexts to request draining, and
// waits for protocol hooks and handlers. On deadline it forcibly closes all
// connections and returns ctx.Err(). Repeated and concurrent calls are safe.
func (e *QUICEndpoint) Shutdown(ctx context.Context) error {
	e.stopAccepting()
	select {
	case <-e.done:
		e.mu.Lock()
		defer e.mu.Unlock()
		return e.shutdownErr
	case <-ctx.Done():
		_ = e.Close()
		return ctx.Err()
	case <-e.hardCtx.Done():
		_ = e.closeTransport()
		return context.Canceled
	}
}

// Close immediately closes all connections and the QUIC transport, leaving a
// caller-supplied UDP socket open. It does not wait for application handlers
// that ignore cancellation. Repeated and concurrent calls are safe.
func (e *QUICEndpoint) Close() error {
	e.stopAccepting()
	e.mu.Lock()
	connections := make([]*quic.Conn, 0, len(e.connections))
	for conn := range e.connections {
		connections = append(connections, conn)
	}
	e.mu.Unlock()
	for _, conn := range connections {
		_ = conn.CloseWithError(0, "QUIC endpoint closed")
	}
	e.hardCancel()
	return e.closeTransport()
}

func (e *QUICEndpoint) closeTransport() error {
	e.transportOnce.Do(func() {
		if e.transport != nil {
			e.transportErr = e.transport.Close()
		}
	})
	return e.transportErr
}
