package http

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
)

func TestEventStreamReportsDatabaseChanges(t *testing.T) {
	s, database := newTestServer(t, constants.SecurityModeEnforce)
	ts := httptest.NewServer(s.engine)
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/api/events", nil)
	req.Header.Set(constants.HeaderGatewayAPIKey, testGWKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d, content type %q, body %s", resp.StatusCode, resp.Header.Get("Content-Type"), b)
	}

	lines := make(chan string, 16)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()

	// The stream announces itself, then reports a change once one happens.
	select {
	case <-lines:
	case <-time.After(2 * time.Second):
		t.Fatal("no first line from the stream")
	}
	if err := database.CreateNode(db.NewNode("second", "http://second:8082", "key", false)); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(3 * time.Second)
	for {
		select {
		case line := <-lines:
			if strings.HasPrefix(line, "data:") && strings.Contains(line, `"kind":"nodes"`) {
				return
			}
		case <-deadline:
			t.Fatal("no node change event arrived")
		}
	}
}
