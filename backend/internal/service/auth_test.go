package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/your-org/prd-generator/backend/internal/config"
	"github.com/your-org/prd-generator/backend/internal/domain"
)

type memoryUsers struct {
	usersByID map[string]*domain.User
	emailToID map[string]string
	refresh   map[string]domain.RefreshToken
}

func newMemoryUsers() *memoryUsers {
	return &memoryUsers{usersByID: map[string]*domain.User{}, emailToID: map[string]string{}, refresh: map[string]domain.RefreshToken{}}
}
func (r *memoryUsers) CreateUser(_ context.Context, u *domain.User) error {
	u.ID = "user-1"
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt
	copy := *u
	r.usersByID[u.ID] = &copy
	r.emailToID[u.Email] = u.ID
	return nil
}
func (r *memoryUsers) UserByEmail(_ context.Context, email string) (*domain.User, error) {
	id, ok := r.emailToID[email]
	if !ok {
		return nil, errors.New("not found")
	}
	u := *r.usersByID[id]
	return &u, nil
}
func (r *memoryUsers) UserByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := r.usersByID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	copy := *u
	return &copy, nil
}
func (r *memoryUsers) UpdateUser(_ context.Context, u *domain.User) error {
	r.usersByID[u.ID].Name = u.Name
	return nil
}
func (r *memoryUsers) UpdatePassword(_ context.Context, id, hash string) error {
	r.usersByID[id].PasswordHash = hash
	return nil
}
func (r *memoryUsers) SaveRefreshToken(_ context.Context, t domain.RefreshToken) error {
	r.refresh[t.Hash] = t
	return nil
}
func (r *memoryUsers) ConsumeRefreshToken(_ context.Context, hash string) (string, error) {
	t, ok := r.refresh[hash]
	if !ok || time.Now().After(t.ExpiresAt) {
		return "", errors.New("not found")
	}
	delete(r.refresh, hash)
	return t.UserID, nil
}
func (r *memoryUsers) RevokeRefreshToken(_ context.Context, hash string) error {
	delete(r.refresh, hash)
	return nil
}
func (r *memoryUsers) RevokeUserRefreshTokens(_ context.Context, id string) error {
	for hash, t := range r.refresh {
		if t.UserID == id {
			delete(r.refresh, hash)
		}
	}
	return nil
}

func TestAuthRegisterLoginRefreshAndLogout(t *testing.T) {
	repo := newMemoryUsers()
	cfg := config.Config{JWTSecret: strings.Repeat("s", 32), AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour}
	svc := New(repo, nil, nil, nil, nil, cfg, nil)
	u, access, refresh, err := svc.Register(context.Background(), "user@example.com", "User", "long-enough-password")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == "" || access == "" || refresh == "" {
		t.Fatal("registration did not issue account and tokens")
	}
	owner, err := svc.VerifyAccess(access)
	if err != nil || owner != u.ID {
		t.Fatalf("access token verification failed: %q, %v", owner, err)
	}
	if _, _, _, err = svc.Login(context.Background(), "USER@example.com", "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("invalid password should be rejected: %v", err)
	}
	if _, _, _, err = svc.Login(context.Background(), "USER@example.com", "long-enough-password"); err != nil {
		t.Fatalf("valid login failed: %v", err)
	}
	_, rotatedAccess, rotatedRefresh, err := svc.Refresh(context.Background(), refresh)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.VerifyAccess(rotatedAccess); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = svc.Refresh(context.Background(), refresh); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("rotated refresh token was reusable: %v", err)
	}
	if err = svc.Logout(context.Background(), rotatedRefresh); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = svc.Refresh(context.Background(), rotatedRefresh); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("logged-out refresh token was reusable: %v", err)
	}
}

func TestAuthRejectsInvalidRegistration(t *testing.T) {
	svc := New(newMemoryUsers(), nil, nil, nil, nil, config.Config{}, nil)
	if _, _, _, err := svc.Register(context.Background(), "not-an-email", "User", "short"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid registration accepted: %v", err)
	}
}
