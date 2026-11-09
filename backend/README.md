# AI PRD Generator API

Go/Gin REST API (`/api/v1`) with PostgreSQL persistence. The backend implements account authentication, project and PRD management, saved clarification interviews, structured AI generation and review actions, version history, usage quotas, Markdown/PDF/DOCX export, print view, and revocable share links.

## Run locally with Docker Compose

From the repository root:

```sh
docker compose up --build
```

Compose starts PostgreSQL, applies the ordered SQL migrations, then starts the API at `http://localhost:8080`. The OpenAPI contract is available at `http://localhost:8080/openapi.yaml`; `GET /healthz` checks process health and `GET /readyz` checks database readiness.

AI requests require a provider key. Set `AI_API_KEY` in the shell or root `.env` before starting Compose. `AI_BASE_URL` and `AI_MODEL` configure an OpenAI-compatible provider. The key is only read by the backend.

## Local Go workflow

Use Go 1.25+. Set the variables listed in `.env.example` in the shell (the application does not load `.env` files automatically), start PostgreSQL, then run:

```sh
migrate -path migrations -database "$DATABASE_URL" up
go run ./cmd/api
```

Apply migrations as a separate deployment step. The Docker Compose migration service does this before the API starts.

## API overview

All private endpoints use the `access_token` HttpOnly cookie. Refresh tokens are random opaque values, stored as SHA-256 hashes, rotated on refresh, and revoked on logout or password change. JSON responses use a `data` envelope; errors use `{ "error": { "code": "...", "message": "..." } }`. Paths in the endpoint table are relative to `/api/v1`.

| Area | Endpoints |
| --- | --- |
| Auth | `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`; `GET /auth/me`; `PATCH /auth/profile`; `PUT /auth/password` |
| Dashboard | `GET /dashboard` (project/PRD totals, drafts, and recent documents) |
| Projects | `POST/GET /projects`; `GET/PATCH/DELETE /projects/:projectId`; `POST /projects/:projectId/archive`, `/duplicate`; `GET /projects/:projectId/prds` |
| PRDs | `GET /prds` (search/filter/page); `POST /projects/:projectId/prds`; `GET/PATCH/DELETE /prds/:prdId`; `POST /prds/:prdId/duplicate`; `GET /prds/:prdId/versions`; `POST /prds/:prdId/restore/:versionId` |
| Export | `GET /prds/:prdId/export/markdown`, `/export/pdf`, `/export/docx`; `GET /prds/:prdId/print` |
| Sharing | `POST/GET /prds/:prdId/share-links`; `DELETE /prds/:prdId/share-links/:shareId`; public read `GET /shared/:token` |
| AI | `POST /ai/clarify`; `POST /ai/generate-prd`; `POST /ai/actions`; shortcuts `POST /ai/regenerate-section`, `/ai/review-prd`, `/ai/generate-tasks` |
| Usage | `GET /usage`; `GET /usage/history` |
| Plans | `GET /plans` (Free/Pro quota configuration; read-only) |

Project and PRD listing support `q`, `status`, `dateFrom`, `dateTo`, `sort`, `page`, and `limit` query parameters. PRD updates require the current `version`; stale writes return HTTP 409. Generation may use a clarification `sessionId`; sessions are private to their owner. Send an optional `Idempotency-Key` header (8–128 characters) to make retries return the original PRD without spending another quota unit; reusing a key with different input returns HTTP 409. The clarification API accepts `skip: true` to proceed with explicit assumptions.

Supported `/ai/actions` action values: `improve_prd`, `find_missing_requirements`, `detect_ambiguity`, `generate_user_stories`, `generate_acceptance_criteria`, `identify_edge_cases`, `review_security`, `generate_tasks`, `create_mvp_scope`, `estimate_complexity`, `draft_api_contract`, `draft_database_schema`, `regenerate_section`, `simplify_language`, and `translate_document`.

## Plans and limits

The migrations seed `ai_plans` with Free (10 AI calls/month) and Pro (1,000/month) limits. Usage enforcement is atomic against a locked user row. Billing is not integrated; assigning a Pro plan is an administrative database operation until billing is implemented. Usage logs record provider token counts when returned.

## Environment

See `.env.example`. `JWT_SECRET` must be at least 32 characters. Set `COOKIE_SECURE=true` behind HTTPS in production. `FRONTEND_ORIGIN` is the single allowed browser origin; `PUBLIC_API_URL` is used to construct share URLs; `TRUSTED_PROXIES` optionally lists trusted proxy CIDRs for client-IP rate limiting. Keep database and AI credentials in a secret manager in deployed environments.

## Verification

```sh
go test ./...
go vet ./...
go build ./...
```

The test suite includes unit coverage for PRD Markdown rendering, export package structure, and generated-document validation. PostgreSQL behavior should also be exercised in an environment with the migrations applied.

Set `TEST_DATABASE_URL` to a dedicated PostgreSQL database to enable the isolated-schema repository integration test; it creates and drops a randomly named schema only inside that database.

## Operational notes

- AI generation is synchronous with a bounded provider timeout and one retry for transient network/429/5xx failures. The HTTP server allows a longer write timeout than the configured AI timeout.
- PDF export uses built-in Helvetica to avoid a runtime font dependency; glyphs outside Latin-1 are replaced with `?`. DOCX is a minimal Office Open XML document.
- Rate limiting is in-process and suitable for a single API instance. Use a shared Redis-backed limiter before horizontal scaling.
- No payment processing or plan self-service is enabled. Back up PostgreSQL, protect secrets, and review migration rollback scripts before production use.
