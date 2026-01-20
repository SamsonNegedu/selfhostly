# Improvements & Known Issues

## Documentation
- [ ] Explanation of why cloudflare token is being generated multiple times in 1 app creation
- [x] **Document security model and single-user design limitation** (see `docs/SECURITY.md`)
- [x] **Recommend Cloudflare Zero Trust over GitHub OAuth** (see `docs/CLOUDFLARE_ZERO_TRUST.md`)
  - GitHub OAuth is unnecessary overhead when deploying behind CF Zero Trust
  - Simpler setup, better security, less code to maintain

## Error Handling & Reliability
- [ ] Look for error handling everywhere and handle with grace
- [ ] Better error messages for common issues (Docker not running, Cloudflare API failures)
- [ ] Retry logic for transient failures

## Configuration & Setup
- [ ] Apps directory should not be in current working directory....It should be stored in a dedicated directory like `/opt/selfhostly/apps`
- [ ] env.example should not be committed to the repository (move to docs or keep as template)
- [ ] Let's get started screen if any of the setup steps are missing
- [ ] Configuration validation on startup

## Cloudflare Integration
- [ ] Add cloudflare create-tunnel route
- [ ] Cleanup process should delete the cloudflare tunnel as well (currently implemented, verify)
- [ ] Support for custom domains with Cloudflare tunnels
- [ ] DNS record management UI

## Security Improvements (Future - Multi-User Support)
**⚠️ SECURITY LIMITATION: Single-user design only**  
See `docs/SECURITY.md` for full details and migration path.

Current state:
- ✅ Authentication via GitHub OAuth
- ❌ No resource-level authorization
- ❌ Any authenticated user can manage ALL resources

Future multi-user requirements:
- [ ] Add `user_id` to all resource models (App, CloudflareTunnel, Settings)
- [ ] Implement resource ownership checks in all API handlers
- [ ] Add `GetUserApps()`, `GetUserApp()` etc. to database layer
- [ ] Per-user settings instead of global settings
- [ ] Role-based access control (admin vs regular user)
- [ ] User management UI
- [ ] Audit logging for compliance
- [ ] API rate limiting per user

## UI/UX Improvements
- [ ] Better loading states and skeleton screens
- [ ] Real-time status updates via WebSocket or SSE
- [ ] Bulk operations (start/stop multiple apps)
- [ ] App templates/marketplace
- [ ] Better mobile responsiveness
- [ ] Dark mode support

## Features
- [ ] App health checks and automatic restart
- [ ] Backup and restore functionality
- [ ] Environment variable management per app
- [ ] Volume management UI
- [ ] Network configuration UI
- [ ] Container resource limits (CPU, memory)
- [ ] Application metrics and monitoring
- [ ] Notification system (email, webhook) for app failures

## Performance
- [ ] Database connection pooling
- [ ] Caching for frequently accessed data
- [ ] Pagination for app lists
- [ ] Background job queue for long-running operations

## DevOps & Operations
- [ ] Prometheus metrics endpoint
- [ ] Structured logging with log levels
- [ ] Health check endpoints for monitoring
- [ ] Graceful shutdown handling
- [ ] Database migrations system (currently inline in code)
- [ ] Automated testing (unit, integration, e2e)
- [ ] CI/CD pipeline
- [ ] Docker image optimization (multi-stage builds)

## Code Quality
- [ ] Add comprehensive tests
- [ ] API documentation (OpenAPI/Swagger)
- [ ] Code comments and inline documentation
- [ ] Consistent error handling patterns
- [ ] Input validation middleware
- [ ] Dependency updates and security scanning

---

**Legend:**
- [ ] Not started
- [x] Completed
- ⚠️ Known limitation by design
