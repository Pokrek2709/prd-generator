CREATE TABLE ai_plans (
    code text PRIMARY KEY CHECK (code IN ('free','pro')),
    monthly_generation_limit integer NOT NULL CHECK (monthly_generation_limit > 0),
    updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO ai_plans(code,monthly_generation_limit) VALUES ('free',10),('pro',1000);
