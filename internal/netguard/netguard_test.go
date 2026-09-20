package netguard

import (
	"net"
	"testing"
)

func TestCheckIP(t *testing.T) {
	p, _ := NewPolicy(false, nil)
	blocked := []string{"169.254.169.254", "0.0.0.0", "127.0.0.1", "::1", "224.0.0.1", "fe80::1"}
	for _, s := range blocked {
		if p.CheckIP(net.ParseIP(s)) == nil {
			t.Errorf("%s should be blocked", s)
		}
	}
	for _, s := range []string{"192.168.1.10", "10.0.0.5", "172.18.0.3", "100.64.1.1", "8.8.8.8"} {
		if err := p.CheckIP(net.ParseIP(s)); err != nil {
			t.Errorf("%s should be allowed: %v", s, err)
		}
	}
}

func TestLoopbackAllowedWhenConfigured(t *testing.T) {
	p, _ := NewPolicy(true, nil)
	if err := p.CheckIP(net.ParseIP("127.0.0.1")); err != nil {
		t.Fatal(err)
	}
}

func TestCIDRRestriction(t *testing.T) {
	p, err := NewPolicy(false, []string{"192.168.0.0/16"})
	if err != nil {
		t.Fatal(err)
	}
	if p.CheckIP(net.ParseIP("192.168.5.5")) != nil {
		t.Error("inside CIDR must pass")
	}
	if p.CheckIP(net.ParseIP("10.0.0.1")) == nil {
		t.Error("outside CIDR must fail")
	}
	if _, err := NewPolicy(false, []string{"nope"}); err == nil {
		t.Error("bad CIDR must error")
	}
}

func TestValidateEndpoint(t *testing.T) {
	p, _ := NewPolicy(false, nil)
	bad := []string{"ftp://x", "http://169.254.169.254/latest", "http://user:pw@host", "not a url", "http://127.0.0.1:1"}
	for _, u := range bad {
		if p.ValidateEndpoint(u) == nil {
			t.Errorf("%q should be rejected", u)
		}
	}
	for _, u := range []string{"http://primary:8082", "https://node1.example.com", "http://192.168.1.10:8080"} {
		if err := p.ValidateEndpoint(u); err != nil {
			t.Errorf("%q should pass: %v", u, err)
		}
	}
}
