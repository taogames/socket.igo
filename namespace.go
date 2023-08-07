package socketigo

import (
	"encoding/json"
	"sync"

	"go.uber.org/zap"
)

const MainNamespace = "/"

type Namespace struct {
	name    string
	parser  Parser
	adapter Adapter

	onConnection SocketFunction

	sync.RWMutex
	sockets map[string]*Socket

	logger *zap.SugaredLogger
}

type SocketFunction func(*Socket)

func NewNamespace(s *Server, name string) *Namespace {
	nsp := &Namespace{
		name:    name,
		parser:  s.parser,
		sockets: make(map[string]*Socket),
		logger:  s.logger.With("Namespace", name),
	}
	nsp.adapter = s.adapterInit(nsp)

	return nsp
}

func (nsp *Namespace) OnConnection(f SocketFunction) {
	nsp.onConnection = f
}

func (nsp *Namespace) Name() string {
	return nsp.name
}

func (nsp *Namespace) Connect(sid string, conn *Connection, handshake []byte) {
	socket := &Socket{
		Id:   sid,
		conn: conn,
		nsp:  nsp,
		eh: EventManager{
			m: make(map[string]*handler),
		},
		Custom: make(map[string]interface{}),
		logger: nsp.logger.With("Socket", sid),
	}
	socket.connected.Store(true)
	socket.Handshake.Auth = make(map[string]interface{})
	if len(handshake) > 0 {
		if err := json.Unmarshal([]byte(handshake), &socket.Handshake.Auth); err != nil {
			nsp.logger.Error(err)
		}
	}

	conn.socketIds[nsp.name] = socket
	nsp.Lock()
	nsp.sockets[socket.Id] = socket
	nsp.Unlock()

	if nsp.onConnection != nil {
		nsp.onConnection(socket)
	}
	nsp.adapter.Join(socket.Id, socket.Id)
}

func (nsp *Namespace) Remove(sid string) {
	nsp.Lock()
	defer nsp.Unlock()
	delete(nsp.sockets, sid)
	nsp.adapter.LeaveAll(sid)
}
