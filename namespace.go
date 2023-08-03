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

func (nsp *Namespace) Disconnect(sid string) {
	nsp.Lock()
	delete(nsp.sockets, sid)
	nsp.adapter.LeaveAll(sid)
	nsp.Unlock()
}

// TODO: add packet type: Ack & ConnectError
func (nsp *Namespace) Dispatch(sid string, packet *Packet) {
	nsp.logger.Debug("Dispatch", packet)

	socket, ok := nsp.sockets[sid]
	if !ok {
		nsp.logger.Errorf("sid=%v not exist", sid)
		return
	}

	switch packet.Type {
	case PacketDisconnect:
		nsp.Disconnect(sid)
	case PacketEvent, PacketBinaryEvent:
		name, err := socket.conn.parser.ParseEventName(packet)
		if err != nil {
			nsp.logger.Errorf("ParseEventName %v: %v", packet, err)
			return
		}
		h := socket.eh.GetHandler(name)

		if h == nil {
			return
		}

		args, err := socket.conn.parser.ParseEventArgs(packet, h.types, h.f.Type().IsVariadic())
		if err != nil {
			nsp.logger.Errorf("ParseEventArgs %v: %v", packet, err)
			return
		}
		h.f.Call(args)
	}
}
