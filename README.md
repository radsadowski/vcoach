# vcoach

Personal AI athletic training coach — chat-first, self-hosted, provider-neutral.

vcoach ingests your Garmin Connect data (Personal API), builds an athlete
profile, and lets an LLM coach (any OpenAI-compatible provider — OpenRouter by
default) create structured goals and versioned training plans through chat.
Completed workouts flow back in as compliance; climbing goals track from a
manual climbing log.

**Status: work in progress (MVP).**

## Quick start

```bash
cp .env.example .env   # fill in LLM_API_KEY and GARMIN_API_KEY when you have them
docker compose up --build
```

- Web UI: http://localhost:3000
- Go API: http://localhost:8080/healthz

## Development (hot loop)

```bash
docker compose up -d db   # postgres only
make dev                  # Go API (:8080) + Vite dev server (:5173)
```

## Layout

```
server/            Go API (chi, sqlx, PostgreSQL)
  cmd/server/        entrypoint + wiring
  db/migrations/     numbered SQL migrations
  internal/
    config/          env configuration
    entities/        domain types
    errors/          domain error types
    repository/      data access (sqlx)
    server/          HTTP handlers (translation only)
    service/         business logic (coach, sync, garmin, goals, plans, ...)
    testing/         testcontainers helpers + entity builders
web/               Vite React SPA (TS, Tailwind) + Node Express static/proxy server
packages/shared/   TypeScript types + API client shared with the future RN app
scripts/           dev/e2e scripts
```

See `.kimchi/plans/` for the full design.
