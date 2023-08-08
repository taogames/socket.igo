package socketigo

import (
	"encoding/json"

	engineigo "github.com/taogames/engine.igo"
	"github.com/taogames/engine.igo/message"
	"go.uber.org/zap"
)

type Connection struct {
	server    *Server
	session   *engineigo.Session
	parser    Parser
	socketIds map[string]*Socket // map<Namespace, socketId>

	logger *zap.SugaredLogger
}

func (conn *Connection) Connect(nsp *Namespace, handshake []byte) {
	sid := conn.session.ID()
	rData := connReply{
		Sid: sid,
	}
	rPacket := &Packet{
		Type:      PacketConnect,
		Namespace: nsp.Name(),
		Data:      rData,
	}
	msgs, err := conn.parser.Encode(rPacket)
	if err != nil {
		conn.logger.Error("conn.parser.Encode: ", err)
		return
	}
	conn.WriteToEngine(msgs)

	nsp.Connect(sid, conn, handshake)
}

func (conn *Connection) WriteToEngine(msgs []*message.Message) error {
	for _, msg := range msgs {
		if err := conn.session.WriteMessage(msg); err != nil {
			return err
		}
	}
	return nil
}

func (conn *Connection) ConnectError(namespace string, errMsg interface{}) {
	rPacket := &Packet{
		Type:      PacketConnectError,
		Namespace: namespace,
		Data:      errMsg,
	}

	msgs, err := conn.parser.Encode(rPacket)
	if err != nil {
		conn.logger.Error("conn.parser.Encode", err)
		return
	}
	conn.WriteToEngine(msgs)
}

func (conn *Connection) Start() {
	for {
		mt, bs, err := conn.session.ReadMessage()
		if err != nil {
			conn.logger.Error("conn.session.NextReader:", err)

			for _, socket := range conn.socketIds {
				socket.disconnect(true, DRTransportError)
			}

			conn.Close()
			return
		}

		conn.onPacket(mt, bs)
	}
}

func (conn *Connection) onPacket(mt message.MessageType, data []byte) {
	packet, err := conn.parser.Decode(&message.Message{Type: mt, Data: data})
	if err != nil {
		conn.logger.Error("conn.parser.Decode:", err)
		conn.Close()
		return
	}
	if packet == nil {
		// Binary payload concatenating
		return
	}

	nsp, ok := conn.server.nsps[packet.Namespace]
	if !ok {
		conn.ConnectError(packet.Namespace, ErrInvalidNamespace)
		conn.Close()
	}

	switch packet.Type {
	case PacketConnect:
		handshake, _ := json.Marshal(packet.Data)
		conn.Connect(nsp, handshake)
	case PacketDisconnect:
		socket := conn.socketIds[packet.Namespace]
		socket.disconnect(false, DRClientNamespaceDisconnect)
	case PacketEvent, PacketBinaryEvent:
		socket := conn.socketIds[packet.Namespace]
		socket.dispatch(packet)
	default:
		// Not supported
	}

}

func (conn *Connection) Close() {
	conn.session.Close()
}
