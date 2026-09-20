// Package auth holds the identity checks that sit on top of go-pkgz/auth: the GitHub allow-list
// and Cloudflare Access token verification.
package auth

import (
	"context"
	"crypto/sha1" //nolint:gosec // matches go-pkgz/auth's user ID derivation, not used for security
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/go-pkgz/auth/token"
	"github.com/selfhostly/internal/constants"
)

// AllowList authorizes GitHub users by their stable user ID.
//
// go-pkgz/auth sets token.User.Name to the free-text profile display name, which any GitHub user
// can set to someone else's login. The ID is derived from the immutable login instead, so it is
// the only field that may be used for authorization.
type AllowList struct {
	mu  sync.RWMutex
	ids map[string]struct{}
}

// UserIDForLogin returns the token.User.ID go-pkgz/auth assigns to a GitHub login.
func UserIDForLogin(login string) string {
	return constants.GitHubUserIDPrefix + token.HashID(sha1.New(), login)
}

// NewAllowList builds an allow-list from logins (or ready-made github_<hash> IDs). It does not
// touch the network; call Resolve to add canonical-case logins.
func NewAllowList(entries []string) *AllowList {
	a := &AllowList{ids: make(map[string]struct{})}
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if strings.HasPrefix(e, constants.GitHubUserIDPrefix) {
			a.add(e)
			continue
		}
		// GitHub logins are case-insensitive and unique, so any casing that hashes to a user's ID
		// identifies that same user.
		a.add(UserIDForLogin(e))
		a.add(UserIDForLogin(strings.ToLower(e)))
	}
	return a
}

func (a *AllowList) add(id string) {
	a.mu.Lock()
	a.ids[id] = struct{}{}
	a.mu.Unlock()
}

// Allows reports whether the authenticated user ID is on the list.
func (a *AllowList) Allows(userID string) bool {
	if a == nil || userID == "" {
		return false
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, ok := a.ids[userID]
	return ok
}

// Size returns the number of distinct IDs (several may map to one person).
func (a *AllowList) Size() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.ids)
}

// Resolve looks each login up on GitHub and adds the ID for its canonical casing. Lookup failures
// are logged and skipped: the as-typed and lower-case IDs added by NewAllowList still apply.
func (a *AllowList) Resolve(ctx context.Context, client *http.Client, baseURL string, logins []string) {
	if client == nil {
		client = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = constants.GitHubAPIBaseURL
	}
	for _, login := range logins {
		login = strings.TrimSpace(login)
		if login == "" || strings.HasPrefix(login, constants.GitHubUserIDPrefix) {
			continue
		}
		canonical, err := fetchLogin(ctx, client, baseURL, login)
		if err != nil {
			slog.Warn("could not resolve GitHub login for allow-list; relying on typed casing", "login", login, "error", err)
			continue
		}
		a.add(UserIDForLogin(canonical))
	}
}

func fetchLogin(ctx context.Context, client *http.Client, baseURL, login string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, constants.AllowListResolveTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/users/"+url.PathEscape(login), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github returned %d", resp.StatusCode)
	}
	var body struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body.Login == "" {
		return "", fmt.Errorf("unexpected github response")
	}
	return body.Login, nil
}
