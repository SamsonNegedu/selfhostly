# Cloudflare Package

This package contains functionality for managing Cloudflare tunnels and DNS records.

## Components

### Tunnel Manager (`manager.go`)

The `TunnelManager` struct handles Cloudflare tunnel operations with database integration:

- Creating tunnels and storing metadata
- Updating tunnel configurations
- Deleting tunnels and cleaning up resources
- Managing DNS records for tunnels

### API Manager (`tunnel.go`)

The `Manager` struct handles direct API calls to Cloudflare:

- Creating, configuring, and deleting tunnels
- Managing DNS records
- Handling authentication and API requests

### HTTP Client Abstraction

For testability and production-grade architecture, the package uses an HTTP client abstraction:

- `HTTPClient` interface defines methods for HTTP operations
- `RealHTTPClient` implements the interface for production use
- `MockHTTPClient` implements the interface for testing

This abstraction allows:

1. **Testability**: Tests can use mock implementations without making real HTTP requests
2. **Flexibility**: Different implementations can be used in different environments
3. **Isolation**: Tests can run without network dependencies

## Test Coverage

The package includes comprehensive unit tests:

### Tunnel Tests (`tunnel_test.go`)

- `TestCreateTunnel`: Tests creating a tunnel with mock HTTP client
- `TestCreateTunnelError`: Tests error handling when creating a tunnel
- `TestCreateIngressConfiguration`: Tests configuring tunnel ingress rules
- `TestCreateIngressConfigurationError`: Tests error handling for ingress configuration
- `TestDeleteTunnel`: Tests deleting a tunnel with mock HTTP client

### HTTP Client Tests (`http_client_test.go`)

- Mock implementation of HTTPClient interface
- Helper methods for setting up mock responses
- Verification methods to check HTTP requests

## Running Tests

To run all tests for the cloudflare package:

```bash
go test ./internal/cloudflare -v
```

To run with coverage:

```bash
go test ./internal/cloudflare -cover
```

To run a specific test:

```bash
go test ./internal/cloudflare -run TestName
```

## Production-Grade Features

1. **Dependency Injection**: The Manager can be created with a custom HTTPClient for testing
2. **Error Handling**: All functions properly handle and report API errors
3. **Comprehensive Logging**: Operations are logged with appropriate detail
4. **Resource Management**: Proper cleanup of resources after operations

## Usage Example

### Production Code

```go
manager := cloudflare.NewManager("api-token", "account-id")
tunnelManager := cloudflare.NewTunnelManager("api-token", "account-id", database)

tunnelID, token, err := manager.CreateTunnel("my-app")
if err != nil {
    log.Printf("Failed to create tunnel: %v", err)
    return
}

err = manager.CreateIngressConfiguration(tunnelID, []cloudflare.IngressRule{
    {
        Service: "http://localhost:8080",
        Hostname: "example.com",
    },
})
if err != nil {
    log.Printf("Failed to configure tunnel: %v", err)
    return
}
```

### Test Code

```go
mockClient := cloudflare.NewMockHTTPClient()
manager := cloudflare.NewManagerWithClient("api-token", "account-id", mockClient)

// Set up mock response
response := cloudflare.CreateTunnelResponse{
    Success: true,
    Result: struct {
        ID     string `json:"id"`
        Name   string `json:"name"`
        Token  string `json:"token,omitempty"`
        Status string `json:"status"`
    }{
        ID:     "tunnel-123",
        Name:   "my-app",
        Token:  "token-456",
        Status: "active",
    },
}

mockClient.SetJSONMockResponse(
    "https://api.cloudflare.com/client/v4/accounts/account-id/cfd_tunnel",
    http.StatusOK,
    response,
)

tunnelID, token, err := manager.CreateTunnel("my-app")

// Verify HTTP request was made
if !mockClient.AssertRequestMade("POST", "https://api.cloudflare.com/client/v4/accounts/account-id/cfd_tunnel") {
    t.Error("Expected POST request to create tunnel")
}
```

## Integration with Docker Package

The cloudflare package integrates with the docker package to provide a complete solution for self-hosting applications:

1. Docker Manager handles application lifecycle (start, stop, update)
2. Cloudflare Manager handles tunnel creation and configuration
3. Together they provide public access to self-hosted applications

## Future Improvements

1. Add integration tests that work with real Cloudflare API
2. Implement retry logic for transient network failures
3. Add support for streaming response handling for large responses
4. Implement request/response caching for improved performance
5. Add metrics collection for API call performance and error rates
