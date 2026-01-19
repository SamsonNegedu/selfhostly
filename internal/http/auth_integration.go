package http

// This file previously contained GoBetterAuth integration code.
// Now using go-pkgz/auth - all auth logic is in server.go
//
// Auth flow with go-pkgz/auth:
// 1. User visits /auth/github/login -> redirected to GitHub OAuth
// 2. GitHub redirects back to /auth/github/callback
// 3. go-pkgz/auth sets JWT cookie and redirects to app
// 4. All /api/* requests require valid JWT cookie
// 5. User can logout via /auth/logout
