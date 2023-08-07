package socketigo

import (
	"reflect"
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
	s.disconnect(closeConn, DRServerNamespaceDisconnect)
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

func (s *Socket) disconnect(closeConn bool, reason DisconnectReason) {
	s.nsp.Remove(s.Id)
	delete(s.conn.socketIds, s.Id)

	if closeConn {
		s.conn.Close()
	}

	if s.onDisconnect == nil {
		return
	}
	s.onDisconnect(reason)
}

func (s *Socket) dispatch(packet *Packet) {
	s.logger.Debug("dispatch", packet)

	name, err := s.conn.parser.ParseEventName(packet)
	if err != nil {
		s.logger.Errorf("ParseEventName %v: %v", packet, err)
		return
	}
	h := s.eh.GetHandler(name)

	if h == nil {
		return
	}

	args, err := s.conn.parser.ParseEventArgs(packet, h.types, h.f.Type().IsVariadic())
	if err != nil {
		s.logger.Errorf("ParseEventArgs %v: %v", packet, err)
		return
	}

	if packet.Id != nil {
		ack := func(args ...interface{}) {
			ackPacket := &Packet{
				Type:      PacketAck,
				Namespace: packet.Namespace,
				Data:      args,
				Id:        packet.Id,
			}

			s.logger.Debugf("Acking packet %v: %v", *ackPacket.Id, ackPacket)
			msgs, err := s.conn.parser.Encode(ackPacket)
			if err != nil {
				s.logger.Error("s.conn.parser.Encode: ", err)
				return
			}
			s.conn.WriteToEngine(msgs)
		}
		args = append(args, reflect.ValueOf(ack))
	}

	h.f.Call(args)
}
