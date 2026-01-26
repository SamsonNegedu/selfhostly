# Deploying with Cloudflare Zero Trust

This is the **recommended authentication method** for selfhostly. It's simpler, more secure, and requires zero OAuth configuration.

## Why Cloudflare Zero Trust?

### ✅ Advantages Over GitHub OAuth

| Feature | Cloudflare Zero Trust | GitHub OAuth |
|---------|----------------------|--------------|
| **Setup Complexity** | Simple (3 steps) | Complex (OAuth app + secrets) |
| **Identity Providers** | Multiple (Google, GitHub, email, SSO) | GitHub only |
| **Secret Management** | None needed | JWT secret required |
| **Cookie/Token Handling** | Handled by Cloudflare | App-level management |
| **Audit Logs** | Built-in | Need to implement |
| **Session Management** | Managed by Cloudflare | App-level |
| **Cost** | Free (up to 50 users) | Free |
| **Security** | Edge-level, before app | App-level |
| **Works with Tunnels** | Perfect integration | Independent |

### Perfect for Single-User Deployments

Since selfhostly is designed for single-user use (you managing your own apps), Cloudflare Zero Trust is ideal:
- You're already using Cloudflare Tunnels for your apps
- No need to maintain OAuth apps or JWT secrets
- Cloudflare authenticates before traffic reaches your server
- Zero Trust security model is perfect for personal infrastructure

## Prerequisites

1. Cloudflare account (free tier works)
2. Domain managed by Cloudflare
3. Cloudflare Tunnel set up (you're already doing this!)

## Setup Guide

### Step 1: Enable Zero Trust

1. Go to your Cloudflare Dashboard
2. Navigate to **Zero Trust** (left sidebar)
3. If first time: Complete the onboarding wizard
   - Choose a team name (e.g., "samson-homelab")
   - This is free for up to 50 users

### Step 2: Create an Access Application

1. In Zero Trust dashboard, go to **Access → Applications**
2. Click **Add an application**
3. Choose **Self-hosted**

**Configure the application:**

```yaml
Application name: Selfhost selfhostly
Session Duration: 24 hours (or your preference)
Application domain: 
  - selfhostly.yourdomain.com
```

### Step 3: Configure Authentication

**Add an identity provider** (if you haven't already):

1. Go to **Settings → Authentication**
2. Click **Add new** under Login methods
3. Choose your preferred method:
   - **One-time PIN** (email-based, easiest)
   - **Google** (OAuth)
   - **GitHub** (OAuth)
   - Many others...

**For email OTP (simplest):**
- No configuration needed
- Just enter your email when logging in
- Receive a code, paste it in
- Done!

### Step 4: Create Access Policy

1. Back in your application settings
2. Under **Policies**, click **Add a policy**

**Policy Configuration:**

```yaml
Policy name: Allow myself
Action: Allow
Session duration: 24 hours

Include rules:
  - Selector: Emails
  - Value: your@email.com
```

Or for multiple emails:
```yaml
Include rules:
  - Selector: Emails
  - Value: email1@domain.com, email2@domain.com
```

Or for entire domain:
```yaml
Include rules:
  - Selector: Emails ending in
  - Value: @yourdomain.com
```

### Step 5: Configure Your App

Update your `.env` file:

```env
# Disable built-in authentication
AUTH_ENABLED=false

# Your Cloudflare credentials (for tunnel management)
CLOUDFLARE_API_TOKEN=your_api_token
CLOUDFLARE_ACCOUNT_ID=your_account_id
```

### Step 6: Deploy and Test

1. Deploy your app with the tunnel pointing to your domain
2. Visit `https://selfhostly.yourdomain.com`
3. You'll be redirected to Cloudflare's login page
4. Authenticate with your chosen method
5. You're in! 🎉

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Internet                                 │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ HTTPS Request
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Cloudflare Edge                               │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │          Cloudflare Zero Trust (Access)                    │ │
│  │  • Checks authentication                                   │ │
│  │  • Validates session                                       │ │
│  │  • If not authenticated → redirect to login                │ │
│  │  • If authenticated → pass through with user identity      │ │
│  └────────────────────────────────────────────────────────────┘ │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ Authenticated Request Only
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Cloudflare Tunnel                               │
│  • Secure connection to your infrastructure                      │
│  • No inbound ports needed                                       │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             │ Encrypted Tunnel
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│              Your Infrastructure (Raspberry Pi)                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │              Selfhost selfhostly                            │ │
│  │  • AUTH_ENABLED=false                                      │ │
│  │  • No OAuth configuration needed                           │ │
│  │  • Cloudflare already authenticated the user               │ │
│  │  • App just serves content                                 │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

**Key Benefits:**
- ✅ Authentication happens at Cloudflare's edge (before your server)
- ✅ Unauthenticated requests never reach your infrastructure
- ✅ Your app remains simple (no auth code complexity)
- ✅ Perfect security model for personal infrastructure

## Access Policies Examples

### Single User (You)

```yaml
Policy: Allow only me
Action: Allow
Include:
  - Emails: your@email.com
```

### Multiple Specific Users

```yaml
Policy: Allow team
Action: Allow
Include:
  - Emails: admin@example.com, user@example.com
```

### Email Domain

```yaml
Policy: Allow organization
Action: Allow
Include:
  - Emails ending in: @yourcompany.com
```

### IP Restriction (Extra Security)

```yaml
Policy: Allow from home network
Action: Allow
Include:
  - Emails: your@email.com
  - IP ranges: 1.2.3.4/32
```

### Time-Based Access

```yaml
Policy: Business hours only
Action: Allow
Include:
  - Emails: your@email.com
Require:
  - Time: Monday-Friday, 9am-5pm UTC
```

## Advanced Configuration

### Session Duration

You can configure how long users stay logged in:
- **24 hours** (recommended for personal use)
- **7 days** (convenient for frequently used apps)
- **30 minutes** (high security)

### Multiple Applications

You can protect multiple apps with different policies:
```yaml
Application 1: selfhostly.domain.com
  - Policy: Allow admin only

Application 2: app1.domain.com  
  - Policy: Allow anyone in @family.com

Application 3: public-demo.domain.com
  - Policy: Allow everyone (but collect email)
```

### Audit Logs

View who accessed your apps and when:
1. Go to **Logs → Access**
2. See all authentication attempts
3. Filter by application, user, or date

## Troubleshooting

### "Access Denied" Error

**Check:**
- Your email is in the Access policy
- The policy action is "Allow" not "Block"
- Session hasn't expired

**Fix:**
1. Go to Access → Applications
2. Edit your application
3. Check the policy includes your email
4. Try logging out and back in

### Redirect Loop

**Causes:**
- Application domain doesn't match tunnel domain
- DNS not pointing to Cloudflare

**Fix:**
1. Verify DNS is proxied (orange cloud)
2. Check tunnel routes match application domain
3. Clear browser cookies and try again

### Can't Access Login Page

**Check:**
- Domain is proxied through Cloudflare (orange cloud)
- Tunnel is running and connected
- DNS propagation complete (use `dig` or `nslookup`)

## Cost

Cloudflare Zero Trust is **FREE** for:
- Up to 50 users
- Unlimited applications
- Basic identity providers
- Standard audit logs

Perfect for personal use! 🎉

## Migration from GitHub OAuth

If you're currently using GitHub OAuth:

1. **Set up Cloudflare Zero Trust** (follow steps above)
2. **Test access** by visiting your app
3. **Once confirmed working**, update `.env`:
   ```env
   AUTH_ENABLED=false
   ```
4. **Remove OAuth environment variables** (no longer needed):
   ```env
   # Delete or comment out:
   # GITHUB_CLIENT_ID=...
   # GITHUB_CLIENT_SECRET=...
   ```
5. **Restart your application**

That's it! Much simpler. ✨

## Comparison with Other Auth Methods

| Method | Setup | Maintenance | Security | Cost |
|--------|-------|-------------|----------|------|
| **Cloudflare Zero Trust** | ⭐⭐⭐ Easy | ⭐⭐⭐ Minimal | ⭐⭐⭐ Edge-level | Free |
| GitHub OAuth | ⭐⭐ Moderate | ⭐⭐ Some overhead | ⭐⭐ App-level | Free |
| Basic Auth | ⭐⭐⭐ Very easy | ⭐⭐⭐ Minimal | ⭐ Weak (no MFA) | Free |
| VPN | ⭐ Complex | ⭐ High maintenance | ⭐⭐⭐ Network-level | Varies |

## Resources

- [Cloudflare Zero Trust Documentation](https://developers.cloudflare.com/cloudflare-one/)
- [Access Policies Guide](https://developers.cloudflare.com/cloudflare-one/policies/access/)
- [Identity Providers Setup](https://developers.cloudflare.com/cloudflare-one/identity/idp-integration/)

## Summary

**For single-user selfhostly deployments, Cloudflare Zero Trust is the ideal choice:**

✅ **Simple**: 3 steps, no OAuth configuration  
✅ **Secure**: Authentication at the edge, zero trust model  
✅ **Integrated**: Already using Cloudflare Tunnels  
✅ **Free**: Up to 50 users  
✅ **Flexible**: Multiple identity providers  
✅ **Maintainable**: No secrets to rotate, no cookies to manage  

**Bottom line:** If you're using Cloudflare Tunnels (and you should be), use Cloudflare Zero Trust. It's the best choice.

---

**Questions?** See [SECURITY.md](./SECURITY.md) for architecture details or [INTEGRATION_GUIDE.md](./INTEGRATION_GUIDE.md) for GitHub OAuth alternative.
