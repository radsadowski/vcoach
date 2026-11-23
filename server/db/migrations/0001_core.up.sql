CREATE TABLE athletes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL DEFAULT 'Athlete',
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Guarantees at most one default athlete; the athlete resolver
-- GetOrCreateDefaultAthlete relies on this constraint.
CREATE UNIQUE INDEX athletes_one_default ON athletes (is_default) WHERE is_default;

CREATE TABLE app_settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
