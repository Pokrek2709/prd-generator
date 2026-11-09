package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment, HTTPAddr, DatabaseURL, JWTSecret, FrontendOrigin, PublicAPIURL string
	TrustedProxies                                                              []string
	AccessTokenTTL, RefreshTokenTTL, AITimeout                                  time.Duration
	CookieSecure                                                                bool
	AIBaseURL, AIAPIKey, AIModel                                                string
}

func Load() (Config, error) {
	c := Config{Environment: env("APP_ENV", "development"), HTTPAddr: env("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), JWTSecret: os.Getenv("JWT_SECRET"), FrontendOrigin: env("FRONTEND_ORIGIN", "http://localhost:3000"), PublicAPIURL: env("PUBLIC_API_URL", "http://localhost:8080"), AIBaseURL: env("AI_BASE_URL", "https://api.openai.com/v1"), AIAPIKey: os.Getenv("AI_API_KEY"), AIModel: env("AI_MODEL", "gpt-4o-mini")}
	for _, proxy := range strings.Split(os.Getenv("TRUSTED_PROXIES"), ",") {
		if proxy = strings.TrimSpace(proxy); proxy != "" {
			c.TrustedProxies = append(c.TrustedProxies, proxy)
		}
	}
	var err error
	if c.AccessTokenTTL, err = duration("ACCESS_TOKEN_TTL", "15m"); err != nil {
		return c, err
	}
	if c.RefreshTokenTTL, err = duration("REFRESH_TOKEN_TTL", "720h"); err != nil {
		return c, err
	}
	if c.AITimeout, err = duration("AI_TIMEOUT", "45s"); err != nil {
		return c, err
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.JWTSecret) < 32 {
		return c, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	secure, err := strconv.ParseBool(env("COOKIE_SECURE", strconv.FormatBool(c.Environment == "production")))
	if err != nil {
		return c, fmt.Errorf("COOKIE_SECURE: %w", err)
	}
	c.CookieSecure = secure
	return c, nil
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func duration(key, fallback string) (time.Duration, error) {
	v, err := time.ParseDuration(env(key, fallback))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}
