package socketio

// BroadcastOperator provides a fluent API for broadcasting events to rooms
// within a namespace, with support for excluding specific sockets.
// Equivalent to NestJS Socket.IO's server.to("room").emit() pattern.
//
// Usage:
//
//	// Broadcast to a room
//	ns.To("room1").Emit("event", data)
//
//	// Broadcast to multiple rooms
//	ns.To("room1", "room2").Emit("event", data)
//
//	// Broadcast to a room, excluding certain sockets
//	ns.To("room1").Except("socketID").Emit("event", data)
//
//	// From a socket: send to room excluding self
//	socket.To("room1").Emit("event", data)
//
//	// From a socket: broadcast to all except self
//	socket.Broadcast().Emit("event", data)
type BroadcastOperator struct {
	namespace *Namespace
	rooms     map[string]bool // target rooms (nil = all sockets in namespace)
	except    map[string]bool // excluded socket IDs
}

// To adds target rooms to the broadcast.
func (b *BroadcastOperator) To(rooms ...string) *BroadcastOperator {
	if b.rooms == nil {
		b.rooms = make(map[string]bool)
	}
	for _, r := range rooms {
		b.rooms[r] = true
	}
	return b
}

// In is an alias for To.
func (b *BroadcastOperator) In(rooms ...string) *BroadcastOperator {
	return b.To(rooms...)
}

// Except excludes specific socket IDs from the broadcast.
func (b *BroadcastOperator) Except(ids ...string) *BroadcastOperator {
	for _, id := range ids {
		b.except[id] = true
	}
	return b
}

// Emit sends an event to all matching sockets.
func (b *BroadcastOperator) Emit(event string, data any) {
	targets := b.getTargets()
	for _, sock := range targets {
		_ = sock.Emit(event, data)
	}
}

// EmitToAll sends an event to all matching sockets and returns the count.
func (b *BroadcastOperator) EmitToAll(event string, data any) int {
	targets := b.getTargets()
	count := 0
	for _, sock := range targets {
		if err := sock.Emit(event, data); err == nil {
			count++
		}
	}
	return count
}

// SocketCount returns the number of sockets that would receive the broadcast.
func (b *BroadcastOperator) SocketCount() int {
	return len(b.getTargets())
}

// getTargets resolves the set of sockets matching the room/except filters.
func (b *BroadcastOperator) getTargets() []*Socket {
	b.namespace.mu.RLock()
	defer b.namespace.mu.RUnlock()

	// If no specific rooms, target all sockets in the namespace
	if b.rooms == nil || len(b.rooms) == 0 {
		targets := make([]*Socket, 0, len(b.namespace.sockets))
		for _, sock := range b.namespace.sockets {
			if !b.except[sock.Client.ID] {
				targets = append(targets, sock)
			}
		}
		return targets
	}

	// Collect unique sockets from targeted rooms
	seen := make(map[string]bool)
	var targets []*Socket
	for room := range b.rooms {
		sockets, ok := b.namespace.rooms[room]
		if !ok {
			continue
		}
		for id, sock := range sockets {
			if seen[id] || b.except[id] {
				continue
			}
			seen[id] = true
			targets = append(targets, sock)
		}
	}
	return targets
}
