# Volume Path Whitelist Configuration

## Overview

By default, Selfhostly blocks mounting certain host paths (like `/home`) in Docker Compose files for security reasons. However, you can configure a whitelist of trusted paths that should be allowed for your specific use case.

## Use Case

If you want all your applications to store data in a unified location for easier backups (e.g., `/home/user/Documents/apps`), you can whitelist this path to allow your apps to mount it.

## Configuration

Set the `ALLOWED_VOLUME_PATHS` environment variable with a comma-separated list of paths you want to whitelist:

```bash
ALLOWED_VOLUME_PATHS=/home/user/Documents/opq,/home/user/backup
```

### Example `.env` file:

```env
# Allow apps to mount paths under your apps directory
ALLOWED_VOLUME_PATHS=/home/user/Documents/opq

# Other configuration...
SERVER_ADDRESS=:8080
DATABASE_PATH=./data/selfhostly.db
```

## How It Works

1. **Critical paths are ALWAYS blocked** - Even if whitelisted, the following paths can never be mounted:
   - `/var/run/docker.sock` (Docker socket)
   - `/` (root filesystem)
   - `/etc` (system configuration)
   - `/root` (root home directory)
   - `/sys`, `/proc`, `/dev` (kernel interfaces)
   - `/boot` (boot partition)
   - `/var/lib/docker` (Docker internal storage)
   - `/var/lib/kubelet`, `/var/lib/rancher` (orchestration storage)

2. **Whitelist overrides non-critical restrictions** - The whitelist can override blocks on:
   - `/home/*` paths (user directories)
   - Other non-critical paths

3. **Subdirectories are automatically included** - If you whitelist `/home/user/Documents/apps`, then:
   - `/home/user/Documents/apps` is allowed
   - `/home/user/Documents/apps/app1` is allowed
   - `/home/user/Documents/apps/app1/data` is allowed
   - But `/home/user/Documents/other` is still blocked

## Examples

### Example 1: Unified Backup Directory

**Scenario**: You want all apps to store data in `/home/user/Documents/opq` for unified backups.

**Configuration**:
```env
ALLOWED_VOLUME_PATHS=/home/user/Documents/opq
```

**Docker Compose** (now allowed):
```yaml
version: '3.8'
services:
  finkit:
    image: myapp/finkit:latest
    volumes:
      - /home/user/Documents/opq/finkit/data:/data
      - /home/user/Documents/opq/finkit/config:/config
    ports:
      - "8080:8080"
```

### Example 2: Multiple Whitelisted Paths

**Scenario**: You have separate directories for app data and backups.

**Configuration**:
```env
ALLOWED_VOLUME_PATHS=/home/user/apps,/home/user/backups,/mnt/external/data
```

**Docker Compose** (now allowed):
```yaml
version: '3.8'
services:
  app:
    image: nginx
    volumes:
      - /home/user/apps/nginx/html:/usr/share/nginx/html
      - /home/user/backups/nginx:/backups
      - /mnt/external/data:/data
```

### Example 3: What's Still Blocked

Even with whitelisting, these are NEVER allowed:

```yaml
version: '3.8'
services:
  # BLOCKED: Docker socket (critical path)
  attacker:
    image: alpine
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock

  # BLOCKED: Root filesystem (critical path)
  bad:
    image: alpine
    volumes:
      - /:/host

  # BLOCKED: Not in whitelist
  unauthorized:
    image: alpine
    volumes:
      - /home/otheruser/data:/data
```

## Security Considerations

1. **Be specific with your whitelist** - Only whitelist the exact paths you need. Don't whitelist broad paths like `/home` or `/home/user`.

2. **Understand the risks** - Whitelisting a path means all apps can read/write to it. Make sure you trust the Docker images you're deploying.

3. **Critical paths cannot be overridden** - The system will always block mounting critical system paths, even if you add them to the whitelist.

4. **Path traversal is prevented** - The system uses `filepath.Clean()` to resolve `..` and `.` in paths, so attempts to escape the whitelist via path traversal are blocked.

5. **Backup your data** - Since multiple apps can access the whitelisted paths, ensure you have proper backups in place.

## Troubleshooting

### Error: "mounting /home paths is not allowed"

**Problem**: You're trying to mount a path under `/home` but it's not whitelisted.

**Solution**: Add the path to `ALLOWED_VOLUME_PATHS`:
```env
ALLOWED_VOLUME_PATHS=/home/youruser/Documents/apps
```

### Error: "mounting ... is not allowed (grants full Docker control)"

**Problem**: You're trying to mount a critical path like `/var/run/docker.sock`.

**Solution**: This cannot be whitelisted for security reasons. These paths are always blocked:
- Docker socket
- Root filesystem
- System directories (`/etc`, `/sys`, `/proc`, `/dev`)
- Docker internal storage

### Whitelist Not Working

1. **Check environment variable syntax**:
   ```env
   # Correct
   ALLOWED_VOLUME_PATHS=/home/user/data,/mnt/backup
   
   # Incorrect (no spaces after commas)
   ALLOWED_VOLUME_PATHS=/home/user/data, /mnt/backup
   ```

2. **Restart the application** after changing the environment variable.

3. **Check the path is correct** - Use absolute paths, not relative paths.

4. **Verify subdirectory structure** - If you whitelist `/home/user/data`, then `/home/user/data/app1` is allowed, but `/home/user/other` is not.

## Best Practices

1. **Use a dedicated directory structure**:
   ```
   /home/user/apps/
   ├── app1/
   │   ├── data/
   │   └── config/
   ├── app2/
   │   ├── data/
   │   └── config/
   └── backups/
   ```

2. **Set proper permissions**:
   ```bash
   mkdir -p ~/apps
   chmod 755 ~/apps
   ```

3. **Document your whitelist** in your deployment documentation.

4. **Use named volumes when possible** - They don't require whitelisting:
   ```yaml
   volumes:
     - app_data:/data  # Named volume, always allowed
   
   volumes:
     app_data:
   ```

5. **Consider alternatives**:
   - Use `/opt/apps` instead of `/home/user/apps` (no whitelist needed)
   - Use `/data/apps` instead of `/home/user/apps` (no whitelist needed)
   - Use Docker named volumes (no whitelist needed)

## Migration Guide

If you have existing apps using `/home` paths:

1. **Option A: Add whitelist** (recommended if you want unified backups):
   ```env
   ALLOWED_VOLUME_PATHS=/home/user/Documents/apps
   ```

2. **Option B: Move data to allowed paths**:
   ```bash
   # Move data to /opt
   sudo mkdir -p /opt/apps
   sudo chown $USER:$USER /opt/apps
   mv ~/Documents/apps/* /opt/apps/
   
   # Update compose files
   # Before: /home/user/Documents/apps/app:/data
   # After:  /opt/apps/app:/data
   ```

3. **Option C: Use named volumes**:
   ```yaml
   # Before
   volumes:
     - /home/user/Documents/apps/app:/data
   
   # After
   volumes:
     - app_data:/data
   
   volumes:
     app_data:
   ```

## Related Documentation

- [Security Documentation](./SECURITY.md) - Full security model and blocked configurations
- [Security Quick Reference](./SECURITY_QUICK_REFERENCE.md) - Quick guide to security validations
