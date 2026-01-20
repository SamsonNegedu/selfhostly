# Documentation Index

Welcome to the selfhost-automaton documentation.

## Getting Started

- **[Main README](../README.md)** - Project overview, quick start, and feature list
- **[General Idea](../general-idea.md)** - Original concept and motivation

## Setup & Configuration

- **[Cloudflare Zero Trust Setup](./CLOUDFLARE_ZERO_TRUST.md)** - **⭐ RECOMMENDED** - Deploy with edge authentication (no OAuth needed)
- **[Integration Guide](./INTEGRATION_GUIDE.md)** - Alternative: GitHub OAuth setup (more overhead)
- **[GitHub Whitelist Guide](./GITHUB_WHITELIST.md)** - Restrict access to specific GitHub users (required if using GitHub OAuth)

## Security & Architecture

- **[Security Documentation](./SECURITY.md)** - **⚠️ IMPORTANT: Read this before deploying**
  - Current security model (single-user design)
  - Authentication vs Authorization
  - Known limitations and vulnerabilities
  - Multi-user migration path
  - Deployment recommendations
  - Security checklist

## Project Improvements

- **[Improvements & Roadmap](../improvements.md)** - Known issues, planned features, and future enhancements

## Key Documentation Files

### 📘 SECURITY.md
**Must-read for anyone deploying this system.**

This document explains:
- Why this system is designed for **single-user only**
- What "authentication without authorization" means
- Security implications and acceptable use cases
- How to migrate to multi-user (if needed in future)

**TL;DR:** Any authenticated user can see and manage ALL resources. This is by design for personal Raspberry Pi hosting.

### 📗 INTEGRATION_GUIDE.md
**Technical guide for authentication setup.**

Covers:
- GitHub OAuth configuration
- Environment variables
- Authentication flow
- Frontend integration examples
- Troubleshooting

### 📗 GITHUB_WHITELIST.md
**Guide for restricting access with GitHub username whitelist.**

Covers:
- How the whitelist works
- Configuration and setup
- Security features (fail-secure design)
- Adding/removing users
- Common issues and troubleshooting
- Best practices

## Architecture Overview

```
selfhost-automaton/
├── cmd/server/          # Backend entry point
├── internal/            # Private application code
│   ├── config/         # Configuration management
│   ├── db/             # Database models and operations
│   ├── docker/         # Docker compose operations
│   ├── cloudflare/     # Cloudflare API integration
│   ├── http/           # HTTP handlers and middleware
│   ├── cleanup/        # Resource cleanup logic
│   └── domain/         # Domain entities
├── web/                # Frontend (React + TypeScript)
│   └── src/
│       ├── features/   # Feature-based components
│       └── shared/     # Shared utilities
└── docs/               # Documentation (you are here)
```

## Security Model Summary

| Aspect | Status | Details |
|--------|--------|---------|
| **Authentication** | ✅ Implemented | GitHub OAuth via go-pkgz/auth |
| **Authorization** | ❌ Not Implemented | All users see all resources |
| **Multi-User** | ❌ Not Supported | Single-user design by choice |
| **Resource Isolation** | ❌ None | No user_id on resources |

**Use Case:** Personal infrastructure management (Raspberry Pi, home lab)  
**Not For:** Multi-tenant SaaS, shared infrastructure, team environments

## Quick Links

### For Users
- [Quick Start](../README.md#-quick-start-docker)
- [Configuration Guide](../README.md#️-configuration)
- [Security Notice](./SECURITY.md)

### For Developers
- [Development Setup](../README.md#-development-setup)
- [API Endpoints](../README.md#api-endpoints)
- [Contributing & Improvements](../improvements.md)

### For Security Auditors
- [Security Documentation](./SECURITY.md)
- [Known Vulnerabilities](./SECURITY.md#known-vulnerabilities)
- [Migration Path](./SECURITY.md#migration-path-future)

## Need Help?

1. Check the [Security Documentation](./SECURITY.md) if you have questions about:
   - Multi-user support
   - Resource access control
   - Deployment scenarios

2. Review [Improvements](../improvements.md) for known issues and planned features

3. Check the [Integration Guide](./INTEGRATION_GUIDE.md) for authentication setup issues

## Contributing

See [improvements.md](../improvements.md) for areas that need work, including:
- Error handling improvements
- UI/UX enhancements
- Feature additions
- Multi-user support (major undertaking)

---

**Last Updated:** 2026-01-20
