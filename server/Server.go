package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go/http3"
	"github.com/wangshiben/QuicFrameWork/server/RouteDisPatch"
	"github.com/wangshiben/QuicFrameWork/server/Session"
	"github.com/wangshiben/QuicFrameWork/server/Session/defaultSessionImp"
	"github.com/wangshiben/QuicFrameWork/server/consts"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// Server is the HTTP server implementation.
type Server struct {
	Server       *http.Server
	quicServer   *http3.Server
	listener     net.Listener
	lock         sync.Mutex
	Route        *RouteDisPatch.Route
	Session      Session.ServerSession
	generateFunc Session.GenerateItemInterFace
	otherConfig  *Config
}

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

	ln, err := net.Listen("tcp", s.Server.Addr)
	if tcpLn, ok := ln.(*net.TCPListener); ok {
		ln = tcpKeepAliveListener{TCPListener: tcpLn}
	}

	return tls.NewListener(ln, s.Server.TLSConfig), err
}

// listenPacket creates udp connection for QUIC if it is enabled,
func (s *Server) listenPacket() (net.PacketConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", s.Server.Addr)
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp", udpAddr)

}

// Serve serves requests on ln. It blocks until ln is closed.
func (s *Server) Serve(ln net.Listener) error {
	s.lock.Lock()
	s.listener = ln
	s.lock.Unlock()

	err := s.Server.Serve(ln)
	if err == http.ErrServerClosed {
		err = nil
	}
	if s.quicServer != nil {
		s.quicServer.Close()
	}
	return err
}

// ServePacket serves QUIC requests on pc until it is closed.
func (s *Server) ServePacket(pc net.PacketConn) error {
	if s.quicServer != nil {
		fmt.Println("QUIC server is already running")
		err := s.quicServer.Serve(pc.(*net.UDPConn))
		return fmt.Errorf("serving QUIC connections: %v", err)
	}
	return nil
}
func (s *Server) initContext(parent context.Context) context.Context {
	child := withParent(parent)
	child.SetValue(GetSession, s.Session)
	child.SetValue(InitSessionFunc, s.generateFunc)
	child.SetValue(MaxSessionMemo, s.otherConfig.maxMemo)
	return child
}

func (s *Server) wrapWithSvcHeaders(previousHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(s.initContext(r.Context()))
		s.quicServer.SetQUICHeaders(w.Header())
		previousHandler.ServeHTTP(w, r)
	}
}
func (s *Server) serveHttp(previousHandler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.WithContext(s.initContext(r.Context()))
		previousHandler.ServeHTTP(w, r)
	}
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
		// MinVersion:               tls.VersionTLS13,
		// CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		// PreferServerCipherSuites: true,
		Certificates: []tls.Certificate{cer},
		NextProtos:   []string{"h3", "h2", "h3-29"},
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
	s := &Server{
		Server: &http.Server{
			Addr:      addr,
			TLSConfig: config,
		},
		generateFunc: defaultSessionImp.NewMemoItemInterFace,
		Session:      defaultSessionImp.NewMemoServerSession(),
	}
	handler := RouteDisPatch.InitHandler()
	s.quicServer = &http3.Server{TLSConfig: config, Addr: addr, Handler: s.wrapWithSvcHeaders(handler)}
	s.Server.Handler = s.wrapWithSvcHeaders(handler)
	//s.quicServer.Handler=
	s.Route = handler.Routes
	// s.Server.Handler = s
	return s
}

func NewHttpServer(addr string) *Server {
	fmt.Println("已启动http服务")
	handler := RouteDisPatch.InitHandler()
	s := &Server{
		Server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
	s.Server.Handler = s.serveHttp(handler)
	s.Route = handler.Routes
	return s
}

func (s *Server) StartServer() {
	if s.otherConfig == nil {
		s.otherConfig = defaultConfig
	}
	pc, err := s.listenPacket()
	if err != nil {
		panic(err.Error())
	}
	ln, err := s.listen()

	go func() {
		s.Serve(ln)
	}()
	s.ScheduledTask()
	s.closeListener()
	s.ServePacket(pc)
}
func (s *Server) StartHttpSerer() {
	//pc, err := s.listenPacket()
	//if err != nil {
	//	panic(err.Error())
	//}
	//ln, err := s.listen()
	//if err != nil {
	//	panic(err.Error())
	//}
	//s.Serve(ln)
	if s.otherConfig == nil {
		s.otherConfig = defaultConfig
	}
	s.ScheduledTask()
	s.closeListener()
	s.Server.ListenAndServe()
}
func (s *Server) SetGenerateItemInterFace(generateFunc Session.GenerateItemInterFace) {
	s.generateFunc = generateFunc
}
