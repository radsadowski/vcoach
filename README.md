# vcoach

Personal AI athletic training coach — chat-first, self-hosted, provider-neutral.

vcoach ingests your Garmin Connect data (Personal API), builds an athlete
profile, and lets an LLM coach (any OpenAI-compatible provider — OpenRouter by
default) create structured goals and versioned training plans through chat.
Completed workouts flow back in as compliance; climbing goals track from a
manual climbing log.

## Quick start

```bash
cp .env.example .env   # fill in LLM_API_KEY and GARMIN_API_KEY when you have them
docker compose up --build
```

- Web UI: http://localhost:3000
- Go API: http://localhost:8080/healthz

The stack runs with no keys configured — sync and the coach will surface clear
errors until you add them.

## Configuration (`.env`)

| Variable            | Purpose                                                            |
| ------------------- | ------------------------------------------------------------------ |
| `DATABASE_URL`      | Postgres connection string (local dev default provided)            |
| `POSTGRES_PASSWORD` | Password for the compose Postgres                                  |
| `PORT`             | Go API port (default 8080)                                         |
| `LLM_BASE_URL`     | Any OpenAI-compatible endpoint. Default: OpenRouter                 |
| `LLM_API_KEY`      | Provider API key                                                   |
| `LLM_MODEL`        | Model identifier, e.g. `anthropic/claude-sonnet-4.5` on OpenRouter |
| `GARMIN_API_KEY`   | Garmin Connect Personal API key (see below)                        |
| `SYNC_INTERVAL_MIN`| Scheduled Garmin sync interval in minutes (default 60)              |

Swapping LLM providers is pure configuration: point `LLM_BASE_URL` at OpenAI,
OpenRouter, Groq, Together, DeepSeek, or a local Ollama/LM Studio instance.

## Garmin Personal API key

vcoach reads **your own** Garmin data via the Personal API (pull-only, no
application process):

1. Log in to [Garmin Connect Web](https://connect.garmin.com) → Settings →
   **Personal API Tokens** (direct link:
   <https://connect.garmin.com/app/settings/personalApiToken>).
2. Create a key — choose the longest expiry (1 year). **The key is shown
   once**: copy it into `.env` as `GARMIN_API_KEY` immediately.
3. Run `docker compose up -d api` (or `make compose-up`) to pick it up.

Expiry runbook: keys last 30 days to 1 year; Garmin emails you ~a week before
expiry. When the key expires, syncs start failing — the app shows the error in
the sidebar sync panel. Generate a new key, update `.env`, restart the api
container. Sync state and all data are unaffected.

The first sync backfills 90 days of activities and daily health; afterwards
the scheduler pulls on the configured interval, and "Sync now" triggers an
immediate pull.

## Development

```bash
docker compose up -d db   # postgres only
make dev                  # Go API (:8080) + Vite dev server (:5173)
```

Tests (require Docker for testcontainers):

```bash
make test    # go test -race + web vitest + web build
```

Layout:

```
server/            Go API (chi, sqlx, PostgreSQL)
  cmd/server/        entrypoint + wiring
  db/migrations/     numbered SQL migrations (append-only)
  internal/
    config/          env configuration
    entities/        domain types (incl. Date, Font grade scales)
    errors/          domain error types -> HTTP mapping
    repository/      data access (sqlx), compliance matching SQL
    server/          HTTP handlers (translation only) + httpx helpers
    service/         business logic: coach, sync, garmin, goals, plans,
                     climbing, profile, chat
    testing/         testcontainers helpers + entity builders
web/               Vite React SPA (TS, Tailwind) + Node Express server
packages/shared/   TS types + API client + SSE client (shared with RN app)
scripts/           dev.sh, e2e.sh
```

### Architecture

- **Chat-first coaching.** The LLM coach is the only writer of goals and
  plans, via validated tool calls; the UI renders, deletes and regenerates.
- **Hybrid LLM context.** Every turn injects a compact athlete snapshot
  (profile, goals, active plan compliance, last 7 days); read tools fetch
  deeper history on demand (bounded to 31-day ranges).
- **Versioned plans.** Adapting a plan creates a new version and archives
  the old one; history is append-only.
- **Compliance.** Synced Garmin activities match planned sessions by sport
  family and date window (±1 day); unmatched past sessions are flagged
  missed.
- **Idempotent sync.** Activity upserts are keyed on the Garmin activity ID;
  daily health rows merge per day and never erase known metrics.

### Testing conventions

Go services and handlers are tested against real PostgreSQL via
testcontainers-go, with functional-options entity builders for test data and
mocks only for external systems (Garmin client, LLM client). Garmin and LLM
behaviors are pinned by httptest fixtures — CI never calls live services.

## Roadmap (phase 2+)

- React Native + Expo client over the same API (shared `@vcoach/shared`
  package is the seam)
- Yazio nutrition connector (seam reserved; no public API exists today)
- Board-app integrations (Kilter/Stökt) for climbing data
- Multi-user accounts (athlete_id is on every table; auth swaps the athlete
  resolver)
- Nightly compliance evaluation with drafted plan adaptations

## Design

The full design document (delivery-framework structure with Mermaid
architecture diagram, decision log and risks) lives in
`.kimchi/plans/vcoach-personal-ai-athletic-training-coach-web-m.md`.
