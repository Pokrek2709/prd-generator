CREATE TABLE user_ai_settings (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    provider text NOT NULL DEFAULT 'openai',
    model text NOT NULL DEFAULT 'gpt-4o-mini',
    base_url text NOT NULL DEFAULT '',
    api_key text NOT NULL DEFAULT '',
    custom_model text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
