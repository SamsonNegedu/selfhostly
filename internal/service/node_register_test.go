package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

const testRegistrationToken = "cluster-registration-token"

func newRegistrationService(t *testing.T) (domain.NodeService, *db.DB) {
	t.Helper()
	f, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	database, err := db.Init(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close(); os.Remove(f.Name()) })
	cfg := &config.Config{Node: config.NodeConfig{IsPrimary: true, RegistrationToken: testRegistrationToken}}
	return NewNodeService(database, cfg, slog.Default()), database
}

func registration(token string) domain.AutoRegisterRequest {
	return domain.AutoRegisterRequest{ID: "n1", Name: "pi", APIEndpoint: "http://10.0.0.5:8082", APIKey: "node-key-0123456789", Token: token}
}

func TestAutoRegisterRejectsAnUnknownToken(t *testing.T) {
	svc, database := newRegistrationService(t)
	_, err := svc.AutoRegisterNode(context.Background(), registration("wrong"))
	if !errors.Is(err, domain.ErrRegistrationUnauthorized) {
		t.Fatalf("got %v", err)
	}
	if n, err := database.GetNode("n1"); err == nil && n != nil {
		t.Fatal("a rejected registration must not create a node")
	}
}

func TestAutoRegisterSavesTheNodeBeforeItsHealthCheck(t *testing.T) {
	svc, database := newRegistrationService(t)
	reg, err := svc.AutoRegisterNode(context.Background(), registration(testRegistrationToken))
	if err != nil {
		t.Fatal(err)
	}
	if !reg.Created {
		t.Fatal("expected a new node")
	}
	n, err := database.GetNode("n1")
	if err != nil || n == nil {
		t.Fatalf("the node must be on record: %v", err)
	}
	if n.Status == "" {
		t.Fatal("status must be recorded")
	}
}

func TestAutoRegisterSingleUseJoinTokenWorksOnce(t *testing.T) {
	svc, database := newRegistrationService(t)
	tok, _, err := database.CreateJoinToken(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AutoRegisterNode(context.Background(), registration(tok)); err != nil {
		t.Fatal(err)
	}
	other := registration(tok)
	other.ID, other.Name = "n2", "other"
	if _, err := svc.AutoRegisterNode(context.Background(), other); !errors.Is(err, domain.ErrRegistrationUnauthorized) {
		t.Fatalf("a spent token must be refused, got %v", err)
	}
}

func TestAutoRegisterReRegistrationNeedsTheKeyOnRecord(t *testing.T) {
	svc, _ := newRegistrationService(t)
	ctx := context.Background()
	if _, err := svc.AutoRegisterNode(ctx, registration(testRegistrationToken)); err != nil {
		t.Fatal(err)
	}
	same := registration(testRegistrationToken)
	same.APIEndpoint = "http://10.0.0.6:8082"
	reg, err := svc.AutoRegisterNode(ctx, same)
	if err != nil || reg.Created || reg.Node.APIEndpoint != same.APIEndpoint {
		t.Fatalf("the same key may refresh the endpoint: %v %+v", err, reg)
	}
	swapped := registration(testRegistrationToken)
	swapped.APIKey = "a-different-key-0123456789"
	_, err = svc.AutoRegisterNode(ctx, swapped)
	var de *domain.DomainError
	if !errors.As(err, &de) || de.Code != domain.CodeNodeIDTaken {
		t.Fatalf("a different key must be a conflict, got %v", err)
	}
}

func TestAutoRegisterRefusesADuplicateName(t *testing.T) {
	svc, _ := newRegistrationService(t)
	ctx := context.Background()
	if _, err := svc.AutoRegisterNode(ctx, registration(testRegistrationToken)); err != nil {
		t.Fatal(err)
	}
	dup := registration(testRegistrationToken)
	dup.ID = "n2"
	_, err := svc.AutoRegisterNode(ctx, dup)
	var de *domain.DomainError
	if !errors.As(err, &de) || de.Code != domain.CodeNodeNameTaken {
		t.Fatalf("got %v", err)
	}
}

func TestSecurityServiceJoinTokensSessionsAndAudit(t *testing.T) {
	_, database := newRegistrationService(t)
	svc := NewSecurityService(database)
	ctx := context.Background()

	tok, expires, err := svc.CreateJoinToken(ctx)
	if err != nil || tok == "" || !expires.After(time.Now()) {
		t.Fatalf("token %q expires %v err %v", tok, expires, err)
	}
	if _, err := svc.RevokeSessions(ctx); err != nil {
		t.Fatal(err)
	}
	e := db.AuditEntry{Action: "node.join", TargetType: "node", TargetID: "n1", Time: time.Now(), Actor: "node:n1", Method: "GET", Path: "/x", Status: 200}
	if err := svc.RecordAudit(ctx, e); err != nil {
		t.Fatal(err)
	}
	entries, err := svc.ListAudit(ctx, 10)
	if err != nil || len(entries) == 0 || entries[0].Action != "node.join" {
		t.Fatalf("audit not recorded: %v %+v", err, entries)
	}
}
