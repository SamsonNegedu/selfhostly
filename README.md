# Selfhostly

<p>
	<a href="https://github.com/user-attachments/assets/4d8df016-2bbe-4975-9dfd-474a79b12c85" target="_blank">
		<img alt="Screenshot 2026-02-10 at 20 57 15" src="https://github.com/user-attachments/assets/4d8df016-2bbe-4975-9dfd-474a79b12c85" width="45%" height="auto">
	</a>
	<a href="https://github.com/user-attachments/assets/822001e0-ab5c-42e7-9b08-e993b7bc4283" target="_blank">
		<img alt="Screenshot 2026-02-10 at 20 59 52" src="https://github.com/user-attachments/assets/822001e0-ab5c-42e7-9b08-e993b7bc4283" width="45%" height="auto">
	</a>
</p>

A web platform for running self-hosted apps on a Raspberry Pi or Linux server. Deploy Docker Compose apps,
publish them through Cloudflare tunnels, watch resource use, and manage containers, across one or more machines,
from one place.

- **Apps:** deploy and edit Compose files, with version history and rollback, updates and scheduled start and stop
- **Several machines:** add secondary nodes and manage all of them from one UI
- **Access:** create Cloudflare tunnels, routes and DNS records from the UI
- **Insights:** CPU, memory, disk and per-container usage, with warnings

## Security notice

**Single-user design only.** It is meant for personal use, such as a home server. It is not suitable for
multi-user or multi-tenant setups.

Put it behind [Cloudflare Zero Trust](docs/operations/cloudflare-zero-trust.md) (recommended) or turn on
[GitHub sign-in with an allow list](docs/security/github-allowlist.md). See the
[security overview](docs/security/overview.md) for the model and its known limits.

## Requirements

Docker 20.10+ with Compose, on Linux or macOS (Windows via WSL2), amd64 or arm64 (Raspberry Pi 4 and 5 work),
and at least 1 GB of RAM. A Cloudflare account is optional and only needed for tunnels.

## Quick start

`selfhostlyctl` is the command line for installing and operating it. Install it, then run the guided setup:

```bash
curl -fsSL https://raw.githubusercontent.com/samsonnegedu/selfhostly/main/scripts/install.sh | sh
selfhostlyctl setup      # checks the machine, writes .env, starts everything
```

Or by hand:

```bash
git clone https://github.com/samsonnegedu/selfhostly.git && cd selfhostly
cp env.example .env      # then edit it
make prod
```

Open `http://localhost:8080`. Every `selfhostlyctl` command explains itself with `--help`.

## Documentation

Everything else lives in [`docs/`](docs/README.md):

- **Deploy and run:** [deploy](docs/operations/deploy.md),
  [upgrade an existing install](docs/operations/upgrade.md) (keeps your edits and rolls back on failure),
  [safe restart](docs/operations/safe-restart.md), [troubleshooting](docs/operations/troubleshooting.md)
- **Several machines:** [multi-node](docs/operations/multi-node.md),
  [how secondary nodes connect](docs/rfcs/node-link.md)
- **Settings:** [`env.example`](env.example) lists every environment variable
- **Develop:** [development guide](docs/development/getting-started.md). `make help` lists the commands.
- **Design:** [architecture](docs/design/architecture.md), [UI](docs/design/ui.md)

## Built with

Go, Gin, SQLite ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), pure Go), the Docker API,
[gopsutil](https://github.com/shirou/gopsutil), and [go-pkgz/auth](https://github.com/go-pkgz/auth) on the backend.
React, TypeScript, TanStack Query, Tailwind, [Radix UI](https://www.radix-ui.com/) and
[CodeMirror](https://codemirror.net/) on the frontend.

## Contributing

Pull requests are welcome. Read [`AGENTS.md`](AGENTS.md) first: it points to the project's rules and conventions.

## License

MIT, see [LICENSE](./LICENSE).
