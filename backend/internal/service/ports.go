package service

import (
	"context"
	"encoding/json"
	"github.com/your-org/prd-generator/backend/internal/domain"
	"time"
)

type UserRepository interface {
	CreateUser(context.Context, *domain.User) error
	UserByEmail(context.Context, string) (*domain.User, error)
	UserByID(context.Context, string) (*domain.User, error)
	UpdateUser(context.Context, *domain.User) error
	UpdatePassword(context.Context, string, string) error
	SaveRefreshToken(context.Context, domain.RefreshToken) error
	ConsumeRefreshToken(context.Context, string) (string, error)
	RevokeRefreshToken(context.Context, string) error
	RevokeUserRefreshTokens(context.Context, string) error
}
type ProjectRepository interface {
	CreateProject(context.Context, *domain.Project) error
	ListProjects(context.Context, string, string, string, string, string, string, int, int) ([]domain.Project, int, error)
	GetProject(context.Context, string, string) (*domain.Project, error)
	UpdateProject(context.Context, *domain.Project) error
	DeleteProject(context.Context, string, string) error
	DuplicateProject(context.Context, string, string) (*domain.Project, error)
	ArchiveProject(context.Context, string, string) error
	Dashboard(context.Context, string) (map[string]any, error)
}
type PRDRepository interface {
	ListPRDs(context.Context, string, string, string, string, string, string, int, int) ([]domain.PRD, int, error)
	CreatePRD(context.Context, *domain.PRD) error
	GetPRD(context.Context, string, string) (*domain.PRD, error)
	UpdatePRD(context.Context, *domain.PRD, int) error
	DeletePRD(context.Context, string, string) error
	ListVersions(context.Context, string, string) ([]domain.PRDVersion, error)
	RestoreVersion(context.Context, string, string, string) (*domain.PRD, error)
	ListProjectPRDs(context.Context, string, string, int, int) ([]domain.PRD, int, error)
	DuplicatePRD(context.Context, string, string) (*domain.PRD, error)
}
type GenerationRepository interface {
	SaveGeneratedPRD(context.Context, string, string, string, json.RawMessage, string, string, string, int, int) (*domain.PRD, error)
	IdempotentPRD(context.Context, string, string, string) (*domain.PRD, error)
	UsageSummary(context.Context, string) (map[string]any, error)
	UsageHistory(context.Context, string, int, int) ([]map[string]any, error)
	RecordAIUsage(context.Context, string, string, string, int, int) error
	Plans(context.Context) ([]map[string]any, error)
}
type AIProvider interface {
	Generate(context.Context, domain.Idea) (domain.PRDDocument, int, int, error)
	GenerateStream(context.Context, domain.Idea, func(string)) (domain.PRDDocument, int, int, error)
	Clarify(context.Context, domain.Idea, map[string]string) ([]domain.ClarificationQuestion, bool, int, int, error)
	Act(context.Context, string, json.RawMessage) (json.RawMessage, int, int, error)
}
type CollaborationRepository interface {
	SaveClarification(context.Context, *domain.ClarificationSession, string) error
	GetClarification(context.Context, string, string) (*domain.ClarificationSession, error)
	CreateShareLink(context.Context, string, string, string, time.Time) (*domain.ShareLink, error)
	ListShareLinks(context.Context, string, string) ([]domain.ShareLink, error)
	RevokeShareLink(context.Context, string, string, string) error
	SharedPRD(context.Context, string) (*domain.PRD, error)
}
type AISettingsRepository interface {
	GetAISettings(ctx context.Context, userID string) (*domain.UserAISettings, error)
	UpsertAISettings(ctx context.Context, settings *domain.UserAISettings) error
}
