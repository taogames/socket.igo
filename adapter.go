package socketigo

type Adapter interface {
	Join(sid string, rooms ...string)
	Leave(sid string, rooms ...string)
	LeaveAll(sid string)

	Broadcast(packet *Packet, opts *BroadcastOptions)
}

type BroadcastOptions struct {
	IncludeAll bool
	Includes   []string
	Excludes   map[string]struct{}
}
