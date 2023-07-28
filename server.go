package socketigo

import (
	"io"
	"net/http"
	"time"

	engineigo "engine.igo/v4"
	"go.uber.org/zap"
)

type ServerOption func(o *Server)

func WithPingInterval(intv time.Duration) ServerOption {
	return func(s *Server) {
		s.engineOpts = append(s.engineOpts, engineigo.WithPingInterval(intv))
	}
}

func WithPingTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.engineOpts = append(s.engineOpts, engineigo.WithPingTimeout(timeout))
	}
}

func WithMaxPayload(payload int64) ServerOption {
	return func(s *Server) {
		s.engineOpts = append(s.engineOpts, engineigo.WithMaxPayload(payload))
	}
}

func WithLogger(logger *zap.SugaredLogger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

type Server struct {
	engine      *engineigo.Server
	engineOpts  []engineigo.ServerOption
	adapterInit AdapterIniter
	nsps        map[string]*Namespace
	parser      Parser

	logger *zap.SugaredLogger

	closed chan struct{}
}

// TODO refactor constructor
func NewServer(opts ...ServerOption) *Server {
	srv := &Server{adapterInit: NewInMemoryAdapterIniter(),
		nsps:   make(map[string]*Namespace),
		parser: DefaultParser,
	}

	for _, o := range opts {
		o(srv)
	}

	if srv.logger == nil {
		logger, err := zap.NewProduction()
		if err != nil {
			panic(err)
		}
		srv.logger = logger.Sugar()
	}

	srv.engineOpts = append(srv.engineOpts, engineigo.WithLogger(srv.logger))
	srv.engine = engineigo.NewServer(srv.engineOpts...)

	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.engine.ServeHTTP(w, r)
}

func (s *Server) Accept() {
	for {
		select {
		case <-s.closed:
			return
		case e := <-s.engine.Accept():
			s.logger.Info("Engine.IO connection received")
			conn := &Connection{
				session:   e,
				server:    s,
				parser:    s.parser,
				socketIds: make(map[string]*Socket),
				logger:    s.logger.With("Connection", e.ID()),
			}

			// Init
			go func() {
				_, _, r, err := conn.session.NextReader()
				if err != nil {
					s.logger.Error("conn.session.NextReader(): ", err)
					return
				}
				bs, err := io.ReadAll(r)
				if err != nil {
					s.logger.Error("io.ReadAll: ", err)
					return
				}
				r.Close()

				packet, err := conn.parser.Decode(bs)
				if err != nil {
					s.logger.Error("parser.Decode error: ", err)
					conn.Close()
					return
				}
				if packet.Type != PacketConnect {
					s.logger.Errorf("first packet is %v, not connect", packet.Type)
					conn.Close()
					return
				}

				nsp, ok := conn.server.nsps[packet.Namespace]
				if !ok {
					conn.ConnectError(packet.Namespace, ErrInvalidNamespace)
					conn.Close()
					return
				}

				conn.Connect(nsp, packet.DataBytes)
				go conn.Start()

			}()
		}
	}
}

type errMsg struct {
	Message string `json:"message"`
}

var ErrInvalidNamespace errMsg = errMsg{
	Message: "Invalid namespace",
}

type connReply struct {
	Sid string `json:"sid"`
}

func (s *Server) Close() {
	close(s.closed)
}

func (s *Server) Of(name string) *Namespace {
	nsp, ok := s.nsps[name]
	if ok {
		return nsp
	}

	nsp = NewNamespace(s, name)
	s.nsps[name] = nsp

	return nsp
}
