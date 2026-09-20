// Package events tells the browser that something changed, so it can refresh what it shows without polling.
package events

import "sync"

// Kinds of change. The event says what changed, never the new values: the client asks for those the normal way,
// so nothing sensitive travels on the stream and the normal access checks still apply.
const (
	KindApps  = "apps"
	KindJobs  = "jobs"
	KindNodes = "nodes"
)

// Event is one change. ID is the app, job or node that changed.
type Event struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

const subscriberBuffer = 64

// Bus fans events out to everyone listening. It never blocks the code that publishes: a listener that cannot
// keep up misses events, and the client makes up for that by refreshing when it reconnects and on a slow timer.
type Bus struct {
	mu   sync.Mutex
	subs map[chan Event]struct{}
}

func NewBus() *Bus {
	return &Bus{subs: make(map[chan Event]struct{})}
}

// Subscribe returns a channel of events and a function that stops listening.
func (b *Bus) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, subscriberBuffer)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.subs, ch)
		b.mu.Unlock()
	}
}

// Publish sends an event to every listener that has room for it.
func (b *Bus) Publish(e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs {
		select {
		case ch <- e:
		default:
		}
	}
}
