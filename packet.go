package socketigo

import (
	"fmt"
	"reflect"
)

type Packet struct {
	Type             PacketType
	Namespace        string
	Data             any
	DataKind         reflect.Kind
	Id               int
	NumOfAttachments int
}

type PacketType int

const (
	PacketConnect PacketType = iota
	PacketDisconnect
	PacketEvent
	PacketAck
	PacketConnectError
	PacketBinaryEvent
	PacketBinaryAck
)

func (pt PacketType) Byte() byte {
	return byte(pt) + '0'
}

func ParsePacketType(b byte) (PacketType, error) {
	pt := PacketType(b - '0')
	if pt < PacketConnect || pt > PacketBinaryAck {
		return 0, fmt.Errorf("socket packet type invalid: %c", b)
	}
	return pt, nil
}

type DisconnectReason string

const (
	DRUnknown                   DisconnectReason = "to be replaced"
	DRServerNamespaceDisconnect DisconnectReason = "server namespace disconnect"
	DRClientNamespaceDisconnect DisconnectReason = "client namespace disconnect"

	DRTransportClose DisconnectReason = "transport close"
	DRTransportError DisconnectReason = "transport error"
)
