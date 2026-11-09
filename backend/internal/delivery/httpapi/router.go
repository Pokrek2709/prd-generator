package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/your-org/prd-generator/backend/internal/config"
	"github.com/your-org/prd-generator/backend/internal/domain"
	"github.com/your-org/prd-generator/backend/internal/infrastructure/postgres"
	"github.com/your-org/prd-generator/backend/internal/service"
)

type API struct {
	cfg    config.Config
	log    *slog.Logger
	svc    *service.Service
	limits *limiter
}

func New(cfg config.Config, log *slog.Logger, svc *service.Service) *gin.Engine {
	a := &API{cfg: cfg, log: log, svc: svc, limits: newLimiter()}
	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Error("invalid trusted proxy configuration", "error", err)
		_ = r.SetTrustedProxies(nil)
	}
	r.Use(gin.Recovery(), a.requestLog(), a.securityHeaders(), a.cors())
	r.MaxMultipartMemory = 8 << 20
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/openapi.yaml", func(c *gin.Context) {
		spec, err := openAPISpec()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", spec)
	})
	v := r.Group("/api/v1")
	auth := v.Group("/auth")
	auth.Use(a.rateLimit(10, time.Minute))
	auth.POST("/register", a.register)
	auth.POST("/login", a.login)
	auth.POST("/refresh", a.refresh)
	auth.POST("/logout", a.logout)
	private := v.Group("")
	private.Use(a.authenticate())
	private.GET("/dashboard", a.dashboard)
	private.GET("/auth/me", a.me)
	private.PATCH("/auth/profile", a.updateProfile)
	private.PUT("/auth/password", a.changePassword)
	private.POST("/projects", a.createProject)
	private.GET("/projects", a.listProjects)
	private.GET("/projects/:projectId", a.getProject)
	private.PATCH("/projects/:projectId", a.updateProject)
	private.POST("/projects/:projectId/archive", a.archiveProject)
	private.POST("/projects/:projectId/duplicate", a.duplicateProject)
	private.DELETE("/projects/:projectId", a.deleteProject)
	private.GET("/projects/:projectId/prds", a.listProjectPRDs)
	private.POST("/projects/:projectId/prds", a.createPRD)
	private.GET("/prds", a.listPRDs)
	private.GET("/prds/:prdId", a.getPRD)
	private.GET("/prds/:prdId/export/markdown", a.exportPRDMarkdown)
	private.GET("/prds/:prdId/export/pdf", a.exportPRDPDF)
	private.GET("/prds/:prdId/export/docx", a.exportPRDDOCX)
	private.GET("/prds/:prdId/export/jira", a.exportPRDJira)
	private.GET("/prds/:prdId/export/linear", a.exportPRDLinear)
	private.GET("/prds/:prdId/print", a.printPRD)
	private.PATCH("/prds/:prdId", a.updatePRD)
	private.DELETE("/prds/:prdId", a.deletePRD)
	private.POST("/prds/:prdId/duplicate", a.duplicatePRD)
	private.GET("/prds/:prdId/versions", a.versions)
	private.POST("/prds/:prdId/restore/:versionId", a.restore)
	private.POST("/prds/:prdId/share-links", a.createShareLink)
	private.GET("/prds/:prdId/share-links", a.listShareLinks)
	private.DELETE("/prds/:prdId/share-links/:shareId", a.revokeShareLink)
	private.POST("/ai/generate-prd", a.rateLimit(5, time.Minute), a.generatePRD)
	private.POST("/ai/generate-prd/stream", a.rateLimit(5, time.Minute), a.generatePRDStream)
	private.POST("/ai/clarify", a.rateLimit(10, time.Minute), a.clarify)
	private.POST("/ai/actions", a.rateLimit(10, time.Minute), a.aiAction)
	private.POST("/ai/regenerate-section", a.rateLimit(10, time.Minute), func(c *gin.Context) { a.fixedAIAction(c, "regenerate_section") })
	private.POST("/ai/review-prd", a.rateLimit(10, time.Minute), func(c *gin.Context) { a.fixedAIAction(c, "review_prd") })
	private.POST("/ai/generate-tasks", a.rateLimit(10, time.Minute), func(c *gin.Context) { a.fixedAIAction(c, "generate_tasks") })
	private.GET("/settings/ai", a.getAISettings)
	private.PUT("/settings/ai", a.updateAISettings)
	private.POST("/settings/ai/test", a.testAISettings)
	private.GET("/usage", a.usage)
	private.GET("/usage/history", a.usageHistory)
	v.GET("/shared/:token", a.publicShare)
	v.GET("/plans", a.plans)
	v.GET("/ai/providers", a.getAIProviders)
	return r
}

func (a *API) requestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		a.log.Info("http request", "method", c.Request.Method, "path", path, "status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds())
	}
}
func (a *API) securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}
func (a *API) cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && origin == a.cfg.FrontendOrigin {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, If-Match, Idempotency-Key")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
func (a *API) rateLimit(max int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.limits.allow(c.ClientIP(), max, window) {
			fail(c, http.StatusTooManyRequests, "rate_limited", "Terlalu banyak permintaan. Coba lagi nanti.")
			return
		}
		c.Next()
	}
}
func (a *API) authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		if cookie, err := c.Cookie("access_token"); err == nil {
			token = cookie
		}
		if token == "" {
			token = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}
		id, err := a.svc.VerifyAccess(token)
		if err != nil {
			fail(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid atau sudah berakhir.")
			return
		}
		c.Set("userID", id)
		c.Next()
	}
}
func userID(c *gin.Context) string { id, _ := c.Get("userID"); v, _ := id.(string); return v }
func (a *API) setCookies(c *gin.Context, access, refresh string) {
	http.SetCookie(c.Writer, &http.Cookie{Name: "access_token", Value: access, Path: "/", HttpOnly: true, Secure: a.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: int(a.cfg.AccessTokenTTL.Seconds())})
	http.SetCookie(c.Writer, &http.Cookie{Name: "refresh_token", Value: refresh, Path: "/api/v1/auth", HttpOnly: true, Secure: a.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: int(a.cfg.RefreshTokenTTL.Seconds())})
}
func clearCookies(c *gin.Context) {
	for _, name := range []string{"access_token", "refresh_token"} {
		path := "/"
		if name == "refresh_token" {
			path = "/api/v1/auth"
		}
		http.SetCookie(c.Writer, &http.Cookie{Name: name, Value: "", Path: path, HttpOnly: true, MaxAge: -1, SameSite: http.SameSiteLaxMode})
	}
}
func (a *API) register(c *gin.Context) {
	var in struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if !bind(c, &in) {
		return
	}
	u, access, refresh, err := a.svc.Register(c.Request.Context(), in.Email, in.Name, in.Password)
	if err != nil {
		a.handleError(c, err)
		return
	}
	a.setCookies(c, access, refresh)
	u.PasswordHash = ""
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"user": u}})
}
func (a *API) login(c *gin.Context) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !bind(c, &in) {
		return
	}
	u, access, refresh, err := a.svc.Login(c.Request.Context(), in.Email, in.Password)
	if err != nil {
		a.handleError(c, err)
		return
	}
	a.setCookies(c, access, refresh)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"user": u}})
}
func (a *API) refresh(c *gin.Context) {
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		fail(c, http.StatusUnauthorized, "unauthorized", "Refresh token tidak valid.")
		return
	}
	u, access, refresh, err := a.svc.Refresh(c.Request.Context(), cookie)
	if err != nil {
		clearCookies(c)
		a.handleError(c, err)
		return
	}
	a.setCookies(c, access, refresh)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"user": u}})
}
func (a *API) logout(c *gin.Context) {
	cookie, _ := c.Cookie("refresh_token")
	_ = a.svc.Logout(c.Request.Context(), cookie)
	clearCookies(c)
	c.Status(http.StatusNoContent)
}
func (a *API) me(c *gin.Context) {
	u, err := a.svc.Me(c.Request.Context(), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": u})
}
func (a *API) updateProfile(c *gin.Context) {
	var in struct {
		Name string `json:"name"`
	}
	if !bind(c, &in) {
		return
	}
	u, err := a.svc.UpdateProfile(c.Request.Context(), userID(c), in.Name)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": u})
}
func (a *API) changePassword(c *gin.Context) {
	var in struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if !bind(c, &in) {
		return
	}
	if err := a.svc.ChangePassword(c.Request.Context(), userID(c), in.CurrentPassword, in.NewPassword); err != nil {
		a.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (a *API) createProject(c *gin.Context) {
	var in struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !bind(c, &in) {
		return
	}
	p, err := a.svc.CreateProject(c.Request.Context(), userID(c), in.Name, in.Description)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": p})
}
func paging(c *gin.Context) (int, int) {
	limit := intParam(c, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	page := intParam(c, "page", 1)
	if page < 1 {
		page = 1
	}
	return limit, (page - 1) * limit
}
func intParam(c *gin.Context, key string, def int) int {
	var x int
	if _, err := fmt.Sscanf(c.Query(key), "%d", &x); err != nil || x < 1 {
		return def
	}
	return x
}
func (a *API) listProjects(c *gin.Context) {
	limit, offset := paging(c)
	items, total, err := a.svc.ListProjects(c.Request.Context(), userID(c), c.Query("q"), c.Query("status"), c.DefaultQuery("sort", "updated_desc"), c.Query("dateFrom"), c.Query("dateTo"), limit, offset)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"total": total, "limit": limit, "page": offset/limit + 1}})
}
func (a *API) dashboard(c *gin.Context) {
	data, err := a.svc.Dashboard(c.Request.Context(), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}
func (a *API) archiveProject(c *gin.Context) {
	if err := a.svc.ArchiveProject(c.Request.Context(), c.Param("projectId"), userID(c)); err != nil {
		a.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (a *API) duplicateProject(c *gin.Context) {
	p, err := a.svc.DuplicateProject(c.Request.Context(), c.Param("projectId"), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": p})
}
func (a *API) listProjectPRDs(c *gin.Context) {
	limit, offset := paging(c)
	items, total, err := a.svc.ListProjectPRDs(c.Request.Context(), c.Param("projectId"), userID(c), limit, offset)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"total": total, "limit": limit, "page": offset/limit + 1}})
}
func (a *API) listPRDs(c *gin.Context) {
	limit, offset := paging(c)
	items, total, err := a.svc.ListPRDs(c.Request.Context(), userID(c), c.Query("q"), c.Query("status"), c.Query("dateFrom"), c.Query("dateTo"), c.DefaultQuery("sort", "updated_desc"), limit, offset)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"total": total, "limit": limit, "page": offset/limit + 1}})
}
func (a *API) getProject(c *gin.Context) {
	p, err := a.svc.GetProject(c.Request.Context(), c.Param("projectId"), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}
func (a *API) updateProject(c *gin.Context) {
	var in struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if !bind(c, &in) {
		return
	}
	p, err := a.svc.PatchProject(c.Request.Context(), c.Param("projectId"), userID(c), in.Name, in.Description)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}
func (a *API) deleteProject(c *gin.Context) {
	if err := a.svc.DeleteProject(c.Request.Context(), c.Param("projectId"), userID(c)); err != nil {
		a.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (a *API) createPRD(c *gin.Context) {
	var in struct {
		Title string `json:"title"`
	}
	if !bind(c, &in) {
		return
	}
	p, err := a.svc.CreatePRD(c.Request.Context(), c.Param("projectId"), userID(c), in.Title)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": p})
}
func (a *API) getPRD(c *gin.Context) {
	p, err := a.svc.GetPRD(c.Request.Context(), c.Param("prdId"), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}

func (a *API) exportPRDMarkdown(c *gin.Context) {
	p, markdown, err := a.svc.ExportPRDMarkdown(c.Request.Context(), c.Param("prdId"), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	filename := markdownFilename(p.Title)
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", markdown)
}
func (a *API) exportPRDPDF(c *gin.Context) {
	p, markdown, err := a.svc.ExportPRDMarkdown(c.Request.Context(), c.Param("prdId"), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	body, err := service.RenderPRDPDF(string(markdown))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+strings.TrimSuffix(markdownFilename(p.Title), ".md")+`.pdf"`)
	c.Data(http.StatusOK, "application/pdf", body)
}
func (a *API) exportPRDDOCX(c *gin.Context) {
	p, markdown, err := a.svc.ExportPRDMarkdown(c.Request.Context(), c.Param("prdId"), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	body, err := service.RenderPRDDOCX(string(markdown))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+strings.TrimSuffix(markdownFilename(p.Title), ".md")+`.docx"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", body)
}
func (a *API) exportPRDJira(c *gin.Context) {
	p, err := a.svc.GetPRD(c.Request.Context(), c.Param("prdId"), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	body, err := service.RenderPRDJiraCSV(p)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+strings.TrimSuffix(markdownFilename(p.Title), ".md")+`-jira.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", body)
}

func (a *API) exportPRDLinear(c *gin.Context) {
	p, err := a.svc.GetPRD(c.Request.Context(), c.Param("prdId"), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	body, err := service.RenderPRDLinearCSV(p)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+strings.TrimSuffix(markdownFilename(p.Title), ".md")+`-linear.csv"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", body)
}

func (a *API) printPRD(c *gin.Context) {
	p, markdown, err := a.svc.ExportPRDMarkdown(c.Request.Context(), c.Param("prdId"), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	body, err := service.RenderPrintHTML(p.Title, string(markdown))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", body)
}

func markdownFilename(title string) string {
	var out strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			out.WriteRune(r)
			lastDash = false
		} else if !lastDash && out.Len() > 0 {
			out.WriteByte('-')
			lastDash = true
		}
	}
	name := strings.Trim(out.String(), "-")
	if name == "" {
		name = "prd"
	}
	return name + ".md"
}
func (a *API) updatePRD(c *gin.Context) {
	var in struct {
		Title   string          `json:"title"`
		Status  string          `json:"status"`
		Content json.RawMessage `json:"content"`
		Version int             `json:"version"`
	}
	if !bind(c, &in) {
		return
	}
	p, err := a.svc.UpdatePRD(c.Request.Context(), c.Param("prdId"), userID(c), in.Title, in.Status, in.Content, in.Version)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}
func (a *API) deletePRD(c *gin.Context) {
	if err := a.svc.DeletePRD(c.Request.Context(), c.Param("prdId"), userID(c)); err != nil {
		a.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (a *API) versions(c *gin.Context) {
	v, err := a.svc.Versions(c.Request.Context(), c.Param("prdId"), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": v})
}
func (a *API) restore(c *gin.Context) {
	p, err := a.svc.Restore(c.Request.Context(), c.Param("prdId"), userID(c), c.Param("versionId"))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}
func (a *API) duplicatePRD(c *gin.Context) {
	p, err := a.svc.DuplicatePRD(c.Request.Context(), c.Param("prdId"), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": p})
}
func (a *API) createShareLink(c *gin.Context) {
	var in struct {
		ExpiresInHours int `json:"expiresInHours"`
	}
	if c.Request.ContentLength > 0 && !bind(c, &in) {
		return
	}
	ttl := time.Duration(in.ExpiresInHours) * time.Hour
	link, err := a.svc.CreateShareLink(c.Request.Context(), c.Param("prdId"), userID(c), ttl)
	if err != nil {
		a.handleError(c, err)
		return
	}
	linkURL := strings.TrimRight(a.cfg.PublicAPIURL, "/") + "/api/v1/shared/" + link.Token
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"id": link.ID, "prdId": link.PRDID, "url": linkURL, "token": link.Token, "expiresAt": link.ExpiresAt}})
}
func (a *API) listShareLinks(c *gin.Context) {
	links, err := a.svc.ListShareLinks(c.Request.Context(), c.Param("prdId"), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": links})
}
func (a *API) revokeShareLink(c *gin.Context) {
	if err := a.svc.RevokeShareLink(c.Request.Context(), c.Param("prdId"), userID(c), c.Param("shareId")); err != nil {
		a.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (a *API) publicShare(c *gin.Context) {
	p, err := a.svc.SharedPRD(c.Request.Context(), c.Param("token"))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": p})
}
func (a *API) clarify(c *gin.Context) {
	var in struct {
		SessionID string            `json:"sessionId"`
		Idea      domain.Idea       `json:"idea"`
		Answers   map[string]string `json:"answers"`
		Skip      bool              `json:"skip"`
	}
	if !bind(c, &in) {
		return
	}
	session, err := a.svc.Clarify(c.Request.Context(), userID(c), in.SessionID, in.Idea, in.Answers, in.Skip)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": session})
}
func (a *API) aiAction(c *gin.Context) {
	var in struct {
		Action string          `json:"action"`
		PRDID  string          `json:"prdId"`
		Input  json.RawMessage `json:"input"`
	}
	if !bind(c, &in) {
		return
	}
	result, err := a.svc.AIAction(c.Request.Context(), userID(c), in.Action, in.PRDID, in.Input)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}
func (a *API) fixedAIAction(c *gin.Context, action string) {
	var in struct {
		PRDID string          `json:"prdId"`
		Input json.RawMessage `json:"input"`
	}
	if !bind(c, &in) {
		return
	}
	result, err := a.svc.AIAction(c.Request.Context(), userID(c), action, in.PRDID, in.Input)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}
func (a *API) generatePRD(c *gin.Context) {
	var in struct {
		ProjectID      string      `json:"projectId"`
		SessionID      string      `json:"sessionId"`
		IdempotencyKey string      `json:"idempotencyKey"`
		Provider       string      `json:"provider"`
		Model          string      `json:"model"`
		Idea           domain.Idea `json:"idea"`
	}
	if !bind(c, &in) {
		return
	}
	if header := c.GetHeader("Idempotency-Key"); header != "" {
		if in.IdempotencyKey != "" && in.IdempotencyKey != header {
			fail(c, http.StatusBadRequest, "invalid_input", "Idempotency-Key header dan body tidak cocok.")
			return
		}
		in.IdempotencyKey = header
	}
	var p *domain.PRD
	var err error
	if in.SessionID != "" {
		p, err = a.svc.GeneratePRDFromSessionWithModel(c.Request.Context(), userID(c), in.ProjectID, in.SessionID, in.IdempotencyKey, in.Provider, in.Model)
	} else {
		p, err = a.svc.GeneratePRDWithModel(c.Request.Context(), userID(c), in.ProjectID, in.Idea, in.IdempotencyKey, in.Provider, in.Model)
	}
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": p})
}
func (a *API) generatePRDStream(c *gin.Context) {
	var in struct {
		ProjectID      string      `json:"projectId"`
		SessionID      string      `json:"sessionId"`
		IdempotencyKey string      `json:"idempotencyKey"`
		Provider       string      `json:"provider"`
		Model          string      `json:"model"`
		Idea           domain.Idea `json:"idea"`
	}
	if !bind(c, &in) {
		return
	}
	if header := c.GetHeader("Idempotency-Key"); header != "" {
		if in.IdempotencyKey != "" && in.IdempotencyKey != header {
			fail(c, http.StatusBadRequest, "invalid_input", "Idempotency-Key header dan body tidak cocok.")
			return
		}
		in.IdempotencyKey = header
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	sendSSE := func(event string, payload any) error {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}

	onChunk := func(chunk string) {
		_ = sendSSE("chunk", gin.H{"chunk": chunk})
	}
	onProgress := func(step, message string) {
		_ = sendSSE("progress", gin.H{"step": step, "message": message})
	}

	_ = sendSSE("progress", gin.H{"step": "init", "message": "Menghubungkan ke layanan AI generator..."})

	var p *domain.PRD
	var err error
	if in.SessionID != "" {
		p, err = a.svc.GeneratePRDFromSessionStreamWithModel(c.Request.Context(), userID(c), in.ProjectID, in.SessionID, in.IdempotencyKey, onChunk, onProgress, in.Provider, in.Model)
	} else {
		p, err = a.svc.GeneratePRDStreamWithModel(c.Request.Context(), userID(c), in.ProjectID, in.Idea, in.IdempotencyKey, onChunk, onProgress, in.Provider, in.Model)
	}

	if err != nil {
		a.log.Error("streaming prd generation failed", "error", err)
		code, msg := a.describeError(err)
		_ = sendSSE("error", gin.H{"code": code, "message": msg})
		return
	}

	_ = sendSSE("done", gin.H{"data": p})
}
func (a *API) getAIProviders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": a.svc.GetAIProviders()})
}
func (a *API) getAISettings(c *gin.Context) {
	settings, err := a.svc.GetAISettings(c.Request.Context(), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings})
}
func (a *API) updateAISettings(c *gin.Context) {
	var in domain.UserAISettings
	if !bind(c, &in) {
		return
	}
	settings, err := a.svc.UpdateAISettings(c.Request.Context(), userID(c), in)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings})
}
func (a *API) testAISettings(c *gin.Context) {
	var in domain.UserAISettings
	if !bind(c, &in) {
		return
	}
	if err := a.svc.TestAISettings(c.Request.Context(), in); err != nil {
		fail(c, http.StatusBadRequest, "ai_test_failed", fmt.Sprintf("Koneksi gagal: %v", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "connected", "message": "Koneksi berhasil! Provider dan model siap digunakan."}})
}
func (a *API) describeError(err error) (string, string) {
	if errors.Is(err, context.DeadlineExceeded) {
		return "upstream_timeout", "AI provider tidak merespons tepat waktu. Silakan coba lagi."
	}
	if errors.Is(err, domain.ErrQuotaExceeded) {
		return "quota_exceeded", "Kuota AI bulan ini telah habis."
	}
	if errors.Is(err, domain.ErrIdempotencyConflict) {
		return "idempotency_conflict", "Idempotency key sudah dipakai untuk permintaan yang berbeda."
	}
	switch {
	case errors.Is(err, service.ErrAIProvider):
		return "ai_provider_error", "Provider AI gagal memproses permintaan. Silakan coba lagi."
	case errors.Is(err, service.ErrClarificationRequired):
		return "clarification_required", "Selesaikan wawancara klarifikasi atau lewati pertanyaan sebelum membuat PRD."
	case errors.Is(err, service.ErrInvalidInput):
		return "invalid_input", "Periksa kembali data yang dikirim."
	case errors.Is(err, postgres.ErrNotFound):
		return "not_found", "Data tidak ditemukan."
	default:
		return "internal_error", "Terjadi kesalahan pada server: " + err.Error()
	}
}
func (a *API) usage(c *gin.Context) {
	v, err := a.svc.Usage(c.Request.Context(), userID(c))
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": v})
}
func (a *API) plans(c *gin.Context) {
	data, err := a.svc.Plans(c.Request.Context())
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}
func (a *API) usageHistory(c *gin.Context) {
	limit, offset := paging(c)
	v, err := a.svc.UsageHistory(c.Request.Context(), userID(c), limit, offset)
	if err != nil {
		a.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": v, "meta": gin.H{"limit": limit, "page": offset/limit + 1}})
}
func bind(c *gin.Context, v any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)
	if err := c.ShouldBindJSON(v); err != nil {
		fail(c, http.StatusBadRequest, "invalid_request", "Body JSON tidak valid.")
		return false
	}
	return true
}
func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
func (a *API) handleError(c *gin.Context, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		fail(c, http.StatusGatewayTimeout, "upstream_timeout", "AI provider tidak merespons tepat waktu. Silakan coba lagi.")
		return
	}
	if errors.Is(err, domain.ErrQuotaExceeded) {
		fail(c, http.StatusPaymentRequired, "quota_exceeded", "Kuota AI bulan ini telah habis.")
		return
	}
	if errors.Is(err, domain.ErrIdempotencyConflict) {
		fail(c, http.StatusConflict, "idempotency_conflict", "Idempotency key sudah dipakai untuk permintaan yang berbeda.")
		return
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		fail(c, http.StatusConflict, "conflict", "Email atau data unik lainnya sudah digunakan.")
		return
	}
	switch {
	case errors.Is(err, service.ErrAIProvider):
		fail(c, http.StatusBadGateway, "ai_provider_error", "Provider AI gagal memproses permintaan. Silakan coba lagi.")
	case errors.Is(err, service.ErrClarificationRequired):
		fail(c, http.StatusConflict, "clarification_required", "Selesaikan wawancara klarifikasi atau lewati pertanyaan sebelum membuat PRD.")
	case errors.Is(err, service.ErrInvalidCredentials):
		fail(c, http.StatusUnauthorized, "invalid_credentials", "Email atau kata sandi tidak valid.")
	case errors.Is(err, service.ErrInvalidInput):
		fail(c, http.StatusBadRequest, "invalid_input", "Periksa kembali data yang dikirim.")
	case errors.Is(err, service.ErrVersionConflict):
		fail(c, http.StatusConflict, "version_conflict", "PRD telah berubah. Muat ulang sebelum menyimpan lagi.")
	case errors.Is(err, postgres.ErrNotFound):
		fail(c, http.StatusNotFound, "not_found", "Data tidak ditemukan.")
	case errors.Is(err, service.ErrConflict):
		fail(c, http.StatusConflict, "conflict", "Data sudah terdaftar.")
	default:
		a.log.Error("request failed", "error", err)
		fail(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan pada server.")
	}
}

type limiter struct {
	mu    sync.Mutex
	hits  map[string][]time.Time
	calls uint64
}

func newLimiter() *limiter { return &limiter{hits: make(map[string][]time.Time)} }
func (l *limiter) allow(key string, max int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.calls++
	if l.calls%512 == 0 {
		for ip, times := range l.hits {
			keep := times[:0]
			for _, at := range times {
				if now.Sub(at) < 10*time.Minute {
					keep = append(keep, at)
				}
			}
			if len(keep) == 0 {
				delete(l.hits, ip)
			} else {
				l.hits[ip] = keep
			}
		}
	}
	old := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if now.Sub(t) < window {
			old = append(old, t)
		}
	}
	if len(old) >= max {
		l.hits[key] = old
		return false
	}
	l.hits[key] = append(old, now)
	return true
}
