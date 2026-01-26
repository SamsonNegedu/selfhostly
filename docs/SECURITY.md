# Security Documentation

## Overview

This document outlines the security architecture, current limitations, and future considerations for the selfhostly project.

## Current Security Model

### Authentication ✅

**Implementation:** GitHub OAuth via `go-pkgz/auth`

The system implements **authentication** to verify user identity:

- **Provider:** GitHub OAuth
- **Token Type:** JWT stored in HTTP-only cookies
- **Token Duration:** 24 hours (7 day cookie)
- **Protected Routes:** All `/api/*` endpoints (except `/api/health`)
- **Middleware:** Applied globally to all API routes

**Configuration:**
```env
AUTH_ENABLED=true
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_ALLOWED_USERS=your-github-username,trusted-user2
AUTH_BASE_URL=https://your-domain.com
AUTH_SECURE_COOKIE=true
```

**⚠️ CRITICAL:** You **MUST** set `GITHUB_ALLOWED_USERS` to restrict access. Without this, the system will reject all login attempts (fail-secure design).

**How it works:**
1. User visits the application
2. Clicks "Login with GitHub"
3. Redirects to GitHub OAuth flow
4. GitHub redirects back with authorization code
5. Backend exchanges code for user info and generates JWT
6. JWT stored in HTTP-only cookie
7. All subsequent API requests include cookie for authentication

### Authorization ⚠️ 

**Status:** BASIC WHITELIST IMPLEMENTED (Single-User Design)

The system implements **GitHub username whitelist** for access control, but does **NOT** implement resource-level authorization.

#### What This Means

**GitHub username whitelist provides:**
- ✅ Only whitelisted GitHub users can log in
- ✅ Unauthorized users are rejected at authentication
- ✅ Fail-secure: If no users configured, all access denied

**But any whitelisted user can:**
- ✅ View **all** applications in the system
- ✅ Create, update, delete **any** application
- ✅ Start, stop, update **any** application
- ✅ View and modify **all** global settings (Cloudflare credentials, etc.)
- ✅ Manage **all** Cloudflare tunnel configurations
- ✅ Access logs for **any** application

**There is still no concept of:**
- Resource ownership (which user created which app)
- User-specific resource filtering
- Role-based access control (admin vs regular user)
- Permission checks on individual resources

#### Technical Details

**Missing from data models:**
```go
// Current - no user association
type App struct {
    ID             string
    Name           string
    // ... no user_id or owner_id field
}

// What's needed for multi-user
type App struct {
    ID             string
    UserID         string  // MISSING: who owns this app
    Name           string
    // ...
}
```

**Database queries return ALL resources:**
```go
// Current implementation
func (db *DB) GetAllApps() ([]*App, error) {
    // Returns ALL apps regardless of who's requesting
    rows, err := db.Query("SELECT * FROM apps ORDER BY created_at DESC")
    // ...
}

// What's needed for multi-user
func (db *DB) GetUserApps(userID string) ([]*App, error) {
    // Return only apps owned by this user
    rows, err := db.Query("SELECT * FROM apps WHERE user_id = ? ORDER BY created_at DESC", userID)
    // ...
}
```

**No ownership checks in API handlers:**
```go
// Current implementation
func (s *Server) deleteApp(c *gin.Context) {
    id := c.Param("id")
    app, err := s.database.GetApp(id)
    // No check: does this user own this app?
    // Deletes any app by ID
}

// What's needed for multi-user
func (s *Server) deleteApp(c *gin.Context) {
    id := c.Param("id")
    user, _ := getUserFromContext(c)
    app, err := s.database.GetApp(id)
    
    // Check ownership
    if app.UserID != user.ID {
        c.JSON(http.StatusForbidden, ErrorResponse{Error: "Not authorized"})
        return
    }
    // Proceed with deletion
}
```

## Use Case Analysis

### ✅ Single-User Deployment (Current Design Target)

**Acceptable for:**
- Personal Raspberry Pi hosting
- Single admin managing all apps
- Private network deployment
- Learning/hobby projects

**Reasoning:**
- Only one person has access
- Authentication handled at edge (Cloudflare Zero Trust recommended)
- All apps belong to the same person anyway
- Simpler architecture, less overhead

**Recommended Deployment:**
Deploy behind **Cloudflare Zero Trust** with `AUTH_ENABLED=false`. This is simpler, more secure, and has no OAuth/JWT overhead.

**Security checklist:**
- [x] Edge authentication via Cloudflare Zero Trust (recommended)
- [x] OR GitHub OAuth if not using Cloudflare (less ideal)
- [x] Sensitive credentials stored in database (Cloudflare API tokens)
- [x] HTTPS via Cloudflare Tunnel
- [ ] Resource-level authorization (not needed for single user)

### ❌ Multi-User Deployment (Requires Major Changes)

**NOT suitable for:**
- Multiple users/teams managing separate apps
- Hosted service (SaaS)
- Shared infrastructure
- Enterprise/organizational use

**Required changes:**

1. **Database Schema Changes**
   ```sql
   -- Add user_id to apps table
   ALTER TABLE apps ADD COLUMN user_id TEXT NOT NULL;
   ALTER TABLE apps ADD FOREIGN KEY (user_id) REFERENCES users(id);
   
   -- Add user_id index for performance
   CREATE INDEX idx_apps_user_id ON apps(user_id);
   
   -- Add user_id to cloudflare_tunnels
   ALTER TABLE cloudflare_tunnels ADD COLUMN user_id TEXT NOT NULL;
   
   -- Make settings per-user OR add role-based access
   ALTER TABLE settings ADD COLUMN user_id TEXT;
   ```

2. **Model Updates**
   ```go
   type App struct {
       ID             string
       UserID         string    `json:"user_id" db:"user_id"` // NEW
       Name           string
       // ...
   }
   ```

3. **Database Layer Changes**
   ```go
   // Add user filtering to all queries
   func (db *DB) GetUserApps(userID string) ([]*App, error)
   func (db *DB) GetUserApp(appID, userID string) (*App, error)
   func (db *DB) DeleteUserApp(appID, userID string) error
   // etc...
   ```

4. **API Handler Changes**
   ```go
   // Add ownership checks to all handlers
   func (s *Server) deleteApp(c *gin.Context) {
       user, _ := getUserFromContext(c)
       
       // Verify ownership before any operation
       if !s.database.UserOwnsApp(appID, user.ID) {
           c.JSON(403, ErrorResponse{Error: "Forbidden"})
           return
       }
       // ...
   }
   ```

5. **Role-Based Access Control (Optional)**
   ```go
   type User struct {
       ID       string
       Role     string // "admin", "user"
       // ...
   }
   
   // Admin can see all resources
   // Regular users see only their own
   ```

## Current Security Measures

### What IS Protected

1. **Authentication Required**
   - All API endpoints require valid GitHub login
   - Invalid/expired tokens rejected

2. **Secure Token Storage**
   - JWT stored in HTTP-only cookies (prevents XSS)
   - Secure flag enabled in production (HTTPS only)

3. **Security Headers**
   ```go
   X-Content-Type-Options: nosniff
   X-Frame-Options: DENY
   X-XSS-Protection: 1; mode=block
   Referrer-Policy: strict-origin-when-cross-origin
   Strict-Transport-Security: max-age=31536000 (HTTPS only)
   ```

4. **CORS Protection**
   - Configurable allowed origins
   - Credentials support for same-origin requests

5. **Input Validation**
   - Docker compose YAML parsing and validation
   - Request body size limits (10MB)
   - Bind validation on API requests

6. **Sensitive Data Handling**
   - Cloudflare API tokens stored in database
   - Tunnel tokens not exposed in logs (only length logged)
   - User passwords excluded from JSON responses (`json:"-"`)

### What is NOT Protected

1. **Resource Authorization** (main limitation)
   - No ownership checks
   - No resource-level permissions
   - No user isolation

2. **API Rate Limiting**
   - No protection against abuse
   - No request throttling

3. **Audit Logging**
   - No record of who performed what action
   - No compliance trail

4. **Secret Rotation**
   - No mechanism to rotate JWT secrets
   - No Cloudflare token expiration handling

## Migration Path (Future)

If multi-user support is needed, follow these steps:

### Phase 1: Database Schema Migration
1. Add `user_id` columns to relevant tables
2. Populate existing rows with a default admin user
3. Add foreign key constraints
4. Create indexes for performance

### Phase 2: Application Code Updates
1. Update models to include `UserID`
2. Modify all database queries to filter by user
3. Add ownership verification middleware
4. Update all API handlers to check ownership

### Phase 3: Feature Additions
1. Implement user management UI
2. Add role-based access control
3. Add admin dashboard for system-wide management
4. Implement audit logging

### Phase 4: Testing
1. Test resource isolation
2. Verify no cross-user access possible
3. Load testing with multiple users
4. Security audit/penetration testing

## Deployment Recommendations

### Single-User Setup (Current)

#### ✅ RECOMMENDED: Cloudflare Zero Trust (No GitHub OAuth needed)

**This is the ideal setup for single-user deployments.** Deploy behind Cloudflare Zero Trust (formerly Cloudflare Access) to handle authentication at the edge.

**Why this is better:**
- ✅ Authentication handled by Cloudflare before requests reach your app
- ✅ No OAuth configuration needed
- ✅ No JWT secret management
- ✅ No cookies, no tokens to manage
- ✅ Support for multiple identity providers (Google, GitHub, email OTP, etc.)
- ✅ Zero Trust security model
- ✅ Built-in audit logs
- ✅ Works perfectly with Cloudflare Tunnels (already using them!)

**Setup:**
1. Deploy your app with auth disabled
2. Create a Cloudflare Access application
3. Add authentication policies (email, domain, etc.)
4. Done! Cloudflare handles everything

**Environment Variables:**
```env
# Auth disabled - Cloudflare handles it
AUTH_ENABLED=false

# Optional: Cloudflare API for tunnel management
CLOUDFLARE_API_TOKEN=xxx
CLOUDFLARE_ACCOUNT_ID=xxx
```

**Cloudflare Zero Trust Setup:**
```bash
# 1. Create Cloudflare Access application
# Dashboard → Zero Trust → Access → Applications → Add an application

# 2. Configure:
# - Application domain: selfhostly.yourdomain.com
# - Session duration: 24 hours
# - Identity providers: Email, Google, GitHub, etc.

# 3. Add policy:
# - Policy name: Allow yourself
# - Action: Allow
# - Include: Emails → your@email.com
```

**Cost:** Free for up to 50 users

---

#### Alternative: GitHub OAuth (Not Recommended)

If you can't use Cloudflare Zero Trust, you can enable GitHub OAuth authentication.

**Requirements:**
- Valid GitHub OAuth app credentials
- HTTPS in production
- JWT secret management

**Environment Variables:**
```env
AUTH_ENABLED=true
GITHUB_CLIENT_ID=xxx
GITHUB_CLIENT_SECRET=xxx
AUTH_BASE_URL=https://your-domain.com
SECURE_COOKIES=true
```

**Why this is less ideal:**
- More configuration overhead
- Need to manage OAuth app and secrets
- Cookie/token management complexity
- Only supports GitHub for authentication

**Best Practices:**
- Use strong, random JWT secret (32+ characters)
- Enable HTTPS to protect tokens in transit
- Regularly update dependencies
- Monitor logs for suspicious activity
- Keep Docker images up to date

### Multi-User Setup (Future)

**Additional Requirements:**
- Complete authorization implementation (see Migration Path)
- User management system
- Admin role separation
- Audit logging
- Rate limiting
- Database backups with user data

## Known Vulnerabilities

| Vulnerability | Severity | Impact | Mitigation | Status |
|--------------|----------|---------|-----------|---------|
| No resource authorization | **MEDIUM** | Any whitelisted user can manage all resources | GitHub username whitelist + single-user deployment | Partially Mitigated ⚠️ |
| No rate limiting | Medium | Potential DoS via API abuse | Deploy behind reverse proxy with rate limiting | Open 🔴 |
| No audit logging | Low | No compliance trail for actions | Not critical for single-user | Open 🔴 |
| Shared global settings | **MEDIUM** in multi-user | All whitelisted users share Cloudflare credentials | Limit whitelist to trusted users only | Partially Mitigated ⚠️ |

## Security Checklist

### For Current Deployment
- [ ] GitHub OAuth configured with correct callback URL
- [ ] **GitHub username whitelist configured (`GITHUB_ALLOWED_USERS`)**
- [ ] **Verify only trusted users in whitelist**
- [ ] Strong JWT secret set (32+ characters)
- [ ] HTTPS enabled in production
- [ ] Secure cookies enabled (`AUTH_SECURE_COOKIE=true`)
- [ ] Docker socket protected (not exposed to network)
- [ ] Database file has restricted permissions
- [ ] Regular backups configured
- [ ] Dependencies kept up to date

### For Multi-User Future
- [ ] Database schema updated with user_id
- [ ] All queries filter by user
- [ ] Ownership checks in all handlers
- [ ] Role-based access control implemented
- [ ] User management UI
- [ ] Admin dashboard
- [ ] Audit logging
- [ ] Rate limiting
- [ ] API versioning
- [ ] Security audit completed

## Responsible Disclosure

This is open-source software provided as-is. Security limitations are documented transparently.

**If you discover a security vulnerability:**
1. Do not open a public issue
2. Contact the maintainer directly
3. Provide details and reproduction steps
4. Allow reasonable time for fix before public disclosure

## Conclusion

The current security model is **appropriate for single-user deployments** where one person manages all applications on their own infrastructure. Authentication via GitHub OAuth provides adequate protection against unauthorized external access.

**However**, this system is **NOT suitable for multi-user deployments** without implementing comprehensive authorization and resource isolation.

Choose your deployment model accordingly, and refer to the Migration Path section if multi-user support becomes necessary in the future.

---

**Last Updated:** 2026-01-20  
**Security Model Version:** 1.0 (Single-user)  
**Next Review:** When multi-user support is considered
