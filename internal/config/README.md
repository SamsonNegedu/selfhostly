# Config Package

This package contains configuration structures and functions for the Automaton application.

## Components

### Configuration Structures

- `Config`: Main configuration structure containing all application settings
- `CORSConfig`: CORS-related configuration
- `CloudflareConfig`: Cloudflare API configuration
- `AuthConfig`: Authentication-related configuration
- `GitHubOAuthConfig`: GitHub OAuth-specific configuration

### Functions

- `Load()`: Loads configuration from environment variables with defaults
- `parseCommaSeparatedList()`: Parses a comma-separated string into a slice
- `getEnv()`: Gets an environment variable with a default value

## Environment Variables

The following environment variables can be used to configure the application:

- `SERVER_ADDRESS`: Server address (default: ":8080")
- `DATABASE_PATH`: Path to the SQLite database (default: "./data/automaton.db")
- `APPS_DIR`: Directory for application files (default: "./apps")
- `CORS_ALLOWED_ORIGINS`: Comma-separated list of allowed CORS origins (default: "http://localhost:5173,http://localhost:3000,http://localhost:8080")
- `AUTO_START_APPS`: Whether to auto-start applications (default: "false")
- `CLOUDFLARE_API_TOKEN`: Cloudflare API token (default: "")
- `CLOUDFLARE_ACCOUNT_ID`: Cloudflare account ID (default: "")
- `AUTH_ENABLED`: Whether authentication is enabled (default: "false")
- `JWT_SECRET`: JWT secret key (default: "change-me-in-production-secret-key")
- `AUTH_COOKIE_DOMAIN`: Authentication cookie domain (default: "localhost")
- `AUTH_SECURE_COOKIE`: Whether to use secure cookies (default: "false")
- `AUTH_BASE_URL`: Base URL for OAuth callbacks (default: "http://localhost:8080")
- `GITHUB_CLIENT_ID`: GitHub OAuth client ID (default: "")
- `GITHUB_CLIENT_SECRET`: GitHub OAuth client secret (default: "")

## Test Coverage

The package includes comprehensive unit tests:

### Config Tests (`config_test.go`)

- `TestLoad`: Tests loading configuration with default values
- `TestLoadWithCustomEnv`: Tests loading configuration with custom environment variables
- `TestParseCommaSeparatedList`: Tests parsing comma-separated lists with various inputs
- `TestGetEnv`: Tests getting environment variables with defaults

## Running Tests

To run all tests for the config package:

```bash
go test ./internal/config -v
```

To run with coverage:

```bash
go test ./internal/config -cover
```

To run a specific test:

```bash
go test ./internal/config -run TestName
```

## Security Considerations

- The default JWT secret should be changed in production environments
- Secure cookies should be enabled in production environments
- Proper CORS origins should be configured based on your frontend deployment
- Authentication should be enabled in production environments
- Environment variables containing sensitive information should be properly secured