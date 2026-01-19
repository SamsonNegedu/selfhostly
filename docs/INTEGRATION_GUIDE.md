# Authentication Guide

This guide explains how authentication works with GitHub OAuth using go-pkgz/auth.

## 🏗️ Architecture Overview

### Backend (Go + Gin + go-pkgz/auth)
```
┌─────────────────────────────────────────────────┐
│         Backend Server                          │
│  ┌────────────────────────────────────┐         │
│  │     go-pkgz/auth                   │         │
│  │  ┌────────────────────────────┐    │         │
│  │  │ Auth Middleware            │    │         │
│  │  │ - Validates JWT cookies    │    │         │
│  │  │ - Extracts user info       │    │         │
│  │  └────────────────────────────┘    │         │
│  │                                    │         │
│  │  /auth/* Routes                    │         │
│  │  - /auth/github/login              │         │
│  │  - /auth/github/callback           │         │
│  │  - /auth/logout                    │         │
│  │                                    │         │
│  │  /api/* Routes (Protected)         │         │
│  │  - /api/apps/*                     │         │
│  │  - /api/settings/*                 │         │
│  │  - /api/me                         │         │
│  └────────────────────────────────────┘         │
└─────────────────────────────────────────────────┘
           ↑
    HTTP + JWT Cookies
```

## 🔐 Authentication Flow

### GitHub OAuth Flow
```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  User    │     │ Frontend │     │ Backend  │     │ GitHub   │
└────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │                │
     │ Click Login    │                │                │
     │───────────────>│                │                │
     │                │                │                │
     │                │ GET /auth/github/login          │
     │                │───────────────>│                │
     │                │                │                │
     │                │ 302 Redirect to GitHub          │
     │<────────────────────────────────│                │
     │                                 │                │
     │ Authorize App                   │                │
     │────────────────────────────────────────────────>│
     │                                 │                │
     │ Redirect with code              │                │
     │<────────────────────────────────────────────────│
     │                                 │                │
     │ GET /auth/github/callback?code=xxx              │
     │────────────────────────────────>│                │
     │                                 │                │
     │                                 │ Exchange code  │
     │                                 │───────────────>│
     │                                 │                │
     │                                 │ User info      │
     │                                 │<───────────────│
     │                                 │                │
     │ Set JWT Cookie + Redirect       │                │
     │<────────────────────────────────│                │
     │                │                │                │
     │                │ Access /api/*  │                │
     │                │───────────────>│                │
     │                │                │ Validate JWT   │
     │                │ 200 OK + Data  │                │
     │                │<───────────────│                │
```

## 📝 Environment Configuration

### Required Environment Variables

```bash
# Enable authentication
AUTH_ENABLED=true

# GitHub OAuth App credentials
# Create at: https://github.com/settings/developers
GITHUB_CLIENT_ID=your_client_id
GITHUB_CLIENT_SECRET=your_client_secret

# JWT Secret (use a strong random string in production)
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# Cookie settings
AUTH_COOKIE_DOMAIN=localhost
AUTH_SECURE_COOKIE=false  # Set to true for HTTPS

# CORS (include your frontend origin)
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:8080
```

### GitHub OAuth App Setup

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click "New OAuth App"
3. Fill in:
   - **Application name**: SelfHost Automaton
   - **Homepage URL**: `http://localhost:8080`
   - **Authorization callback URL**: `http://localhost:8080/auth/github/callback`
4. Copy the Client ID and Client Secret to your environment

## 📊 API Endpoints Reference

### Auth Endpoints (go-pkgz/auth)
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/auth/github/login` | GET | Redirects to GitHub OAuth |
| `/auth/github/callback` | GET | OAuth callback (handled automatically) |
| `/auth/logout` | GET | Clears session cookie |
| `/api/me` | GET | Get current authenticated user |

### Protected API Endpoints
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/apps` | GET/POST | List/create apps |
| `/api/apps/:id` | GET/PUT/DELETE | Get/update/delete app |
| `/api/apps/:id/start` | POST | Start app |
| `/api/apps/:id/stop` | POST | Stop app |
| `/api/settings` | GET/PUT | Get/update settings |

## 💻 Frontend Integration

### Login with GitHub

```typescript
// Redirect to GitHub OAuth
function loginWithGitHub() {
  window.location.href = '/auth/github/login';
}
```

### Check Authentication

```typescript
// Get current user
export function useCurrentUser() {
  return useQuery<User | null>({
    queryKey: ['currentUser'],
    queryFn: async () => {
      const response = await fetch('/api/me', {
        credentials: 'include',
      });
      if (!response.ok) {
        if (response.status === 401) {
          return null; // Not authenticated
        }
        throw new Error('Failed to fetch user');
      }
      return response.json();
    },
  });
}
```

### Logout

```typescript
function logout() {
  window.location.href = '/auth/logout';
}
```

### Making Authenticated API Calls

```typescript
// All API calls must include credentials
const response = await fetch('/api/apps', {
  credentials: 'include',  // Required for cookies
});
```

## 🛡️ Security Features

### JWT Cookies
- HttpOnly cookies prevent XSS attacks
- Secure flag (enable in production) prevents transmission over HTTP
- Configurable expiration (default: 24h token, 7d cookie)

### CORS
- Strict origin validation
- Credentials allowed only for configured origins

## 🚀 Development Setup

### 1. Start Backend

```bash
# Set environment variables
export AUTH_ENABLED=true
export GITHUB_CLIENT_ID=your_client_id
export GITHUB_CLIENT_SECRET=your_client_secret
export JWT_SECRET=dev-secret-key

# Run the server
go run ./cmd/server/main.go
```

### 2. Start Frontend

```bash
cd web
npm install
npm run dev
```

### 3. Access the App

1. Open http://localhost:5173
2. Click "Login with GitHub"
3. Authorize the app on GitHub
4. You'll be redirected back and authenticated

## 🔍 Troubleshooting

### 401 Unauthorized
- Check if `AUTH_ENABLED=true`
- Verify GitHub OAuth credentials
- Ensure cookies are being sent (check `credentials: 'include'`)

### OAuth Callback Error
- Verify callback URL matches exactly: `http://localhost:8080/auth/github/callback`
- Check GitHub OAuth app settings

### CORS Issues
- Add frontend origin to `CORS_ALLOWED_ORIGINS`
- Ensure backend and frontend use same domain in production
