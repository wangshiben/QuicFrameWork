// This example serves HTTP/1.1, HTTP/2, HTTP/3, echo/1 and upper/1 at one
// address. See docs/native-quic.md for certificate and client commands.
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/wangshiben/QuicFrameWork/RouteDisPatch"
	"github.com/wangshiben/QuicFrameWork/server"
)

func main() {
	mode := flag.String("mode", "server", "server or client")
	addr := flag.String("addr", "localhost:4445", "shared TCP and UDP address")
	cert := flag.String("cert", "cert.pem", "server certificate PEM")
	key := flag.String("key", "cert.key", "server private key PEM")
	ca := flag.String("ca", "cert.pem", "trusted CA or self-signed certificate PEM")
	alpn := flag.String("alpn", "echo/1", "client protocol: h3, echo/1 or upper/1")
	message := flag.String("message", "hello QUIC", "custom protocol request")
	flag.Parse()
	var err error
	switch *mode {
	case "server":
		err = serve(*addr, *cert, *key)
	case "client":
		err = call(*addr, *ca, *alpn, *message)
	default:
		err = fmt.Errorf("unknown mode %q", *mode)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func serve(addr, cert, key string) error {
	s := server.NewServer(cert, key, addr)
	s.QUIC.QUICConfig = &quic.Config{MaxIncomingStreams: 64, MaxIdleTimeout: 30 * time.Second}
	s.AddHttpHandler("/hello", http.MethodGet, func(w http.ResponseWriter, r *RouteDisPatch.Request) {
		_, _ = fmt.Fprintf(w, "hello over %s\n", r.GetRequest().Proto)
	})
	for _, p := range []server.QUICProtocol{
		streamProtocol("echo/1", func(data []byte) []byte { return data }),
		streamProtocol("upper/1", bytes.ToUpper),
	} {
		if err := s.RegisterQUICProtocol(p); err != nil {
			return err
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result := make(chan error, 1)
	go func() { result <- s.ListenAndServe(context.Background()) }()
	fmt.Printf("starting TCP+QUIC at %s; ALPNs=%v\n", addr, s.QUIC.ALPNs())
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.Shutdown(shutdownCtx)
	return errors.Join(err, <-result)
}

// This example protocol uses stream FIN to delimit a request. Its framing and
// 1 MiB body limit belong to the application, not to the framework dispatcher.
func streamProtocol(alpn string, transform func([]byte) []byte) server.QUICProtocol {
	return server.QUICProtocolFunc(alpn, func(ctx context.Context, conn *quic.Conn) error {
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
				_ = stream.SetDeadline(time.Now().Add(10 * time.Second))
				data, err := io.ReadAll(io.LimitReader(stream, (1<<20)+1))
				if err != nil || len(data) > 1<<20 {
					stream.CancelRead(1)
					stream.CancelWrite(1)
					return
				}
				if _, err := stream.Write(transform(data)); err != nil {
					stream.CancelWrite(1)
					return
				}
				_ = stream.Close()
			}()
		}
	})
}

func call(addr, ca, alpn, message string) error {
	certificate, err := os.ReadFile(ca)
	if err != nil {
		return err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certificate) {
		return errors.New("no certificates found in CA file")
	}
	tlsConfig := &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS13, NextProtos: []string{alpn}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if alpn == http3.NextProtoH3 {
		transport := &http3.Transport{TLSClientConfig: tlsConfig}
		defer transport.Close()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+addr+"/hello", nil)
		if err != nil {
			return err
		}
		response, err := (&http.Client{Transport: transport}).Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		if err == nil {
			fmt.Printf("ALPN=h3 HTTP=%s response=%q\n", response.Proto, data)
		}
		return err
	}
	conn, err := quic.DialAddr(ctx, addr, tlsConfig, nil)
	if err != nil {
		return err
	}
	defer conn.CloseWithError(0, "client finished")
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return err
	}
	_ = stream.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := stream.Write([]byte(message)); err != nil {
		return err
	}
	if err := stream.Close(); err != nil {
		return err
	}
	response, err := io.ReadAll(io.LimitReader(stream, 1<<20))
	if err == nil {
		fmt.Printf("ALPN=%s response=%q\n", conn.ConnectionState().TLS.NegotiatedProtocol, response)
	}
	return err
}
