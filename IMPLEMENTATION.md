# cronhealth — Implementation Log

Last updated: 2026-03-21

---

## Completed

### Go Backend (11 files, all compiling, `go vet` clean)

| File | Description | Status |
|------|-------------|--------|
| `go.mod` / `go.sum` | Module `github.com/colinbruner/cronhealth`. Deps: gin, pgx/v5, aws-sdk-go-v2 (ses, sns, config), go-oidc/v3, oauth2, golang-jwt/jwt/v5, uuid | Done |
| `cmd/api/main.go` | API server entrypoint. Wires config → db → sse hub → sse listener → auth → handlers → gin router. Graceful shutdown on SIGINT/SIGTERM. Routes: `/health`, `/ready`, `/ping/:slug` (unauth), `/auth/*` (unauth), `/api/*` (auth middleware) | Done |
| `cmd/poller/main.go` | Poller entrypoint. Wires config → db → notifier (AWS or noop) → poller. Graceful shutdown | Done |
| `internal/config/config.go` | Loads all env vars into `Config` struct. Validates `DATABASE_URL` required. Defaults: poll interval 30s, cooldown 60m, port 8080 | Done |
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

### SvelteKit Frontend (`ui/`)

| File | Description | Status |
|------|-------------|--------|
| `ui/package.json` / `ui/svelte.config.js` / `ui/vite.config.ts` | SvelteKit 2 + Svelte 5, static adapter (SPA mode), Tailwind CSS 4, vitest | Done |
| `ui/tailwind.config.js` | Design system tokens: dark theme colors, status colors, Inter + JetBrains Mono fonts, sharp border radii | Done |
| `ui/src/app.html` / `ui/src/app.css` | Shell HTML with Google Fonts preconnect, Tailwind base styles | Done |
| `ui/src/lib/types.ts` | TypeScript types matching Go JSON models: Check, Ping, Alert, Silence, User, NotificationChannel, request/response types | Done |
| `ui/src/lib/api.ts` | Typed fetch wrapper for all `/api/*` endpoints. 401 → redirect to `/auth/login`. Methods: listChecks, getCheck, createCheck, updateCheck, deleteCheck, listPings, snoozeCheck, silenceCheck, removeSilence, listAlerts, getAlert, me | Done |
| `ui/src/lib/sse.ts` | EventSource client connecting to `/api/events`. Listens for `ping_received`, `status_changed`, `alert_fired`. Callback dispatch pattern | Done |
| `ui/src/lib/stores/auth.ts` | Auth store: loads user via `/api/me`, tracks loading state, 401 handled by api.ts redirect | Done |
| `ui/src/lib/stores/checks.ts` | Checks store: list with SSE live updates (updateCheckStatus), add/remove/update methods | Done |
| `ui/src/lib/stores/toast.ts` | Toast store: success/error/info with auto-dismiss (4s), unique IDs | Done |
| `ui/src/lib/components/*.svelte` | 10 shared components: StatusBadge, TimeAgo, Toast, ToastContainer, Modal, SkeletonCard, SkeletonRow, EmptyState, CheckCard, Nav | Done |
| `ui/src/routes/+layout.svelte` | Root layout: auth guard, SSE connection, Nav, ToastContainer, mobile bottom padding | Done |
| `ui/src/routes/+page.svelte` | Dashboard: status summary bar, failing-first layout, mobile accordion for healthy checks, skeleton loading, first-run empty state | Done |
| `ui/src/routes/checks/[id]/+page.svelte` | Check detail: two-panel desktop layout, ping URL with copy, snooze modal (30m/1h/4h/24h presets), silence/delete actions, ping history, alert log | Done |
| `ui/src/routes/checks/new/+page.svelte` | New check form: name, period (min), grace (min) | Done |
| `ui/src/routes/checks/[id]/edit/+page.svelte` | Edit check form: pre-fills from existing check | Done |
| `ui/src/routes/alerts/+page.svelte` | Alerts feed: active alerts (top, red) with inline snooze, resolved alerts (muted) | Done |
| `ui/src/routes/settings/+page.svelte` | Settings: user profile display, logout | Done |
| `ui/static/favicon.svg` | Green checkmark favicon on dark background | Done |

**Known gaps (require backend work first):**
- Notification channel CRUD (settings page) — needs `GET/POST /api/channels`, `PUT/DELETE /api/channels/:id`
- Channel selection in new/edit check forms — `channel_ids` field exists in API but no UI to list available channels

---

### Dockerfiles (2 files)

| File | Description | Status |
|------|-------------|--------|
| `Dockerfile` | Multi-stage Go build. Builder stage compiles `cronhealth-api` + `cronhealth-poller` binaries. Two named targets (`api`, `poller`) from `scratch` with CA certs only | Done |
| `ui/Dockerfile` | Multi-stage: `node:22-alpine` build → `nginx:1.27-alpine` serving static SvelteKit build | Done |
| `ui/nginx.conf` | nginx config: serves static SPA with `try_files` fallback, proxies `/api/*`, `/auth/*`, `/ping/*`, `/health`, `/ready` to `cronhealth-api:8080`. SSE support with buffering disabled and 24h read timeout | Done |

### Helm Chart (`charts/cronhealth/`)

Packaged as a Helm chart for ArgoCD deployment. All Kubernetes resources are templated with standard labels, configurable via `values.yaml`.

| File | Description | Status |
|------|-------------|--------|
| `charts/cronhealth/Chart.yaml` | Helm chart metadata: name `cronhealth`, appVersion `1.0.0` | Done |
| `charts/cronhealth/values.yaml` | All configurable values: image repos/tags, replica counts, resource limits, ingress (Traefik, `cronhealth.internal`), config (env vars), secrets (supports `existingSecret` for external secret management) | Done |
| `charts/cronhealth/templates/_helpers.tpl` | Shared template helpers: common labels, secret name resolution | Done |
| `charts/cronhealth/templates/configmap.yaml` | ConfigMap for non-secret env vars: `PORT`, `POLL_INTERVAL_SECONDS`, `ALERT_COOLDOWN_MINUTES`, `AWS_REGION` | Done |
| `charts/cronhealth/templates/secret.yaml` | Secret (conditionally created if `existingSecret` not set): `DATABASE_URL`, OIDC creds, `SESSION_SECRET`, AWS creds. Designed for override via sealed-secrets or ArgoCD secret plugin | Done |
| `charts/cronhealth/templates/api-deployment.yaml` | API Deployment: 1 replica, port 8080, liveness (`/health`) and readiness (`/ready`) probes, envFrom configmap + secret | Done |
| `charts/cronhealth/templates/api-service.yaml` | API ClusterIP Service, port 8080 | Done |
| `charts/cronhealth/templates/poller-deployment.yaml` | Poller Deployment: 1 replica, Recreate strategy (prevents duplicate alert processing), envFrom configmap + secret, no ports (background worker) | Done |
| `charts/cronhealth/templates/ui-deployment.yaml` | UI Deployment: 1 replica, port 80, liveness/readiness probes on `/` | Done |
| `charts/cronhealth/templates/ui-service.yaml` | UI ClusterIP Service, port 80 | Done |
| `charts/cronhealth/templates/ingress.yaml` | Ingress (conditional): routes `/api`, `/auth`, `/ping`, `/health`, `/ready` → api service; `/` → ui service. Supports `ingressClassName`, TLS, annotations | Done |

### Other

| File | Description | Status |
|------|-------------|--------|
| `.gitignore` | Go binaries, node_modules, build artifacts, .env files, IDE files | Done |

---

### CI Pipeline (1 file)

| File | Description | Status |
|------|-------------|--------|
| `.github/workflows/ci.yaml` | GitHub Actions CI: 4 jobs — **Go** (vet, build api+poller, `go test ./...`), **UI** (`npm ci`, `svelte-check`, `npm run build`), **Docker** (build all 3 images via multi-stage targets, runs after Go+UI pass), **Helm** (lint `charts/cronhealth`). Triggers on push/PR to `master` | Done |

---

### Go Tests (3 files, 19 tests)

| File | Description | Status |
|------|-------------|--------|
| `internal/config/config_test.go` | 6 tests: env var loading, defaults (port, region, poll interval, cooldown), custom values, OIDC fields, DATABASE_URL required validation, invalid int env vars | Done |
| `internal/notify/notify_test.go` | 9 tests: formatAlertBody (with/without last ping), formatRecoveryBody, formatAlertSMS, formatRecoverySMS, timeSince (seconds/minutes/hours/days), NoopNotifier SendAlert/SendRecovery | Done |
| `internal/sse/hub_test.go` | 7 tests: NewHub, register/unregister, broadcast to single/multiple clients, slow client skipping (non-blocking), concurrent broadcast safety, no-clients-no-panic | Done |

### Svelte Tests (3 files, 31 tests)

| File | Description | Status |
|------|-------------|--------|
| `ui/src/lib/stores/toast.test.ts` | 8 tests: empty initial state, success/error/info toast types, auto-dismiss after 4s (fake timers), manual dismiss, multiple concurrent toasts, unique IDs | Done |
| `ui/src/lib/stores/checks.test.ts` | 8 tests: initial state, load from API (mocked), error toast on load failure, updateCheck, updateCheckStatus (existing + non-existent), removeCheck, addCheck sorted by name | Done |
| `ui/src/lib/api.test.ts` | 15 tests: all API methods (listChecks, getCheck, createCheck, updateCheck, deleteCheck, listPings with default/custom params, snoozeCheck, silenceCheck, removeSilence, listAlerts, me), 401 redirect to login, error message extraction, statusText fallback | Done |
