// Package selfhostly carries the deployment files that ship with a release, so selfhostlyctl can write
// them on a machine that has no clone of the repository.
package selfhostly

import "embed"

// Files are the compose files and the example settings file, exactly as in the repository
//
//go:embed docker-compose.prod.yml docker-compose.secondary.yml docker-compose.secondary-direct.yml docker-compose.socket-proxy.yml env.example
var Files embed.FS
