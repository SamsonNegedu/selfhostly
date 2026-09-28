package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

// deploySourceVerifiers holds any extra, kind-specific check a deploy hook's SourceKind needs beyond
// the token match every hook already requires (ConsumeDeployHookToken already scopes the match to
// one app, so this is purely additional). A kind with no entry here, including the only kind that
// exists today (constants.DeploySourceGeneric), gets that shared check and nothing more. Adding a
// kind that needs more, for example one that also checks a caller's identity, means adding an entry
// here: no change to the schema, the route, or any other kind's behavior.
var deploySourceVerifiers = map[string]func(ctx context.Context, hook *db.DeployHook, callerIP string) error{}

// deployHookService implements domain.DeployHookService
type deployHookService struct {
	database   *db.DB
	appService domain.AppService
	logger     *slog.Logger
}

// NewDeployHookService creates the service that manages per-app deploy hooks and verifies-and-
// triggers the update one authorizes. It calls into AppService for the trigger itself, so a
// hook-authenticated request starts the exact same job the manual Update button starts.
func NewDeployHookService(database *db.DB, appService domain.AppService, logger *slog.Logger) domain.DeployHookService {
	return &deployHookService{database: database, appService: appService, logger: logger}
}

func (s *deployHookService) CreateDeployHook(ctx context.Context, appID, name string) (string, *domain.DeployHook, error) {
	if _, err := s.database.GetApp(appID); err != nil {
		return "", nil, domain.WrapAppNotFound(appID, err)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, domain.WrapValidationError("name", errors.New("name cannot be empty"))
	}

	plain, hook, err := s.database.CreateDeployHook(appID, name, constants.DeploySourceGeneric)
	if err != nil {
		return "", nil, domain.WrapDatabaseOperation("create deploy hook", err)
	}
	s.logger.InfoContext(ctx, "deploy hook created", "appID", appID, "hookID", hook.ID, "name", hook.Name)
	return plain, toDomainDeployHook(hook), nil
}

func (s *deployHookService) ListDeployHooks(ctx context.Context, appID string) ([]*domain.DeployHook, error) {
	hooks, err := s.database.ListDeployHooks(appID)
	if err != nil {
		return nil, domain.WrapDatabaseOperation("list deploy hooks", err)
	}
	out := make([]*domain.DeployHook, 0, len(hooks))
	for _, h := range hooks {
		out = append(out, toDomainDeployHook(h))
	}
	return out, nil
}

func (s *deployHookService) RevokeDeployHook(ctx context.Context, appID, hookID string) error {
	err := s.database.RevokeDeployHook(appID, hookID)
	if errors.Is(err, db.ErrDeployHookNotFound) {
		return domain.ErrDeployHookNotFound
	}
	if err != nil {
		return domain.WrapDatabaseOperation("revoke deploy hook", err)
	}
	s.logger.InfoContext(ctx, "deploy hook revoked", "appID", appID, "hookID", hookID)
	return nil
}

func (s *deployHookService) TriggerDeploy(ctx context.Context, appID, presentedToken, callerIP string) (*db.Job, *domain.DeployHook, error) {
	hook, err := s.database.ConsumeDeployHookToken(appID, presentedToken, callerIP)
	if err != nil {
		return nil, nil, domain.WrapDatabaseOperation("verify deploy hook", err)
	}
	if hook == nil {
		return nil, nil, domain.ErrDeployTriggerUnauthorized
	}
	if verify, ok := deploySourceVerifiers[hook.SourceKind]; ok {
		if err := verify(ctx, hook, callerIP); err != nil {
			return nil, nil, err
		}
	}

	job, err := s.appService.UpdateAppContainersAsync(ctx, appID)
	if err != nil {
		return nil, nil, err
	}
	s.logger.InfoContext(ctx, "deploy hook triggered an update", "appID", appID, "hookID", hook.ID, "name", hook.Name, "callerIP", callerIP)
	return job, toDomainDeployHook(hook), nil
}

func toDomainDeployHook(h *db.DeployHook) *domain.DeployHook {
	return &domain.DeployHook{
		ID:         h.ID,
		AppID:      h.AppID,
		Name:       h.Name,
		SourceKind: h.SourceKind,
		CreatedAt:  h.CreatedAt,
		LastUsedAt: h.LastUsedAt,
		LastUsedIP: h.LastUsedIP,
	}
}
