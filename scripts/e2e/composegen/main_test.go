package main

import (
	"strings"
	"testing"
)

const sample = `services:
  gateway:
    image: x
    container_name: selfhostly-gateway
    environment:
      KEEP: "1"
  cloudflared:
    image: cloudflare/cloudflared
    container_name: selfhostly-cloudflared
    depends_on:
      gateway:
        condition: service_healthy

networks:
  selfhostly-network:
    name: selfhostly-network # Use exact name, no project prefix
`

func TestTransformRenamesLabelsAndDropsCloudflared(t *testing.T) {
	got := transform(sample, nil)
	for _, want := range []string{"container_name: e2e-gateway", "selfhostly.e2e: \"1\"", "name: e2e-network", "KEEP: \"1\"", "selfhostly-network:"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, gone := range []string{"cloudflared", "selfhostly-gateway", "name: selfhostly-network"} {
		if strings.Contains(got, gone) {
			t.Errorf("%q must be gone:\n%s", gone, got)
		}
	}
}

func TestTransformKeepsWhatFollowsTheDroppedService(t *testing.T) {
	if got := transform(sample, nil); !strings.Contains(got, "networks:") {
		t.Fatalf("the top level after the dropped service must stay:\n%s", got)
	}
}

func TestTransformDropsRequestedLines(t *testing.T) {
	if got := transform(sample, []string{"KEEP"}); strings.Contains(got, "KEEP") {
		t.Fatalf("got:\n%s", got)
	}
}
