package socketigo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Parser interface {
	Decode([]byte) (*Packet, error)
	Encode(*Packet) ([]byte, error)

	ParseEventName([]byte) (string, []byte, error)
	ParseEventArgs([]byte, []reflect.Type, bool) ([]reflect.Value, error)
}

var DefaultParser *defaultParser = &defaultParser{}

type defaultParser struct {
}

func (p *defaultParser) Decode(bs []byte) (*Packet, error) {
	i := 0
	packet := &Packet{}

	// Packet type
	if i == len(bs) {
		return nil, fmt.Errorf("0 invalid packet %v", string(bs))
	}
	pt, err := ParsePacketType(bs[0])
	if err != nil {
		return nil, err
	}
	packet.Type = pt
	i++

	// Num of attchments
	if pt == PacketBinaryEvent || pt == PacketBinaryAck {
		begin := i
		for {
			i++
			if i == len(bs) {
				return nil, fmt.Errorf("1 invalid packet %v", string(bs))
			}
			if bs[i] == '-' {
				att, err := strconv.Atoi(string(bs[begin:i]))
				if err != nil {
					return nil, err
				}
				packet.Attachments = att
				break
			}
		}
		i++
	}

	// Namespace
	if i < len(bs) && bs[i] == '/' {
		begin := i
		for {
			i++
			if i == len(bs) {
				packet.Namespace = string(bs[begin:i])
				break
			}
			if bs[i] == ',' {
				packet.Namespace = string(bs[begin:i])
				i++
				break
			}
		}
	} else {
		packet.Namespace = MainNamespace
	}

	// Id
	if i < len(bs) && isDigit(bs[i]) {
		begin := i
		for {
			i++
			if i == len(bs) {
				return nil, fmt.Errorf("3 invalid packet %v", string(bs))
			}
			if !isDigit(bs[i]) {
				id, err := strconv.Atoi(string(bs[begin:i]))
				if err != nil {
					return nil, err
				}
				packet.Id = id
				break
			}
		}
	}

	// Data
	packet.DataBytes = bs[i:]

	if !isPayloadValid(packet.Type, packet.DataBytes) {
		return nil, fmt.Errorf("4 invalid packet %v", string(bs))
	}

	return packet, nil
}

func isPayloadValid(pt PacketType, payload []byte) bool {
	switch pt {
	case PacketConnect:
		return len(payload) == 0 || isJsonObject(payload)
	case PacketDisconnect:
		return len(payload) == 0
	case PacketConnectError:
		return !isJsonArray(payload)
	case PacketEvent, PacketBinaryEvent:
		return isJsonArray(payload)
	case PacketAck, PacketBinaryAck:
		return isJsonArray(payload)
	default:
		return false
	}
}

func isJsonArray(bs []byte) bool {
	return len(bs) >= 2 && bs[0] == '[' && bs[len(bs)-1] == ']'
}
func isJsonObject(bs []byte) bool {
	return len(bs) >= 2 && bs[0] == '{' && bs[len(bs)-1] == '}'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func itob(i int) byte {
	return '0' + byte(i)
}

func (p *defaultParser) Encode(packet *Packet) ([]byte, error) {
	var builder strings.Builder

	// Type
	builder.WriteByte(itob(int(packet.Type)))

	// Bin
	if packet.Type == PacketBinaryEvent || packet.Type == PacketBinaryAck {
		builder.Write([]byte{itob(packet.Attachments), '-'})
	}

	// Nsp
	if packet.Namespace != "/" {
		builder.WriteString(packet.Namespace)
		builder.WriteByte(',')
	}

	// Ack
	if packet.Id != 0 {
		builder.WriteByte(itob(packet.Id))
	}

	// Data
	bs, err := json.Marshal(packet.Data)
	if err != nil {
		return nil, err
	}
	builder.Write(bs)

	return []byte(builder.String()), nil
}

func (p *defaultParser) ParseEventName(data []byte) (string, []byte, error) {
	var decoded []interface{}

	if err := json.Unmarshal(data, &decoded); err != nil {
		return "", nil, err
	}

	if len(decoded) == 0 {
		return "", nil, fmt.Errorf("invalid event data: %v", string(data))
	}

	name, ok := decoded[0].(string)
	if !ok {
		return "", nil, fmt.Errorf("invalid event data: %v", string(data))
	}

	bs, err := json.Marshal(decoded[1:])

	return name, bs, err
}

func (p *defaultParser) ParseEventArgs(data []byte, types []reflect.Type, isVariadic bool) ([]reflect.Value, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	_, err := dec.Token()
	if err != nil {
		return nil, err
	}

	pointers := make([]reflect.Value, 0)
	for i := 0; dec.More(); i++ {
		var t reflect.Type
		if isVariadic && i >= len(types)-1 {
			t = types[len(types)-1].Elem()
		} else {
			if i >= len(types) {
				return nil, fmt.Errorf("invalid event args")
			}
			t = types[i]
		}

		p := reflect.New(t)
		pointers = append(pointers, p)

		recv := p.Interface()

		if err := dec.Decode(&recv); err != nil {
			return nil, err
		}
	}

	args := make([]reflect.Value, len(pointers))
	for i := range pointers {
		args[i] = pointers[i].Elem()
	}

	return args, nil
}
