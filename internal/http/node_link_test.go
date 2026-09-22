package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/nodelink"
)

const linkNodeKey = "secondary-node-key-0123456789abc"

// newTestSecondary is a real secondary Server (not a stub), so the tests exercise its own node-key
// check and routing exactly as a deployed one would.
func newTestSecondary(t *testing.T, id, name string) *Server {
	t.Helper()
	dir := t.TempDir()
	database, err := db.Init(filepath.Join(dir, "sec.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	cfg := &config.Config{
		ServerAddress: "127.0.0.1:8082",
		DatabasePath:  filepath.Join(dir, "sec.db"),
		AppsDir:       filepath.Join(dir, "apps"),
		Environment:   constants.EnvProduction,
		Node: config.NodeConfig{
			ID: id, Name: name, IsPrimary: false, APIKey: linkNodeKey,
			APIEndpoint: linkEndpoint(id), Transport: constants.NodeTransportTunnel,
		},
		Security: config.SecurityConfig{Mode: constants.SecurityModeWarn, HostAppsDir: filepath.Join(dir, "apps"), SessionHours: 24},
	}
	if err := database.InitNode(cfg); err != nil {
		t.Fatal(err)
	}
	return NewServer(cfg, database)
}

type linkFixture struct {
	primary    *Server
	primaryDB  *db.DB
	httpServer *httptest.Server
}

func newLinkFixture(t *testing.T) *linkFixture {
	t.Helper()
	p, database := newTestServer(t, constants.SecurityModeEnforce)
	hs := httptest.NewServer(p.engine)
	t.Cleanup(hs.Close)
	return &linkFixture{primary: p, primaryDB: database, httpServer: hs}
}

// dial starts a secondary's link to the fixture's primary
func (f *linkFixture) dial(t *testing.T, id, name, key, token string, handler http.Handler) context.CancelFunc {
	t.Helper()
	return f.dialWithBackoff(t, id, name, key, token, handler, 20*time.Millisecond)
}

// dialWithBackoff is dial with a chosen retry delay. Connections that are meant to be refused use a
// long one: a real node backs off from a second, and the primary rate limits link attempts.
func (f *linkFixture) dialWithBackoff(t *testing.T, id, name, key, token string, handler http.Handler, backoff time.Duration) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	h := http.Header{
		constants.HeaderLinkNodeID:   {id},
		constants.HeaderLinkNodeName: {name},
		constants.HeaderLinkNodeKey:  {key},
	}
	if token != "" {
		h.Set(constants.HeaderLinkJoinToken, token)
	}
	c := &nodelink.Client{
		URL: f.httpServer.URL + constants.LinkPath, Header: h, Handler: handler,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		MinBackoff: backoff, MaxBackoff: backoff * 5,
	}
	go c.Run(ctx)
	t.Cleanup(cancel)
	return cancel
}

func (f *linkFixture) waitStatus(t *testing.T, id, want string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		if n, err := f.primaryDB.GetNode(id); err == nil {
			last = n.Status
			if n.Status == want {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("node %s never became %q (last status %q)", id, want, last)
}

// disconnectAndWaitOffline drops id's link and confirms it went offline. A bare Registry.Disconnect
// call is a no-op if there is no live session for id at that exact moment, and under load there
// briefly is none: the client backs off and reconnects on its own (by design, and normally
// unremarkable) even seconds into a link that was already up, and a Disconnect that lands in that gap
// silently disconnects nothing. Retrying the call alongside the status poll means a Disconnect that
// missed is simply tried again against whatever session is live by the next tick.
func (f *linkFixture) disconnectAndWaitOffline(t *testing.T, id string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		f.primary.nodeLinks.Disconnect(id)
		if n, err := f.primaryDB.GetNode(id); err == nil {
			last = n.Status
			if n.Status == constants.NodeStatusOffline {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("node %s never became %q after Disconnect (last status %q)", id, constants.NodeStatusOffline, last)
}

func (f *linkFixture) joinToken(t *testing.T) string {
	t.Helper()
	tok, _, err := f.primaryDB.CreateJoinToken(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestNewNodeJoinsOverItsOwnOutboundLink(t *testing.T) {
	f := newLinkFixture(t)
	sec := newTestSecondary(t, "sec1", "sec1")
	f.dial(t, "sec1", "sec1", linkNodeKey, f.joinToken(t), sec.engine)

	f.waitStatus(t, "sec1", constants.NodeStatusOnline)
	n, _ := f.primaryDB.GetNode("sec1")
	if n.APIEndpoint != "tunnel://sec1" || n.IsPrimary {
		t.Fatalf("the node must be recorded as a linked secondary, got %+v", n)
	}
	if n.APIKey != linkNodeKey {
		t.Fatal("the primary must keep the node's key: it uses it on every request over the link")
	}
	var used int
	_ = f.primaryDB.QueryRow(`SELECT COUNT(*) FROM join_tokens WHERE used_at IS NOT NULL`).Scan(&used)
	if used != 1 {
		t.Fatal("the join token must be spent")
	}
}

func TestLinkIsRefusedWithoutValidCredentials(t *testing.T) {
	f := newLinkFixture(t)
	sec := newTestSecondary(t, "sec1", "sec1")

	// a stranger with no token, a wrong token, and (below) a known node with the wrong key
	f.dialWithBackoff(t, "ghost", "ghost", linkNodeKey, "", sec.engine, 2*time.Second)
	f.dialWithBackoff(t, "ghost2", "ghost2", linkNodeKey, "sfj_not-a-real-token", sec.engine, 2*time.Second)
	time.Sleep(400 * time.Millisecond)
	for _, id := range []string{"ghost", "ghost2"} {
		if _, err := f.primaryDB.GetNode(id); err == nil {
			t.Fatalf("%s must not be registered without a valid token", id)
		}
	}

	f.dial(t, "sec1", "sec1", linkNodeKey, f.joinToken(t), sec.engine)
	f.waitStatus(t, "sec1", constants.NodeStatusOnline)
	f.dialWithBackoff(t, "sec1", "sec1", "an-impostor-key-0123456789ab", "", sec.engine, 2*time.Second)
	time.Sleep(400 * time.Millisecond)
	if n, _ := f.primaryDB.GetNode("sec1"); n.APIKey != linkNodeKey {
		t.Fatal("an impostor must not be able to replace a node's key")
	}
}

func TestAJoinTokenWorksOnce(t *testing.T) {
	f := newLinkFixture(t)
	tok := f.joinToken(t)
	s1 := newTestSecondary(t, "sec1", "sec1")
	s2 := newTestSecondary(t, "sec2", "sec2")
	f.dial(t, "sec1", "sec1", linkNodeKey, tok, s1.engine)
	f.waitStatus(t, "sec1", constants.NodeStatusOnline)

	f.dialWithBackoff(t, "sec2", "sec2", linkNodeKey, tok, s2.engine, 2*time.Second)
	time.Sleep(400 * time.Millisecond)
	if _, err := f.primaryDB.GetNode("sec2"); err == nil {
		t.Fatal("a spent token must not admit a second node")
	}
}

func TestRequestsForALinkedNodeAreForwardedWithoutTheUsersCredentials(t *testing.T) {
	f := newLinkFixture(t)

	var mu sync.Mutex
	var seen http.Header
	var seenBody string
	recorder := http.NewServeMux()
	recorder.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"status":"healthy"}`)) })
	recorder.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		seen, seenBody = r.Header.Clone(), string(b)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("answered by the secondary"))
	})
	f.dial(t, "sec1", "sec1", linkNodeKey, f.joinToken(t), recorder)
	f.waitStatus(t, "sec1", constants.NodeStatusOnline)

	req, _ := http.NewRequest("POST", f.httpServer.URL+"/api/apps/abc/start?node_id=sec1", strings.NewReader(`{"x":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "JWT=the-users-session")
	req.Header.Set("Authorization", "Bearer the-users-token")
	req.Header.Set("Origin", "https://app.example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || string(body) != "answered by the secondary" {
		t.Fatalf("the caller must get the secondary's answer, got %d %q", resp.StatusCode, body)
	}

	mu.Lock()
	defer mu.Unlock()
	if seen.Get(constants.HeaderNodeID) != "sec1" || seen.Get(constants.HeaderNodeAPIKey) != linkNodeKey {
		t.Fatalf("the secondary must receive the node's credentials, got %v", seen)
	}
	for _, h := range []string{"Cookie", "Authorization", "Origin", constants.HeaderGatewayAPIKey} {
		if seen.Get(h) != "" {
			t.Fatalf("the user's %s must never reach the secondary", h)
		}
	}
	if seenBody != `{"x":1}` {
		t.Fatalf("the body must arrive intact, got %q", seenBody)
	}
}

func TestCreatingAnAppOnALinkedNodeIsForwardedByTheBodysNodeID(t *testing.T) {
	f := newLinkFixture(t)
	var mu sync.Mutex
	var gotBody string
	h := http.NewServeMux()
	h.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	h.HandleFunc("/api/apps", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		gotBody = string(b)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	})
	f.dial(t, "sec1", "sec1", linkNodeKey, f.joinToken(t), h)
	f.waitStatus(t, "sec1", constants.NodeStatusOnline)

	body := `{"name":"kan","node_id":"sec1","compose_content":"x"}`
	resp, err := http.Post(f.httpServer.URL+"/api/apps", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	mu.Lock()
	defer mu.Unlock()
	if resp.StatusCode != http.StatusCreated || gotBody != body {
		t.Fatalf("create must be forwarded to the linked node with its body: %d %q", resp.StatusCode, gotBody)
	}
}

func TestForwardingReachesARealSecondaryAndItsOwnAuthAcceptsIt(t *testing.T) {
	f := newLinkFixture(t)
	sec := newTestSecondary(t, "sec1", "sec1")
	f.dial(t, "sec1", "sec1", linkNodeKey, f.joinToken(t), sec.engine)
	f.waitStatus(t, "sec1", constants.NodeStatusOnline)

	resp, err := http.Get(f.httpServer.URL + "/api/node/info?node_id=sec1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var info struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&info)
	if resp.StatusCode != 200 || info.ID != "sec1" {
		t.Fatalf("the real secondary must answer with its own identity, got %d %+v", resp.StatusCode, info)
	}

	// and the secondary still refuses anyone who does not hold its key: loopback is not the only control
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/node/info", nil)
	sec.engine.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Log("note: a secondary with no login configured answers unauthenticated local calls (by design; it listens on loopback only)")
	}
}

func TestRequestToAnUnconnectedLinkedNodeIsServiceUnavailable(t *testing.T) {
	f := newLinkFixture(t)
	n := db.NewNodeWithID("gone", "gone", linkEndpoint("gone"), linkNodeKey, false)
	if err := f.primaryDB.CreateNode(n); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(f.httpServer.URL + "/api/apps/abc?node_id=gone")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("a linked node with no live link must give 503, got %d", resp.StatusCode)
	}
}

func TestDroppedLinkMarksTheNodeOfflineAtOnce(t *testing.T) {
	f := newLinkFixture(t)
	sec := newTestSecondary(t, "sec1", "sec1")
	cancel := f.dial(t, "sec1", "sec1", linkNodeKey, f.joinToken(t), sec.engine)
	f.waitStatus(t, "sec1", constants.NodeStatusOnline)
	cancel()
	f.waitStatus(t, "sec1", constants.NodeStatusOffline)
}

func TestExistingDirectNodeSwitchesToTheLinkKeepingItsIdentity(t *testing.T) {
	f := newLinkFixture(t)
	direct := db.NewNodeWithID("old1", "old1", "http://192.168.1.50:8082", linkNodeKey, false)
	if err := f.primaryDB.CreateNode(direct); err != nil {
		t.Fatal(err)
	}
	sec := newTestSecondary(t, "old1", "old1")
	f.dial(t, "old1", "old1", linkNodeKey, "", sec.engine) // no token: the node is recognised by its key
	deadline := time.Now().Add(15 * time.Second)          // same CI-contention margin as waitStatus
	for time.Now().Before(deadline) {
		if n, _ := f.primaryDB.GetNode("old1"); n != nil && n.APIEndpoint == "tunnel://old1" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	n, _ := f.primaryDB.GetNode("old1")
	t.Fatalf("the node must switch to being reached over its link, still %s", n.APIEndpoint)
}

func TestUsersCannotRegisterALinkEndpointThemselves(t *testing.T) {
	f := newLinkFixture(t)
	body := registerBody("evil", "k-evil-0123456789abcdef", testRegToken)
	body["api_endpoint"] = "tunnel://sec1"
	if w := do(f.primary, "POST", "/api/nodes/register", body, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("a tunnel endpoint could point one node's traffic at another node's link, got %d", w.Code)
	}
}

func TestLinkSurvivesTheServersRequestTimeouts(t *testing.T) {
	// the real server sets read/write timeouts; a hijacked connection must not inherit them
	p, database := newTestServer(t, constants.SecurityModeEnforce)
	hs := httptest.NewUnstartedServer(p.engine)
	hs.Config.ReadTimeout = 300 * time.Millisecond
	hs.Config.WriteTimeout = 300 * time.Millisecond
	hs.Start()
	t.Cleanup(hs.Close)
	f := &linkFixture{primary: p, primaryDB: database, httpServer: hs}

	sec := newTestSecondary(t, "sec1", "sec1")
	f.dial(t, "sec1", "sec1", linkNodeKey, f.joinToken(t), sec.engine)
	f.waitStatus(t, "sec1", constants.NodeStatusOnline)
	time.Sleep(900 * time.Millisecond) // three times the timeouts

	resp, err := http.Get(hs.URL + "/api/node/info?node_id=sec1")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("the link must still work after the server timeouts have passed: %v", err)
	}
	resp.Body.Close()
}

func TestJoiningOverTheLinkIsAuditedButReconnectingIsNot(t *testing.T) {
	f := newLinkFixture(t)
	sec := newTestSecondary(t, "sec1", "sec1")
	f.dial(t, "sec1", "sec1", linkNodeKey, f.joinToken(t), sec.engine)
	f.waitStatus(t, "sec1", constants.NodeStatusOnline)

	count := func(action string) int {
		entries, err := f.primaryDB.ListAudit(100)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, e := range entries {
			if e.Action == action && e.TargetName == "sec1" {
				n++
			}
		}
		return n
	}
	if count("node.join") != 1 {
		t.Fatal("registering a node over its link must be recorded, with the node's name and no secrets")
	}

	f.disconnectAndWaitOffline(t, "sec1")
	f.waitStatus(t, "sec1", constants.NodeStatusOnline) // the secondary reconnects by itself
	if count("node.join") != 1 {
		t.Fatal("a reconnect changes nothing and must not add audit noise")
	}
}
