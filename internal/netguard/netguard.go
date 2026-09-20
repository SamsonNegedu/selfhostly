// Package netguard restricts where server-side HTTP clients may connect, to prevent the platform
// from being used to reach cloud metadata services or other unintended internal targets.
package netguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// Policy describes which destinations are acceptable for node-to-node traffic.
type Policy struct {
	// AllowLoopback permits 127.0.0.0/8 and ::1 (development and single-host setups)
	AllowLoopback bool
	// AllowedCIDRs, when non-empty, additionally requires the destination to fall inside one of them
	AllowedCIDRs []*net.IPNet
}

// NewPolicy parses CIDR strings into a Policy
func NewPolicy(allowLoopback bool, cidrs []string) (*Policy, error) {
	p := &Policy{AllowLoopback: allowLoopback}
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(strings.TrimSpace(c))
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", c, err)
		}
		p.AllowedCIDRs = append(p.AllowedCIDRs, n)
	}
	return p, nil
}

// CheckIP returns an error when ip is never an acceptable node destination.
func (p *Policy) CheckIP(ip net.IP) error {
	switch {
	case ip.IsUnspecified():
		return fmt.Errorf("destination %s is an unspecified address", ip)
	case ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast():
		return fmt.Errorf("destination %s is link-local (cloud metadata range)", ip)
	case ip.IsMulticast():
		return fmt.Errorf("destination %s is a multicast address", ip)
	case ip.IsLoopback() && !p.AllowLoopback:
		return fmt.Errorf("destination %s is loopback", ip)
	}
	if len(p.AllowedCIDRs) > 0 && !(ip.IsLoopback() && p.AllowLoopback) {
		for _, n := range p.AllowedCIDRs {
			if n.Contains(ip) {
				return nil
			}
		}
		return fmt.Errorf("destination %s is outside NODE_ENDPOINT_ALLOWED_CIDRS", ip)
	}
	return nil
}

// ValidateEndpoint checks the scheme and, for IP literals, the address. Hostnames are checked at
// connection time by Control, which also defeats DNS rebinding.
func (p *Policy) ValidateEndpoint(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return errors.New("endpoint must be an absolute http(s) URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("endpoint scheme must be http or https")
	}
	if u.User != nil {
		return errors.New("endpoint must not contain credentials")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil {
		return p.CheckIP(ip)
	}
	return nil
}

// Control is a net.Dialer.Control hook that vets the resolved address just before connecting.
func (p *Policy) Control(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("unparseable destination %q", host)
	}
	return p.CheckIP(ip)
}

// NewHTTPClient returns a client that enforces the policy on every connection and never follows
// redirects (a redirect would carry node credentials to an unvetted host).
func (p *Policy) NewHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, Control: p.Control}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
