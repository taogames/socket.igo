package socketigo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/taogames/engine.igo/message"
)

type Parser interface {
	Decode(message.MessageType, []byte) (*Packet, error)
	Encode(*Packet) ([]byte, error)

	ParseEventName(*Packet) (string, error)
	ParseEventArgs(*Packet, []reflect.Type, bool) ([]reflect.Value, error)
}

var DefaultParser *defaultParser = &defaultParser{
	recon: &reconstructor{},
}

type defaultParser struct {
	recon *reconstructor
}

type reconstructor struct {
	packet  *Packet
	buffers [][]byte
}

func (recon *reconstructor) reset(packet *Packet) {
	recon.packet = packet
	recon.buffers = nil
}

func (recon *reconstructor) takeBinary(data []byte) (bool, *Packet) {
	recon.buffers = append(recon.buffers, data)
	if len(recon.buffers) == recon.packet.NumOfAttachments {
		return true, recon.build()
	}
	return false, nil
}

func (recon *reconstructor) build() *Packet {
	data := recon.packet.Data.([]interface{})

	var bufIdx int
	for placeIdx := range data {
		if bufIdx >= len(recon.buffers) {
			break
		}

		m, ok := data[placeIdx].(map[string]interface{})
		if ok && m["_placeholder"] != nil {
			data[placeIdx] = recon.buffers[bufIdx]
			bufIdx++
		}
	}

	return recon.packet
}

func (p *defaultParser) Decode(mt message.MessageType, bs []byte) (*Packet, error) {
	switch mt {
	case message.MTText:
		packet, err := p.decodeString(bs)
		if err != nil {
			return nil, err
		}
		switch packet.Type {
		case PacketBinaryEvent, PacketBinaryAck:
			if packet.NumOfAttachments == 0 {
				return packet, nil
			} else {
				p.recon.reset(packet)
				return nil, nil
			}
		default:
			return packet, nil
		}

	case message.MTBinary:
		isFull, packet := p.recon.takeBinary(bs)
		if isFull {
			return packet, nil
		}
		return nil, nil

	default:
		return nil, errors.New("invalid message type")
	}
}

func (p *defaultParser) decodeString(bs []byte) (*Packet, error) {
	i := 0
	packet := &Packet{}

	// Packet type
	if i == len(bs) {
		return nil, fmt.Errorf("empty packet %v", string(bs))
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
				return nil, fmt.Errorf("empty binary packet %v", string(bs))
			}
			if bs[i] == '-' {
				n, err := strconv.Atoi(string(bs[begin:i]))
				if err != nil {
					return nil, err
				}
				packet.NumOfAttachments = n
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
				return nil, fmt.Errorf("invalid packet id %v", string(bs))
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

	if len(bs[i:]) > 0 {
		var payload any
		dec := json.NewDecoder(bytes.NewReader(bs[i:]))
		dec.UseNumber()
		if err := dec.Decode(&payload); err != nil {
			return nil, err
		}

		packet.Data = payload
		packet.DataKind = reflect.ValueOf(payload).Kind()

		if !p.isPayloadValid(packet) {
			return nil, fmt.Errorf("invalid packet payload %v", string(bs))
		}
	}

	return packet, nil
}

func (p *defaultParser) isPayloadValid(packet *Packet) bool {
	switch packet.Type {
	case PacketConnect:
		return packet.DataKind == reflect.Map
	case PacketDisconnect:
		return false
	case PacketConnectError:
		return packet.DataKind == reflect.Map || packet.DataKind == reflect.String
	case PacketEvent, PacketBinaryEvent:
		if packet.DataKind == reflect.Slice && reflect.ValueOf(packet.Data).Len() > 0 {
			_, ok := packet.Data.([]interface{})[0].(string)
			if ok {
				return true
			}
		}
		return false
	case PacketAck, PacketBinaryAck:
		return packet.DataKind == reflect.Slice
	default:
		return false
	}
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
		builder.Write([]byte{itob(packet.NumOfAttachments), '-'})
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

func (p *defaultParser) ParseEventName(packet *Packet) (string, error) {
	return packet.Data.([]interface{})[0].(string), nil
}

func (p *defaultParser) ParseEventArgs(packet *Packet, types []reflect.Type, isVariadic bool) ([]reflect.Value, error) {
	data, _ := json.Marshal(packet.Data.([]interface{})[1:])
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
