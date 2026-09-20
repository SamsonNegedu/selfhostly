package nodelink

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// primaryFor serves the link endpoint the way the real primary will: authenticate, then hand over
type primaryFor struct {
	*httptest.Server
	Reg      *Registry
	attempts atomic.Int32
	mu       sync.Mutex
	events   []string
}

func newPrimary(t *testing.T, key string) *primaryFor {
	t.Helper()
	p := &primaryFor{}
	p.Reg = NewRegistry(func(id string, up bool) {
		p.mu.Lock()
		defer p.mu.Unlock()
		if up {
			p.events = append(p.events, id+":up")
		} else {
			p.events = append(p.events, id+":down")
		}
	})
	p.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.attempts.Add(1)
		if r.Header.Get("X-Key") != key {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		_ = p.Reg.Serve(w, r, r.Header.Get("X-Node"))
	}))
	t.Cleanup(p.Close)
	return p
}

func (p *primaryFor) eventLog() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return strings.Join(p.events, ",")
}

func secondaryHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hi from " + r.Header.Get("X-Who")))
	})
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(w, r.Body)
	})
	release := make(chan struct{})
	mux.HandleFunc("/release", func(w http.ResponseWriter, _ *http.Request) { close(release) })
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: first\n\n"))
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
		}
		_, _ = w.Write([]byte("data: last\n\n"))
	})
	return mux
}

func startSecondary(t *testing.T, primaryURL, node, key string) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		URL:        primaryURL,
		Header:     http.Header{"X-Node": {node}, "X-Key": {key}},
		Handler:    secondaryHandler(),
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		MinBackoff: 20 * time.Millisecond,
		MaxBackoff: 100 * time.Millisecond,
		PingEvery:  200 * time.Millisecond,
	}
	go c.Run(ctx)
	t.Cleanup(cancel)
	return cancel
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func nodeClient(reg *Registry) *http.Client {
	tr := &http.Transport{}
	tr.RegisterProtocol(Scheme, reg.RoundTripper())
	return &http.Client{Transport: tr, Timeout: 10 * time.Second}
}

func TestRequestReachesASecondaryThatOnlyDialledOut(t *testing.T) {
	p := newPrimary(t, "k")
	startSecondary(t, p.URL, "n1", "k")
	waitFor(t, "the link", func() bool { return p.Reg.Connected("n1") })

	req, _ := http.NewRequest("GET", "tunnel://n1/hello", nil)
	req.Header.Set("X-Who", "the primary")
	resp, err := nodeClient(p.Reg).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(body) != "hi from the primary" {
		t.Fatalf("got %d %q", resp.StatusCode, body)
	}
}

func TestManyConcurrentRequestsShareOneLink(t *testing.T) {
	p := newPrimary(t, "k")
	startSecondary(t, p.URL, "n1", "k")
	waitFor(t, "the link", func() bool { return p.Reg.Connected("n1") })

	client := nodeClient(p.Reg)
	var wg sync.WaitGroup
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get("tunnel://n1/hello")
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			if b, _ := io.ReadAll(resp.Body); !strings.HasPrefix(string(b), "hi from") {
				errs <- errors.New("bad body " + string(b))
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if p.attempts.Load() != 1 {
		t.Fatalf("100 requests must ride one connection, but the primary saw %d connection attempts", p.attempts.Load())
	}
}

func TestStreamingResponsesArriveWhileTheSecondaryIsStillWriting(t *testing.T) {
	p := newPrimary(t, "k")
	startSecondary(t, p.URL, "n1", "k")
	waitFor(t, "the link", func() bool { return p.Reg.Connected("n1") })
	client := nodeClient(p.Reg)

	resp, err := client.Get("tunnel://n1/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 64)
	n := make(chan int, 1)
	go func() { k, _ := resp.Body.Read(buf); n <- k }()
	select {
	case k := <-n:
		if !strings.Contains(string(buf[:k]), "first") {
			t.Fatalf("unexpected first chunk %q", buf[:k])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the first chunk was held back while the secondary was still streaming")
	}
	if r, err := client.Get("tunnel://n1/release"); err == nil {
		r.Body.Close()
	}
	rest, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(rest), "last") {
		t.Fatalf("the stream must finish, got %q", rest)
	}
}

func TestLargeBodiesSurviveFlowControl(t *testing.T) {
	p := newPrimary(t, "k")
	startSecondary(t, p.URL, "n1", "k")
	waitFor(t, "the link", func() bool { return p.Reg.Connected("n1") })

	payload := bytes.Repeat([]byte("0123456789abcdef"), 512*1024) // 8 MiB, far above HTTP/2's initial window
	resp, err := nodeClient(p.Reg).Post("tunnel://n1/echo", "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("an 8 MiB body must round-trip intact: %d bytes back (%v)", len(got), err)
	}
}

func TestLinkRecoversAfterItIsDropped(t *testing.T) {
	p := newPrimary(t, "k")
	startSecondary(t, p.URL, "n1", "k")
	waitFor(t, "the link", func() bool { return p.Reg.Connected("n1") })

	p.Reg.Disconnect("n1")
	waitFor(t, "the drop to be noticed", func() bool { return strings.Contains(p.eventLog(), "n1:down") })
	waitFor(t, "the secondary to reconnect by itself", func() bool { return p.Reg.Connected("n1") })

	resp, err := nodeClient(p.Reg).Get("tunnel://n1/hello")
	if err != nil {
		t.Fatalf("requests must work again after the reconnect: %v", err)
	}
	resp.Body.Close()
	if got := p.eventLog(); got != "n1:up,n1:down,n1:up" {
		t.Fatalf("expected up, down, up; got %s", got)
	}
}

func TestRequestToADisconnectedNodeFailsCleanly(t *testing.T) {
	p := newPrimary(t, "k")
	_, err := nodeClient(p.Reg).Get("tunnel://ghost/hello")
	if err == nil || !errors.Is(err, ErrNodeNotConnected) {
		t.Fatalf("expected ErrNodeNotConnected, got %v", err)
	}
}

func TestWrongCredentialsAreRefusedAndRetried(t *testing.T) {
	p := newPrimary(t, "right")
	startSecondary(t, p.URL, "n1", "wrong")
	waitFor(t, "several attempts", func() bool { return p.attempts.Load() >= 3 })
	if p.Reg.Connected("n1") {
		t.Fatal("a node with the wrong key must not get a link")
	}
}

func TestANewLinkReplacesTheOldOne(t *testing.T) {
	p := newPrimary(t, "k")
	startSecondary(t, p.URL, "n1", "k")
	waitFor(t, "the first link", func() bool { return p.Reg.Connected("n1") })
	startSecondary(t, p.URL, "n1", "k") // a second process claiming the same node
	waitFor(t, "replacement", func() bool { return strings.Count(p.eventLog(), "n1:up") >= 2 })
	waitFor(t, "a single live link", func() bool { return p.Reg.Connected("n1") })

	resp, err := nodeClient(p.Reg).Get("tunnel://n1/hello")
	if err != nil {
		t.Fatalf("the node must still answer after a replacement: %v", err)
	}
	resp.Body.Close()
}

func TestSecondaryStopsWhenCancelled(t *testing.T) {
	p := newPrimary(t, "k")
	cancel := startSecondary(t, p.URL, "n1", "k")
	waitFor(t, "the link", func() bool { return p.Reg.Connected("n1") })
	cancel()
	waitFor(t, "the link to go away", func() bool { return !p.Reg.Connected("n1") })
}
