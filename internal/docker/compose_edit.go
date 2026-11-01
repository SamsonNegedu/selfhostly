package docker

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/tunnel"
)

// Editing a user's compose file means editing their YAML, not rebuilding it from a typed model.
// ComposeFile covers only some compose fields and rewrites others (a bind mount "./data" becomes
// "data", which compose reads as a named volume), so anything we write back must go through the
// functions below. They change only the nodes they own and leave the rest of the file as it was,
// comments and anchors included.

const (
	tunnelServiceName = "tunnel"
	defaultNetwork    = "default"
	composeIndent     = 2

	keyServices    = "services"
	keyNetworks    = "networks"
	keyNetworkMode = "network_mode"
	keyMergeKey    = "<<"
)

// InjectTunnelIntoCompose adds the tunnel provider's sidecar container to the compose text and
// returns the updated text. An existing "tunnel" service is replaced in place. It returns false and
// the original text when the provider needs no container.
func InjectTunnelIntoCompose(content, appName string, containerConfig *tunnel.ContainerConfig) (string, bool, error) {
	if containerConfig == nil {
		return content, false, nil
	}
	doc, err := openComposeDoc(content)
	if err != nil {
		return "", false, err
	}
	doc.injectTunnel(appName, containerConfig)
	updated, err := doc.render()
	if err != nil {
		return "", false, err
	}
	return updated, true, nil
}

// RemoveTunnelFromCompose deletes the "tunnel" service from the compose text. It returns false and
// the original text when there is no such service.
func RemoveTunnelFromCompose(content string) (string, bool, error) {
	doc, err := openComposeDoc(content)
	if err != nil {
		return "", false, err
	}
	if !doc.removeService(tunnelServiceName) {
		return content, false, nil
	}
	updated, err := doc.render()
	if err != nil {
		return "", false, err
	}
	return updated, true, nil
}

type composeDoc struct {
	doc      yaml.Node
	root     *yaml.Node
	services *yaml.Node
}

func openComposeDoc(content string) (*composeDoc, error) {
	d := &composeDoc{}
	if err := yaml.Unmarshal([]byte(content), &d.doc); err != nil {
		return nil, fmt.Errorf("invalid compose file: %w", err)
	}
	if d.doc.Kind != yaml.DocumentNode || len(d.doc.Content) == 0 || d.doc.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("invalid compose file: top level must be a mapping")
	}
	d.root = d.doc.Content[0]
	d.services = resolveAlias(mapValue(d.root, keyServices))
	if d.services == nil || d.services.Kind != yaml.MappingNode {
		return nil, errors.New("invalid compose file: services section is required")
	}
	return d, nil
}

func (d *composeDoc) render() (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(composeIndent)
	if err := enc.Encode(&d.doc); err != nil {
		return "", fmt.Errorf("failed to write compose file: %w", err)
	}
	if err := enc.Close(); err != nil {
		return "", fmt.Errorf("failed to write compose file: %w", err)
	}
	return buf.String(), nil
}

func (d *composeDoc) removeService(name string) bool {
	i := mapIndex(d.services, name)
	if i < 0 {
		return false
	}
	d.services.Content = append(d.services.Content[:i], d.services.Content[i+2:]...)
	return true
}

// appNetworks lists the networks the app uses: those its services name, then those the file
// defines, in document order. Empty means the app relies on Compose's implicit default network.
func (d *composeDoc) appNetworks() []string {
	var names []string
	seen := map[string]bool{}
	add := func(n string) {
		if n != "" && !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	for i := 0; i+1 < len(d.services.Content); i += 2 {
		if d.services.Content[i].Value == tunnelServiceName {
			continue // a tunnel being replaced must not decide which network the new one joins
		}
		svc := resolveAlias(d.services.Content[i+1])
		if svc == nil || svc.Kind != yaml.MappingNode {
			continue
		}
		for _, n := range networkNames(mapValue(svc, keyNetworks)) {
			add(n)
		}
	}
	for _, n := range networkNames(mapValue(d.root, keyNetworks)) {
		add(n)
	}
	return names
}

func (d *composeDoc) injectTunnel(appName string, cfg *tunnel.ContainerConfig) {
	network := defaultNetwork
	if names := d.appNetworks(); len(names) > 0 {
		network = names[0]
	}

	// A named network must exist and every service must be on it, or the tunnel could not reach
	// the services that rely on the implicit default network. The implicit default needs neither.
	if network != defaultNetwork {
		d.ensureNetworkDefined(network, map[string]string{"driver": "bridge"})
		d.attachBareServices(network)
	}

	networks := append([]string(nil), cfg.Networks...)
	if len(networks) == 0 {
		networks = []string{network}
	}
	if !containsString(networks, constants.CoreAPINetwork) && network != constants.CoreAPINetwork {
		networks = append(networks, constants.CoreAPINetwork)
		def := map[string]string{"driver": "bridge"}
		if checkDockerNetworkExists(constants.CoreAPINetwork) {
			def["external"] = "true"
		}
		d.ensureNetworkDefined(constants.CoreAPINetwork, def)
	}

	svc := mapping(
		"image", scalar(cfg.Image),
		"container_name", scalar(fmt.Sprintf("%s-tunnel", appName)),
		"restart", scalar("unless-stopped"),
		"networks", stringSeq(networks),
	)
	if len(cfg.Environment) > 0 {
		svc.Content = append(svc.Content, scalar("environment"), stringMap(cfg.Environment))
	}
	if len(cfg.Command) > 0 {
		svc.Content = append(svc.Content, scalar("command"), scalar(strings.Join(cfg.Command, " ")))
	}
	if len(cfg.Volumes) > 0 {
		svc.Content = append(svc.Content, scalar("volumes"), stringSeq(cfg.Volumes))
	}
	if len(cfg.Ports) > 0 {
		svc.Content = append(svc.Content, scalar("ports"), stringSeq(cfg.Ports))
	}

	if i := mapIndex(d.services, tunnelServiceName); i >= 0 {
		d.services.Content[i+1] = svc
		return
	}
	d.services.Content = append(d.services.Content, scalar(tunnelServiceName), svc)
}

// ensureNetworkDefined adds a top-level network unless the file already defines it.
func (d *composeDoc) ensureNetworkDefined(name string, fields map[string]string) {
	nets := mapValue(d.root, keyNetworks)
	if nets == nil {
		nets = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		d.root.Content = append(d.root.Content, scalar(keyNetworks), nets)
	}
	nets = resolveAlias(nets)
	if nets.Kind != yaml.MappingNode {
		return // a null `networks:` or an unexpected shape: leave it for Compose to report
	}
	if mapIndex(nets, name) >= 0 {
		return
	}
	def := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := scalar(fields[k])
		if fields[k] == "true" {
			v = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}
		}
		def.Content = append(def.Content, scalar(k), v)
	}
	nets.Content = append(nets.Content, scalar(name), def)
}

// attachBareServices puts services that name no network on the given one. Services that take
// their networking from elsewhere (network_mode, a merge key, an alias) are left alone.
func (d *composeDoc) attachBareServices(network string) {
	for i := 0; i+1 < len(d.services.Content); i += 2 {
		if d.services.Content[i].Value == tunnelServiceName {
			continue
		}
		svc := d.services.Content[i+1]
		if svc.Kind != yaml.MappingNode {
			continue
		}
		if mapIndex(svc, keyNetworks) >= 0 || mapIndex(svc, keyNetworkMode) >= 0 || mapIndex(svc, keyMergeKey) >= 0 {
			continue
		}
		svc.Content = append(svc.Content, scalar(keyNetworks), stringSeq([]string{network}))
	}
}

// networkNames reads a service's or the file's `networks` value, which is a list of names or a
// map keyed by name.
func networkNames(n *yaml.Node) []string {
	n = resolveAlias(n)
	if n == nil {
		return nil
	}
	var names []string
	switch n.Kind {
	case yaml.SequenceNode:
		for _, item := range n.Content {
			if item.Kind == yaml.ScalarNode {
				names = append(names, item.Value)
			}
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			names = append(names, n.Content[i].Value)
		}
	}
	return names
}

func resolveAlias(n *yaml.Node) *yaml.Node {
	for n != nil && n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	return n
}

func mapIndex(m *yaml.Node, key string) int {
	if m == nil || m.Kind != yaml.MappingNode {
		return -1
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

func mapValue(m *yaml.Node, key string) *yaml.Node {
	if i := mapIndex(m, key); i >= 0 {
		return m.Content[i+1]
	}
	return nil
}

func scalar(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
}

func mapping(kv ...interface{}) *yaml.Node {
	m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for i := 0; i+1 < len(kv); i += 2 {
		m.Content = append(m.Content, scalar(kv[i].(string)), kv[i+1].(*yaml.Node))
	}
	return m
}

func stringSeq(items []string) *yaml.Node {
	s := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, it := range items {
		s.Content = append(s.Content, scalar(it))
	}
	return s
}

func stringMap(m map[string]string) *yaml.Node {
	out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out.Content = append(out.Content, scalar(k), scalar(m[k]))
	}
	return out
}

func containsString(list []string, s string) bool {
	for _, it := range list {
		if it == s {
			return true
		}
	}
	return false
}
