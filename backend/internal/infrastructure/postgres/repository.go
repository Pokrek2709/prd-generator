package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/prd-generator/backend/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Repository struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) CreateUser(ctx context.Context, u *domain.User) error {
	u.ID = uuid.NewString()
	return r.db.QueryRow(ctx, `INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,$3,$4) RETURNING created_at,updated_at`, u.ID, strings.ToLower(u.Email), u.Name, u.PasswordHash).Scan(&u.CreatedAt, &u.UpdatedAt)
}
func (r *Repository) UserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT id,email,name,password_hash,created_at,updated_at FROM users WHERE email=$1`, strings.ToLower(email)))
}
func (r *Repository) UserByID(ctx context.Context, id string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT id,email,name,password_hash,created_at,updated_at FROM users WHERE id=$1`, id))
}
func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}
func (r *Repository) UpdateUser(ctx context.Context, u *domain.User) error {
	return r.db.QueryRow(ctx, `UPDATE users SET name=$2,updated_at=now() WHERE id=$1 RETURNING updated_at`, u.ID, u.Name).Scan(&u.UpdatedAt)
}
func (r *Repository) UpdatePassword(ctx context.Context, id, hash string) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET password_hash=$2,updated_at=now() WHERE id=$1`, id, hash)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
func (r *Repository) SaveRefreshToken(ctx context.Context, t domain.RefreshToken) error {
	_, err := r.db.Exec(ctx, `INSERT INTO refresh_tokens(token_hash,user_id,expires_at) VALUES($1,$2,$3)`, t.Hash, t.UserID, t.ExpiresAt)
	return err
}
func (r *Repository) ConsumeRefreshToken(ctx context.Context, hash string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `DELETE FROM refresh_tokens WHERE token_hash=$1 AND expires_at>now() RETURNING user_id`, hash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}
func (r *Repository) RevokeRefreshToken(ctx context.Context, hash string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash=$1`, hash)
	return err
}
func (r *Repository) RevokeUserRefreshTokens(ctx context.Context, user string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id=$1`, user)
	return err
}

func (r *Repository) CreateProject(ctx context.Context, p *domain.Project) error {
	p.ID = uuid.NewString()
	p.Status = "active"
	return r.db.QueryRow(ctx, `INSERT INTO projects(id,user_id,name,description,status) VALUES($1,$2,$3,$4,$5) RETURNING created_at,updated_at`, p.ID, p.UserID, p.Name, p.Description, p.Status).Scan(&p.CreatedAt, &p.UpdatedAt)
}
func (r *Repository) ListProjects(ctx context.Context, user, search, status, sort, dateFrom, dateTo string, limit, offset int) ([]domain.Project, int, error) {
	if status != "active" && status != "archived" {
		status = ""
	}
	orderBy := "updated_at DESC"
	switch sort {
	case "created_asc":
		orderBy = "created_at ASC"
	case "created_desc":
		orderBy = "created_at DESC"
	case "name_asc":
		orderBy = "name ASC"
	case "name_desc":
		orderBy = "name DESC"
	case "updated_asc":
		orderBy = "updated_at ASC"
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM projects WHERE user_id=$1 AND ($2='' OR status=$2) AND ($3='' OR name ILIKE '%'||$3||'%' OR description ILIKE '%'||$3||'%') AND (NULLIF($4,'')::date IS NULL OR created_at>=NULLIF($4,'')::date) AND (NULLIF($5,'')::date IS NULL OR created_at<NULLIF($5,'')::date+INTERVAL '1 day')`, user, status, search, dateFrom, dateTo).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT id,user_id,name,description,status,created_at,updated_at FROM projects WHERE user_id=$1 AND ($2='' OR status=$2) AND ($3='' OR name ILIKE '%'||$3||'%' OR description ILIKE '%'||$3||'%') AND (NULLIF($4,'')::date IS NULL OR created_at>=NULLIF($4,'')::date) AND (NULLIF($5,'')::date IS NULL OR created_at<NULLIF($5,'')::date+INTERVAL '1 day') ORDER BY `+orderBy+` LIMIT $6 OFFSET $7`, user, status, search, dateFrom, dateTo, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []domain.Project{}
	for rows.Next() {
		var p domain.Project
		if err = rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}
func (r *Repository) GetProject(ctx context.Context, id, user string) (*domain.Project, error) {
	var p domain.Project
	err := r.db.QueryRow(ctx, `SELECT id,user_id,name,description,status,created_at,updated_at FROM projects WHERE id=$1 AND user_id=$2`, id, user).Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}
func (r *Repository) UpdateProject(ctx context.Context, p *domain.Project) error {
	err := r.db.QueryRow(ctx, `UPDATE projects SET name=$3,description=$4,updated_at=now() WHERE id=$1 AND user_id=$2 RETURNING updated_at`, p.ID, p.UserID, p.Name, p.Description).Scan(&p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
func (r *Repository) DeleteProject(ctx context.Context, id, user string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM projects WHERE id=$1 AND user_id=$2`, id, user)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (r *Repository) ArchiveProject(ctx context.Context, id, user string) error {
	tag, err := r.db.Exec(ctx, `UPDATE projects SET status='archived',updated_at=now() WHERE id=$1 AND user_id=$2`, id, user)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (r *Repository) DuplicateProject(ctx context.Context, id, user string) (*domain.Project, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var p domain.Project
	p.ID = uuid.NewString()
	err = tx.QueryRow(ctx, `INSERT INTO projects(id,user_id,name,description,status) SELECT $3,user_id,name || ' (copy)',description,'active' FROM projects WHERE id=$1 AND user_id=$2 RETURNING user_id,name,description,status,created_at,updated_at`, id, user, p.ID).Scan(&p.UserID, &p.Name, &p.Description, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT title,status,content FROM prds WHERE project_id=$1`, id)
	if err != nil {
		return nil, err
	}
	type prdCopy struct {
		title, status string
		content       []byte
	}
	copies := []prdCopy{}
	for rows.Next() {
		var item prdCopy
		if err = rows.Scan(&item.title, &item.status, &item.content); err != nil {
			rows.Close()
			return nil, err
		}
		copies = append(copies, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, item := range copies {
		if _, err = tx.Exec(ctx, `INSERT INTO prds(id,project_id,title,status,content) VALUES($1,$2,$3,$4,$5)`, uuid.NewString(), p.ID, item.title, item.status, item.content); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) Dashboard(ctx context.Context, user string) (map[string]any, error) {
	var projects, prds, drafts int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM projects WHERE user_id=$1 AND status='active'`, user).Scan(&projects); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE d.status='draft') FROM prds d JOIN projects p ON p.id=d.project_id WHERE p.user_id=$1`, user).Scan(&prds, &drafts); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT d.id,d.title,d.status,d.updated_at,p.id,p.name FROM prds d JOIN projects p ON p.id=d.project_id WHERE p.user_id=$1 ORDER BY d.updated_at DESC LIMIT 8`, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	recent := []map[string]any{}
	for rows.Next() {
		var id, title, status, projectID, projectName string
		var updated time.Time
		if err = rows.Scan(&id, &title, &status, &updated, &projectID, &projectName); err != nil {
			return nil, err
		}
		recent = append(recent, map[string]any{"prdId": id, "title": title, "status": status, "updatedAt": updated, "projectId": projectID, "projectName": projectName})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"totalProjects": projects, "totalPRDs": prds, "draftPRDs": drafts, "recentDocuments": recent}, nil
}

func (r *Repository) CreatePRD(ctx context.Context, p *domain.PRD) error {
	p.ID = uuid.NewString()
	if len(p.Content) == 0 {
		p.Content = json.RawMessage(`{}`)
	}
	return r.db.QueryRow(ctx, `INSERT INTO prds(id,project_id,title,status,content) SELECT $1,id,$3,$4,$5 FROM projects WHERE id=$2 RETURNING version,created_at,updated_at`, p.ID, p.ProjectID, p.Title, p.Status, p.Content).Scan(&p.Version, &p.CreatedAt, &p.UpdatedAt)
}
func scanPRD(row pgx.Row) (*domain.PRD, error) {
	var p domain.PRD
	err := row.Scan(&p.ID, &p.ProjectID, &p.Title, &p.Status, &p.Content, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}
func (r *Repository) GetPRD(ctx context.Context, id, user string) (*domain.PRD, error) {
	return scanPRD(r.db.QueryRow(ctx, `SELECT d.id,d.project_id,d.title,d.status,d.content,d.version,d.created_at,d.updated_at FROM prds d JOIN projects p ON p.id=d.project_id WHERE d.id=$1 AND p.user_id=$2`, id, user))
}
func (r *Repository) UpdatePRD(ctx context.Context, p *domain.PRD, expected int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var current int
	err = tx.QueryRow(ctx, `SELECT version FROM prds d JOIN projects p ON p.id=d.project_id WHERE d.id=$1 AND p.user_id=$2 FOR UPDATE`, p.ID, p.UserID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if current != expected {
		return fmt.Errorf("version_conflict: current version is %d", current)
	}
	_, err = tx.Exec(ctx, `INSERT INTO prd_versions(prd_id,version,title,status,content) SELECT id,version,title,status,content FROM prds WHERE id=$1`, p.ID)
	if err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `UPDATE prds SET title=$2,status=$3,content=$4,version=version+1,updated_at=now() WHERE id=$1 RETURNING version,updated_at`, p.ID, p.Title, p.Status, p.Content).Scan(&p.Version, &p.UpdatedAt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) DeletePRD(ctx context.Context, id, user string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM prds d USING projects p WHERE d.project_id=p.id AND d.id=$1 AND p.user_id=$2`, id, user)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
func (r *Repository) ListProjectPRDs(ctx context.Context, project, user string, limit, offset int) ([]domain.PRD, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM prds d JOIN projects p ON p.id=d.project_id WHERE p.id=$1 AND p.user_id=$2`, project, user).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT d.id,d.project_id,d.title,d.status,d.content,d.version,d.created_at,d.updated_at FROM prds d JOIN projects p ON p.id=d.project_id WHERE p.id=$1 AND p.user_id=$2 ORDER BY d.updated_at DESC LIMIT $3 OFFSET $4`, project, user, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []domain.PRD{}
	for rows.Next() {
		var p domain.PRD
		if err = rows.Scan(&p.ID, &p.ProjectID, &p.Title, &p.Status, &p.Content, &p.Version, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}
func (r *Repository) ListPRDs(ctx context.Context, user, search, status, dateFrom, dateTo, sort string, limit, offset int) ([]domain.PRD, int, error) {
	if status != "draft" && status != "in_progress" && status != "completed" && status != "archived" {
		status = ""
	}
	orderBy := "d.updated_at DESC"
	switch sort {
	case "updated_asc":
		orderBy = "d.updated_at ASC"
	case "created_asc":
		orderBy = "d.created_at ASC"
	case "created_desc":
		orderBy = "d.created_at DESC"
	case "title_asc":
		orderBy = "d.title ASC"
	case "title_desc":
		orderBy = "d.title DESC"
	}
	filter := `p.user_id=$1 AND ($2='' OR d.status=$2) AND ($3='' OR d.title ILIKE '%'||$3||'%') AND (NULLIF($4,'')::date IS NULL OR d.created_at>=NULLIF($4,'')::date) AND (NULLIF($5,'')::date IS NULL OR d.created_at<NULLIF($5,'')::date+INTERVAL '1 day')`
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM prds d JOIN projects p ON p.id=d.project_id WHERE `+filter, user, status, search, dateFrom, dateTo).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT d.id,d.project_id,d.title,d.status,d.content,d.version,d.created_at,d.updated_at FROM prds d JOIN projects p ON p.id=d.project_id WHERE `+filter+` ORDER BY `+orderBy+` LIMIT $6 OFFSET $7`, user, status, search, dateFrom, dateTo, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []domain.PRD{}
	for rows.Next() {
		var p domain.PRD
		if err = rows.Scan(&p.ID, &p.ProjectID, &p.Title, &p.Status, &p.Content, &p.Version, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}
func (r *Repository) DuplicatePRD(ctx context.Context, id, user string) (*domain.PRD, error) {
	p := &domain.PRD{ID: uuid.NewString()}
	err := r.db.QueryRow(ctx, `INSERT INTO prds(id,project_id,title,status,content) SELECT $3,d.project_id,d.title || ' (copy)','draft',d.content FROM prds d JOIN projects p ON p.id=d.project_id WHERE d.id=$1 AND p.user_id=$2 RETURNING id,project_id,title,status,content,version,created_at,updated_at`, id, user, p.ID).Scan(&p.ID, &p.ProjectID, &p.Title, &p.Status, &p.Content, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}
func (r *Repository) ListVersions(ctx context.Context, id, user string) ([]domain.PRDVersion, error) {
	rows, err := r.db.Query(ctx, `SELECT v.id,v.prd_id,v.version,v.title,v.status,v.content,v.created_at FROM prd_versions v JOIN prds d ON d.id=v.prd_id JOIN projects p ON p.id=d.project_id WHERE d.id=$1 AND p.user_id=$2 ORDER BY v.version DESC`, id, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.PRDVersion{}
	for rows.Next() {
		var v domain.PRDVersion
		if err = rows.Scan(&v.ID, &v.PRDID, &v.Version, &v.Title, &v.Status, &v.Content, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *Repository) RestoreVersion(ctx context.Context, id, user, versionID string) (*domain.PRD, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var content []byte
	var title, status string
	err = tx.QueryRow(ctx, `SELECT v.content,v.title,v.status FROM prd_versions v JOIN prds d ON d.id=v.prd_id JOIN projects p ON p.id=d.project_id WHERE d.id=$1 AND p.user_id=$2 AND v.id=$3`, id, user, versionID).Scan(&content, &title, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var p domain.PRD
	err = tx.QueryRow(ctx, `SELECT id,project_id,title,status,content,version,created_at,updated_at FROM prds WHERE id=$1 FOR UPDATE`, id).Scan(&p.ID, &p.ProjectID, &p.Title, &p.Status, &p.Content, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO prd_versions(prd_id,version,title,status,content) SELECT id,version,title,status,content FROM prds WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	err = tx.QueryRow(ctx, `UPDATE prds SET title=$2,status=$3,content=$4,version=version+1,updated_at=now() WHERE id=$1 RETURNING id,project_id,title,status,content,version,created_at,updated_at`, id, title, status, content).Scan(&p.ID, &p.ProjectID, &p.Title, &p.Status, &p.Content, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, tx.Commit(ctx)
}

func (r *Repository) IdempotentPRD(ctx context.Context, user, key, requestHash string) (*domain.PRD, error) {
	if key == "" {
		return nil, nil
	}
	var p domain.PRD
	var existingHash string
	err := r.db.QueryRow(ctx, `SELECT d.id,d.project_id,d.title,d.status,d.content,d.version,d.created_at,d.updated_at,g.request_hash FROM ai_generation_requests g JOIN prds d ON d.id=g.prd_id JOIN projects p ON p.id=d.project_id WHERE g.user_id=$1 AND g.idempotency_key=$2 AND p.user_id=$1`, user, key).Scan(&p.ID, &p.ProjectID, &p.Title, &p.Status, &p.Content, &p.Version, &p.CreatedAt, &p.UpdatedAt, &existingHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if existingHash != requestHash {
		return nil, domain.ErrIdempotencyConflict
	}
	return &p, nil
}
func (r *Repository) SaveGeneratedPRD(ctx context.Context, user, project, title string, content json.RawMessage, model, idempotencyKey, requestHash string, inputTokens, outputTokens int) (*domain.PRD, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if idempotencyKey != "" {
		var userID string
		if err = tx.QueryRow(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, user).Scan(&userID); errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
		var existingID, existingHash string
		err = tx.QueryRow(ctx, `SELECT prd_id,request_hash FROM ai_generation_requests WHERE user_id=$1 AND idempotency_key=$2`, user, idempotencyKey).Scan(&existingID, &existingHash)
		if err == nil {
			if existingHash != requestHash {
				return nil, domain.ErrIdempotencyConflict
			}
			p, e := scanPRD(tx.QueryRow(ctx, `SELECT id,project_id,title,status,content,version,created_at,updated_at FROM prds WHERE id=$1`, existingID))
			if e != nil {
				return nil, e
			}
			if e = tx.Commit(ctx); e != nil {
				return nil, e
			}
			return p, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	if err = reserveUsage(ctx, tx, user); err != nil {
		return nil, err
	}
	var projectID string
	err = tx.QueryRow(ctx, `SELECT id FROM projects WHERE id=$1 AND user_id=$2`, project, user).Scan(&projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	var p domain.PRD
	err = tx.QueryRow(ctx, `INSERT INTO prds(id,project_id,title,status,content) VALUES($1,$2,$3,'draft',$4) RETURNING id,project_id,title,status,content,version,created_at,updated_at`, id, projectID, title, content).Scan(&p.ID, &p.ProjectID, &p.Title, &p.Status, &p.Content, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO ai_usage_logs(user_id,model,input_tokens,output_tokens,action) VALUES($1,$2,$3,$4,'generate_prd')`, user, model, inputTokens, outputTokens); err != nil {
		return nil, err
	}
	if idempotencyKey != "" {
		if _, err = tx.Exec(ctx, `INSERT INTO ai_generation_requests(user_id,idempotency_key,request_hash,prd_id) VALUES($1,$2,$3,$4)`, user, idempotencyKey, requestHash, p.ID); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &p, nil
}
func (r *Repository) UsageSummary(ctx context.Context, user string) (map[string]any, error) {
	var requests, generations, ins, outs, limit int
	var plan string
	err := r.db.QueryRow(ctx, `SELECT u.plan,pl.monthly_generation_limit,coalesce(x.requests,0),coalesce(x.generations,0),coalesce(x.input_tokens,0),coalesce(x.output_tokens,0) FROM users u JOIN ai_plans pl ON pl.code=u.plan LEFT JOIN (SELECT user_id,count(*) requests,count(*) FILTER(WHERE action='generate_prd') generations,sum(input_tokens) input_tokens,sum(output_tokens) output_tokens FROM ai_usage_logs WHERE created_at>=date_trunc('month',now()) GROUP BY user_id) x ON x.user_id=u.id WHERE u.id=$1`, user).Scan(&plan, &limit, &requests, &generations, &ins, &outs)
	return map[string]any{"period": "month", "plan": plan, "monthlyAILimit": limit, "remaining": max(limit-requests, 0), "aiRequests": requests, "generations": generations, "inputTokens": ins, "outputTokens": outs}, err
}
func (r *Repository) UsageHistory(ctx context.Context, user string, limit, offset int) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, `SELECT id,model,action,input_tokens,output_tokens,created_at FROM ai_usage_logs WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, user, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, model, action string
		var in, outs int
		var at time.Time
		if err = rows.Scan(&id, &model, &action, &in, &outs, &at); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "model": model, "action": action, "inputTokens": in, "outputTokens": outs, "createdAt": at})
	}
	return out, rows.Err()
}

func reserveUsage(ctx context.Context, tx pgx.Tx, user string) error {
	var limit int
	if err := tx.QueryRow(ctx, `SELECT pl.monthly_generation_limit FROM users u JOIN ai_plans pl ON pl.code=u.plan WHERE u.id=$1 FOR UPDATE OF u`, user).Scan(&limit); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM ai_usage_logs WHERE user_id=$1 AND created_at>=date_trunc('month',now())`, user).Scan(&count); err != nil {
		return err
	}
	if count >= limit {
		return domain.ErrQuotaExceeded
	}
	return nil
}
func (r *Repository) RecordAIUsage(ctx context.Context, user, model, action string, inputTokens, outputTokens int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = reserveUsage(ctx, tx, user); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO ai_usage_logs(user_id,model,action,input_tokens,output_tokens) VALUES($1,$2,$3,$4,$5)`, user, model, action, inputTokens, outputTokens); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) Plans(ctx context.Context) ([]map[string]any, error) {
	rows, err := r.db.Query(ctx, `SELECT code,monthly_generation_limit FROM ai_plans ORDER BY monthly_generation_limit`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var code string
		var limit int
		if err = rows.Scan(&code, &limit); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"code": code, "monthlyAILimit": limit})
	}
	return out, rows.Err()
}

func (r *Repository) SaveClarification(ctx context.Context, session *domain.ClarificationSession, user string) error {
	idea, err := json.Marshal(session.Idea)
	if err != nil {
		return err
	}
	questions, err := json.Marshal(session.Questions)
	if err != nil {
		return err
	}
	answers, err := json.Marshal(session.Answers)
	if err != nil {
		return err
	}
	if session.ID == "" {
		session.ID = uuid.NewString()
		return r.db.QueryRow(ctx, `INSERT INTO ai_clarification_sessions(id,user_id,idea,questions,answers,ready) VALUES($1,$2,$3,$4,$5,$6) RETURNING created_at,updated_at`, session.ID, user, idea, questions, answers, session.Ready).Scan(&session.CreatedAt, &session.UpdatedAt)
	}
	err = r.db.QueryRow(ctx, `UPDATE ai_clarification_sessions SET idea=$3,questions=$4,answers=$5,ready=$6,updated_at=now() WHERE id=$1 AND user_id=$2 RETURNING created_at,updated_at`, session.ID, user, idea, questions, answers, session.Ready).Scan(&session.CreatedAt, &session.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
func (r *Repository) GetClarification(ctx context.Context, id, user string) (*domain.ClarificationSession, error) {
	var s domain.ClarificationSession
	var idea, questions, answers []byte
	err := r.db.QueryRow(ctx, `SELECT id,idea,questions,answers,ready,created_at,updated_at FROM ai_clarification_sessions WHERE id=$1 AND user_id=$2`, id, user).Scan(&s.ID, &idea, &questions, &answers, &s.Ready, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(idea, &s.Idea); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(questions, &s.Questions); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(answers, &s.Answers); err != nil {
		return nil, err
	}
	return &s, nil
}
func (r *Repository) CreateShareLink(ctx context.Context, prd, user, hash string, expires time.Time) (*domain.ShareLink, error) {
	s := &domain.ShareLink{ID: uuid.NewString(), PRDID: prd, ExpiresAt: expires}
	err := r.db.QueryRow(ctx, `INSERT INTO share_links(id,prd_id,token_hash,expires_at) SELECT $1,d.id,$3,$4 FROM prds d JOIN projects p ON p.id=d.project_id WHERE d.id=$2 AND p.user_id=$5 RETURNING created_at`, s.ID, prd, hash, expires, user).Scan(&s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}
func (r *Repository) ListShareLinks(ctx context.Context, prd, user string) ([]domain.ShareLink, error) {
	rows, err := r.db.Query(ctx, `SELECT s.id,s.prd_id,s.expires_at,s.created_at FROM share_links s JOIN prds d ON d.id=s.prd_id JOIN projects p ON p.id=d.project_id WHERE d.id=$1 AND p.user_id=$2 AND s.revoked_at IS NULL AND s.expires_at>now() ORDER BY s.created_at DESC`, prd, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ShareLink{}
	for rows.Next() {
		var s domain.ShareLink
		if err = rows.Scan(&s.ID, &s.PRDID, &s.ExpiresAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func (r *Repository) RevokeShareLink(ctx context.Context, prd, user, id string) error {
	tag, err := r.db.Exec(ctx, `UPDATE share_links s SET revoked_at=now() FROM prds d JOIN projects p ON p.id=d.project_id WHERE s.prd_id=d.id AND d.id=$1 AND p.user_id=$2 AND s.id=$3 AND s.revoked_at IS NULL`, prd, user, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
func (r *Repository) SharedPRD(ctx context.Context, tokenHash string) (*domain.PRD, error) {
	return scanPRD(r.db.QueryRow(ctx, `SELECT d.id,d.project_id,d.title,d.status,d.content,d.version,d.created_at,d.updated_at FROM share_links s JOIN prds d ON d.id=s.prd_id WHERE s.token_hash=$1 AND s.revoked_at IS NULL AND s.expires_at>now()`, tokenHash))
}

func (r *Repository) GetAISettings(ctx context.Context, userID string) (*domain.UserAISettings, error) {
	var s domain.UserAISettings
	err := r.db.QueryRow(ctx, `SELECT user_id, provider, model, base_url, api_key, custom_model, created_at, updated_at FROM user_ai_settings WHERE user_id=$1`, userID).
		Scan(&s.UserID, &s.Provider, &s.Model, &s.BaseURL, &s.APIKey, &s.CustomModel, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &s, err
}

func (r *Repository) UpsertAISettings(ctx context.Context, s *domain.UserAISettings) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO user_ai_settings(user_id, provider, model, base_url, api_key, custom_model, updated_at)
		VALUES($1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (user_id) DO UPDATE SET
			provider=EXCLUDED.provider,
			model=EXCLUDED.model,
			base_url=EXCLUDED.base_url,
			api_key=EXCLUDED.api_key,
			custom_model=EXCLUDED.custom_model,
			updated_at=now()
		RETURNING created_at, updated_at
	`, s.UserID, s.Provider, s.Model, s.BaseURL, s.APIKey, s.CustomModel).Scan(&s.CreatedAt, &s.UpdatedAt)
}
