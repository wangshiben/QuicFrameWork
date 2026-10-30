package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/quic-go/quic-go/http3"
	"github.com/wangshiben/QuicFrameWork/RouteDisPatch"
	"github.com/wangshiben/QuicFrameWork/Session"
	"github.com/wangshiben/QuicFrameWork/Session/defaultSessionImp"
	"github.com/wangshiben/QuicFrameWork/consts"
)

// Server serves HTTP over TCP and dispatches QUIC application protocols over UDP.
type Server struct {
	Server       *http.Server
	QUIC         *QUICEndpoint
	quicServer   *http3.Server
	listener     net.Listener
	lock         sync.Mutex
	started      bool
	shuttingDown bool
	closeOnce    sync.Once
	closeErr     error
	stopCh       chan struct{}
	Route        *RouteDisPatch.Route
	Session      Session.ServerSession
	generateFunc Session.GenerateItemInterFace
	otherConfig  *Config
	taskCtx      context.Context
	taskCancel   context.CancelFunc
	taskOnce     sync.Once
	taskWG       sync.WaitGroup
	sessionOnce  sync.Once
	sessionErr   error
	handler      *RouteDisPatch.ServerHandler
}

const (
	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultReadTimeout       = 30 * time.Second
	DefaultIdleTimeout       = 2 * time.Minute
)

const (
	InitSessionFunc = consts.InitSessionFunc
	GetSession      = consts.GetSession
	MaxSessionMemo  = consts.MaxSessionMemo
)

// listen creates an active listener for s that can be
// used to serve requests.
func (s *Server) listen() (net.Listener, error) {
	if s.Server == nil {
		return nil, fmt.Errorf("Server field is nil")
	}
	if s.Server.TLSConfig == nil {
		return nil, errors.New("TLS configuration is missing; check certificate and key")
	}

	ln, err := net.Listen("tcp", s.Server.Addr)
	if err != nil {
		return nil, err
	}
	if tcpLn, ok := ln.(*net.TCPListener); ok {
		ln = tcpKeepAliveListener{TCPListener: tcpLn}
	}

	return tls.NewListener(ln, s.Server.TLSConfig), nil
}

// Serve serves requests on ln. It blocks until ln is closed.
func (s *Server) Serve(ln net.Listener) error {
	s.lock.Lock()
	s.listener = ln
	s.lock.Unlock()

	err := s.Server.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	s.lock.Lock()
	closeQUIC := !s.shuttingDown
	s.lock.Unlock()
	if closeQUIC && s.QUIC != nil {
		// Preserve Serve's original coupled close behavior for callers using
		// separate TCP/UDP Serve methods. A coordinated Shutdown drains QUIC.
		_ = s.QUIC.Close()
	}
	return err
}

// ServePacket serves every registered QUIC protocol on pc. The caller owns pc;
// keep it open until Shutdown completes, then close it.
func (s *Server) ServePacket(pc net.PacketConn) error {
	if s.QUIC == nil {
		return nil // Preserve the original HTTP-only ServePacket no-op.
	}
	return s.QUIC.Serve(context.Background(), pc)
}

// RegisterQUICProtocol adds a developer-defined ALPN and connection handler.
// NewServer registers h3 first. Duplicate names (including h3) are rejected.
func (s *Server) RegisterQUICProtocol(protocol QUICProtocol) error {
	if s.QUIC == nil {
		return errors.New("QUIC endpoint is not configured")
	}
	return s.QUIC.RegisterProtocol(protocol)
}
func (s *Server) initContext(parent context.Context) context.Context {
	ctx := context.WithValue(parent, GetSession, s.Session)
	ctx = context.WithValue(ctx, InitSessionFunc, s.generateFunc)
	return context.WithValue(ctx, MaxSessionMemo, s.otherConfig.maxMemo)
}

func (s *Server) wrapWithSvcHeaders(previousHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(s.initContext(r.Context()))
		// ServeQUICConn does not register a listener with http3.Server, so
		// advertise the shared endpoint's actual port, including when using :0.
		if s.QUIC != nil {
			if addr := s.QUIC.Addr(); addr != nil {
				if _, port, err := net.SplitHostPort(addr.String()); err == nil {
					if number, err := strconv.Atoi(port); err == nil && number > 0 {
						w.Header().Add("Alt-Svc", fmt.Sprintf(`h3=":%d"; ma=2592000`, number))
					}
				}
			}
		}
		previousHandler.ServeHTTP(w, r)
	}
}
func (s *Server) serveHttp(previousHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(s.initContext(r.Context()))
		previousHandler.ServeHTTP(w, r)
	}
}
func (s *Server) Close() error {
	s.lock.Lock()
	s.shuttingDown = true
	s.lock.Unlock()
	s.closeOnce.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
		}

		var errs []error
		if s.Server != nil {
			if err := s.Server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errs = append(errs, err)
			}
		}
		if s.QUIC != nil {
			if err := s.QUIC.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				errs = append(errs, err)
			}
		}
		s.lock.Lock()
		listener := s.listener
		s.lock.Unlock()
		if listener != nil {
			if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				errs = append(errs, err)
			}
		}
		if err := s.closeSession(); err != nil {
			errs = append(errs, err)
		}
		s.closeErr = errors.Join(errs...)
	})
	return s.closeErr
}

// Shutdown drains HTTP and every QUIC protocol concurrently. Expiring ctx
// forces connection closure. Signal handling belongs in the calling program.
func (s *Server) Shutdown(ctx context.Context) error {
	s.lock.Lock()
	s.shuttingDown = true
	s.lock.Unlock()
	results := make(chan error, 2)
	go func() {
		if s.Server == nil {
			results <- nil
			return
		}
		results <- s.Server.Shutdown(ctx)
	}()
	go func() {
		if s.QUIC == nil {
			results <- nil
			return
		}
		results <- s.QUIC.Shutdown(ctx)
	}()
	err := errors.Join(<-results, <-results)
	if err != nil {
		return errors.Join(err, s.Close())
	}
	return s.closeSession()
}

func (s *Server) closeSession() error {
	s.sessionOnce.Do(func() {
		s.lock.Lock()
		if s.taskCancel != nil {
			s.taskCancel()
		}
		s.lock.Unlock()
		s.taskWG.Wait()
		if s.Session != nil {
			s.sessionErr = s.Session.Close()
		}
	})
	return s.sessionErr
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

// Accept accepts the connection with a keep-alive enabled.
func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func (ln tcpKeepAliveListener) File() (*os.File, error) {
	return ln.TCPListener.File()
}

func loadTLS(TLSPem, TLSKey string) *tls.Config {
	cer, err := tls.LoadX509KeyPair(TLSPem, TLSKey)
	if err != nil {
		log.Println(err)
		return nil
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cer},
		NextProtos:   []string{"h2", "http/1.1"},
	}
	return config
}

func NewServer(TLSPem, TLSKey, addr string) *Server {
	if len(TLSPem) == 0 || len(TLSKey) == 0 {
		fmt.Println("未提供TLS证书，已自动生成")
		CreateESDATLS()
		TLSPem = DefaultPem
		TLSKey = DefaultKey
	}
	config := loadTLS(TLSPem, TLSKey)
	s := initDefaultServer()
	s.Server = &http.Server{
		Addr:              addr,
		TLSConfig:         config,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		ReadTimeout:       DefaultReadTimeout,
		IdleTimeout:       DefaultIdleTimeout,
	}
	handler := RouteDisPatch.InitHandler()
	s.handler = handler
	s.quicServer = &http3.Server{
		TLSConfig:   config,
		Addr:        addr,
		Handler:     s.wrapWithSvcHeaders(handler),
		IdleTimeout: DefaultIdleTimeout,
	}
	s.QUIC = NewQUICEndpoint(config, nil)
	_ = s.RegisterQUICProtocol(NewHTTP3Protocol(s.quicServer))
	s.Server.Handler = s.wrapWithSvcHeaders(handler)
	//s.quicServer.Handler=
	s.Route = handler.Routes
	// s.Server.Handler = s
	return s
}

func NewHttpServer(addr string) *Server {
	fmt.Println("已启动http服务")
	handler := RouteDisPatch.InitHandler()
	s := initDefaultServer()
	s.Server = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		ReadTimeout:       DefaultReadTimeout,
		IdleTimeout:       DefaultIdleTimeout,
	}
	s.handler = handler
	s.Server.Handler = s.serveHttp(handler)
	s.Route = handler.Routes
	return s
}

// ListenAndServe starts TLS/TCP and the shared QUIC/UDP endpoint. When Addr
// uses port 0, both transports bind the same chosen port. Cancellation forces
// closure; use Shutdown with its own deadline for graceful draining instead.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.QUIC == nil {
		return errors.New("TLS QUIC endpoint is not configured; use NewServer")
	}
	s.lock.Lock()
	if s.shuttingDown {
		s.lock.Unlock()
		return http.ErrServerClosed
	}
	if s.started {
		s.lock.Unlock()
		return errors.New("server has already been started")
	}
	s.started = true
	s.lock.Unlock()
	ln, err := s.listen()
	if err != nil {
		return err
	}
	defer ln.Close()
	pc, err := net.ListenPacket("udp", ln.Addr().String())
	if err != nil {
		return err
	}
	defer pc.Close()
	s.ScheduledTask()
	stop := context.AfterFunc(ctx, func() { _ = s.Close() })
	defer stop()
	results := make(chan error, 2)
	go func() { results <- s.Serve(ln) }()
	go func() { results <- s.QUIC.Serve(ctx, pc) }()
	var errs []error
	for range 2 {
		err := <-results
		if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, ctx.Err()) {
			errs = append(errs, err)
			_ = s.Close()
		}
	}
	// Keep UDP open until HTTP/3 GOAWAY and developer protocols finish draining.
	if len(errs) == 0 {
		if err := s.QUIC.Shutdown(ctx); err != nil && !errors.Is(err, ctx.Err()) && !errors.Is(err, context.Canceled) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// StartServer preserves the original blocking API and automatic signal
// handling. Prefer ListenAndServe for explicit lifecycle management.
func (s *Server) StartServer() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := s.ListenAndServe(ctx); err != nil {
		panic(err.Error())
	}
}

func initDefaultServer() *Server {
	taskCtx, taskCancel := context.WithCancel(context.Background())
	return &Server{
		Server:       &http.Server{},
		generateFunc: defaultSessionImp.NewMemoItemInterFace,
		Session:      defaultSessionImp.NewServerSession(),
		otherConfig:  defaultConfig,
		taskCtx:      taskCtx,
		taskCancel:   taskCancel,
		stopCh:       make(chan struct{}),
	}
}
func (s *Server) StartHttpSerer() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	stopClose := context.AfterFunc(ctx, func() { _ = s.Close() })
	defer stopClose()
	s.ScheduledTask()
	if err := s.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server stopped: %v", err)
	}
}
func (s *Server) SetGenerateItemInterFace(generateFunc Session.GenerateItemInterFace) {
	if generateFunc != nil {
		s.generateFunc = generateFunc
	}
}
func (s *Server) SetMemoServerSession(sessionMemo Session.ServerSession) {
	if sessionMemo != nil {
		s.Session = sessionMemo
	}
}
