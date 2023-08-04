package socketigo

import (
	"sync/atomic"

	"go.uber.org/zap"
)

type Socket struct {
	Id string

	connected atomic.Bool

	Handshake struct {
		Auth map[string]interface{}
	}

	Custom map[string]interface{}

	nsp *Namespace

	conn *Connection
	eh   EventManager

	onDisconnect func(reason DisconnectReason)

	logger *zap.SugaredLogger
}

func (s *Socket) Disconnect(closeConn bool) {
	if !s.connected.CompareAndSwap(true, false) {
		return
	}
	s._disconnect(closeConn, DRServerNamespaceDisconnect)
}

func (s *Socket) _disconnect(closeConn bool, reason DisconnectReason) {

	s.nsp.Disconnect(s.Id)
	delete(s.conn.socketIds, s.Id)

	if closeConn {
		s.conn.Close()
	}

	if s.onDisconnect == nil {
		return
	}
	s.onDisconnect(reason)
}

func (s *Socket) Join(rooms ...string) {
	s.nsp.adapter.Join(s.Id, rooms...)
}
func (s *Socket) Leave(rooms ...string) {
	s.nsp.adapter.Leave(s.Id, rooms...)
}

func (s *Socket) To(rooms ...string) *Broadcast {
	return &Broadcast{
		nsp:      s.nsp,
		includes: rooms,
		excludes: map[string]struct{}{
			s.Id: {},
		},
	}
}

func (s *Socket) Broadcast() *Broadcast {
	return &Broadcast{
		nsp:        s.nsp,
		includeAll: true,
		excludes:   map[string]struct{}{s.Id: {}},
	}
}

func (s *Socket) Emit(eName string, args ...interface{}) {
	s.logger.Debug("Emit %s: %v", eName, args)

	data := append([]interface{}{eName}, args...)

	packet := &Packet{
		Type:      PacketEvent,
		Namespace: s.nsp.Name(),
		Data:      data,
	}

	msgs, err := s.conn.parser.Encode(packet)
	if err != nil {
		s.logger.Error("s.conn.parser.Encode: ", err)
		return
	}

	s.conn.WriteToEngine(msgs)
}

func (s *Socket) On(eName string, h any) {
	s.eh.Register(eName, h)
}

func (s *Socket) OnDisconnect(f func(reason DisconnectReason)) {
	s.onDisconnect = f
}
