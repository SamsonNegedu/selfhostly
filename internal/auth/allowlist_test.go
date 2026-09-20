package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDisplayNameCannotImpersonateLogin(t *testing.T) {
	a := NewAllowList([]string{"samson"})
	// The attacker's login differs, so their ID differs even if their display name is "samson".
	if a.Allows(UserIDForLogin("attacker")) {
		t.Fatal("a different login must not be allowed")
	}
	if !a.Allows(UserIDForLogin("samson")) {
		t.Fatal("the listed login must be allowed")
	}
}

func TestCaseInsensitiveEntries(t *testing.T) {
	a := NewAllowList([]string{"SamsonNegedu"})
	if !a.Allows(UserIDForLogin("samsonnegedu")) || !a.Allows(UserIDForLogin("SamsonNegedu")) {
		t.Fatal("as-typed and lower-case IDs must both match")
	}
}

func TestExplicitIDEntry(t *testing.T) {
	id := UserIDForLogin("someone")
	a := NewAllowList([]string{id})
	if !a.Allows(id) {
		t.Fatal("explicit github_ ID entries must be honoured")
	}
}

func TestEmptyIDAndNilList(t *testing.T) {
	var nilList *AllowList
	if nilList.Allows("x") || NewAllowList(nil).Allows("") {
		t.Fatal("empty list or ID must deny")
	}
}

func TestResolveAddsCanonicalCasing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/nedson202" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"login":"Nedson202"}`))
	}))
	defer srv.Close()

	entries := []string{"nedson202"}
	a := NewAllowList(entries)
	if a.Allows(UserIDForLogin("Nedson202")) {
		t.Fatal("canonical casing should not match before resolving")
	}
	a.Resolve(context.Background(), srv.Client(), srv.URL, entries)
	if !a.Allows(UserIDForLogin("Nedson202")) {
		t.Fatal("canonical casing must match after resolving")
	}
}

func TestResolveFailureKeepsTypedIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	a := NewAllowList([]string{"abc"})
	a.Resolve(context.Background(), srv.Client(), srv.URL, []string{"abc"})
	if !a.Allows(UserIDForLogin("abc")) {
		t.Fatal("typed ID must survive a failed lookup")
	}
}
