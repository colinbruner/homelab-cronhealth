-- cronhealth initial schema
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

-- Users (populated from OIDC on first login)
CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL UNIQUE,
    name       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Registered health checks
CREATE TABLE checks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    period_seconds  INT NOT NULL,
    grace_seconds   INT NOT NULL DEFAULT 300,
    status          TEXT NOT NULL DEFAULT 'new',
    last_ping_at    TIMESTAMPTZ,
    last_alerted_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID REFERENCES users(id)
);

-- Raw ping events
CREATE TABLE pings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    check_id    UUID NOT NULL REFERENCES checks(id) ON DELETE CASCADE,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_ip   TEXT,
    exit_code   INT
);

-- Alert history log (append-only)
CREATE TABLE alerts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    check_id    UUID NOT NULL REFERENCES checks(id) ON DELETE CASCADE,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    alert_count INT NOT NULL DEFAULT 1
);

-- Snooze / silence records
CREATE TABLE silences (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    check_id    UUID NOT NULL REFERENCES checks(id) ON DELETE CASCADE,
    silenced_by UUID REFERENCES users(id),
    starts_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at     TIMESTAMPTZ,
    reason      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Notification channel definitions (global to user)
CREATE TABLE notification_channels (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label      TEXT NOT NULL,
    type       TEXT NOT NULL,
    target     TEXT NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-check notification channel mapping
CREATE TABLE check_notification_channels (
    check_id   UUID NOT NULL REFERENCES checks(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES notification_channels(id) ON DELETE CASCADE,
    PRIMARY KEY (check_id, channel_id)
);

-- Indexes
CREATE INDEX idx_checks_slug ON checks(slug);
CREATE INDEX idx_checks_status ON checks(status);
CREATE INDEX idx_checks_last_ping_at ON checks(last_ping_at);
CREATE INDEX idx_pings_check_id ON pings(check_id);
CREATE INDEX idx_pings_received_at ON pings(received_at DESC);
CREATE INDEX idx_alerts_check_id ON alerts(check_id);
CREATE INDEX idx_silences_check_id_ends_at ON silences(check_id, ends_at);
CREATE INDEX idx_check_notification_channels_check ON check_notification_channels(check_id);
