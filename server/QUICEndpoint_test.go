package server

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
)

func testTLS(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour),
		DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	return &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}},
		&tls.Config{RootCAs: roots, ServerName: "localhost"}
}

func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker")
		var zero T
		return zero
	}
}

func waitForQUIC(t *testing.T, e *QUICEndpoint, serveErr <-chan error) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if addr := e.Addr(); addr != nil {
			return addr.String()
		}
		select {
		case err := <-serveErr:
			t.Fatalf("server stopped before listening: %v", err)
		case <-ctx.Done():
			t.Fatal("QUIC endpoint did not start")
		case <-ticker.C:
		}
	}
}

func startEndpoint(t *testing.T, e *QUICEndpoint) (net.PacketConn, <-chan error) {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- e.Serve(context.Background(), pc) }()
	t.Cleanup(func() {
		_ = e.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = e.Shutdown(ctx)
		_ = pc.Close()
	})
	waitForQUIC(t, e, serveErr)
	return pc, serveErr
}

func dialProtocol(t *testing.T, addr string, clientTLS *tls.Config, alpns ...string) *quic.Conn {
	t.Helper()
	config := clientTLS.Clone()
	config.NextProtos = alpns
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := quic.DialAddr(ctx, addr, config, &quic.Config{EnableDatagrams: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.CloseWithError(0, "test complete") })
	return conn
}

func echoProtocol(alpn string, transform func([]byte) []byte) QUICProtocol {
	return QUICProtocolFunc(alpn, func(ctx context.Context, conn *quic.Conn) error {
		var workers sync.WaitGroup
		defer workers.Wait()
		for {
			stream, err := conn.AcceptStream(ctx)
			if err != nil {
				return nil
			}
			workers.Add(1)
			go func() {
				defer workers.Done()
				payload, err := io.ReadAll(io.LimitReader(stream, 4097))
				if err != nil || len(payload) > 4096 {
					stream.CancelRead(1)
					stream.CancelWrite(1)
					return
				}
				_, _ = stream.Write(transform(payload))
				_ = stream.Close()
			}()
		}
	})
}

func exchange(conn *quic.Conn, payload string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return "", err
	}
	_ = stream.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err = stream.Write([]byte(payload)); err != nil {
		return "", err
	}
	if err = stream.Close(); err != nil {
		return "", err
	}
	response, err := io.ReadAll(stream)
	return string(response), err
}

func TestQUICProtocolRegistration(t *testing.T) {
	e := NewQUICEndpoint(nil, nil)
	var typedNil *http3Protocol
	for _, invalid := range []QUICProtocol{
		nil, typedNil, QUICProtocolFunc("nil/1", nil),
		echoProtocol("", bytes.ToUpper), echoProtocol(strings.Repeat("x", 256), bytes.ToUpper),
	} {
		if err := e.RegisterProtocol(invalid); err == nil {
			t.Fatalf("accepted invalid registration: %#v", invalid)
		}
	}
	for _, name := range []string{"one/1", "two/1", strings.Repeat("x", 255)} {
		if err := e.RegisterProtocol(echoProtocol(name, bytes.ToUpper)); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.RegisterProtocol(echoProtocol("one/1", bytes.ToUpper)); err == nil {
		t.Fatal("accepted duplicate ALPN")
	}
	names := e.ALPNs()
	names[0] = "modified"
	if !slices.Equal(e.ALPNs(), []string{"one/1", "two/1", strings.Repeat("x", 255)}) {
		t.Fatal("registry order or isolation was lost")
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	if err := e.RegisterProtocol(echoProtocol("late/1", bytes.ToUpper)); err == nil {
		t.Fatal("accepted registration after close")
	}
}

func TestQUICEndpointALPNAndSNI(t *testing.T) {
	serverTLS, clientTLS := testTLS(t)
	selected := serverTLS.Clone()
	selected.NextProtos = []string{"unregistered/1"}
	serverTLS.NextProtos = []string{"h2", "http/1.1"}
	serverTLS.GetConfigForClient = func(*tls.ClientHelloInfo) (*tls.Config, error) { return selected, nil }
	e := NewQUICEndpoint(serverTLS, nil)
	for _, alpn := range []string{"first/1", "second/1"} {
		if err := e.RegisterProtocol(echoProtocol(alpn, bytes.ToUpper)); err != nil {
			t.Fatal(err)
		}
	}
	pc, serveErr := startEndpoint(t, e)
	conn := dialProtocol(t, pc.LocalAddr().String(), clientTLS, "second/1", "first/1")
	if got := conn.ConnectionState().TLS.NegotiatedProtocol; got != "first/1" {
		t.Fatalf("negotiated ALPN = %q", got)
	}
	if got, err := exchange(conn, "hello"); err != nil || got != "HELLO" {
		t.Fatalf("response = %q, error = %v", got, err)
	}
	if err := e.RegisterProtocol(echoProtocol("late/1", bytes.ToUpper)); err == nil {
		t.Fatal("accepted registration after Serve")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	badTLS := clientTLS.Clone()
	badTLS.NextProtos = []string{"unknown/1"}
	if conn, err := quic.DialAddr(ctx, pc.LocalAddr().String(), badTLS, nil); err == nil {
		_ = conn.CloseWithError(0, "unexpected")
		t.Fatal("unknown ALPN completed TLS negotiation")
	}
	if !slices.Equal(serverTLS.NextProtos, []string{"h2", "http/1.1"}) || !slices.Equal(selected.NextProtos, []string{"unregistered/1"}) {
		t.Fatal("endpoint mutated caller TLS configurations")
	}
	_ = conn.CloseWithError(0, "done")
	if err := e.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := receive(t, serveErr); err != nil {
		t.Fatal(err)
	}
}

func TestQUICEndpointConcurrentStreamsAndReset(t *testing.T) {
	serverTLS, clientTLS := testTLS(t)
	reset := make(chan error, 1)
	e := NewQUICEndpoint(serverTLS, &quic.Config{MaxIncomingStreams: 64})
	p := QUICProtocolFunc("streams/1", func(ctx context.Context, conn *quic.Conn) error {
		var workers sync.WaitGroup
		defer workers.Wait()
		for {
			stream, err := conn.AcceptStream(ctx)
			if err != nil {
				return nil
			}
			workers.Add(1)
			go func() {
				defer workers.Done()
				payload, err := io.ReadAll(io.LimitReader(stream, 4096))
				if err != nil {
					select {
					case reset <- err:
					default:
					}
					return
				}
				_, _ = stream.Write(payload)
				_ = stream.Close()
			}()
		}
	})
	if err := e.RegisterProtocol(p); err != nil {
		t.Fatal(err)
	}
	pc, _ := startEndpoint(t, e)
	conn := dialProtocol(t, pc.LocalAddr().String(), clientTLS, "streams/1")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = stream.Write([]byte("incomplete"))
	stream.CancelWrite(42)
	stream.CancelRead(42)
	var resetErr *quic.StreamError
	if err := receive(t, reset); !errors.As(err, &resetErr) || resetErr.ErrorCode != 42 {
		t.Fatalf("reset error = %v", err)
	}
	results := make(chan error, 24)
	for i := range 24 {
		go func() {
			payload := fmt.Sprintf("request-%d", i)
			got, err := exchange(conn, payload)
			if err == nil && got != payload {
				err = fmt.Errorf("response %q != %q", got, payload)
			}
			results <- err
		}()
	}
	for range 24 {
		if err := receive(t, results); err != nil {
			t.Fatal(err)
		}
	}
}

func TestQUICEndpointUnidirectionalStreamsAndDatagrams(t *testing.T) {
	serverTLS, clientTLS := testTLS(t)
	e := NewQUICEndpoint(serverTLS, &quic.Config{EnableDatagrams: true})
	p := QUICProtocolFunc("messages/1", func(ctx context.Context, conn *quic.Conn) error {
		var workers sync.WaitGroup
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				payload, err := conn.ReceiveDatagram(ctx)
				if err != nil {
					return
				}
				_ = conn.SendDatagram(append([]byte("datagram:"), payload...))
			}
		}()
		defer workers.Wait()
		for {
			stream, err := conn.AcceptUniStream(ctx)
			if err != nil {
				return nil
			}
			payload, err := io.ReadAll(io.LimitReader(stream, 1024))
			if err != nil {
				return err
			}
			response, err := conn.OpenUniStreamSync(ctx)
			if err != nil {
				return err
			}
			_, _ = response.Write(append([]byte("uni:"), payload...))
			_ = response.Close()
		}
	})
	if err := e.RegisterProtocol(p); err != nil {
		t.Fatal(err)
	}
	pc, _ := startEndpoint(t, e)
	conn := dialProtocol(t, pc.LocalAddr().String(), clientTLS, "messages/1")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stream, err := conn.OpenUniStreamSync(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = stream.Write([]byte("hello"))
	_ = stream.Close()
	response, err := conn.AcceptUniStream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.SetReadDeadline(time.Now().Add(time.Second))
	data, err := io.ReadAll(response)
	if err != nil || string(data) != "uni:hello" {
		t.Fatalf("unidirectional response = %q, %v", data, err)
	}
	if !conn.ConnectionState().SupportsDatagrams.Remote {
		t.Fatal("datagrams were not negotiated")
	}
	if err := conn.SendDatagram([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	data, err = conn.ReceiveDatagram(ctx)
	if err != nil || string(data) != "datagram:hello" {
		t.Fatalf("datagram response = %q, %v", data, err)
	}
}

type testQUICProtocol struct {
	alpn     string
	serve    QUICProtocolHandlerFunc
	shutdown func(context.Context) error
}

func (p *testQUICProtocol) ALPN() string { return p.alpn }
func (p *testQUICProtocol) ServeQUICConn(ctx context.Context, conn *quic.Conn) error {
	return p.serve(ctx, conn)
}
func (p *testQUICProtocol) Shutdown(ctx context.Context) error { return p.shutdown(ctx) }

func TestQUICEndpointGracefulShutdownKeepsUDPAlive(t *testing.T) {
	serverTLS, clientTLS := testTLS(t)
	entered, draining, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	hookCalled := make(chan struct{})
	e := NewQUICEndpoint(serverTLS, nil)
	p := &testQUICProtocol{alpn: "drain/1", serve: func(ctx context.Context, conn *quic.Conn) error {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return err
		}
		if _, err := io.ReadFull(stream, make([]byte, 1)); err != nil {
			return err
		}
		close(entered)
		<-ctx.Done()
		close(draining)
		select {
		case <-release:
		case <-conn.Context().Done():
			return nil
		}
		_, _ = stream.Write([]byte("drained"))
		_ = stream.Close()
		<-conn.Context().Done()
		return nil
	}, shutdown: func(context.Context) error { close(hookCalled); return nil }}
	if err := e.RegisterProtocol(p); err != nil {
		t.Fatal(err)
	}
	pc, serveErr := startEndpoint(t, e)
	conn := dialProtocol(t, pc.LocalAddr().String(), clientTLS, "drain/1")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_ = stream.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = stream.Write([]byte{1})
	_ = stream.Close()
	receive(t, entered)
	shutdownErr := make(chan error, 1)
	go func() { shutdownErr <- e.Shutdown(ctx) }()
	receive(t, draining)
	receive(t, hookCalled)
	if err := receive(t, serveErr); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-shutdownErr:
		t.Fatalf("shutdown returned before active handler: %v", err)
	default:
	}
	close(release)
	data, err := io.ReadAll(stream)
	if err != nil || string(data) != "drained" {
		t.Fatalf("draining response = %q, %v", data, err)
	}
	_ = conn.CloseWithError(0, "drained")
	if err := receive(t, shutdownErr); err != nil {
		t.Fatal(err)
	}
	if err := e.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	// The caller's PacketConn is still usable, with no QUIC read loop left.
	sender, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer sender.Close()
	_, _ = sender.WriteTo([]byte("owned by caller"), pc.LocalAddr())
	_ = pc.SetReadDeadline(time.Now().Add(time.Second))
	buffer := make([]byte, 1500)
	for {
		n, _, err := pc.ReadFrom(buffer)
		if err != nil {
			t.Fatal(err)
		}
		if string(buffer[:n]) == "owned by caller" {
			break
		}
	}
}

func TestQUICEndpointShutdownDeadlineAndConcurrentClose(t *testing.T) {
	serverTLS, clientTLS := testTLS(t)
	entered, exited := make(chan struct{}), make(chan struct{})
	e := NewQUICEndpoint(serverTLS, nil)
	if err := e.RegisterProtocol(QUICProtocolFunc("blocked/1", func(_ context.Context, conn *quic.Conn) error {
		close(entered)
		<-conn.Context().Done()
		close(exited)
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	pc, serveErr := startEndpoint(t, e)
	conn := dialProtocol(t, pc.LocalAddr().String(), clientTLS, "blocked/1")
	receive(t, entered)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := e.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown = %v", err)
	}
	receive(t, conn.Context().Done())
	receive(t, exited)
	if err := receive(t, serveErr); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 8)
	for range 8 {
		go func() { results <- e.Close() }()
	}
	for range 8 {
		if err := receive(t, results); err != nil {
			t.Fatal(err)
		}
	}
}

func TestQUICEndpointPanicAndApplicationErrorIsolation(t *testing.T) {
	serverTLS, clientTLS := testTLS(t)
	e := NewQUICEndpoint(serverTLS, nil)
	e.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, protocol := range []QUICProtocol{
		QUICProtocolFunc("panic/1", func(context.Context, *quic.Conn) error { panic("boom") }),
		QUICProtocolFunc("error/1", func(context.Context, *quic.Conn) error {
			return &quic.ApplicationError{ErrorCode: 77, ErrorMessage: "application refused"}
		}),
		echoProtocol("healthy/1", bytes.ToUpper),
	} {
		if err := e.RegisterProtocol(protocol); err != nil {
			t.Fatal(err)
		}
	}
	pc, _ := startEndpoint(t, e)
	for _, tc := range []struct {
		alpn string
		code quic.ApplicationErrorCode
	}{{"panic/1", 1}, {"error/1", 77}} {
		conn := dialProtocol(t, pc.LocalAddr().String(), clientTLS, tc.alpn)
		receive(t, conn.Context().Done())
		var appErr *quic.ApplicationError
		if err := context.Cause(conn.Context()); !errors.As(err, &appErr) || appErr.ErrorCode != tc.code {
			t.Fatalf("%s close = %v", tc.alpn, err)
		}
	}
	conn := dialProtocol(t, pc.LocalAddr().String(), clientTLS, "healthy/1")
	if got, err := exchange(conn, "healthy"); err != nil || got != "HEALTHY" {
		t.Fatalf("healthy protocol = %q, %v", got, err)
	}
}

func TestQUICEndpointServeContextCancellation(t *testing.T) {
	serverTLS, clientTLS := testTLS(t)
	e := NewQUICEndpoint(serverTLS, nil)
	entered, canceled := make(chan struct{}), make(chan struct{})
	if err := e.RegisterProtocol(QUICProtocolFunc("cancel/1", func(ctx context.Context, _ *quic.Conn) error {
		close(entered)
		<-ctx.Done()
		close(canceled)
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveErr := make(chan error, 1)
	go func() { serveErr <- e.ListenAndServe(ctx, "127.0.0.1:0") }()
	t.Cleanup(func() { _ = e.Close() })
	addr := waitForQUIC(t, e, serveErr)
	conn := dialProtocol(t, addr, clientTLS, "cancel/1")
	receive(t, entered)
	cancel()
	if err := receive(t, serveErr); !errors.Is(err, context.Canceled) {
		t.Fatalf("Serve = %v", err)
	}
	receive(t, canceled)
	receive(t, conn.Context().Done())
}

func TestQUICEndpointStreamLimit(t *testing.T) {
	serverTLS, clientTLS := testTLS(t)
	accepted := make(chan struct{}, 2)
	e := NewQUICEndpoint(serverTLS, &quic.Config{MaxIncomingStreams: 2})
	if err := e.RegisterProtocol(QUICProtocolFunc("limited/1", func(ctx context.Context, conn *quic.Conn) error {
		for {
			_, err := conn.AcceptStream(ctx)
			if err != nil {
				return nil
			}
			accepted <- struct{}{}
		}
	})); err != nil {
		t.Fatal(err)
	}
	pc, _ := startEndpoint(t, e)
	conn := dialProtocol(t, pc.LocalAddr().String(), clientTLS, "limited/1")
	for range 2 {
		stream, err := conn.OpenStream()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := stream.Write([]byte{1}); err != nil {
			t.Fatal(err)
		}
		receive(t, accepted)
	}
	if _, err := conn.OpenStream(); err == nil {
		t.Fatal("third stream exceeded configured concurrent stream limit")
	}
}

func TestQUICEndpointCloseDoesNotWaitForUncooperativeHandler(t *testing.T) {
	serverTLS, clientTLS := testTLS(t)
	e := NewQUICEndpoint(serverTLS, nil)
	entered, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	if err := e.RegisterProtocol(QUICProtocolFunc("hold/1", func(context.Context, *quic.Conn) error {
		close(entered)
		<-release
		return nil
	})); err != nil {
		t.Fatal(err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- e.ListenAndServe(context.Background(), "127.0.0.1:0") }()
	addr := waitForQUIC(t, e, serveErr)
	conn := dialProtocol(t, addr, clientTLS, "hold/1")
	receive(t, entered)
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	if err := receive(t, serveErr); err != nil {
		t.Fatal(err)
	}
	receive(t, conn.Context().Done())
}
