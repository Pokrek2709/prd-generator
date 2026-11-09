ALTER TABLE users ADD COLUMN plan text NOT NULL DEFAULT 'free' CHECK (plan IN ('free','pro'));
ALTER TABLE projects ADD COLUMN status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','archived'));
ALTER TABLE ai_usage_logs ADD COLUMN action text NOT NULL DEFAULT 'generate_prd';

CREATE TABLE ai_clarification_sessions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    idea jsonb NOT NULL,
    questions jsonb NOT NULL DEFAULT '[]'::jsonb,
    answers jsonb NOT NULL DEFAULT '{}'::jsonb,
    ready boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ai_clarification_owner_idx ON ai_clarification_sessions(user_id, updated_at DESC);

CREATE TABLE share_links (
    id uuid PRIMARY KEY,
    prd_id uuid NOT NULL REFERENCES prds(id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX share_links_prd_idx ON share_links(prd_id, created_at DESC);
