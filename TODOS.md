# TODOS

## v2 backlog

### Prometheus metrics endpoint

**What:** Add `GET /metrics` (Prometheus text format) — check status counts, pings/min, alert fire rate.

**Why:** Lets you graph cronhealth health in Grafana alongside other homelab services without any new infrastructure, if you already scrape Prometheus in your cluster.

**Pros:** Zero new infra; data is already computed by the poller every 30s; `prometheus/client_golang` is ~50 lines.

**Cons:** Minor scope; only valuable if Prometheus scraping is already set up in the cluster.

**Context:** The poller already reads all check statuses every 30s. Exporting those counts as Prometheus gauges is trivial. The `/metrics` endpoint would be unauthenticated (standard Prometheus convention) and scraped by an in-cluster Prometheus job.

**Depends on:** v1 complete. Prometheus already running in the cluster (optional dep).

---

### Generic webhook notification channel

**What:** Add `webhook` as a `notification_channels.type`. On alert/recovery, POST a JSON payload to the configured URL.

**Why:** A single webhook implementation unlocks Slack (incoming webhooks), Discord, ntfy.sh, and any custom endpoint — without building per-service integrations.

**Pros:** The `notification_channels` table is already generic (`type` + `target`). Adding `webhook` is a schema enum addition + ~30 lines of Go HTTP POST. Covers ntfy.sh (free push notifications, popular in homelab community) which may be more useful than SMS for many users.

**Cons:** No per-service message formatting — payload is a generic JSON blob. Slack/Discord may need adapter middleware to display nicely.

**Context:** JSON payload shape to define: `{"check": "backup-nightly", "status": "alerting", "last_ping": "2026-03-19T02:14:00Z", "message": "..."}`. Recovery events include `"status": "resolved"` and `"resolved_at"`.

**Depends on:** v1 notification system (email + SMS) must be complete first.
