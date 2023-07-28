package socketigo

import (
	"io"

	engineigo "engine.igo/v4"
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
	bs, err := conn.parser.Encode(rPacket)
	if err != nil {
		conn.logger.Error("conn.parser.Encode: ", err)
		return
	}
	conn.session.WriteMessage(bs)

	nsp.Connect(sid, conn, handshake)
}

func (conn *Connection) WriteToEngine(bs []byte) error {

	return conn.session.WriteMessage(bs)
}

func (conn *Connection) ConnectError(namespace string, errMsg interface{}) {
	rPacket := &Packet{
		Type:      PacketConnectError,
		Namespace: namespace,
		Data:      errMsg,
	}

	bs, err := conn.parser.Encode(rPacket)
	if err != nil {
		conn.logger.Error("conn.parser.Encode", err)
		return
	}
	conn.session.WriteMessage(bs)
}

func (conn *Connection) Start() {
	for {
		_, _, r, err := conn.session.NextReader()
		if err != nil {
			conn.logger.Error("conn.session.NextReader:", err)

			for _, socket := range conn.socketIds {
				socket._disconnect(true, DRTransportError)
			}

			conn.Close()
			return
		}

		bs, err := io.ReadAll(r)
		if err != nil {
			conn.logger.Error("io.ReadAll: ", err)
			return
		}
		r.Close()

		packet, err := conn.parser.Decode(bs)
		if err != nil {
			conn.logger.Error("conn.parser.Decode:", err)
			conn.Close()
			return
		}
		nsp, ok := conn.server.nsps[packet.Namespace]
		if !ok {
			conn.ConnectError(packet.Namespace, ErrInvalidNamespace)
			conn.Close()
		}

		switch packet.Type {
		case PacketConnect:
			conn.Connect(nsp, packet.DataBytes)
		default:
			nsp.Dispatch(conn.socketIds[packet.Namespace].Id, packet)
		}
	}
}

func (conn *Connection) Close() {
	conn.session.Close()
}
