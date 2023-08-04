package socketigo

import (
	"encoding/json"
	"fmt"
	"io"

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
		fmt.Println("WriteToEngine: ", msg.Type, string(msg.Data))
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
		mt, _, r, err := conn.session.NextReader()
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

		packet, err := conn.parser.Decode(&message.Message{Type: mt, Data: bs})
		if err != nil {
			conn.logger.Error("conn.parser.Decode:", err)
			conn.Close()
			return
		}
		if packet == nil {
			// Binary payload concatenating
			continue
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
		default:
			nsp.Dispatch(conn.socketIds[packet.Namespace].Id, packet)
		}
	}
}

func (conn *Connection) Close() {
	conn.session.Close()
}
