package server

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"crypto/tls"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/wangshiben/QuicFrameWork/RouteDisPatch"
)

func newQUICFrameworkServer(t *testing.T) (*Server, *tls.Config) {
	t.Helper()
	serverTLS, clientTLS := testTLS(t)
	dir := t.TempDir()
	certPath, keyPath := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	cert := serverTLS.Certificates[0]
	key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), 0600); err != nil {
		t.Fatal(err)
	}
	s := NewServer(certPath, keyPath, "127.0.0.1:0")
	t.Cleanup(func() { _ = s.Close() })
	return s, clientTLS
}

func TestServerSharedQUICProtocols(t *testing.T) {
	s, clientTLS := newQUICFrameworkServer(t)
	for _, p := range []QUICProtocol{
		echoProtocol("echo/1", func(data []byte) []byte { return append([]byte("echo:"), data...) }),
		echoProtocol("upper/1", bytes.ToUpper),
	} {
		if err := s.RegisterQUICProtocol(p); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RegisterQUICProtocol(echoProtocol("h3", bytes.ToUpper)); err == nil {
		t.Fatal("overwrote built-in HTTP/3")
	}
	s.AddHttpHandler("/probe", http.MethodGet, func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		session, err := r.GetSession()
		if err != nil {
			panic(err)
		}
		previous, err := session.GetStoreValue("visited")
		if err != nil {
			panic(err)
		}
		if err := session.Store("visited", true); err != nil {
			panic(err)
		}
		_, _ = fmt.Fprintf(w, "%s visited=%v", r.GetRequest().Proto, previous)
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveErr := make(chan error, 1)
	go func() { serveErr <- s.ListenAndServe(ctx) }()
	addr := waitForQUIC(t, s.QUIC, serveErr)
	s.lock.Lock()
	tcpAddr := s.listener
	s.lock.Unlock()
	// The TCP Serve goroutine may not have published its listener yet; the
	// requests below independently prove TCP uses the advertised UDP port.
	if tcpAddr != nil && tcpAddr.Addr().String() != addr {
		t.Fatalf("TCP and UDP addresses differ: %s / %s", tcpAddr.Addr(), addr)
	}
	h3 := &http3.Transport{TLSClientConfig: clientTLS.Clone()}
	t.Cleanup(func() { _ = h3.Close() })
	jar, _ := cookiejar.New(nil)
	h3Client := &http.Client{Transport: h3, Jar: jar, Timeout: 3 * time.Second}
	for _, want := range []string{"HTTP/3.0 visited=<nil>", "HTTP/3.0 visited=true"} {
		response, err := h3Client.Get("https://" + addr + "/probe")
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil || response.StatusCode != 200 || string(data) != want {
			t.Fatalf("HTTP/3 response: %d %q %v", response.StatusCode, data, err)
		}
	}
	for _, tc := range []struct{ alpn, want string }{{"echo/1", "echo:hello"}, {"upper/1", "HELLO"}} {
		conn := dialProtocol(t, addr, clientTLS, tc.alpn)
		if conn.ConnectionState().TLS.NegotiatedProtocol != tc.alpn {
			t.Fatalf("wrong ALPN for %s", tc.alpn)
		}
		if got, err := exchange(conn, "hello"); err != nil || got != tc.want {
			t.Fatalf("%s response = %q, %v", tc.alpn, got, err)
		}
		_ = conn.CloseWithError(0, "done")
	}
	_, port, _ := net.SplitHostPort(addr)
	for _, tc := range []struct {
		http2 bool
		major int
	}{{false, 1}, {true, 2}} {
		transport := &http.Transport{TLSClientConfig: clientTLS.Clone(), ForceAttemptHTTP2: tc.http2}
		client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
		response, err := client.Get("https://" + addr + "/probe")
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		transport.CloseIdleConnections()
		if err != nil || response.ProtoMajor != tc.major || !strings.Contains(string(data), "visited=") {
			t.Fatalf("HTTP/%d response: %s %q %v", tc.major, response.Proto, data, err)
		}
		if got := response.Header.Get("Alt-Svc"); got != fmt.Sprintf(`h3=":%s"; ma=2592000`, port) {
			t.Fatalf("Alt-Svc = %q", got)
		}
	}
	_ = h3.Close()
	shutdownCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	if err := s.Shutdown(shutdownCtx); err != nil {
		t.Fatal(err)
	}
	if err := receive(t, serveErr); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	t.Log("one UDP port served h3, echo/1, upper/1; the same TCP port served HTTP/1.1 and HTTP/2")
}

func TestHTTP3ShutdownDeadlineClosesActiveConnection(t *testing.T) {
	s, clientTLS := newQUICFrameworkServer(t)
	entered, exited := make(chan struct{}), make(chan struct{})
	s.AddHttpHandler("/wait", http.MethodGet, func(_ http.ResponseWriter, r *RouteDisPatch.Request) {
		close(entered)
		<-r.GetRequest().Context().Done()
		close(exited)
	})
	serveErr := make(chan error, 1)
	go func() { serveErr <- s.ListenAndServe(context.Background()) }()
	addr := waitForQUIC(t, s.QUIC, serveErr)
	transport := &http3.Transport{TLSClientConfig: clientTLS}
	defer transport.Close()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	requestErr := make(chan error, 1)
	go func() {
		response, err := client.Get("https://" + addr + "/wait")
		if response != nil {
			_ = response.Body.Close()
		}
		requestErr <- err
	}()
	receive(t, entered)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	shutdownErr := make(chan error, 1)
	go func() { shutdownErr <- s.Shutdown(ctx) }()
	if err := receive(t, shutdownErr); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown = %v", err)
	}
	receive(t, exited)
	if err := receive(t, requestErr); err == nil {
		t.Fatal("active request unexpectedly succeeded")
	}
	_ = receive(t, serveErr)
}

// PacketConn wrappers are valid inputs; serving must not assert *net.UDPConn.
type wrappedPacketConn struct{ net.PacketConn }

func TestServePacketSupportsPacketConnWrapper(t *testing.T) {
	s, clientTLS := newQUICFrameworkServer(t)
	if err := s.RegisterQUICProtocol(echoProtocol("wrapped/1", bytes.ToUpper)); err != nil {
		t.Fatal(err)
	}
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	serveErr := make(chan error, 1)
	go func() { serveErr <- s.ServePacket(wrappedPacketConn{pc}) }()
	waitForQUIC(t, s.QUIC, serveErr)
	conn := dialProtocol(t, pc.LocalAddr().String(), clientTLS, "wrapped/1")
	if got, err := exchange(conn, "wrapper"); err != nil || got != "WRAPPER" {
		t.Fatalf("wrapped PacketConn = %q, %v", got, err)
	}
	_ = conn.CloseWithError(0, "done")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := receive(t, serveErr); err != nil {
		t.Fatal(err)
	}
}

func TestServerCloseBeforeStartAndHTTPOnly(t *testing.T) {
	s := NewHttpServer("127.0.0.1:0")
	if err := s.RegisterQUICProtocol(QUICProtocolFunc("unused/1", func(context.Context, *quic.Conn) error { return nil })); err == nil {
		t.Fatal("registered QUIC on an HTTP-only server")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s.ScheduledTask() // Must not restart maintenance after Close.
	s, _ = newQUICFrameworkServer(t)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHTTP3GracefulShutdownCompletesActiveRequest(t *testing.T) {
	s, clientTLS := newQUICFrameworkServer(t)
	entered, release := make(chan struct{}), make(chan struct{})
	s.AddHttpHandler("/drain", http.MethodGet, func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		close(entered)
		select {
		case <-release:
			_, _ = w.Write([]byte("completed"))
		case <-r.GetRequest().Context().Done():
		}
	})
	serveErr := make(chan error, 1)
	go func() { serveErr <- s.ListenAndServe(context.Background()) }()
	addr := waitForQUIC(t, s.QUIC, serveErr)
	transport := &http3.Transport{TLSClientConfig: clientTLS}
	defer transport.Close()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	requestErr := make(chan error, 1)
	go func() {
		response, err := client.Get("https://" + addr + "/drain")
		if err == nil {
			payload, readErr := io.ReadAll(response.Body)
			_ = response.Body.Close()
			err = readErr
			if err == nil && string(payload) != "completed" {
				err = fmt.Errorf("response = %q", payload)
			}
		}
		requestErr <- err
	}()
	receive(t, entered)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	shutdownErr := make(chan error, 1)
	go func() { shutdownErr <- s.Shutdown(ctx) }()
	receive(t, s.QUIC.drainCtx.Done())
	select {
	case err := <-shutdownErr:
		t.Fatalf("shutdown interrupted active request: %v", err)
	default:
	}
	close(release)
	if err := receive(t, requestErr); err != nil {
		t.Fatal(err)
	}
	_ = transport.Close()
	if err := receive(t, shutdownErr); err != nil {
		t.Fatal(err)
	}
	if err := receive(t, serveErr); err != nil {
		t.Fatal(err)
	}
}

func TestServerStartupFailureReleasesTCPPort(t *testing.T) {
	s, _ := newQUICFrameworkServer(t)
	occupied, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	s.Server.Addr = occupied.LocalAddr().String()
	if err := s.ListenAndServe(context.Background()); err == nil {
		t.Fatal("startup ignored occupied UDP port")
	}
	listener, err := net.Listen("tcp", s.Server.Addr)
	if err != nil {
		t.Fatalf("TCP listener leaked after UDP startup failure: %v", err)
	}
	_ = listener.Close()
}

func TestLegacyStartServerWithCustomQUICProtocol(t *testing.T) {
	s, clientTLS := newQUICFrameworkServer(t)
	if err := s.RegisterQUICProtocol(echoProtocol("legacy/1", bytes.ToUpper)); err != nil {
		t.Fatal(err)
	}
	serveErr := make(chan error, 1)
	go func() { s.StartServer(); serveErr <- nil }()
	addr := waitForQUIC(t, s.QUIC, serveErr)
	conn := dialProtocol(t, addr, clientTLS, "legacy/1")
	if got, err := exchange(conn, "unchanged API"); err != nil || got != "UNCHANGED API" {
		t.Fatalf("legacy startup response = %q, %v", got, err)
	}
	_ = conn.CloseWithError(0, "done")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := receive(t, serveErr); err != nil {
		t.Fatal(err)
	}
}
