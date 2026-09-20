// Package buildinfo holds values stamped into the binary at build time.
package buildinfo

// Version is the release this binary was built from. The image build sets it with
// -ldflags "-X github.com/selfhostly/internal/buildinfo.Version=1.4.0". A plain `go build` reports "dev".
var Version = "dev"

// DevVersion is what Version holds when nothing was stamped in.
const DevVersion = "dev"
