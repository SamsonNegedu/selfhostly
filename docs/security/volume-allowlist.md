# Volume paths

What an app may bind-mount from the host, and how to allow more. The rules apply when an app is created or redeployed, and
`selfhostlyctl doctor --audit-apps` applies them to what is already deployed.

## Where an app's data may live

| Where | Result |
|---|---|
| Inside the app's own folder (`./data:/data`) | Always allowed. This is the default. |
| A folder listed in `ALLOWED_VOLUME_PATHS` (or anywhere beneath it) | Allowed. |
| Anywhere else on the host | Logged in `warn` mode, blocked in `enforce` mode. |
| A path with a variable (`${DATA}/x`) or `~` | Cannot be verified: logged in `warn`, blocked in `enforce`. Use a literal path. |
| Docker's socket, `/etc`, `/root` and the other paths listed [below](#never-allowed) | Never allowed, in any mode. |
| A relative path that climbs out of the app folder (`../x`), or a symlink under the apps folder that leaves it | Never allowed, in any mode. |

## Allow a folder

```env
ALLOWED_VOLUME_PATHS=/home/me/media,/mnt/external/data
```

Comma-separated, absolute. Subfolders are included: allowing `/home/me/media` allows `/home/me/media/tv/data`, not
`/home/me/other`. Apply it with `selfhostlyctl upgrade --set ALLOWED_VOLUME_PATHS=...` (this **replaces** the whole value, so
include the folders you already allow).

Find what needs allowing before you switch to `enforce`: `selfhostlyctl doctor --audit-apps`. It checks only apps registered
in the database and groups every folder that just needs allowing into one finding with the exact value to set, keeping
the folders you already allow.

## Keep the data in the app folder, or allow an outside folder?

- *In the folder*: nothing to configure. But deleting the app deletes its data. If the container writes files as another user
  (Redis, databases), Selfhostly cannot remove them, so a delete leaves them behind for you to remove with `sudo`. If the app
  uses `build:`, the data sits inside the Docker build context and the build can fail with "no permission to read": put the
  folder's name in a `.dockerignore` next to the app's compose file.
- *Outside, and allowed*: the data survives deleting the app and stays out of build contexts. It costs one line in
  `ALLOWED_VOLUME_PATHS`.

## Never allowed

The list cannot override these: mounting one, or a folder that contains one, is refused.

| Path | Why |
|---|---|
| `/var/run/docker.sock`, `/run/docker.sock`, `/var/run/docker`, `/run/docker` | Full control of the host |
| `/root`, `/etc`, `/boot`, `/sys`, `/proc`, `/dev`, `/host` | System and kernel interfaces |
| `/var/lib/docker`, `/var/lib/kubelet`, `/var/lib/rancher` | Container runtime storage |
| `/usr`, `/bin`, `/sbin`, `/lib`, `/lib64` themselves | The system (a subfolder is judged on its own) |
| `/`, `/var`, `/run` and other folders that contain one of the above | Mounting them would expose it |

The policy is a blocklist plus this allow-list, not a sandbox: see the limitations in the
[security overview](overview.md).

## Example

```env
ALLOWED_VOLUME_PATHS=/home/me/apps,/mnt/external/data
```

```yaml
services:
  app:
    image: nginx
    volumes:
      - ./config:/etc/nginx/conf.d          # in the app folder: always fine
      - /home/me/apps/nginx/html:/html      # under an allowed folder
      - /mnt/external/data:/data            # an allowed folder
      # - /var/run/docker.sock:/s           # never allowed
```

## Messages and what to do

| Message | Do this |
|---|---|
| `... is outside the app directory and ALLOWED_VOLUME_PATHS` | Move the data into the app folder, or add the folder to `ALLOWED_VOLUME_PATHS`. |
| `... must never be mounted into an app` or `... is inside /...` or `... contains /...` | Remove that mount. Nothing allows it. |
| `... contains a variable, so its host path cannot be verified` | Write the path out. The policy reads the file before variables are filled in. |
| `... uses ~` | Write the absolute path. |
| `... escapes the app directory` | A `../` path. Use a folder inside the app, or an allowed absolute path. |
| `... resolves through a symlink that leaves the apps directory` | A folder under the apps directory is a symlink pointing elsewhere. Mount the real path and allow it. |
| The policy cannot judge a path at all | `HOST_APPS_DIR` is unknown. Set it to the host path of the apps folder (`selfhostlyctl doctor` shows whether it was detected). |
