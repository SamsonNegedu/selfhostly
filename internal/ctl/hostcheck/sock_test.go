package hostcheck

import "net"

func netListenUnix(path string) (net.Listener, error) { return net.Listen("unix", path) }
