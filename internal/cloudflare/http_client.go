package cloudflare

import (
	"net/http"
)

// HTTPClient interface abstracts HTTP operations for testing
type HTTPClient interface {
	// Do executes an HTTP request and returns a response
	Do(req *http.Request) (*http.Response, error)
}

// RealHTTPClient is the production implementation that uses the standard http.Client
type RealHTTPClient struct {
	client *http.Client
}

// NewRealHTTPClient creates a new real HTTP client
func NewRealHTTPClient() *RealHTTPClient {
	return &RealHTTPClient{
		client: &http.Client{},
	}
}

// Do executes an HTTP request and returns a response
func (r *RealHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return r.client.Do(req)
}
