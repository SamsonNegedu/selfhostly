package docker

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/selfhostly/internal/constants"
)

// A compose file with the things the typed model cannot carry: relative bind mounts and their
// options, top-level secrets, a described named volume, a list-form command, and comments.
const richCompose = `# my stack
services:
  web:
    image: nginx:latest
    command: ["nginx", "-g", "daemon off;"]
    volumes:
      - ./secrets:/run/secrets:ro   # keep read-only
      - ./data:/data
      - ./cfg:/cfg:z
      - ../shared:/shared
      - cache:/cache
    secrets:
      - api_key
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
secrets:
  api_key:
    file: ./api_key.txt
volumes:
  cache:
    external: true
    name: shared-cache
`

func parseGeneric(t *testing.T, content string) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &out); err != nil {
		t.Fatalf("result is not valid YAML: %v\n%s", err, content)
	}
	return out
}

func serviceOf(t *testing.T, content, name string) map[string]interface{} {
	t.Helper()
	svc, ok := parseGeneric(t, content)["services"].(map[string]interface{})[name].(map[string]interface{})
	if !ok {
		t.Fatalf("service %q missing from:\n%s", name, content)
	}
	return svc
}

func stringsOf(v interface{}) []string {
	var out []string
	for _, it := range v.([]interface{}) {
		out = append(out, it.(string))
	}
	return out
}

// Regression: adding the tunnel rewrote "./secrets:/run/secrets:ro" as "secrets:/run/secrets:ro",
// which Compose reads as a named volume, so the container lost its bind mount.
func TestInjectTunnelKeepsEverythingElseInTheFile(t *testing.T) {
	updated, injected, err := InjectTunnelIntoCompose(richCompose, "web", cloudflaredLikeContainerConfig("tok"))
	if err != nil || !injected {
		t.Fatalf("InjectTunnelIntoCompose() = injected %v, err %v", injected, err)
	}

	for _, want := range []string{
		"./secrets:/run/secrets:ro",
		"./data:/data",
		"./cfg:/cfg:z",
		"../shared:/shared",
		"# my stack",
		"# keep read-only",
		"file: ./api_key.txt",
		"external: true",
		"name: shared-cache",
	} {
		if !strings.Contains(updated, want) {
			t.Errorf("updated compose lost %q:\n%s", want, updated)
		}
	}

	before, after := serviceOf(t, richCompose, "web"), serviceOf(t, updated, "web")
	for key, want := range before {
		got, _ := yaml.Marshal(after[key])
		exp, _ := yaml.Marshal(want)
		if string(got) != string(exp) {
			t.Errorf("web.%s changed:\nwant %s\ngot  %s", key, exp, got)
		}
	}
	if len(after) != len(before) {
		t.Errorf("web gained or lost keys: before %d, after %d", len(before), len(after))
	}

	top := parseGeneric(t, updated)
	for _, key := range []string{"secrets", "volumes"} {
		a, _ := yaml.Marshal(top[key])
		b, _ := yaml.Marshal(parseGeneric(t, richCompose)[key])
		if string(a) != string(b) {
			t.Errorf("top-level %s changed:\nwant %s\ngot  %s", key, b, a)
		}
	}
}

func TestInjectTunnelAddsASidecar(t *testing.T) {
	updated, injected, err := InjectTunnelIntoCompose(richCompose, "myapp", cloudflaredLikeContainerConfig("tok"))
	if err != nil || !injected {
		t.Fatalf("InjectTunnelIntoCompose() = injected %v, err %v", injected, err)
	}
	svc := serviceOf(t, updated, "tunnel")
	if svc["image"] != "cloudflare/cloudflared:latest" || svc["container_name"] != "myapp-tunnel" ||
		svc["restart"] != "unless-stopped" || svc["command"] != "tunnel run" {
		t.Errorf("unexpected tunnel service: %v", svc)
	}
	if svc["environment"].(map[string]interface{})["TUNNEL_TOKEN"] != "tok" {
		t.Errorf("TUNNEL_TOKEN missing: %v", svc["environment"])
	}
}

func TestInjectTunnelOnImplicitDefaultNetwork(t *testing.T) {
	updated, _, err := InjectTunnelIntoCompose("services:\n  web:\n    image: nginx\n", "web", cloudflaredLikeContainerConfig("tok"))
	if err != nil {
		t.Fatal(err)
	}
	nets := stringsOf(serviceOf(t, updated, "tunnel")["networks"])
	if len(nets) != 2 || nets[0] != "default" || nets[1] != constants.CoreAPINetwork {
		t.Errorf("tunnel networks = %v, want [default %s]", nets, constants.CoreAPINetwork)
	}
	if _, has := serviceOf(t, updated, "web")["networks"]; has {
		t.Error("web must stay on the implicit default network, untouched")
	}
	defined := parseGeneric(t, updated)["networks"].(map[string]interface{})
	if _, ok := defined[constants.CoreAPINetwork]; !ok {
		t.Errorf("%s must be defined for the tunnel to join it", constants.CoreAPINetwork)
	}
	if _, ok := defined["default"]; ok {
		t.Error("the implicit default network must not be redefined")
	}
}

func TestInjectTunnelUsesTheAppsOwnNetwork(t *testing.T) {
	in := `services:
  web:
    image: nginx
    networks: [front]
  db:
    image: postgres
networks:
  front:
    driver: bridge
`
	updated, _, err := InjectTunnelIntoCompose(in, "web", cloudflaredLikeContainerConfig("tok"))
	if err != nil {
		t.Fatal(err)
	}
	nets := stringsOf(serviceOf(t, updated, "tunnel")["networks"])
	if len(nets) != 2 || nets[0] != "front" || nets[1] != constants.CoreAPINetwork {
		t.Errorf("tunnel networks = %v, want [front %s]", nets, constants.CoreAPINetwork)
	}
	if got := stringsOf(serviceOf(t, updated, "web")["networks"]); len(got) != 1 || got[0] != "front" {
		t.Errorf("web networks changed: %v", got)
	}
	// A service with no network would otherwise sit on `default`, out of the tunnel's reach.
	if got := stringsOf(serviceOf(t, updated, "db")["networks"]); len(got) != 1 || got[0] != "front" {
		t.Errorf("db must join the tunnel's network, got %v", got)
	}
}

func TestInjectTunnelLeavesServicesWithOtherNetworkingAlone(t *testing.T) {
	in := `services:
  web:
    image: nginx
    networks: [front]
  host:
    image: x
    network_mode: host
networks:
  front: {}
`
	updated, _, err := InjectTunnelIntoCompose(in, "web", cloudflaredLikeContainerConfig("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if _, has := serviceOf(t, updated, "host")["networks"]; has {
		t.Error("a network_mode service cannot take a networks list")
	}
}

func TestInjectTunnelQuickTunnelConfig(t *testing.T) {
	updated, _, err := InjectTunnelIntoCompose("services:\n  web:\n    image: nginx\n", "myapp", quickTunnelContainerConfig(2000))
	if err != nil {
		t.Fatal(err)
	}
	svc := serviceOf(t, updated, "tunnel")
	if svc["command"] != "tunnel --url http://web:80 --metrics 0.0.0.0:2000" {
		t.Errorf("command = %v", svc["command"])
	}
	if ports := stringsOf(svc["ports"]); len(ports) != 1 || ports[0] != "2000:2000" {
		t.Errorf("ports = %v", ports)
	}
}

func TestInjectTunnelReplacesAnExistingTunnelInPlace(t *testing.T) {
	first, _, err := InjectTunnelIntoCompose(richCompose, "web", cloudflaredLikeContainerConfig("old"))
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := InjectTunnelIntoCompose(first, "web", cloudflaredLikeContainerConfig("new"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(second, "tunnel:\n"); n != 1 {
		t.Errorf("want one tunnel service, got %d:\n%s", n, second)
	}
	env := serviceOf(t, second, "tunnel")["environment"].(map[string]interface{})
	if env["TUNNEL_TOKEN"] != "new" {
		t.Errorf("TUNNEL_TOKEN = %v, want new", env["TUNNEL_TOKEN"])
	}
}

func TestInjectTunnelWithoutAContainerChangesNothing(t *testing.T) {
	updated, injected, err := InjectTunnelIntoCompose(richCompose, "web", nil)
	if err != nil || injected || updated != richCompose {
		t.Errorf("got injected %v, err %v, changed %v", injected, err, updated != richCompose)
	}
}

func TestRemoveTunnelFromCompose(t *testing.T) {
	withTunnel, _, err := InjectTunnelIntoCompose(richCompose, "web", cloudflaredLikeContainerConfig("tok"))
	if err != nil {
		t.Fatal(err)
	}
	updated, removed, err := RemoveTunnelFromCompose(withTunnel)
	if err != nil || !removed {
		t.Fatalf("RemoveTunnelFromCompose() = removed %v, err %v", removed, err)
	}
	if _, has := parseGeneric(t, updated)["services"].(map[string]interface{})["tunnel"]; has {
		t.Error("tunnel service should be gone")
	}
	for _, want := range []string{"./secrets:/run/secrets:ro", "./cfg:/cfg:z", "# my stack"} {
		if !strings.Contains(updated, want) {
			t.Errorf("removing the tunnel lost %q:\n%s", want, updated)
		}
	}

	same, removed, err := RemoveTunnelFromCompose(richCompose)
	if err != nil || removed || same != richCompose {
		t.Errorf("no tunnel: got removed %v, err %v, changed %v", removed, err, same != richCompose)
	}
}

func TestEditingRejectsContentThatIsNotACompose(t *testing.T) {
	for name, in := range map[string]string{
		"not yaml":    "services: [",
		"no services": "version: '3'\n",
		"list":        "- a\n- b\n",
	} {
		if _, _, err := InjectTunnelIntoCompose(in, "x", cloudflaredLikeContainerConfig("t")); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
