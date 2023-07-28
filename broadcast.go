package socketigo

type Broadcast struct {
	nsp *Namespace

	includeAll bool
	includes   []string
	excludes   map[string]struct{}
}

func (b *Broadcast) Emit(eName string, args ...interface{}) {
	data := append([]interface{}{eName}, args...)
	packet := &Packet{
		Type:      PacketEvent,
		Namespace: b.nsp.Name(),
		Data:      data,
	}

	b.nsp.adapter.Broadcast(packet, &BroadcastOptions{
		IncludeAll: b.includeAll,
		Includes:   b.includes,
		Excludes:   b.excludes,
	})
}
