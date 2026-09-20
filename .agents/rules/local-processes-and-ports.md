# Local Processes and Ports

## Overview

Rules for starting, stopping and testing against local servers on a developer's machine. This machine
also runs other people's tools: Docker or OrbStack, databases, other projects' dev servers. A port that
looks like "mine" may belong to one of them. Stopping the wrong process breaks the developer's
environment, and it cannot be undone from here.

Written after an agent ran `lsof -ti tcp:8080 | xargs kill` to clear "address already in use". The
port was held by OrbStack's port forwarder, so OrbStack's networking went down more than once in one
session.

## Core Principles

**Never stop a process by port number.** Do not pipe `lsof`, `fuser` or `netstat` output into `kill`.
A port tells you nothing about who owns it. Only stop a process you started yourself, by the PID or
task ID you recorded when you started it.

**When a port is taken, look, then report.** Run `lsof -nP -iTCP:<port> -sTCP:LISTEN` (or
`docker ps` for published container ports), name the owning process to the user, and pick another
port. Do not free the port yourself, even if the owner looks stale. Only the user decides to stop
something they own.

**Test servers use their own ports, never the defaults.** Anything an agent starts for testing
(backend, Vite, fixtures) runs on a high, unusual port set through configuration (`SERVER_ADDRESS`,
`--port`), so it cannot collide with a developer's stack on 8080, 5173, 3000 and similar. Record the
port in the fixture docs.

**Track what you start.** Start long-running servers as background tasks and keep their IDs. Stop
them through the task tool, or with `kill <pid>` for a PID you captured (`$!`). Confirm with the
recorded PID that it is yours before killing it. At the end of the work, stop exactly those and
list what you stopped.

**Never touch container tooling.** Do not stop, restart or kill Docker, OrbStack, Colima, Podman or
their helper processes, and do not run `docker rm`, `docker kill`, `docker system prune` or
`docker compose down -v` against anything the agent did not create in this session. If containers
seem stuck, tell the user.

**Say what a command will affect before running anything broad.** `pkill`, `killall`, `kill -9`
on an unfamiliar PID, and any `xargs kill` are broad. If one seems necessary, show the process it
would hit and ask first.

## Examples

**Bad:**

```bash
lsof -ti tcp:8080 | xargs kill
```

**Good:**

```bash
lsof -nP -iTCP:8080 -sTCP:LISTEN     # who owns it? report it, then use another port
make backend ENV_FILE=docs/dev-fixtures/env.fixtures   # fixture config sets SERVER_ADDRESS=:8090
```

**Bad:** restarting OrbStack or Docker to "fix" a port conflict.

**Good:** "Port 8080 is held by OrbStack Helper (pid 1234), so I started the test server on 8090."

## Scope

Applies to every local process an agent starts or is tempted to stop: dev servers, backends, databases,
browsers driven for tests, and container tooling.
