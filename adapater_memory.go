package socketigo

import (
	"sync"

	"go.uber.org/zap"
)

type AdapterIniter func(nsp *Namespace) Adapter

type InMemoryAdapter struct {
	sync.RWMutex

	nsp *Namespace

	Sids  map[string]map[string]struct{} // Map<SocketId, Set<Room>>
	Rooms map[string]map[string]struct{} // Map<Room, Set<SocketId>>

	logger *zap.SugaredLogger
}

func NewInMemoryAdapterIniter() AdapterIniter {
	return func(nsp *Namespace) Adapter {
		return &InMemoryAdapter{
			nsp:    nsp,
			Sids:   make(map[string]map[string]struct{}),
			Rooms:  make(map[string]map[string]struct{}),
			logger: nsp.logger.With("Adapter", "InMemory"),
		}
	}
}

func (adp *InMemoryAdapter) Join(sid string, rooms ...string) {
	adp.logger.Debugf("%s Join %v", sid, rooms)

	if _, ok := adp.Sids[sid]; !ok {
		adp.Sids[sid] = make(map[string]struct{})
	}

	for _, room := range rooms {
		adp.Sids[sid][room] = struct{}{}

		if _, ok := adp.Rooms[room]; !ok {
			adp.Rooms[room] = make(map[string]struct{})
		}
		adp.Rooms[room][sid] = struct{}{}
	}
}

func (adp *InMemoryAdapter) Leave(sid string, rooms ...string) {
	adp.logger.Debugf("%s Leave %v", sid, rooms)

	for _, room := range rooms {
		delete(adp.Sids[sid], room)
		delete(adp.Rooms[room], sid)
	}
}

func (adp *InMemoryAdapter) LeaveAll(sid string) {
	adp.logger.Debugf("%s LeaveAll", sid)

	for room := range adp.Sids[sid] {
		delete(adp.Rooms[room], sid)
	}
	delete(adp.Sids, sid)
}

func (adp *InMemoryAdapter) Broadcast(packet *Packet, opts *BroadcastOptions) {
	adp.logger.Debugf("Broadcast %v with opts %v", packet, opts)

	adp.RLock()
	defer adp.RUnlock()

	msgs, err := adp.nsp.parser.Encode(packet)
	if err != nil {
		adp.logger.Errorf("Broadcast packet %v: %v", packet, err)
	}

	sids := make(map[string]interface{})

	if opts.IncludeAll {
		for sid := range adp.Sids {
			if _, ok := opts.Excludes[sid]; ok {
				continue
			}
			sids[sid] = struct{}{}
		}
	} else {
		for _, room := range opts.Includes {
			for sid := range adp.Rooms[room] {
				if _, ok := opts.Excludes[sid]; ok {
					continue
				}
				sids[sid] = struct{}{}
			}
		}
	}

	for sid := range sids {
		if err := adp.nsp.sockets[sid].conn.WriteToEngine(msgs); err != nil {
			adp.logger.Errorf("Broadcast sid=%v WriteToEngine: %v", sid, err)
		}
	}

}
