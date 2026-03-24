# cronhealth

A self-hosted cron job health monitoring application. External jobs ping a unique URL after each run; if a ping is missed beyond its configured grace period, cronhealth fires an alert via email or SMS.

## How it works

Each monitored job is represented as a **check** with two time windows:

- **Period** — how often the job is expected to run (e.g. every 3600 seconds)
- **Grace** — how long after the period expires to wait before alerting (e.g. 300 seconds)

Your cron job sends a `POST /ping/:slug` request after each successful execution. The poller runs every 30 seconds and transitions checks through a state machine:

```
new ──(first ping)──► up ──(missed + grace expired)──► alerting
                       ▲                                    │
                       └──────────(recovery ping)───────────┘
                                         │
                             silenced ◄──┴──► (silence expires)
```

| State | Meaning |
|---|---|
| `new` | Created, no pings received yet |
| `up` | Last ping arrived within the expected window |
| `alerting` | Grace period expired; notification has been sent |
| `silenced` | Alerts suppressed until silence expires |

Re-alerts fire on a configurable cooldown (default: 60 minutes) to avoid notification spam.

## Architecture

Three services share a PostgreSQL database:

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  cronhealth │     │  cronhealth  │     │  cronhealth  │
│     ui      │────►│     api      │◄────│    poller    │
│  (nginx)    │     │  (Go/Gin)    │     │  (Go worker) │
└─────────────┘     └──────┬───────┘     └──────┬───────┘
                           │                    │
                    ┌──────▼────────────────────▼──────┐
                    │           PostgreSQL              │
                    └───────────────────────────────────┘
```

- **api** — REST API + SSE for real-time UI updates. Handles authentication and the unauthenticated `/ping/:slug` endpoint.
- **poller** — Background worker that queries for missed checks and sends notifications via AWS SES (email) and SNS (SMS).
- **ui** — SvelteKit SPA served by nginx, which reverse-proxies API traffic to the api service.

## Requirements

### Local development

- [Docker](https://docs.docker.com/get-docker/) with the Compose plugin

### Production

- Kubernetes cluster with [ArgoCD](https://argo-cd.readthedocs.io/) and [Helm](https://helm.sh/)
- PostgreSQL database (tested with Supabase)
- An OIDC provider (tested with [Pocket-ID](https://github.com/pocket-id/pocket-id))
- AWS account with SES and optionally SNS enabled (for email/SMS alerts)

## Local development

### 1. Clone and configure

```bash
git clone https://github.com/colinbruner/homelab-cronhealth.git
cd homelab-cronhealth
cp .env.example .env
```

The default `.env` has `DEV_AUTH_BYPASS=true`, which disables OIDC and treats all requests as authenticated. No OIDC provider is required to run locally.

Edit `.env` if you want to change any defaults (poll interval, AWS credentials for notification testing, etc.).

### 2. Start the stack

```bash
docker compose up --build
```

This starts:
- **PostgreSQL** on `localhost:5432` — schema applied automatically on first start
- **cronhealth-api** on `localhost:8080` (internal only)
- **cronhealth-poller** — background worker, no exposed port
- **cronhealth-ui** on `localhost:8090` — main entry point

Open [http://localhost:8090](http://localhost:8090).

### 3. Sending a test ping

Create a check in the UI, then copy its slug and send a ping:

```bash
curl -X POST http://localhost:8090/ping/<slug>
```

### 4. Resetting the database

```bash
docker compose down -v   # removes the postgres_data volume
docker compose up --build
```

### Testing OIDC locally

Set `DEV_AUTH_BYPASS=false` in `.env` and provide real OIDC credentials:

```env
DEV_AUTH_BYPASS=false
OIDC_ISSUER=https://your-provider.example.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=http://localhost:8090/auth/callback
ALLOWED_OIDC_EMAILS=you@example.com
SESSION_SECRET=<output of: openssl rand -hex 32>
```

## Configuration reference

All configuration is via environment variables. See [`.env.example`](.env.example) for annotated defaults.

| Variable | Service | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | api, poller | — | PostgreSQL connection string |
| `DEV_AUTH_BYPASS` | api | `false` | Disable OIDC; all requests authenticated as `dev@localhost` |
| `SESSION_SECRET` | api | — | Secret for signing session JWTs |
| `OIDC_ISSUER` | api | — | OIDC provider base URL |
| `OIDC_CLIENT_ID` | api | — | OAuth2 client ID |
| `OIDC_CLIENT_SECRET` | api | — | OAuth2 client secret |
| `OIDC_REDIRECT_URL` | api | — | Callback URL registered with the OIDC provider |
| `ALLOWED_OIDC_EMAILS` | api | — | Comma-separated email allowlist |
| `PORT` | api | `8080` | HTTP listen port |
| `POLL_INTERVAL_SECONDS` | poller | `30` | How often the poller checks for missed jobs |
| `ALERT_COOLDOWN_MINUTES` | poller | `60` | Minimum time between repeat alerts for the same check |
| `AWS_REGION` | poller | `us-east-1` | AWS region for SES and SNS |
| `AWS_SES_FROM` | poller | — | Verified SES sender address |
| `AWS_SNS_SMS_ENABLED` | poller | `false` | Enable SMS alerts via SNS |
| `AWS_ACCESS_KEY_ID` | poller | — | AWS credentials |
| `AWS_SECRET_ACCESS_KEY` | poller | — | AWS credentials |

## Production deployment

The Helm chart is in `charts/cronhealth/`. It is intended to be deployed via ArgoCD.

```bash
helm upgrade --install cronhealth ./charts/cronhealth \
  --namespace cronhealth \
  --create-namespace \
  --values charts/cronhealth/values.yaml
```

Sensitive values (`DATABASE_URL`, OIDC credentials, AWS credentials, `SESSION_SECRET`) should be provided via the `existingSecret` parameter rather than plain Helm values.

## Running tests

```bash
# Go tests
go test ./...

# UI tests
cd ui && npm test
```
