package cloudflare

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// MockHTTPClient is a test implementation that records requests and returns mocked responses
type MockHTTPClient struct {
	// Map of URL to mock response
	MockResponses map[string]MockResponse
	// Recorded requests
	RecordedRequests []RequestRecord
}

// MockResponse represents a mocked HTTP response
type MockResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// RequestRecord records a request made to the mock client
type RequestRecord struct {
	Method  string
	URL     string
	Headers map[string][]string
	Body    string
}

// NewMockHTTPClient creates a new mock HTTP client
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		MockResponses:   make(map[string]MockResponse),
		RecordedRequests: make([]RequestRecord, 0),
	}
}

// Do records the request and returns a mocked response
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Record the request
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		bodyStr := string(bodyBytes)
		
		// Restore the request body for subsequent reads
		req.Body = io.NopCloser(strings.NewReader(bodyStr))
		
		record := RequestRecord{
			Method:  req.Method,
			URL:     req.URL.String(),
			Headers: req.Header,
			Body:    bodyStr,
		}
		m.RecordedRequests = append(m.RecordedRequests, record)
	} else {
		record := RequestRecord{
			Method:  req.Method,
			URL:     req.URL.String(),
			Headers: req.Header,
			Body:    "",
		}
		m.RecordedRequests = append(m.RecordedRequests, record)
	}
	
	// Find a matching mock response
	if response, exists := m.MockResponses[req.URL.String()]; exists {
		// Create a new response with the mocked data
		statusCode := response.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		
		mockResp := &http.Response{
			StatusCode: statusCode,
			Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(response.Body)),
		}
		
		// Set headers
		for key, value := range response.Headers {
			mockResp.Header.Set(key, value)
		}
		
		return mockResp, nil
	}
	
	// Default response - 404 Not Found
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Status:     fmt.Sprintf("%d %s", http.StatusNotFound, http.StatusText(http.StatusNotFound)),
		Body:       io.NopCloser(strings.NewReader("Not Found")),
	}, nil
}

// SetMockResponse sets a mock response for a specific URL
func (m *MockHTTPClient) SetMockResponse(url string, response MockResponse) {
	m.MockResponses[url] = response
}

// SetJSONMockResponse sets a mock JSON response for a specific URL
func (m *MockHTTPClient) SetJSONMockResponse(url string, statusCode int, body interface{}) error {
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}
	
	m.MockResponses[url] = MockResponse{
		StatusCode: statusCode,
		Body:       string(jsonBytes),
		Headers: map[string]string{"Content-Type": "application/json"},
	}
	
	return nil
}

// GetRecordedRequests returns all recorded requests
func (m *MockHTTPClient) GetRecordedRequests() []RequestRecord {
	return m.RecordedRequests
}

// Clear clears all recorded requests and mock responses
func (m *MockHTTPClient) Clear() {
	m.RecordedRequests = make([]RequestRecord, 0)
	m.MockResponses = make(map[string]MockResponse)
}

// AssertRequestMade checks if a request was made to a specific URL
func (m *MockHTTPClient) AssertRequestMade(method, url string) bool {
	for _, record := range m.RecordedRequests {
		if record.Method == method && record.URL == url {
			return true
		}
	}
	return false
}

// GetRequestCount returns the number of requests made to a specific URL
func (m *MockHTTPClient) GetRequestCount(method, url string) int {
	count := 0
	for _, record := range m.RecordedRequests {
		if record.Method == method && record.URL == url {
			count++
		}
	}
	return count
}

// GetRequestBody returns the body of the last request to a specific URL
func (m *MockHTTPClient) GetRequestBody(method, url string) string {
	for i := len(m.RecordedRequests) - 1; i >= 0; i-- {
		record := m.RecordedRequests[i]
		if record.Method == method && record.URL == url {
			return record.Body
		}
	}
	return ""
}
