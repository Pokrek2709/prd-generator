package postgres

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/prd-generator/backend/internal/domain"
)

func TestRepositoryVersioningAndOwnershipPostgres(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "prd_test_" + uuid.NewString()[:8]
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
	}()
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	migrationDir := filepath.Join("..", "..", "..", "migrations")
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		t.Fatal(err)
	}
	migrations := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" && filepath.Base(entry.Name()) != "" && entry.Name()[len(entry.Name())-7:] == ".up.sql" {
			migrations = append(migrations, filepath.Join(migrationDir, entry.Name()))
		}
	}
	sort.Strings(migrations)
	for _, path := range migrations {
		sql, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, err = pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
	repo := New(pool)
	u := &domain.User{Email: "tester@example.com", Name: "Tester", PasswordHash: "hash"}
	if err = repo.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	p := &domain.Project{UserID: u.ID, Name: "Project", Description: "Test"}
	if err = repo.CreateProject(ctx, p); err != nil {
		t.Fatal(err)
	}
	prd := &domain.PRD{ProjectID: p.ID, Title: "Draft", Status: "draft", Content: json.RawMessage(`{"executiveSummary":"first"}`)}
	if err = repo.CreatePRD(ctx, prd); err != nil {
		t.Fatal(err)
	}
	updated := &domain.PRD{ID: prd.ID, UserID: u.ID, Title: "Draft v2", Status: "in_progress", Content: json.RawMessage(`{"executiveSummary":"second"}`)}
	if err = repo.UpdatePRD(ctx, updated, 1); err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2, got %d", updated.Version)
	}
	versions, err := repo.ListVersions(ctx, prd.ID, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 || versions[0].Version != 1 || versions[0].Title != "Draft" {
		t.Fatalf("unexpected version history: %#v", versions)
	}
	if _, err = repo.GetPRD(ctx, prd.ID, uuid.NewString()); err != ErrNotFound {
		t.Fatalf("expected owner isolation, got %v", err)
	}
	if _, err = repo.RestoreVersion(ctx, prd.ID, u.ID, versions[0].ID); err != nil {
		t.Fatal(err)
	}
	current, err := repo.GetPRD(ctx, prd.ID, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Title != "Draft" || current.Status != "draft" || current.Version != 3 {
		t.Fatalf("restore did not create a new full version: %#v", current)
	}
	link, err := repo.CreateShareLink(ctx, prd.ID, u.ID, "share-token-hash", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.SharedPRD(ctx, "share-token-hash"); err != nil {
		t.Fatalf("share link did not grant read access: %v", err)
	}
	if err = repo.RevokeShareLink(ctx, prd.ID, u.ID, link.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.SharedPRD(ctx, "share-token-hash"); err != ErrNotFound {
		t.Fatalf("revoked share link remained accessible: %v", err)
	}
	generated, err := repo.SaveGeneratedPRD(ctx, u.ID, p.ID, "Generated", json.RawMessage(`{"executiveSummary":"Generated"}`), "test-model", "request-key-1", "request-hash-1", 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := repo.SaveGeneratedPRD(ctx, u.ID, p.ID, "Generated", json.RawMessage(`{"executiveSummary":"Generated"}`), "test-model", "request-key-1", "request-hash-1", 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != generated.ID {
		t.Fatalf("idempotent retry created another PRD: %s != %s", replayed.ID, generated.ID)
	}
	if _, err = repo.SaveGeneratedPRD(ctx, u.ID, p.ID, "Generated", json.RawMessage(`{"executiveSummary":"Different"}`), "test-model", "request-key-1", "different-hash", 2, 3); err != domain.ErrIdempotencyConflict {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
	for i := 0; i < 9; i++ {
		if err = repo.RecordAIUsage(ctx, u.ID, "test-model", "generate_tasks", 1, 1); err != nil {
			t.Fatalf("record usage %d: %v", i, err)
		}
	}
	if err = repo.RecordAIUsage(ctx, u.ID, "test-model", "generate_tasks", 1, 1); err != domain.ErrQuotaExceeded {
		t.Fatalf("expected Free plan quota enforcement, got %v", err)
	}
}
