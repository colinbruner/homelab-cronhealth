# cronhealth — Implementation Log

Last updated: 2026-03-19

---

## Completed

### Go Backend (11 files, all compiling, `go vet` clean)

| File | Description | Status |
|------|-------------|--------|
| `go.mod` / `go.sum` | Module `github.com/colinbruner/cronhealth`. Deps: gin, pgx/v5, aws-sdk-go-v2 (ses, sns, config), go-oidc/v3, oauth2, golang-jwt/jwt/v5, uuid | Done |
| `cmd/api/main.go` | API server entrypoint. Wires config → db → sse hub → sse listener → auth → handlers → gin router. Graceful shutdown on SIGINT/SIGTERM. Routes: `/health`, `/ready`, `/ping/:slug` (unauth), `/auth/*` (unauth), `/api/*` (auth middleware) | Done |
| `cmd/poller/main.go` | Poller entrypoint. Wires config → db → notifier (AWS or noop) → poller. Graceful shutdown | Done |
| `internal/config/config.go` | Loads all env vars into `Config` struct. Validates `DATABASE_URL` required. Parses `ALLOWED_OIDC_EMAILS` comma-separated. Defaults: poll interval 30s, cooldown 60m, port 8080 | Done |
| `internal/db/models.go` | All data models: `Check` (with status constants: new/up/down/alerting/silenced), `Ping`, `Alert`, `Silence`, `User`, `NotificationChannel`, `CheckWithChannels`. UUID + time.Time fields with JSON tags | Done |
| `internal/db/db.go` | Database connection management. `New()` creates pgxpool (min 2, max 10 conns). `NewListenConn()` creates dedicated non-pooled pgx.Conn for LISTEN/NOTIFY | Done |
| `internal/db/queries.go` | All SQL queries (~350 lines). `ListChecks`, `GetCheck`, `GetCheckBySlug`, `CreateCheck`, `UpdateCheck`, `DeleteCheck`, `RecordPingWithRecovery` (transactional: insert ping + update status + resolve alert + pg_notify), `ListPings`, `ListAlerts`, `GetAlert`, `CreateSilence`, `DeleteSilence`, `UpsertUser`, `GetChannelsForCheck`, `SetCheckChannels`, `GetMissedChecks`, `TransitionToAlerting`, `ReAlert` | Done |
| `internal/api/handlers.go` | All Gin route handlers (~300 lines). Ping (unauth, 503 on DB error), CRUD checks, list pings w/ pagination, snooze/silence/remove-silence, list/get alerts, SSE events endpoint, health/ready probes | Done |
| `internal/auth/auth.go` | OIDC auth with Pocket-ID. Login (state cookie → redirect), Callback (validate state → exchange code → verify ID token → check allowlist → upsert user → JWT session cookie 24h), Logout, MeHandler, Middleware (validate JWT, set user_id/user_email in gin context) | Done |
| `internal/notify/notify.go` | `Notifier` interface (SendAlert/SendRecovery). `AWSNotifier` using SES + SNS. `NoopNotifier` for dev. All send failures logged but non-blocking | Done |
| `internal/sse/hub.go` | SSE fan-out hub. `Hub` manages client channels with mutex. `StartListener()` creates dedicated pgx.Conn, LISTEN check_events, parses JSON payload, broadcasts to hub. Reconnect with exponential backoff (1s→30s max) | Done |
| `internal/poller/poller.go` | Alert state machine. Ticker → `GetMissedChecks` → per-check: up/down → alerting (transition + notify), alerting → re-alert if cooldown expired. Fetches per-check notification channels | Done |

### Database Schema (1 file)

| File | Description | Status |
|------|-------------|--------|
| `supabase/migrations/20260319000000_initial_schema.sql` | Tables: users, checks, pings, alerts, silences, notification_channels, check_notification_channels. All indexes. Status transitions documented in comments | Done |

### Design Documentation (2 files)

| File | Description | Status |
|------|-------------|--------|
| `DESIGN.md` | Full system design: architecture diagrams, component breakdown, DB schema, API spec, alert state machine, visual design system, user journeys, interaction states, responsive/a11y specs, SSE bridge, notification flow, OIDC flow, k8s topology, test strategy, cost estimate | Done |
| `TODOS.md` | v2 backlog: Prometheus metrics endpoint, generic webhook notification channel | Done |

---

## Not Started

### SvelteKit Frontend (`ui/`)

Per DESIGN.md, the UI is a SvelteKit SPA served by nginx with API calls proxied to avoid CORS.

**Screens to build:**
1. **Dashboard** — status summary bar, failing checks surfaced first, check grid, "+ New Check" CTA
2. **Check Detail** — name + status badge, last ping + next expected, action bar (snooze/silence/edit/delete), ping history timeline, alert log
3. **New / Edit Check** — form: name, period, grace period, notification channels
4. **Alerts Feed** — active alerts (top, red), recently resolved (muted), inline snooze/silence
5. **Settings** — user profile, notification channel management (email/SMS)

**Infrastructure to build:**
- SvelteKit project scaffolding (static adapter for SPA mode)
- Tailwind CSS with design system tokens (colors, fonts, spacing from DESIGN.md)
- API client module (`lib/api.ts`) — typed fetch wrapper for all `/api/*` endpoints
- Auth store — session state, redirect to `/auth/login` on 401
- Checks store — list + SSE real-time updates
- SSE integration (`EventSource` connecting to `/api/events`)
- Shared components: StatusBadge, CheckCard, SkeletonCard, Toast, Modal, TimeAgo
- Empty states: first-run dashboard, no pings yet, no alerts
- Responsive layouts: mobile bottom nav, collapsed healthy checks, full-width snooze buttons
- Loading states: skeleton cards/rows (no spinners on page load)
- Error handling: toast notifications on failed API calls

### Kubernetes Manifests (`k8s/`)

| Component | Description |
|-----------|-------------|
| `k8s/api/deployment.yaml` | cronhealth-api Deployment (Go + Gin, port 8080, 1 replica) |
| `k8s/api/service.yaml` | ClusterIP service, port 8080 |
| `k8s/poller/deployment.yaml` | cronhealth-poller Deployment (Go ticker, 1 replica) |
| `k8s/ui/deployment.yaml` | cronhealth-ui Deployment (nginx + SvelteKit build, port 80, 1 replica) |
| `k8s/ui/service.yaml` | ClusterIP service, port 80 |
| `k8s/ingress.yaml` | Ingress routing: /api/*, /auth/*, /ping/* → api; /* → ui |
| `k8s/secrets.yaml` | Secret template: DATABASE_URL, OIDC_CLIENT_SECRET, SESSION_SECRET, AWS creds |

### Dockerfiles

| File | Description |
|------|-------------|
| `Dockerfile` | Multi-stage Go build for cronhealth-api + cronhealth-poller binaries (scratch base) |
| `ui/Dockerfile` | Multi-stage: node build → nginx serving static files |
| `ui/nginx.conf` | nginx config: serve static, proxy /api/* and /auth/* to cronhealth-api |

### Other

| Item | Description |
|------|-------------|
| `.gitignore` | Go binaries, node_modules, build artifacts, .env files |
| Go tests | Integration tests against local Supabase (poller state machine, ping endpoint, auth) |
| Svelte tests | Component tests with vitest + @testing-library/svelte |
| CI pipeline | GitHub Actions: supabase start → go test → vitest |
