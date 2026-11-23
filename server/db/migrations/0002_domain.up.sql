-- Athlete profile basics (manual; Garmin-derived metrics are computed on read).
ALTER TABLE athletes
    ADD COLUMN birth_year INT,
    ADD COLUMN sex TEXT CHECK (sex IN ('male', 'female', 'other')),
    ADD COLUMN weight_kg NUMERIC(5, 2),
    ADD COLUMN running_experience TEXT CHECK (running_experience IN ('beginner', 'intermediate', 'advanced')),
    ADD COLUMN climbing_experience TEXT CHECK (climbing_experience IN ('beginner', 'intermediate', 'advanced'));

CREATE TYPE goal_activity_type AS ENUM ('run', 'boulder', 'lead');
CREATE TYPE goal_status AS ENUM ('active', 'achieved', 'abandoned');

CREATE TABLE goals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    athlete_id UUID NOT NULL REFERENCES athletes(id),
    activity_type goal_activity_type NOT NULL,
    target_metric TEXT NOT NULL,
    target_value TEXT NOT NULL,
    target_date DATE,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    status goal_status NOT NULL DEFAULT 'active',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX goals_one_primary_active ON goals (athlete_id)
    WHERE is_primary AND status = 'active';

CREATE TYPE plan_status AS ENUM ('active', 'archived');

CREATE TABLE plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    athlete_id UUID NOT NULL REFERENCES athletes(id),
    goal_id UUID REFERENCES goals(id),
    version INT NOT NULL,
    status plan_status NOT NULL DEFAULT 'active',
    title TEXT NOT NULL DEFAULT 'Training plan',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Exactly one active plan per athlete: plans are generated around the
-- primary goal and a successor archives its predecessor.
CREATE UNIQUE INDEX plans_one_active ON plans (athlete_id) WHERE status = 'active';

CREATE TABLE plan_weeks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    week_number INT NOT NULL,
    start_date DATE NOT NULL,
    focus TEXT NOT NULL DEFAULT '',
    UNIQUE (plan_id, week_number)
);

CREATE TYPE session_status AS ENUM ('planned', 'completed', 'missed', 'skipped');

CREATE TABLE plan_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    week_id UUID NOT NULL REFERENCES plan_weeks(id) ON DELETE CASCADE,
    athlete_id UUID NOT NULL REFERENCES athletes(id),
    sport TEXT NOT NULL,
    session_type TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    planned_date DATE NOT NULL,
    target_duration_min INT,
    target_distance_km NUMERIC(6, 2),
    target_intensity TEXT,
    status session_status NOT NULL DEFAULT 'planned',
    matched_activity_id UUID,
    completed_at TIMESTAMPTZ
);

CREATE TABLE climbing_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    athlete_id UUID NOT NULL REFERENCES athletes(id),
    session_date DATE NOT NULL,
    venue TEXT NOT NULL DEFAULT '',
    discipline TEXT NOT NULL CHECK (discipline IN ('boulder', 'lead')),
    hardest_sent TEXT NOT NULL,
    hardest_attempted TEXT NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    rpe INT CHECK (rpe BETWEEN 1 AND 10),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    athlete_id UUID NOT NULL REFERENCES athletes(id),
    external_id TEXT NOT NULL,
    sport TEXT NOT NULL,
    sport_family TEXT NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    duration_sec INT NOT NULL,
    distance_m NUMERIC(10, 2),
    avg_hr INT,
    max_hr INT,
    vo2max NUMERIC(5, 1),
    raw JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Idempotent sync: one row per Garmin activity per athlete.
CREATE UNIQUE INDEX activities_external ON activities (athlete_id, external_id);

CREATE TABLE daily_health (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    athlete_id UUID NOT NULL REFERENCES athletes(id),
    day DATE NOT NULL,
    sleep_score INT,
    sleep_hours NUMERIC(4, 2),
    hrv_ms INT,
    resting_hr INT,
    body_battery INT,
    stress_avg INT,
    steps INT,
    weight_kg NUMERIC(5, 2),
    raw JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Idempotent sync: one row per day per athlete.
CREATE UNIQUE INDEX daily_health_day ON daily_health (athlete_id, day);

CREATE TABLE sync_state (
    source TEXT NOT NULL,
    athlete_id UUID NOT NULL REFERENCES athletes(id),
    last_synced_at TIMESTAMPTZ,
    cursor TIMESTAMPTZ,
    last_error TEXT,
    last_error_at TIMESTAMPTZ,
    PRIMARY KEY (source, athlete_id)
);

CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    athlete_id UUID NOT NULL REFERENCES athletes(id),
    title TEXT NOT NULL DEFAULT 'New chat',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    athlete_id UUID NOT NULL REFERENCES athletes(id),
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    content TEXT NOT NULL,
    tool_calls JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX chat_messages_session ON chat_messages (chat_session_id, created_at);
