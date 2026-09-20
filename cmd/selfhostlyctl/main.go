// selfhostlyctl sets up, joins, upgrades and checks a Selfhostly install.
package main

import (
	"os"

	"github.com/selfhostly/internal/ctl"
)

func main() { os.Exit(ctl.Execute(os.Args[1:])) }
