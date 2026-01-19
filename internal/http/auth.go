package http

// Note: Authentication is handled by go-pkgz/auth with GitHub OAuth
// All API routes require authentication via JWT cookie
// Auth endpoints:
//   - GET /auth/github/login  - Start GitHub OAuth flow
//   - GET /auth/github/callback - GitHub OAuth callback
//   - GET /auth/logout - Logout and clear session
//   - GET /api/me - Get current user info
//
// ErrorResponse is defined in app.go
