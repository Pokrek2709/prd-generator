package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/your-org/prd-generator/backend/internal/config"
	"github.com/your-org/prd-generator/backend/internal/domain"
	"github.com/your-org/prd-generator/backend/internal/infrastructure/ai"
	"github.com/your-org/prd-generator/backend/internal/infrastructure/postgres"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrConflict = errors.New("resource conflict")
var ErrVersionConflict = errors.New("version conflict")
var ErrInvalidInput = errors.New("invalid input")
var ErrClarificationRequired = errors.New("clarification required before generation")
var ErrAIProvider = errors.New("AI provider failure")

var supportedAIActions = map[string]bool{
	"improve_prd": true, "review_prd": true, "find_missing_requirements": true, "detect_ambiguity": true, "generate_user_stories": true,
	"generate_acceptance_criteria": true, "identify_edge_cases": true, "review_security": true, "generate_tasks": true,
	"create_mvp_scope": true, "estimate_complexity": true, "draft_api_contract": true, "draft_database_schema": true,
	"regenerate_section": true, "simplify_language": true, "translate_document": true,
}

type Service struct {
	users      UserRepository
	projects   ProjectRepository
	prds       PRDRepository
	usage      GenerationRepository
	collab     CollaborationRepository
	aiSettings AISettingsRepository
	cfg        config.Config
	ai         AIProvider
}

func New(users UserRepository, projects ProjectRepository, prds PRDRepository, usage GenerationRepository, collab CollaborationRepository, cfg config.Config, ai AIProvider) *Service {
	s := &Service{users: users, projects: projects, prds: prds, usage: usage, collab: collab, cfg: cfg, ai: ai}
	if as, ok := any(users).(AISettingsRepository); ok {
		s.aiSettings = as
	}
	return s
}

func (s *Service) WithAISettings(aiSettings AISettingsRepository) *Service {
	s.aiSettings = aiSettings
	return s
}
func (s *Service) Register(ctx context.Context, email, name, password string) (*domain.User, string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	parsedEmail, emailErr := mail.ParseAddress(email)
	if emailErr != nil || parsedEmail.Address != email || len(name) < 1 || len(name) > 100 || len(password) < 10 || len(password) > 72 {
		return nil, "", "", ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", "", err
	}
	u := &domain.User{Email: email, Name: name, PasswordHash: string(hash)}
	if err = s.users.CreateUser(ctx, u); err != nil {
		return nil, "", "", err
	}
	access, refresh, err := s.issueTokens(ctx, u.ID)
	return u, access, refresh, err
}
func (s *Service) Login(ctx context.Context, email, password string) (*domain.User, string, string, error) {
	u, err := s.users.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, "", "", ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, "", "", ErrInvalidCredentials
	}
	access, refresh, err := s.issueTokens(ctx, u.ID)
	return u, access, refresh, err
}
func (s *Service) issueTokens(ctx context.Context, userID string) (string, string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{Subject: userID, ID: uuid.NewString(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTokenTTL))}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", "", err
	}
	raw := make([]byte, 48)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	refresh := base64.RawURLEncoding.EncodeToString(raw)
	if err = s.users.SaveRefreshToken(ctx, domain.RefreshToken{Hash: hashToken(refresh), UserID: userID, ExpiresAt: now.Add(s.cfg.RefreshTokenTTL)}); err != nil {
		return "", "", err
	}
	return token, refresh, nil
}
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func (s *Service) Refresh(ctx context.Context, refresh string) (*domain.User, string, string, error) {
	id, err := s.users.ConsumeRefreshToken(ctx, hashToken(refresh))
	if err != nil {
		return nil, "", "", ErrInvalidCredentials
	}
	u, err := s.users.UserByID(ctx, id)
	if err != nil {
		return nil, "", "", err
	}
	a, r, err := s.issueTokens(ctx, id)
	return u, a, r, err
}
func (s *Service) Logout(ctx context.Context, refresh string) error {
	if refresh == "" {
		return nil
	}
	return s.users.RevokeRefreshToken(ctx, hashToken(refresh))
}
func (s *Service) VerifyAccess(token string) (string, error) {
	parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", err
	}
	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok || !parsed.Valid {
		return "", ErrInvalidCredentials
	}
	return claims.Subject, nil
}
func (s *Service) Me(ctx context.Context, id string) (*domain.User, error) {
	return s.users.UserByID(ctx, id)
}
func (s *Service) UpdateProfile(ctx context.Context, id, name string) (*domain.User, error) {
	if len(strings.TrimSpace(name)) == 0 || len(name) > 100 {
		return nil, ErrInvalidInput
	}
	u, err := s.users.UserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	u.Name = strings.TrimSpace(name)
	if err = s.users.UpdateUser(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}
func (s *Service) ChangePassword(ctx context.Context, id, current, next string) error {
	if len(next) < 10 || len(next) > 72 {
		return ErrInvalidInput
	}
	u, err := s.users.UserByID(ctx, id)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(current)) != nil {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err = s.users.UpdatePassword(ctx, id, string(hash)); err != nil {
		return err
	}
	return s.users.RevokeUserRefreshTokens(ctx, id)
}
func (s *Service) CreateProject(ctx context.Context, user, name, description string) (*domain.Project, error) {
	if len(strings.TrimSpace(name)) < 1 || len(name) > 120 || len(description) > 2000 {
		return nil, ErrInvalidInput
	}
	p := &domain.Project{UserID: user, Name: strings.TrimSpace(name), Description: strings.TrimSpace(description)}
	return p, s.projects.CreateProject(ctx, p)
}
func (s *Service) ListProjects(ctx context.Context, user, search, status, sort, dateFrom, dateTo string, limit, offset int) ([]domain.Project, int, error) {
	for _, value := range []string{dateFrom, dateTo} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return nil, 0, ErrInvalidInput
			}
		}
	}
	return s.projects.ListProjects(ctx, user, search, status, sort, dateFrom, dateTo, limit, offset)
}
func (s *Service) GetProject(ctx context.Context, id, user string) (*domain.Project, error) {
	return s.projects.GetProject(ctx, id, user)
}
func (s *Service) UpdateProject(ctx context.Context, id, user, name, description string) (*domain.Project, error) {
	if len(strings.TrimSpace(name)) < 1 || len(name) > 120 || len(description) > 2000 {
		return nil, ErrInvalidInput
	}
	p := &domain.Project{ID: id, UserID: user, Name: strings.TrimSpace(name), Description: strings.TrimSpace(description)}
	return p, s.projects.UpdateProject(ctx, p)
}
func (s *Service) PatchProject(ctx context.Context, id, user string, name, description *string) (*domain.Project, error) {
	if name == nil && description == nil {
		return nil, ErrInvalidInput
	}
	p, err := s.projects.GetProject(ctx, id, user)
	if err != nil {
		return nil, err
	}
	if name != nil {
		p.Name = strings.TrimSpace(*name)
	}
	if description != nil {
		p.Description = strings.TrimSpace(*description)
	}
	if len(p.Name) < 1 || len(p.Name) > 120 || len(p.Description) > 2000 {
		return nil, ErrInvalidInput
	}
	if err = s.projects.UpdateProject(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
func (s *Service) DeleteProject(ctx context.Context, id, user string) error {
	return s.projects.DeleteProject(ctx, id, user)
}
func (s *Service) ArchiveProject(ctx context.Context, id, user string) error {
	return s.projects.ArchiveProject(ctx, id, user)
}
func (s *Service) DuplicateProject(ctx context.Context, id, user string) (*domain.Project, error) {
	return s.projects.DuplicateProject(ctx, id, user)
}
func (s *Service) Dashboard(ctx context.Context, user string) (map[string]any, error) {
	return s.projects.Dashboard(ctx, user)
}
func (s *Service) CreatePRD(ctx context.Context, project, user, title string) (*domain.PRD, error) {
	if len(strings.TrimSpace(title)) < 1 || len(title) > 200 {
		return nil, ErrInvalidInput
	}
	if _, err := s.projects.GetProject(ctx, project, user); err != nil {
		return nil, err
	}
	p := &domain.PRD{ProjectID: project, Title: strings.TrimSpace(title), Status: "draft", Content: json.RawMessage(`{"sections":[]}`)}
	return p, s.prds.CreatePRD(ctx, p)
}
func (s *Service) GetPRD(ctx context.Context, id, user string) (*domain.PRD, error) {
	return s.prds.GetPRD(ctx, id, user)
}

func (s *Service) ExportPRDMarkdown(ctx context.Context, id, user string) (*domain.PRD, []byte, error) {
	p, err := s.prds.GetPRD(ctx, id, user)
	if err != nil {
		return nil, nil, err
	}
	markdown, err := RenderPRDMarkdown(p)
	if err != nil {
		return nil, nil, err
	}
	return p, markdown, nil
}
func (s *Service) UpdatePRD(ctx context.Context, id, user, title, status string, content json.RawMessage, expected int) (*domain.PRD, error) {
	title = strings.TrimSpace(title)
	if expected < 1 || len(title) < 1 || len(title) > 200 || len(content) > 1_000_000 || !json.Valid(content) {
		return nil, ErrInvalidInput
	}
	if status != "draft" && status != "in_progress" && status != "completed" && status != "archived" {
		return nil, ErrInvalidInput
	}
	p, err := s.prds.GetPRD(ctx, id, user)
	if err != nil {
		return nil, err
	}
	p.UserID = user
	p.Title = title
	p.Status = status
	p.Content = content
	err = s.prds.UpdatePRD(ctx, p, expected)
	if err != nil && strings.HasPrefix(err.Error(), "version_conflict:") {
		return nil, ErrVersionConflict
	}
	return p, err
}
func (s *Service) DeletePRD(ctx context.Context, id, user string) error {
	return s.prds.DeletePRD(ctx, id, user)
}
func (s *Service) Versions(ctx context.Context, id, user string) ([]domain.PRDVersion, error) {
	return s.prds.ListVersions(ctx, id, user)
}
func (s *Service) Restore(ctx context.Context, id, user, version string) (*domain.PRD, error) {
	return s.prds.RestoreVersion(ctx, id, user, version)
}
func (s *Service) ListProjectPRDs(ctx context.Context, project, user string, limit, offset int) ([]domain.PRD, int, error) {
	return s.prds.ListProjectPRDs(ctx, project, user, limit, offset)
}
func (s *Service) ListPRDs(ctx context.Context, user, search, status, dateFrom, dateTo, sort string, limit, offset int) ([]domain.PRD, int, error) {
	for _, value := range []string{dateFrom, dateTo} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return nil, 0, ErrInvalidInput
			}
		}
	}
	return s.prds.ListPRDs(ctx, user, search, status, dateFrom, dateTo, sort, limit, offset)
}
func (s *Service) DuplicatePRD(ctx context.Context, id, user string) (*domain.PRD, error) {
	return s.prds.DuplicatePRD(ctx, id, user)
}

func (s *Service) Clarify(ctx context.Context, user, sessionID string, idea domain.Idea, answers map[string]string, skip bool) (*domain.ClarificationSession, error) {
	if err := validateAnswers(answers); err != nil {
		return nil, err
	}
	var session *domain.ClarificationSession
	var err error
	if sessionID != "" {
		session, err = s.collab.GetClarification(ctx, sessionID, user)
		if err != nil {
			return nil, err
		}
		if idea.ProductName != "" {
			if err = ValidateIdea(idea); err != nil {
				return nil, err
			}
			session.Idea = idea
		}
		if session.Answers == nil {
			session.Answers = map[string]string{}
		}
		for key, value := range answers {
			if len(key) > 80 || len(value) > 2000 {
				return nil, ErrInvalidInput
			}
			session.Answers[key] = value
		}
	} else {
		if err := ValidateIdea(idea); err != nil {
			return nil, ErrInvalidInput
		}
		session = &domain.ClarificationSession{Idea: idea, Answers: answers}
		if session.Answers == nil {
			session.Answers = map[string]string{}
		}
	}
	if skip {
		session.Ready = true
		session.Questions = []domain.ClarificationQuestion{}
	} else {
		if err = s.ensureQuota(ctx, user); err != nil {
			return nil, err
		}
		aiClient, effectiveModel := s.resolveAIProvider(ctx, user, "", "")
		questions, ready, inputTokens, outputTokens, callErr := aiClient.Clarify(ctx, session.Idea, session.Answers)
		if callErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrAIProvider, callErr)
		}
		session.Questions = questions
		session.Ready = ready
		session.Idea.Clarifications = session.Answers
		if err = s.usage.RecordAIUsage(ctx, user, effectiveModel, "clarify", inputTokens, outputTokens); err != nil {
			return nil, err
		}
	}
	if err = s.collab.SaveClarification(ctx, session, user); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) AIAction(ctx context.Context, user, action, prdID string, input json.RawMessage) (json.RawMessage, error) {
	if !supportedAIActions[action] {
		return nil, ErrInvalidInput
	}
	if prdID == "" && (len(input) == 0 || len(input) > 1_000_000 || !json.Valid(input)) {
		return nil, ErrInvalidInput
	}
	if prdID != "" {
		prd, err := s.prds.GetPRD(ctx, prdID, user)
		if err != nil {
			return nil, err
		}
		wrapped, _ := json.Marshal(map[string]any{"prd": prd, "context": json.RawMessage(input)})
		input = wrapped
	}
	if err := s.ensureQuota(ctx, user); err != nil {
		return nil, err
	}
	aiClient, effectiveModel := s.resolveAIProvider(ctx, user, "", "")
	result, in, out, err := aiClient.Act(ctx, action, input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIProvider, err)
	}
	if err = s.usage.RecordAIUsage(ctx, user, effectiveModel, action, in, out); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) CreateShareLink(ctx context.Context, prd, user string, ttl time.Duration) (*domain.ShareLink, error) {
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	if ttl > 90*24*time.Hour {
		return nil, ErrInvalidInput
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	link, err := s.collab.CreateShareLink(ctx, prd, user, hashToken(token), time.Now().Add(ttl))
	if err != nil {
		return nil, err
	}
	link.Token = token
	return link, nil
}
func (s *Service) ListShareLinks(ctx context.Context, prd, user string) ([]domain.ShareLink, error) {
	return s.collab.ListShareLinks(ctx, prd, user)
}
func (s *Service) RevokeShareLink(ctx context.Context, prd, user, id string) error {
	return s.collab.RevokeShareLink(ctx, prd, user, id)
}
func (s *Service) SharedPRD(ctx context.Context, token string) (*domain.PRD, error) {
	if len(token) < 40 || len(token) > 100 {
		return nil, ErrInvalidInput
	}
	return s.collab.SharedPRD(ctx, hashToken(token))
}

func (s *Service) resolveAIProvider(ctx context.Context, userID, overrideProvider, overrideModel string) (AIProvider, string) {
	var userSettings *domain.UserAISettings
	if s.aiSettings != nil && userID != "" {
		settings, err := s.aiSettings.GetAISettings(ctx, userID)
		if err == nil && settings != nil {
			userSettings = settings
		}
	}
	provider, baseURL, apiKey, model := ai.ResolveSettings(userSettings, overrideProvider, overrideModel, s.cfg)
	client := ai.NewCustom(provider, baseURL, apiKey, model, s.cfg.AITimeout)
	return client, model
}

func (s *Service) GetAISettings(ctx context.Context, userID string) (*domain.UserAISettings, error) {
	if s.aiSettings == nil {
		return &domain.UserAISettings{
			UserID:   userID,
			Provider: "openai",
			Model:    s.cfg.AIModel,
			BaseURL:  s.cfg.AIBaseURL,
		}, nil
	}
	settings, err := s.aiSettings.GetAISettings(ctx, userID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return &domain.UserAISettings{
				UserID:   userID,
				Provider: "openai",
				Model:    s.cfg.AIModel,
				BaseURL:  s.cfg.AIBaseURL,
			}, nil
		}
		return nil, err
	}
	if len(settings.APIKey) > 8 {
		settings.APIKey = settings.APIKey[:4] + "..." + settings.APIKey[len(settings.APIKey)-4:]
	} else if len(settings.APIKey) > 0 {
		settings.APIKey = "••••••••"
	}
	return settings, nil
}

func (s *Service) UpdateAISettings(ctx context.Context, userID string, in domain.UserAISettings) (*domain.UserAISettings, error) {
	if s.aiSettings == nil {
		return nil, errors.New("ai settings repository unavailable")
	}
	in.UserID = userID
	in.Provider = strings.TrimSpace(in.Provider)
	in.Model = strings.TrimSpace(in.Model)
	in.BaseURL = strings.TrimSpace(in.BaseURL)
	in.APIKey = strings.TrimSpace(in.APIKey)
	in.CustomModel = strings.TrimSpace(in.CustomModel)

	if in.Provider == "" {
		in.Provider = "openai"
	}
	if in.Model == "" {
		in.Model = "gpt-4o-mini"
	}

	if strings.Contains(in.APIKey, "...") || strings.Contains(in.APIKey, "•") {
		existing, err := s.aiSettings.GetAISettings(ctx, userID)
		if err == nil && existing != nil {
			in.APIKey = existing.APIKey
		}
	}

	if err := s.aiSettings.UpsertAISettings(ctx, &in); err != nil {
		return nil, err
	}
	return s.GetAISettings(ctx, userID)
}

func (s *Service) TestAISettings(ctx context.Context, in domain.UserAISettings) error {
	provider, baseURL, apiKey, model := ai.ResolveSettings(&in, in.Provider, in.Model, s.cfg)
	client := ai.NewCustom(provider, baseURL, apiKey, model, 15*time.Second)
	return client.TestConnection(ctx)
}

func (s *Service) GetAIProviders() []domain.AIProviderInfo {
	return ai.GetProvidersCatalog()
}

func (s *Service) GeneratePRDStream(ctx context.Context, user, project string, idea domain.Idea, idempotencyKey string, onChunk func(string), onProgress func(step, message string)) (*domain.PRD, error) {
	return s.GeneratePRDStreamWithModel(ctx, user, project, idea, idempotencyKey, onChunk, onProgress, "", "")
}

func (s *Service) GeneratePRDStreamWithModel(ctx context.Context, user, project string, idea domain.Idea, idempotencyKey string, onChunk func(string), onProgress func(step, message string), providerOverride, modelOverride string) (*domain.PRD, error) {
	if idempotencyKey != "" && (len(idempotencyKey) < 8 || len(idempotencyKey) > 128) {
		return nil, ErrInvalidInput
	}
	if err := ValidateIdea(idea); err != nil {
		return nil, ErrInvalidInput
	}
	idea.ProductName = strings.TrimSpace(idea.ProductName)
	requestBody, err := json.Marshal(struct {
		ProjectID string      `json:"projectId"`
		Idea      domain.Idea `json:"idea"`
		Provider  string      `json:"provider,omitempty"`
		Model     string      `json:"model,omitempty"`
	}{project, idea, providerOverride, modelOverride})
	if err != nil {
		return nil, err
	}
	requestDigest := sha256.Sum256(requestBody)
	requestHash := hex.EncodeToString(requestDigest[:])
	if idempotencyKey != "" {
		existing, err := s.usage.IdempotentPRD(ctx, user, idempotencyKey, requestHash)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			if onProgress != nil {
				onProgress("done", "Dokumen PRD ditemukan dari cache permintaan sebelumnya.")
			}
			return existing, nil
		}
	}
	if _, err := s.projects.GetProject(ctx, project, user); err != nil {
		return nil, err
	}
	if err := s.ensureQuota(ctx, user); err != nil {
		return nil, err
	}

	aiClient, effectiveModel := s.resolveAIProvider(ctx, user, providerOverride, modelOverride)

	if onProgress != nil {
		onProgress("generating", fmt.Sprintf("AI (%s) sedang menganalisis ide dan menyusun dokumen PRD lengkap...", effectiveModel))
	}
	doc, in, out, err := aiClient.GenerateStream(ctx, idea, onChunk)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIProvider, err)
	}
	if onProgress != nil {
		onProgress("validating", "Memvalidasi kelengkapan seksi dokumen dan kriteria penerimaan...")
	}
	if err := ValidatePRDDocument(doc); err != nil {
		return nil, fmt.Errorf("AI menghasilkan PRD yang tidak lengkap atau tidak valid: %w", err)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	if onProgress != nil {
		onProgress("saving", "Menyimpan dokumen dan riwayat versi ke database...")
	}
	return s.usage.SaveGeneratedPRD(ctx, user, project, idea.ProductName, raw, effectiveModel, idempotencyKey, requestHash, in, out)
}

func (s *Service) GeneratePRD(ctx context.Context, user, project string, idea domain.Idea, idempotencyKey string) (*domain.PRD, error) {
	return s.GeneratePRDStreamWithModel(ctx, user, project, idea, idempotencyKey, nil, nil, "", "")
}

func (s *Service) GeneratePRDWithModel(ctx context.Context, user, project string, idea domain.Idea, idempotencyKey, providerOverride, modelOverride string) (*domain.PRD, error) {
	return s.GeneratePRDStreamWithModel(ctx, user, project, idea, idempotencyKey, nil, nil, providerOverride, modelOverride)
}

func ValidateIdea(idea domain.Idea) error {
	if len(strings.TrimSpace(idea.ProductName)) < 1 || len(idea.ProductName) > 120 || len(strings.TrimSpace(idea.Description)) < 1 || len(idea.Description) > 10000 || len(idea.Problem) > 5000 || len(idea.TargetUsers) > 3000 || len(idea.BusinessGoals) > 3000 || len(idea.Platform) > 80 || len(idea.ProductType) > 120 || len(idea.Technology) > 2000 || len(idea.Constraints) > 3000 || len(idea.Language) > 80 || len(idea.DetailLevel) > 40 || len(idea.Features) > 30 || len(idea.Clarifications) > 30 {
		return ErrInvalidInput
	}
	for _, feature := range idea.Features {
		if len(strings.TrimSpace(feature)) == 0 || len(feature) > 500 {
			return ErrInvalidInput
		}
	}
	for key, value := range idea.Clarifications {
		if len(key) > 80 || len(value) > 2000 {
			return ErrInvalidInput
		}
	}
	return nil
}

func validateAnswers(answers map[string]string) error {
	if len(answers) > 30 {
		return ErrInvalidInput
	}
	for key, value := range answers {
		if len(strings.TrimSpace(key)) == 0 || len(key) > 80 || len(value) > 2000 {
			return ErrInvalidInput
		}
	}
	return nil
}
func (s *Service) GeneratePRDFromSessionStream(ctx context.Context, user, project, sessionID, idempotencyKey string, onChunk func(string), onProgress func(step, message string)) (*domain.PRD, error) {
	return s.GeneratePRDFromSessionStreamWithModel(ctx, user, project, sessionID, idempotencyKey, onChunk, onProgress, "", "")
}

func (s *Service) GeneratePRDFromSessionStreamWithModel(ctx context.Context, user, project, sessionID, idempotencyKey string, onChunk func(string), onProgress func(step, message string), providerOverride, modelOverride string) (*domain.PRD, error) {
	session, err := s.collab.GetClarification(ctx, sessionID, user)
	if err != nil {
		return nil, err
	}
	if !session.Ready {
		return nil, ErrClarificationRequired
	}
	session.Idea.Clarifications = session.Answers
	return s.GeneratePRDStreamWithModel(ctx, user, project, session.Idea, idempotencyKey, onChunk, onProgress, providerOverride, modelOverride)
}

func (s *Service) GeneratePRDFromSession(ctx context.Context, user, project, sessionID, idempotencyKey string) (*domain.PRD, error) {
	return s.GeneratePRDFromSessionStreamWithModel(ctx, user, project, sessionID, idempotencyKey, nil, nil, "", "")
}

func (s *Service) GeneratePRDFromSessionWithModel(ctx context.Context, user, project, sessionID, idempotencyKey, providerOverride, modelOverride string) (*domain.PRD, error) {
	return s.GeneratePRDFromSessionStreamWithModel(ctx, user, project, sessionID, idempotencyKey, nil, nil, providerOverride, modelOverride)
}
func (s *Service) ensureQuota(ctx context.Context, user string) error {
	summary, err := s.usage.UsageSummary(ctx, user)
	if err != nil {
		return err
	}
	if remaining, ok := summary["remaining"].(int); ok && remaining <= 0 {
		return domain.ErrQuotaExceeded
	}
	return nil
}

func ValidatePRDDocument(doc domain.PRDDocument) error {
	if strings.TrimSpace(doc.ExecutiveSummary) == "" || strings.TrimSpace(doc.ProductOverview) == "" || strings.TrimSpace(doc.ProblemStatement) == "" {
		return fmt.Errorf("executive summary, product overview, and problem statement are required")
	}
	seen := map[string]bool{}
	for _, list := range [][]domain.Requirement{doc.UserStories, doc.FunctionalRequirements, doc.NonFunctionalRequirements} {
		for _, r := range list {
			if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.Description) == "" {
				return fmt.Errorf("requirement ID and description are required")
			}
			if seen[r.ID] {
				return fmt.Errorf("duplicate requirement ID %q", r.ID)
			}
			seen[r.ID] = true
			if r.Priority != "P0" && r.Priority != "P1" && r.Priority != "P2" {
				return fmt.Errorf("requirement %s has invalid priority", r.ID)
			}
			if len(r.AcceptanceCriteria) == 0 {
				return fmt.Errorf("requirement %s has no acceptance criteria", r.ID)
			}
			for _, criterion := range r.AcceptanceCriteria {
				if strings.TrimSpace(criterion) == "" {
					return fmt.Errorf("requirement %s has an empty acceptance criterion", r.ID)
				}
			}
		}
	}
	if len(doc.FunctionalRequirements) == 0 {
		return fmt.Errorf("at least one functional requirement is required")
	}
	return nil
}
func (s *Service) Usage(ctx context.Context, user string) (map[string]any, error) {
	return s.usage.UsageSummary(ctx, user)
}
func (s *Service) UsageHistory(ctx context.Context, user string, limit, offset int) ([]map[string]any, error) {
	return s.usage.UsageHistory(ctx, user, limit, offset)
}
func (s *Service) Plans(ctx context.Context) ([]map[string]any, error) { return s.usage.Plans(ctx) }
