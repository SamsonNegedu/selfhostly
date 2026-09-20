// Package nodelink carries a primary's HTTP requests to a secondary over a connection the
// secondary opened outbound.
//
// The secondary dials the primary's WebSocket endpoint and then serves HTTP/2 on that connection,
// while the primary is the HTTP/2 client. The primary's node client keeps speaking ordinary HTTP;
// only the connection underneath changes, so the secondary needs no inbound port and its API can
// listen on loopback.
package nodelink

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/net/http2"
)

// Scheme is the URL scheme of a node reached over a link: tunnel://<node-id>/path
const Scheme = "tunnel"

// ErrNodeNotConnected is returned for a request to a node with no live link
var ErrNodeNotConnected = errors.New("node is not connected")

// Keepalive settings for the HTTP/2 connection. Cloudflare closes idle WebSockets after about 100
// seconds, and a half-dead link should be noticed within a minute.
const (
	readIdleTimeout = 30 * time.Second
	pingTimeout     = 15 * time.Second
)

type session struct {
	cc   *http2.ClientConn
	conn *watchConn
}

// Registry holds the live link of every connected node. It lives on the primary.
type Registry struct {
	mu       sync.RWMutex
	sessions map[string]*session
	onChange func(nodeID string, connected bool)
}

// NewRegistry creates an empty registry. onChange, if set, is called when a node's link comes up or
// goes down (a replaced link reports down then up).
func NewRegistry(onChange func(nodeID string, connected bool)) *Registry {
	return &Registry{sessions: make(map[string]*session), onChange: onChange}
}

// Connected reports whether the node has a live link
func (r *Registry) Connected(nodeID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.sessions[nodeID]
	return ok
}

// Disconnect drops a node's link, if any
func (r *Registry) Disconnect(nodeID string) {
	r.mu.RLock()
	s := r.sessions[nodeID]
	r.mu.RUnlock()
	if s != nil {
		_ = s.conn.Close()
	}
}

// Serve upgrades the request to a link for nodeID and blocks until that link ends. The caller must
// already have authenticated the node. A node has one link: a new connection replaces the old one.
func (r *Registry) Serve(w http.ResponseWriter, req *http.Request, nodeID string) error {
	ws, err := websocket.Accept(w, req, nil)
	if err != nil {
		return fmt.Errorf("accept websocket: %w", err)
	}
	ws.SetReadLimit(-1)

	conn := newWatchConn(websocket.NetConn(req.Context(), ws, websocket.MessageBinary))
	t := &http2.Transport{ReadIdleTimeout: readIdleTimeout, PingTimeout: pingTimeout}
	cc, err := t.NewClientConn(conn)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("start http2 client: %w", err)
	}

	sess := &session{cc: cc, conn: conn}
	r.mu.Lock()
	old := r.sessions[nodeID]
	r.sessions[nodeID] = sess
	r.mu.Unlock()
	if old != nil {
		_ = old.conn.Close()
		r.notify(nodeID, false)
	}
	r.notify(nodeID, true)

	select {
	case <-conn.done:
	case <-req.Context().Done():
	}
	_ = conn.Close()
	_ = cc.Close()

	r.mu.Lock()
	if r.sessions[nodeID] == sess {
		delete(r.sessions, nodeID)
		r.mu.Unlock()
		r.notify(nodeID, false)
	} else {
		r.mu.Unlock() // replaced by a newer link, which already reported itself
	}
	return nil
}

func (r *Registry) notify(nodeID string, connected bool) {
	if r.onChange != nil {
		r.onChange(nodeID, connected)
	}
}

// RoundTripper sends requests for tunnel://<node-id>/... down that node's link. Register it for the
// scheme on the HTTP client the node client uses.
func (r *Registry) RoundTripper() http.RoundTripper { return registryTransport{r} }

type registryTransport struct{ r *Registry }

func (t registryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	nodeID := req.URL.Hostname()
	t.r.mu.RLock()
	s := t.r.sessions[nodeID]
	t.r.mu.RUnlock()
	if s == nil {
		return nil, fmt.Errorf("%w: %s", ErrNodeNotConnected, nodeID)
	}
	out := req.Clone(req.Context())
	out.URL.Scheme = "http" // the link itself is the transport; this only labels the HTTP/2 :scheme
	out.URL.Host = nodeID
	out.Host = ""
	resp, err := s.cc.RoundTrip(out)
	if err != nil {
		if !s.cc.CanTakeNewRequest() {
			return nil, fmt.Errorf("%w: %s: %v", ErrNodeNotConnected, nodeID, err)
		}
		return nil, err
	}
	return resp, nil
}

// watchConn signals when the underlying connection ends, whichever side ends it
type watchConn struct {
	net.Conn
	once sync.Once
	done chan struct{}
}

func newWatchConn(c net.Conn) *watchConn { return &watchConn{Conn: c, done: make(chan struct{})} }

func (w *watchConn) finish() { w.once.Do(func() { close(w.done) }) }

func (w *watchConn) Read(p []byte) (int, error) {
	n, err := w.Conn.Read(p)
	if err != nil {
		w.finish()
	}
	return n, err
}

func (w *watchConn) Write(p []byte) (int, error) {
	n, err := w.Conn.Write(p)
	if err != nil {
		w.finish()
	}
	return n, err
}

func (w *watchConn) Close() error {
	w.finish()
	return w.Conn.Close()
}
