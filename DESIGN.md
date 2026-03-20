# cronhealth — System Design

A self-hosted healthchecks.io alternative for internal cronjob monitoring.

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        HOMELAB KUBERNETES CLUSTER                         │
│                                                                            │
│  ┌─────────────────────────┐   ┌─────────────────────────────────────┐   │
│  │     cronhealth-api      │   │          cronhealth-poller           │   │
│  │       (Go + Gin)        │   │      (Go ticker, 30s interval)       │   │
│  │                         │   │                                      │   │
│  │  POST /ping/:slug       │   │  • Every 30s: query Supabase for     │   │
│  │  GET  /api/checks       │   │    checks where last_ping <          │   │
│  │  POST /api/checks       │   │    NOW() - (period + grace)          │   │
│  │  PUT  /api/checks/:id   │   │  • Dedup: skip if already alerting   │   │
│  │  DEL  /api/checks/:id   │   │    and last_alerted_at < cooldown    │   │
│  │  POST /api/checks/:id/  │   │  • Fire AWS SES (email) on miss      │   │
│  │        snooze           │   │  • Fire AWS SNS (SMS) on miss        │   │
│  │  POST /api/checks/:id/  │   │  • Fire recovery notification when   │   │
│  │        silence          │   │    check recovers (ping received)    │   │
│  │  GET  /api/alerts       │   │                                      │   │
│  │  GET  /auth/login       │   │  Alert state machine:                │   │
│  │  GET  /auth/callback    │   │    ok → failing → alerting           │   │
│  │  GET  /health           │   │    alerting → resolved               │   │
│  └────────────┬────────────┘   └──────────────┬───────────────────────┘   │
│               │                               │                            │
│  ┌────────────▼────────────┐                  │                            │
│  │    cronhealth-ui         │                  │                            │
│  │  (SvelteKit SPA + nginx) │                  │                            │
│  │                          │                  │                            │
│  │  Dashboard (check list)  │                  │                            │
│  │  Check detail + history  │                  │                            │
│  │  Snooze / silence UI     │                  │                            │
│  │  Alert feed              │                  │                            │
│  │  Settings / profile      │                  │                            │
│  └──────────────────────────┘                  │                            │
│                                                │                            │
│  ┌────────────────────────────────────────────────────────────────────┐   │
│  │                     k8s Ingress (Traefik / nginx-ingress)           │   │
│  │   cronhealth.internal → cronhealth-api (port 8080)                  │   │
│  │   cronhealth.internal → cronhealth-ui  (port 80, nginx)             │   │
│  └────────────────────────────────────────────────────────────────────┘   │
│                                                                            │
│  Ping sources (internal network):                                          │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐              │
│  │ k8s CronJobs   │  │ Bare-metal VMs │  │ External hosts │              │
│  │ (same cluster) │  │ (same LAN)     │  │ (VPN/LAN)      │              │
│  └───────┬────────┘  └───────┬────────┘  └───────┬────────┘              │
│          └──────────────────►│◄──────────────────┘                       │
│                    POST /ping/:slug                                        │
└────────────────────────────────────────────────────────────────────────────┘
                │ (Supabase connection)          │ (AWS API calls)
                ▼                                ▼
┌───────────────────────────┐    ┌───────────────────────────────────────┐
│         Supabase           │    │              AWS Services              │
│   (managed Postgres,       │    │                                       │
│    hosted on AWS)          │    │  ┌────────────────────────────────┐   │
│                            │    │  │  SES (Simple Email Service)    │   │
│  Tables:                   │    │  │  • Alert fired email           │   │
│  ├── checks                │    │  │  • Recovery email              │   │
│  ├── pings                 │    │  │  • 3,000 msgs/mo free (in-AWS) │   │
│  ├── alerts                │    │  └────────────────────────────────┘   │
│  ├── users                 │    │  ┌────────────────────────────────┐   │
│  ├── notification_channels │    │  │  SNS (Simple Notif. Service)   │   │
│  └── silences              │    │  │  • SMS on alert                │   │
│                            │    │  │  • 100 msgs/mo free (US)       │   │
│  Free tier:                │    │  └────────────────────────────────┘   │
│  500MB storage             │    └───────────────────────────────────────┘
│  5GB bandwidth             │
│  Unlimited connections     │
└───────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│                            GCP (GCE VM)                                   │
│                                                                            │
│   Pocket-ID (OIDC provider)                                               │
│   Issuer: https://id.yourdomain.com                                       │
│                  │                                                         │
│                  └── Cloudflare Tunnel ──► homelab k8s cluster            │
│                       (Browser auth flows routed through tunnel)           │
└──────────────────────────────────────────────────────────────────────────┘

Public access via Cloudflare Tunnel (second tunnel route):
┌──────────────────────────────────────────────────────────────────────────┐
│   cronhealth.yourdomain.com  ──►  homelab k8s (cronhealth-api + ui)      │
│                                                                            │
│   Used for:  browser access off-LAN (phone on LTE, travel)               │
│              OIDC redirect callback URL                                   │
│              external cronjob pings (from outside homelab)               │
│   Auth gate: OIDC login required for all /api/* and UI routes            │
│   Ping:      /ping/:slug unauthenticated — slug is the shared secret     │
│   LAN path:  cronhealth.internal still works via Traefik (homelab only)  │
│   k8s jobs:  use svc DNS: cronhealth-api-svc.cronhealth.svc.cluster.local│
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Component Breakdown

### 1. cronhealth-api (Go + Gin)

**Responsibilities:**
- Receive ping events from cronjobs via `POST /ping/:slug`
- Serve REST API for the SvelteKit frontend
- Handle OIDC login/callback flow (Pocket-ID)
- Authenticate all `/api/*` routes via session cookie (JWT)

**Key design decisions:**
- Ping endpoint requires NO authentication — slug is the shared secret (keep slugs long, random, e.g. UUID)
- Ping endpoint: if Supabase is unreachable, return 503 (do not return 200 — cronjob should know the ping failed)
- LISTEN connection (SSE bridge): runs in a dedicated goroutine with reconnect loop (exponential backoff, max 30s)
- All `/api/*` endpoints require valid session
- OIDC state stored in a signed cookie (no server-side session storage)
- Connects to Supabase via `pgx` (native Postgres driver, not ORM)
- Uses two connection modes: `pgxpool` for query traffic (pool size: 5–10), plus one dedicated non-pooled `pgx.Conn` for `LISTEN check_events` (SSE bridge). These must be separate — pgxpool connections cannot hold persistent LISTEN state.
- Poller notification query uses a single JOIN (no N+1): `SELECT c.*, nc.* FROM checks c JOIN check_notification_channels cnc ON cnc.check_id = c.id JOIN notification_channels nc ON nc.id = cnc.channel_id WHERE c.status IN ('down','alerting') AND ...`

**Configuration (env vars):**
```
DATABASE_URL=postgres://...supabase.co/postgres
OIDC_ISSUER=https://id.yourdomain.com
OIDC_CLIENT_ID=cronhealth
OIDC_CLIENT_SECRET=...
OIDC_REDIRECT_URL=https://cronhealth.yourdomain.com/auth/callback
SESSION_SECRET=...
AWS_REGION=us-east-1
AWS_SES_FROM=alerts@yourdomain.com
AWS_SNS_ENABLED=true
ALLOWED_OIDC_EMAILS=you@example.com,other@example.com
```

### 2. cronhealth-poller (Go ticker)

**Responsibilities:**
- Tick every 30 seconds (configurable via `POLL_INTERVAL_SECONDS`)
- Query Supabase for all active, non-silenced checks
- For each check, evaluate: `last_ping < NOW() - (period_seconds + grace_seconds)`
- Manage alert state transitions
- Fire notifications via SES (email) and SNS (SMS)
- On status change, publish SSE event so connected dashboards update live

**Deployment:** Separate k8s Deployment from the API. Independent restart, separate resource limits, cleaner separation of concerns.

**Failure handling (critical):**
- Supabase unreachable during tick → log error, skip tick, retry on next interval (do NOT crash)
- SES send failure → log error with check name, skip notification (do not block state transition)
- SNS send failure → same as SES — log and continue
- pg_notify failure → log warning; SSE clients will miss event but will resync on next poll or reconnect
- LISTEN connection drops → reconnect loop with exponential backoff (1s, 2s, 4s, max 30s)

**Deduplication logic (reads/writes checks table only):**
```
for each check where status IN ('down', 'alerting')
  AND last_ping_at < NOW() - (period_seconds + grace_seconds)
  AND (silenced = false OR silence expired):

  if status == 'down':
    # Grace period has expired — first alert
    UPDATE checks SET status='alerting', last_alerted_at=NOW()
    INSERT INTO alerts (check_id, started_at) VALUES (...)
    fire_notification(check)

  elif status == 'alerting':
    if last_alerted_at < NOW() - ALERT_COOLDOWN_MINUTES:
      # Re-alert after cooldown
      UPDATE checks SET last_alerted_at=NOW()
      UPDATE alerts SET alert_count=alert_count+1 WHERE check_id=... AND resolved_at IS NULL
      fire_notification(check)
    # else: skip — still within cooldown window
```

**Configuration (env vars):**
```
DATABASE_URL=postgres://...
POLL_INTERVAL_SECONDS=30
ALERT_COOLDOWN_MINUTES=60
AWS_REGION=us-east-1
AWS_SES_FROM=alerts@yourdomain.com
AWS_SNS_SMS_ENABLED=true
```

### 3. cronhealth-ui (SvelteKit SPA)

**Responsibilities:**
- Serve the dashboard and all user-facing UI
- Authenticate via OIDC (redirects to cronhealth-api `/auth/login`)
- Communicate with cronhealth-api REST endpoints

**Build artifact:**
- Static files built by `vite build`
- Served by an nginx container in the same pod or a separate Deployment
- API calls proxied through nginx to cronhealth-api to avoid CORS

**Screens:**
1. **Dashboard** — check status grid, color-coded (green/red/yellow/silenced)
2. **Check detail** — ping history timeline, alert log, snooze/silence controls
3. **New / edit check** — form: name, slug, period, grace period, notification channels
4. **Alerts feed** — paginated list of active and recent alerts
5. **Settings** — user profile, default notification preferences (email/SMS toggle)

**Navigation Flow:**
```
Login (Pocket-ID OIDC)
        │
        ▼
  [Dashboard]  ◄──── primary home, always-accessible via nav
   │      │
   │      └──► [Check Detail]
   │               ├── Snooze / Silence (modal)
   │               └── Edit Check (inline or modal)
   │
   ├──► [Alerts Feed]   (nav link, secondary)
   ├──► [New Check]     (primary CTA button in nav)
   └──► [Settings]      (nav, avatar/profile link)
```

**Screen Information Hierarchy:**

Dashboard (primary screen):
```
1st: System health summary bar   — "X checks OK, Y down, Z silenced"
2nd: Failing / alerting checks   — surfaced at top, red, with time since last ping
3rd: All other checks grid       — green/silenced, lower visual weight
4th: Primary action              — "+ New Check" button (top right)
```

Check Detail:
```
1st: Check name + current status badge (large, top of page)
2nd: Last ping time + next expected ping countdown
3rd: Action bar — Snooze / Silence / Edit / Delete
4th: Ping history timeline (scrollable)
5th: Alert log (collapsible)
```

Alerts Feed:
```
1st: Active alerts (firing)       — top, red
2nd: Recently resolved            — below, muted
3rd: Snooze / silence controls    — inline per alert row
```

---

## Visual Design System

**Aesthetic:** Dark, terminal-adjacent. A tool made by someone who cares about their infra. No generic SaaS chrome.

```
Color palette:
  Background:     #0f1117  (near-black)
  Surface:        #1a1d24  (card / panel backgrounds)
  Border:         #2a2d36  (subtle, low-contrast)
  Text primary:   #e2e8f0  (off-white)
  Text secondary: #94a3b8  (muted, for timestamps / metadata)

Status colors:
  UP:             #22c55e  (muted green)
  DOWN:           #ef4444  (alert red)
  SILENCED:       #6b7280  (gray — de-emphasized intentionally)
  NEW (waiting):  #f59e0b  (amber — "waiting for first ping")
  ALERTING:       #f97316  (orange — worse than down, notifications firing)

Typography:
  UI font:        Inter (system fallback stack)
  Code font:      JetBrains Mono — used for: slugs, ping URLs,
                  timestamps, ping counts, exit codes
  Body size:      14px (tight — ops tool, not a landing page)

Spacing scale:    4px base unit (Tailwind default)
Border radius:    4px for cards, 2px for badges (sharp, not rounded)
Shadows:          None — flat dark UI, depth via background contrast only

Differentiators (not AI slop):
  • Slugs and ping URLs displayed in monospace, always visible on detail page
  • Timestamps shown as human delta ("3h 12m ago") with ISO tooltip on hover
  • Status badges are text + color, not just colored dots
  • Failing checks in dashboard show a subtle left border accent (red), not full bg color
  • "+ New Check" is the ONLY primary action color in the nav — everything else is neutral
```

---

## User Journey & Emotional Arc

### Journey 1: 2am alert — "Something is broken"
```
STEP | USER DOES                        | USER FEELS          | DESIGN SERVES IT BY
-----|----------------------------------|---------------------|--------------------------------------------
1    | Gets SMS/email alert             | Alarmed, half-asleep| Alert message includes check name + time
2    | Opens cronhealth on phone        | Disoriented         | Mobile layout: check name BIG, status CLEAR
3    | Reads dashboard                  | Scanning for info   | Failing checks pinned top, no noise
4    | Taps check → sees last ping time | Grounding           | "Last ping: 3h 12m ago" (human-readable delta)
5    | Hits Snooze 1h                   | Relief (partial)    | One tap on preset button, no form to fill
6    | Goes back to sleep               | Calm enough         | Toast confirms snooze, no more SMS for 1h
```

### Journey 2: Daily check-in — "Is everything OK?"
```
STEP | USER DOES                        | USER FEELS          | DESIGN SERVES IT BY
-----|----------------------------------|---------------------|--------------------------------------------
1    | Opens dashboard                  | Mildly curious      | Status bar: "17 OK · 0 DOWN" → immediate answer
2    | Scans grid                       | Satisfied (if green)| Green checks fade to background, nothing to do
3    | Closes tab                       | Confident           | Took <5 seconds, no friction
```

### Journey 3: Onboarding a new cronjob — "Add a check"
```
STEP | USER DOES                        | USER FEELS          | DESIGN SERVES IT BY
-----|----------------------------------|---------------------|--------------------------------------------
1    | Clicks "+ New Check"             | Purposeful          | Prominent CTA in nav bar
2    | Fills form (name, period, grace) | Focused             | Minimal form — only required fields
3    | Sees generated slug + ping URL   | Satisfied           | URL shown immediately after save, with copy button
4    | Pastes URL into cron job         | Done                | Example command pre-filled with their slug
5    | Waits for first ping             | Mildly anxious      | Check shows "Waiting for first ping..." (not "DOWN")
6    | Ping arrives, status turns green | Relieved            | Status update is live (SvelteKit store polling or SSE)
```

**5-second / 5-minute / 5-year design:**
- **5 seconds (visceral):** Status bar immediately tells you "everything OK" or "X things broken." No hunting.
- **5 minutes (behavioral):** Snooze/silence are one tap, check detail shows actionable info, ping URL is always copy-able.
- **5 years (reflective):** The tool gets out of the way. It's the thing that pages you when your backup fails and shuts up when things are fine.

---

## Interaction State Coverage

| Feature | Loading | Empty | Error | Success |
|---|---|---|---|---|
| Dashboard | Skeleton cards (gray placeholder shapes, no spinner) | See "First-run empty state" below | Toast: "Failed to load checks. Retry?" + retry button | — (always showing) |
| Check detail | Skeleton for ping timeline | "No pings yet — waiting for first ping" + copy of ping URL | Toast: "Failed to load check" | — |
| Ping received | — | — | HTTP 404 + JSON `{"error": "check not found"}` | HTTP 200 + JSON `{"ok": true}` |
| Snooze action | Button shows spinner, disabled | — | Toast: "Snooze failed" | Toast: "Snoozed for Xh" + badge updates inline. Snooze UX: modal with preset buttons 30m / 1h / 4h / 24h — no text input |
| Silence action | Button shows spinner, disabled | — | Toast: "Silence failed" | Check status badge → "SILENCED" immediately |
| New check form | Submit button spinner | — | Inline field errors (name taken, invalid period) | Redirect to check detail with "Check created" toast |
| Alerts feed | Skeleton rows | "No active alerts — everything looks healthy" (green icon) | Toast: "Failed to load alerts. Retry?" | — |
| OIDC login | Redirect to Pocket-ID (no loading state in app) | — | Full-page error: "Login failed. Try again." + link back | Redirect to dashboard with session set |
| Delete check | Confirm modal → spinner | — | Toast: "Delete failed" | Redirect to dashboard, check removed |

**First-run empty state (dashboard, zero checks):**
```
┌─────────────────────────────────────────────────────┐
│                                                     │
│         No checks configured yet.                   │
│                                                     │
│   Paste this into your first cron job to get        │
│   started:                                          │
│                                                     │
│   curl -fsS -X POST \                              │
│     http://cronhealth.internal/ping/YOUR-SLUG       │
│                                                     │
│              [ + Create your first check ]          │
│                                                     │
└─────────────────────────────────────────────────────┘
```
Warmth: practical, not motivational. Shows a real command so the user can immediately act.

**Skeleton loading pattern:** All list/grid views use gray placeholder shapes at the same dimensions as real content. No spinners on page load — skeletons only. Spinners reserved for button actions (snooze, silence, delete).

---

## Database Schema (Supabase / Postgres)

```sql
-- Registered health checks
-- checks.status is the single source of truth for current check state.
-- The alerts table is an append-only history log, not queried for current state.
--
-- Status transitions:
--   new → up (first ping received)
--   up → down (missed window, within grace period)
--   down → alerting (grace expired, notification fired)
--   alerting → up (recovery ping received)
--   any → silenced (manual snooze/silence applied)
--   silenced → down (silence expires, check still missed)
CREATE TABLE checks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,      -- used in /ping/:slug URL
    period_seconds  INT NOT NULL,              -- expected ping interval
    grace_seconds   INT NOT NULL DEFAULT 300,  -- tolerance before transitioning down→alerting
    status          TEXT NOT NULL DEFAULT 'new', -- new | up | down | alerting | silenced
    last_ping_at    TIMESTAMPTZ,
    last_alerted_at TIMESTAMPTZ,               -- dedup: moved from alerts table
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID REFERENCES users(id)
);

-- Raw ping events
CREATE TABLE pings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    check_id   UUID NOT NULL REFERENCES checks(id) ON DELETE CASCADE,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_ip  TEXT,
    exit_code  INT                             -- optional: cron can POST exit code
);

-- Alert history log (append-only — NOT the source of truth for current state)
-- Current state lives in checks.status and checks.last_alerted_at.
-- This table is for the alert history feed in the UI.
CREATE TABLE alerts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    check_id     UUID NOT NULL REFERENCES checks(id) ON DELETE CASCADE,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- when check entered 'alerting'
    resolved_at  TIMESTAMPTZ,                          -- when recovery ping received
    alert_count  INT NOT NULL DEFAULT 1                -- notifications sent during this incident
);

-- Snooze / silence records
CREATE TABLE silences (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    check_id    UUID NOT NULL REFERENCES checks(id) ON DELETE CASCADE,
    silenced_by UUID REFERENCES users(id),
    starts_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at     TIMESTAMPTZ,                     -- NULL = indefinite silence; set for snooze
    reason      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Users (populated from OIDC on first login)
CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    name       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Notification channel definitions (global to user — e.g. "my email", "my phone")
CREATE TABLE notification_channels (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label      TEXT NOT NULL,                   -- e.g. "my email", "work phone"
    type       TEXT NOT NULL,                   -- email | sms
    target     TEXT NOT NULL,                   -- email address or E.164 phone number
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-check: which notification channels fire when this check fails
-- Zero rows = silent check (no notifications)
CREATE TABLE check_notification_channels (
    check_id   UUID NOT NULL REFERENCES checks(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES notification_channels(id) ON DELETE CASCADE,
    PRIMARY KEY (check_id, channel_id)
);

-- Indexes
CREATE INDEX idx_check_notification_channels_check ON check_notification_channels(check_id);
CREATE INDEX idx_checks_slug ON checks(slug);
CREATE INDEX idx_checks_last_ping_at ON checks(last_ping_at); -- poller's core query filter
CREATE INDEX idx_checks_status ON checks(status);             -- poller filters by status

-- Retention policy (run as a periodic job or Supabase scheduled function):
-- Pings older than 90 days are pruned to control table growth.
-- Alerts history retained indefinitely (low volume, high value).
-- DELETE FROM pings WHERE received_at < NOW() - INTERVAL '90 days';
CREATE INDEX idx_pings_check_id ON pings(check_id);
CREATE INDEX idx_pings_received_at ON pings(received_at DESC);
CREATE INDEX idx_alerts_check_id ON alerts(check_id);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_silences_check_id_ends_at ON silences(check_id, ends_at);
```

---

## Alert State Machine

State lives entirely in `checks.status`. The `alerts` table is append-only history.

```
                    ┌─────────────────────────────────────────────────────┐
                    │                          ping received (recovery)    │
  (created)         │                                                      │
     │              ▼  ping received (first ping)                         │
     │           ┌─────┐ ─────────────────────────────────────► ┌──────────────┐
     └──────────►│ new │                                         │      up      │
                 └─────┘                                         └──────────────┘
                                                                        │
                                                      ping missed (within grace period)
                                                                        │
                                                                        ▼
                                                                 ┌──────────────┐
                                                    ┌────────────│     down     │
                                                    │            └──────────────┘
                                             snooze/│                   │
                                             silence│      grace period expires
                                                    │                   │
                                                    │                   ▼
                                                    │            ┌──────────────┐
                                                    │            │   alerting   │◄──┐
                                                    │            └──────────────┘   │ re-alert
                                                    │                   │           │ after cooldown
                                                    │            snooze/│           │ (ALERT_COOLDOWN_MINUTES)
                                                    │            silence│           │
                                                    │                   ▼           │
                                                    └──────────► ┌──────────────┐  │
                                                                 │   silenced   │  │
                                                                 └──────────────┘  │
                                                                        │          │
                                                          silence expires          │
                                                          (ends_at reached)        │
                                                                        │          │
                                                                        └──────────┘
                                                             (back to alerting if still missed)
```

**Single source of truth:** `checks.status` is the only field that determines current state.
**Deduplication:** `checks.last_alerted_at` tracks when last notification was sent.
**Re-alert rule:** Once in `alerting`, re-notify only after `ALERT_COOLDOWN_MINUTES` (default: 60 min).
**Incident log:** Each transition into `alerting` appends a row to the `alerts` table with `started_at`. Recovery sets `resolved_at`.

---

## API Endpoint Specification

### Ping (unauthenticated)
```
POST /ping/:slug
POST /ping/:slug?exit_code=0

Response: 200 OK
Body: {"ok": true, "check": "my-backup-job"}
```

### Auth
```
GET  /auth/login           → redirect to Pocket-ID OIDC authorization URL
GET  /auth/callback        → handle OIDC code, set session cookie, redirect to UI
POST /auth/logout          → clear session cookie
GET  /auth/me              → return current user info
```

### Checks (authenticated)
```
GET    /api/checks                  → list all checks with current status
POST   /api/checks                  → create check
GET    /api/checks/:id              → get single check + recent pings
PUT    /api/checks/:id              → update check config
DELETE /api/checks/:id              → delete check
GET    /api/checks/:id/pings        → paginated ping history
POST   /api/checks/:id/snooze       → body: {"duration_minutes": 60}
POST   /api/checks/:id/silence      → body: {"reason": "...", "ends_at": "<ISO8601>"}
DELETE /api/checks/:id/silence      → remove active silence
```

### Alerts (authenticated)
```
GET /api/alerts             → list active + recent resolved alerts
GET /api/alerts/:id         → single alert detail
```

### Real-time (SSE)
```
GET /api/events             → SSE stream (authenticated)
                              Events:
                                check.status_changed  {"check_id", "status", "name"}
                                ping.received         {"check_id", "name"}
                                alert.fired           {"check_id", "name"}
                              Reconnect: EventSource auto-reconnects
                              One connection per browser tab

SSE event bridge (poller → API):
  Uses Postgres LISTEN/NOTIFY — zero additional infrastructure.

  Poller (on state change):
    SELECT pg_notify('check_events',
      json_build_object('check_id', id, 'status', status)::text)

  API (/api/events handler):
    conn.Exec(ctx, "LISTEN check_events")
    loop: conn.WaitForNotification(ctx)
      → parse payload
      → fan out to all active SSE client channels

  Data flow:
    poller detects miss
      → UPDATE checks SET status=...
      → pg_notify('check_events', {...})
        → API's LISTEN goroutine wakes
          → SSE broadcast to browser clients

  Note: API holds one dedicated pgx connection for LISTEN
  (separate from the connection pool used for queries).
```

### Admin
```
GET /health                 → liveness probe (unauthenticated)
GET /ready                  → readiness probe (checks DB connectivity)
```

---

## Kubernetes Deployment Topology

```yaml
# Deployments
cronhealth-api      # Go + Gin, replicas: 1, port 8080
cronhealth-poller   # Go ticker, replicas: 1 (no horizontal scaling needed)
cronhealth-ui       # nginx serving SvelteKit build, replicas: 1, port 80

# Migration strategy: Supabase CLI, applied before rollout
# k8s Job or CI step: supabase db push
# File layout: supabase/migrations/<timestamp>_<name>.sql
# Never auto-migrates on pod startup — migration is a deployment gate

# Services
cronhealth-api-svc  # ClusterIP, port 8080
cronhealth-ui-svc   # ClusterIP, port 80

# Ingress
cronhealth.internal →
  /api/*   → cronhealth-api-svc:8080
  /auth/*  → cronhealth-api-svc:8080
  /ping/*  → cronhealth-api-svc:8080
  /health  → cronhealth-api-svc:8080
  /*       → cronhealth-ui-svc:80

# Secrets (k8s Secret, not ConfigMap)
cronhealth-secrets:
  DATABASE_URL
  OIDC_CLIENT_SECRET
  SESSION_SECRET
  AWS_ACCESS_KEY_ID
  AWS_SECRET_ACCESS_KEY
```

---

## Notification Flow

```
Poller detects missed check (checks.status = 'alerting')
         │
         ▼
Query check_notification_channels WHERE check_id = ?
JOIN notification_channels ON channel_id = id
         │
         ├──► type = "email"
         │       └── AWS SES SendEmail API
         │            From: alerts@yourdomain.com
         │            To: user.email
         │            Subject: [ALERT] "my-backup-job" missed its ping
         │
         └──► type = "sms"
                 └── AWS SNS Publish API (SMS)
                      To: +1xxxxxxxxxx
                      Message: "[cronhealth] my-backup-job missed ping. Last seen: 2h ago"

Recovery notification follows same path with subject/message indicating recovery.
```

---

## OIDC Authentication Flow

```
User visits cronhealth.internal
         │
         ▼ (no session cookie)
cronhealth-ui detects unauthenticated → redirects to /auth/login
         │
         ▼
cronhealth-api generates OIDC state, redirects to Pocket-ID:
  https://id.yourdomain.com/authorize?client_id=cronhealth&...
         │
         ▼ (Cloudflare tunnel routes auth to Pocket-ID on GCP)
User authenticates with Pocket-ID
         │
         ▼
Pocket-ID redirects back to:
  https://cronhealth.internal/auth/callback?code=...&state=...
         │
         ▼
cronhealth-api exchanges code for tokens, extracts email
Checks email in ALLOWED_OIDC_EMAILS list
Creates/updates user record in Supabase
Sets signed session cookie (JWT, 24h expiry)
Redirects to UI dashboard
```

---

## Cronjob Integration

Cronjobs send a GET or POST to the ping URL after successful completion:

```bash
# Kubernetes CronJob example
- name: my-backup
  command:
    - /bin/sh
    - -c
    - |
      run-backup.sh && \
      curl -fsS -X POST http://cronhealth-api-svc.cronhealth.svc.cluster.local:8080/ping/YOUR-SLUG-HERE

# Bare-metal cron example
0 2 * * * /usr/local/bin/backup.sh && curl -fsS -X POST http://cronhealth.internal/ping/YOUR-SLUG-HERE

# With exit code reporting
0 2 * * * /usr/local/bin/backup.sh; curl -fsS -X POST "http://cronhealth.internal/ping/YOUR-SLUG?exit_code=$?"
```

---

## Deployment Containerization

### cronhealth-api + cronhealth-poller

```dockerfile
# Multi-stage build
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o cronhealth-api ./cmd/api
RUN CGO_ENABLED=0 go build -o cronhealth-poller ./cmd/poller

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/cronhealth-api /cronhealth-api
COPY --from=builder /app/cronhealth-poller /cronhealth-poller
```

### cronhealth-ui

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY . .
RUN npm ci && npm run build

FROM nginx:alpine
COPY --from=builder /app/build /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
```

---

## Repository Structure

```
cronhealth/
├── cmd/
│   ├── api/         # main.go — starts Gin server
│   └── poller/      # main.go — starts ticker loop
├── internal/
│   ├── api/         # Gin route handlers
│   ├── auth/        # OIDC flow, session management
│   ├── db/          # pgx queries (no ORM)
│   ├── poller/      # ticker logic, alert state machine
│   └── notify/      # SES + SNS notification senders
├── supabase/
│   └── migrations/  # Supabase CLI migration files (supabase migration new)
├── ui/              # SvelteKit application
│   ├── src/
│   │   ├── routes/  # SvelteKit pages
│   │   ├── lib/     # shared components + API client
│   │   └── stores/  # Svelte stores (auth state, checks)
│   └── package.json
├── k8s/             # Kubernetes manifests
│   ├── api/
│   ├── poller/
│   └── ui/
├── DESIGN.md        # this file
└── README.md
```

---

## Service Cost Estimate (monthly, low volume)

| Service | Free Tier | Estimated Usage | Cost |
|---|---|---|---|
| Supabase | 500MB DB, 5GB bandwidth | <10MB DB, <1GB BW | $0 |
| AWS SES | 3,000 msgs/mo (in-AWS) | <100 alert emails | $0 |
| AWS SNS SMS | 100 msgs/mo (US) | <50 SMS | $0 |
| Self-hosted k8s | $0 (your hardware) | 3 small pods | $0 |
| Pocket-ID (GCP GCE) | e2-micro always free | Already running | $0 |
| Cloudflare Tunnel | Free | Already running | $0 |
| **Total** | | | **$0** |

---

## Responsive Layout

**Mobile (< 640px) — Primary use case: checking alerts at 2am**
```
Dashboard:
  • Status bar at top: "X DOWN · Y OK" (single line, large text)
  • Failing checks: full-width cards, stacked vertically
    - Check name (large, bold)
    - Status badge + time since last ping
    - Snooze button (full-width, large touch target)
  • Healthy checks: collapsed into "Y checks OK" accordion
    (tap to expand grid if needed — hidden by default on mobile)
  • Nav: bottom tab bar (Dashboard | Alerts | + New | Settings)

Check detail (mobile):
  • Name + status badge: full width, top
  • Last ping time: prominent, large
  • Snooze / Silence: stacked full-width buttons
  • Ping history: condensed list (time + status only, no IP column)

Forms (mobile):
  • Single-column, full-width inputs
  • Keyboard type hints: numeric keyboard for period/grace fields
```

**Tablet (640px–1024px):** Same as desktop with slightly reduced sidebar. No special layout.

**Desktop (> 1024px):**
```
  Nav: Left sidebar (collapsible) or top nav bar
  Dashboard: 2-column — failing checks left, healthy grid right
             OR alerts-first single column (per design decision above)
  Check detail: Two-panel — info left, ping history right
```

## Accessibility

```
Keyboard navigation:
  • All interactive elements reachable via Tab
  • Snooze modal: focus trapped inside, Escape to close
  • Dashboard check cards: Enter/Space activates (goes to detail)
  • Status badges: not color-only — always include text label (UP/DOWN/etc.)

ARIA:
  • Dashboard: role="main", nav landmarks
  • Status badges: aria-label="Status: DOWN" (not just colored text)
  • Snooze modal: role="dialog", aria-modal="true", aria-labelledby
  • Live status updates: aria-live="polite" on status bar

Color contrast:
  • All text on dark backgrounds: >= 4.5:1 ratio (WCAG AA)
  • Status colors: never relied upon alone — always paired with text
  • DOWN red (#ef4444) on dark surface (#1a1d24): ~5.2:1 ✓

Touch targets (mobile):
  • Minimum 44x44px for all interactive elements
  • Snooze preset buttons: 48px height minimum
  • Check cards: full-row tappable area
```

---

## Test Strategy

**Principle:** No mocked Postgres. The poller's correctness is entirely DB-state-driven — mocks can't catch state transition bugs.

```
Go integration tests (internal/poller, internal/api):
  Setup:      supabase start (local Postgres via Supabase CLI)
  Mocked:     AWS SES, AWS SNS (external APIs only — use interface injection)
  Coverage:   80%+ on poller state machine logic
              100% of checks.status transitions tested

  Critical test cases:
    up → down (ping missed, within grace)
    down → alerting (grace expired, notification fires)
    alerting dedup (last_alerted_at within cooldown → no re-send)
    alerting re-alert (cooldown expired → re-send, alert_count++)
    alerting → up (recovery ping, resolved_at set)
    silenced → skip (active silence)
    silence expired → re-evaluate as if not silenced
    pg_notify → SSE fan-out (use in-process channel for test)

  Ping endpoint tests:
    valid slug → 200, DB updated
    unknown slug → 404
    ping on alerting check → status='up', resolved_at set

  Auth tests:
    email in allowlist → session set
    email not in allowlist → 403
    invalid OIDC state → 400

Svelte component tests (vitest + @testing-library/svelte):
  Dashboard: skeleton loading, empty state, alerts-first ordering
  Snooze modal: preset buttons render, correct duration POSTed
  Check detail: status badge reflects current state
  Error states: failed fetch → toast shown

CI:
  GitHub Actions: supabase start → go test ./... → vitest run
```

---

## Not In Scope (v1)

- Multi-tenant / multi-user access control beyond email allowlist
- Webhook notifications (Slack, PagerDuty, etc.) — can be added later
- Check groups or tags
- Public status page
- API rate limiting on ping endpoint (internal network, trusted)
- Metrics / Prometheus export
- Check dependencies (alert only if parent check is healthy)
