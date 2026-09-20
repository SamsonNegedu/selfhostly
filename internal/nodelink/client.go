package nodelink

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/net/http2"
)

// Defaults for Client. Cloudflare closes an idle WebSocket after about 100 seconds, so the link is
// pinged well inside that; a ping that goes unanswered means the primary is gone.
const (
	defaultMinBackoff = time.Second
	defaultMaxBackoff = time.Minute
	defaultPingEvery  = 25 * time.Second
	pingReplyTimeout  = 10 * time.Second

	// a link that stayed up this many times the longest backoff counts as healthy: the next drop
	// starts the backoff over instead of continuing to grow it
	stableLinkFactor = 2
)

// Client is the secondary's side: it keeps one outbound link to the primary and serves the primary's
// requests on it with Handler, reconnecting with backoff whenever the link drops.
type Client struct {
	// URL of the primary's link endpoint (http or https; the WebSocket upgrade is done for you)
	URL string
	// Header carries the node's credentials
	Header http.Header
	// Handler answers the requests the primary sends over the link
	Handler http.Handler
	// HTTPClient is used to dial (optional)
	HTTPClient *http.Client
	Logger     *slog.Logger

	MinBackoff time.Duration // default defaultMinBackoff
	MaxBackoff time.Duration // default defaultMaxBackoff
	// PingEvery is how often the client checks the link is alive (default defaultPingEvery)
	PingEvery time.Duration

	// OnState, if set, is called when the link comes up (true) or drops (false)
	OnState func(connected bool)
}

// Run keeps the link up until ctx is cancelled. A rejected connection (wrong credentials) is retried
// like any other failure, so a node whose token is fixed later recovers without a restart.
func (c *Client) Run(ctx context.Context) {
	minB, maxB := c.MinBackoff, c.MaxBackoff
	if minB <= 0 {
		minB = defaultMinBackoff
	}
	if maxB <= 0 {
		maxB = defaultMaxBackoff
	}
	log := c.Logger
	if log == nil {
		log = slog.Default()
	}
	backoff := minB
	for ctx.Err() == nil {
		started := time.Now()
		err := c.once(ctx, log)
		if ctx.Err() != nil {
			return
		}
		// A link that stayed up for a while starts the backoff over
		if time.Since(started) > stableLinkFactor*maxB {
			backoff = minB
		}
		wait := backoff/2 + time.Duration(rand.Int63n(int64(backoff)/2+1))
		log.Warn("node link down, reconnecting", "error", err, "retry_in", wait.Round(time.Millisecond))
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		if backoff *= 2; backoff > maxB {
			backoff = maxB
		}
	}
}

func (c *Client) once(ctx context.Context, log *slog.Logger) error {
	ws, resp, err := websocket.Dial(ctx, c.URL, &websocket.DialOptions{HTTPHeader: c.Header, HTTPClient: c.HTTPClient})
	if err != nil {
		if resp != nil {
			return fmt.Errorf("the primary refused the link (%s): %w", resp.Status, err)
		}
		return err
	}
	ws.SetReadLimit(-1)

	linkCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	conn := newWatchConn(websocket.NetConn(linkCtx, ws, websocket.MessageBinary))

	// Notice a dead primary and keep idle intermediaries (Cloudflare) from closing the link
	every := c.PingEvery
	if every <= 0 {
		every = defaultPingEvery
	}
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-linkCtx.Done():
				return
			case <-conn.done:
				return
			case <-t.C:
				pctx, pcancel := context.WithTimeout(linkCtx, pingReplyTimeout)
				err := ws.Ping(pctx)
				pcancel()
				if err != nil {
					_ = conn.Close()
					return
				}
			}
		}
	}()

	if c.OnState != nil {
		c.OnState(true)
		defer c.OnState(false)
	}
	log.Info("node link up")

	srv := &http2.Server{}
	srv.ServeConn(conn, &http2.ServeConnOpts{Context: linkCtx, Handler: c.Handler})
	_ = conn.Close()
	if err := linkCtx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return errors.New("the primary closed the link")
}
