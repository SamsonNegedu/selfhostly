# GitHub Username Whitelist

## Overview

The GitHub username whitelist is a security feature that restricts access to your selfhostly instance to specific GitHub users only.

## How It Works

When GitHub OAuth is enabled:

1. User clicks "Continue with GitHub" on login page
2. Redirects to GitHub for OAuth authorization
3. GitHub validates credentials and redirects back
4. Backend receives user info from GitHub
5. **Backend checks if username is in whitelist**
6. If whitelisted → JWT cookie set, access granted ✅
7. If not whitelisted → No cookie set, access denied ❌
8. **Frontend detects failed authentication and shows error message**

### UI Enforcement

The frontend provides visual feedback when whitelist check fails:

- Detects when OAuth completes but no valid session is established
- Automatically redirects to login page with error parameters
- Displays clear error message explaining why access was denied
- Shows security notice about whitelist requirement
- Provides guidance on contacting administrator

## Configuration

### Required Environment Variables

```bash
AUTH_ENABLED=true
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_ALLOWED_USERS=your-username,trusted-friend,family-member
AUTH_BASE_URL=https://your-domain.com
AUTH_SECURE_COOKIE=true
```

### Finding Your GitHub Username

Your GitHub username is:
- The name in your profile URL: `https://github.com/YOUR-USERNAME`
- Shown on your GitHub profile page
- **Case-insensitive** - `JohnDoe`, `johndoe`, and `JOHNDOE` all work

Example: If your profile is `https://github.com/JohnDoe`, you can use any of:
- `JohnDoe` ✅
- `johndoe` ✅
- `JOHNDOE` ✅

### Adding Multiple Users

Separate usernames with commas (no spaces):

```bash
# Good ✅
GITHUB_ALLOWED_USERS=alice,bob,charlie

# Also works ✅
GITHUB_ALLOWED_USERS=alice, bob, charlie

# Spaces are trimmed automatically
```

## Security Features

### 1. Fail-Secure Design

If `GITHUB_ALLOWED_USERS` is not set or empty, **all access is denied**:

```bash
# This will reject ALL login attempts
AUTH_ENABLED=true
GITHUB_ALLOWED_USERS=
```

This prevents accidentally exposing your system if you forget to configure the whitelist.

### 2. Logging

The system logs authentication attempts:

**Backend logs:**
```
INFO: User authorized, username=alice
WARN: Unauthorized GitHub user attempted access, username=mallory
WARN: GitHub auth enabled but no allowed users configured - rejecting access
```

**Frontend feedback:**
- Error alert displayed on login page after failed OAuth
- Clear message: "Access denied: Your GitHub account is not authorized"
- Security notice explaining whitelist requirement
- Dismissible alert that auto-clears URL parameters

### 3. No Partial Access

There's no "read-only" or "limited" access. Whitelisted users have **full access** to:
- View all apps
- Create/update/delete apps
- Start/stop apps
- Modify global settings
- Manage Cloudflare tunnels

This is a **single-user design** - the whitelist just lets you specify which GitHub accounts YOU use.

## Examples

### Example 1: Solo Developer

```bash
# Just you
GITHUB_ALLOWED_USERS=your-github-username
```

### Example 2: Personal + Work Accounts

```bash
# Allow both your accounts
GITHUB_ALLOWED_USERS=john-personal,john-work
```

### Example 3: Shared with Trusted Person

```bash
# You and your spouse/partner
GITHUB_ALLOWED_USERS=alice,bob
```

⚠️ **Warning:** Both users have **full control** over everything. Only add people you completely trust.

## Common Issues

### Issue 1: Login Rejected Despite Being in Whitelist

**Cause:** Username not actually in whitelist or typo

**Solution:**
1. Go to `https://github.com/YOUR-USERNAME`
2. Copy the username from the URL
3. Check for typos in your `.env` file
4. Case doesn't matter - `JohnDoe` and `johndoe` are treated the same

```bash
# All of these work ✅
GITHUB_ALLOWED_USERS=JohnDoe
GITHUB_ALLOWED_USERS=johndoe
GITHUB_ALLOWED_USERS=JOHNDOE

# Common mistakes ❌
GITHUB_ALLOWED_USERS=John Doe     # No spaces allowed
GITHUB_ALLOWED_USERS="johndoe"    # Don't use quotes
GITHUB_ALLOWED_USERS=@johndoe     # No @ symbol
```

### Issue 2: All Logins Rejected

**Cause:** Empty or missing `GITHUB_ALLOWED_USERS`

**Solution:** Set the environment variable:

```bash
GITHUB_ALLOWED_USERS=your-github-username
```

Check server logs:
```
WARNING: GitHub auth enabled but no allowed users configured - all access will be denied
```

### Issue 3: Username with Special Characters

GitHub usernames can only contain alphanumeric characters and hyphens. If your username contains these, no escaping needed:

```bash
# These all work fine
GITHUB_ALLOWED_USERS=john-doe,alice_123,bob-dev-2024
```

## Testing Your Configuration

### Step 1: Start the Server

```bash
go run cmd/server/main.go
```

### Step 2: Check Logs

You should see:
```
INFO: Auth enabled: true
INFO: GitHub OAuth configured: ClientID=Ov23liOs...
INFO: GitHub whitelist configured: 2 user(s) allowed
```

If you see this warning:
```
WARNING: GitHub auth enabled but no allowed users configured - all access will be denied
```

Then fix your `GITHUB_ALLOWED_USERS` configuration.

### Step 3: Test Login

1. Navigate to `http://localhost:8080` (or your domain)
2. Click "Continue with GitHub"
3. Authorize the app on GitHub
4. Check results:

**Success:**
- Redirected to dashboard
- Backend logs: `INFO: User authorized, username=your-username`
- User can access all features

**Failure (Not in Whitelist):**
- Redirected back to login page
- **Red error alert displayed:**
  ```
  Access Denied
  Access denied: Your GitHub account is not authorized to access this system.
  
  Security Notice
  Only specific GitHub accounts are authorized to access this system.
  If you believe you should have access, contact your system administrator
  to add your GitHub username to the whitelist.
  ```
- Backend logs: `WARN: Unauthorized GitHub user attempted access, username=wrong-user`

## Updating the Whitelist

### Adding Users

1. Stop the server
2. Update `GITHUB_ALLOWED_USERS` in `.env`
3. Restart the server

**No database changes needed** - this is configured purely via environment variables.

### Removing Users

1. Remove username from `GITHUB_ALLOWED_USERS`
2. Restart the server
3. Removed users will be denied on next login attempt

**Note:** Existing sessions may remain valid until JWT expires (default: 24 hours)

## Security Best Practices

### ✅ Do This

- **Use strong JWT secret** (32+ random characters)
- **Enable HTTPS** (`AUTH_SECURE_COOKIE=true`)
- **Limit whitelist** to only accounts you control or completely trust
- **Review logs** periodically for unauthorized attempts
- **Use different secrets** for dev/staging/production

### ❌ Don't Do This

- Don't share your `.env` file
- Don't commit secrets to git
- Don't add users you don't fully trust (they get FULL access)
- Don't use weak JWT secrets
- Don't expose without HTTPS in production
- Don't reuse JWT secrets across environments

## Comparison with Other Solutions

### GitHub Whitelist vs No Auth

| Feature | No Auth | GitHub Whitelist |
|---------|---------|------------------|
| Setup complexity | ⭐ Easy | ⭐⭐ Medium |
| Security | ❌ None | ✅ User restriction |
| Internet required | No | Yes (GitHub OAuth) |
| Best for | Local network only | Internet-exposed |

### GitHub Whitelist vs Cloudflare Zero Trust

| Feature | GitHub Whitelist | Cloudflare Zero Trust |
|---------|------------------|----------------------|
| Setup complexity | ⭐⭐ Medium | ⭐⭐⭐ More complex |
| Security | ✅ Good | ✅✅ Excellent |
| Edge protection | ❌ No | ✅ Yes |
| Extra features | None | DDoS, WAF, etc. |
| Best for | Simple deployments | Production/serious use |

**Recommendation:** Use Cloudflare Zero Trust for production, or GitHub whitelist for simpler deployments.

## Troubleshooting

### Enable Debug Logging

Check what the system sees:

1. Look at server startup logs
2. Check authentication attempt logs
3. Verify user count shows correctly

### Verify Configuration

```bash
# Print your configuration (sanitized)
echo "Auth enabled: $AUTH_ENABLED"
echo "Allowed users count: $(echo $GITHUB_ALLOWED_USERS | tr ',' '\n' | wc -l)"
echo "Users: $GITHUB_ALLOWED_USERS"  # Don't share this output publicly
```

### Test GitHub OAuth Setup

Before testing whitelist, verify GitHub OAuth works:

1. Temporarily add a test user to whitelist
2. Have them try to log in
3. Check what username appears in logs
4. Use that exact username in your whitelist

## FAQ

### Q: Can I use GitHub organizations?

**A:** Not directly. You must list individual usernames. Organizations/teams are not supported.

### Q: What happens if I remove all users?

**A:** All logins will be rejected (fail-secure). The system logs a warning on startup.

### Q: Can I use emails instead of usernames?

**A:** No, must be GitHub usernames. The system validates against `claims.User.Name`.

### Q: How do I know my GitHub username?

**A:** Visit `https://github.com/YOUR-USERNAME` - the username is in the URL.

### Q: Does this work with GitHub Enterprise?

**A:** The current implementation uses public GitHub OAuth. GitHub Enterprise would require code changes.

### Q: Can I whitelist by organization membership?

**A:** Not currently. Would require additional GitHub API calls and permissions.

---

**Last Updated:** 2026-01-20  
**Related Documentation:**
- [Security Documentation](./SECURITY.md)
- [Integration Guide](./INTEGRATION_GUIDE.md)
- [Main README](../README.md)
