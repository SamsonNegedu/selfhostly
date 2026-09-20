// Command composegen writes the compose file an end-to-end install runs from docker-compose.prod.yml. It renames
// what the file fixes (container and network names) so the stack cannot collide with a real install, and drops
// cloudflared, which the harness replaces with a reverse proxy. Nothing else changes: the install and the release
// images both carry this file, so the update flow sees the same compose file a real one would.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const (
	realPrefix = "selfhostly-"
	e2ePrefix  = "e2e-"
	labelLines = "    labels:\n      selfhostly.e2e: \"1\"\n"
)

type dropFlags []string

func (d *dropFlags) String() string     { return strings.Join(*d, ",") }
func (d *dropFlags) Set(v string) error { *d = append(*d, v); return nil }

func main() {
	in := flag.String("in", "", "docker-compose.prod.yml")
	out := flag.String("out", "", "file to write")
	var drop dropFlags
	flag.Var(&drop, "drop-line", "drop every line containing this text (repeatable)")
	flag.Parse()
	if *in == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: composegen --in FILE --out FILE [--drop-line TEXT]...")
		os.Exit(2)
	}
	b, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "composegen:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, []byte(transform(string(b), drop)), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "composegen:", err)
		os.Exit(1)
	}
}

func transform(src string, drop []string) string {
	var out strings.Builder
	skipping := false
	for _, line := range strings.SplitAfter(src, "\n") {
		trimmed := strings.TrimRight(line, "\n")
		// a service block ends at the next line that is not indented
		if skipping && trimmed != "" && !strings.HasPrefix(trimmed, " ") && !strings.HasPrefix(trimmed, "#") {
			skipping = false
		}
		if trimmed == "  cloudflared:" {
			skipping = true
		}
		if skipping || containsAny(line, drop) {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "    container_name: "+realPrefix):
			out.WriteString(strings.Replace(line, realPrefix, e2ePrefix, 1))
			out.WriteString(labelLines)
		case strings.HasPrefix(trimmed, "    name: "+realPrefix+"network"):
			out.WriteString("    name: " + e2ePrefix + "network\n")
		default:
			out.WriteString(line)
		}
	}
	return out.String()
}

func containsAny(line string, subs []string) bool {
	for _, s := range subs {
		if strings.Contains(line, s) {
			return true
		}
	}
	return false
}
