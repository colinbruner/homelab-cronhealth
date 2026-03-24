package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// --- Checks ---

func (d *DB) ListChecks(ctx context.Context) ([]Check, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT id, name, slug, period_seconds, grace_seconds, status,
		       last_ping_at, last_alerted_at, created_at, created_by
		FROM checks ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []Check
	for rows.Next() {
		var c Check
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.PeriodSeconds,
			&c.GraceSeconds, &c.Status, &c.LastPingAt, &c.LastAlertedAt,
			&c.CreatedAt, &c.CreatedBy); err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

func (d *DB) GetCheck(ctx context.Context, id uuid.UUID) (*Check, error) {
	var c Check
	err := d.Pool.QueryRow(ctx, `
		SELECT id, name, slug, period_seconds, grace_seconds, status,
		       last_ping_at, last_alerted_at, created_at, created_by
		FROM checks WHERE id = $1`, id).Scan(
		&c.ID, &c.Name, &c.Slug, &c.PeriodSeconds, &c.GraceSeconds,
		&c.Status, &c.LastPingAt, &c.LastAlertedAt, &c.CreatedAt, &c.CreatedBy)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (d *DB) GetCheckBySlug(ctx context.Context, slug string) (*Check, error) {
	var c Check
	err := d.Pool.QueryRow(ctx, `
		SELECT id, name, slug, period_seconds, grace_seconds, status,
		       last_ping_at, last_alerted_at, created_at, created_by
		FROM checks WHERE slug = $1`, slug).Scan(
		&c.ID, &c.Name, &c.Slug, &c.PeriodSeconds, &c.GraceSeconds,
		&c.Status, &c.LastPingAt, &c.LastAlertedAt, &c.CreatedAt, &c.CreatedBy)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

type CreateCheckParams struct {
	Name          string
	Slug          string
	PeriodSeconds int
	GraceSeconds  int
	CreatedBy     *uuid.UUID
}

func (d *DB) CreateCheck(ctx context.Context, p CreateCheckParams) (*Check, error) {
	var c Check
	err := d.Pool.QueryRow(ctx, `
		INSERT INTO checks (name, slug, period_seconds, grace_seconds, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, slug, period_seconds, grace_seconds, status,
		          last_ping_at, last_alerted_at, created_at, created_by`,
		p.Name, p.Slug, p.PeriodSeconds, p.GraceSeconds, p.CreatedBy).Scan(
		&c.ID, &c.Name, &c.Slug, &c.PeriodSeconds, &c.GraceSeconds,
		&c.Status, &c.LastPingAt, &c.LastAlertedAt, &c.CreatedAt, &c.CreatedBy)
	return &c, err
}

type UpdateCheckParams struct {
	ID            uuid.UUID
	Name          string
	PeriodSeconds int
	GraceSeconds  int
}

func (d *DB) UpdateCheck(ctx context.Context, p UpdateCheckParams) (*Check, error) {
	var c Check
	err := d.Pool.QueryRow(ctx, `
		UPDATE checks SET name = $2, period_seconds = $3, grace_seconds = $4
		WHERE id = $1
		RETURNING id, name, slug, period_seconds, grace_seconds, status,
		          last_ping_at, last_alerted_at, created_at, created_by`,
		p.ID, p.Name, p.PeriodSeconds, p.GraceSeconds).Scan(
		&c.ID, &c.Name, &c.Slug, &c.PeriodSeconds, &c.GraceSeconds,
		&c.Status, &c.LastPingAt, &c.LastAlertedAt, &c.CreatedAt, &c.CreatedBy)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (d *DB) DeleteCheck(ctx context.Context, id uuid.UUID) error {
	tag, err := d.Pool.Exec(ctx, `DELETE FROM checks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("check not found")
	}
	return nil
}

// --- Pings ---

func (d *DB) RecordPing(ctx context.Context, checkID uuid.UUID, sourceIP *string, exitCode *int) error {
	now := time.Now().UTC()
	batch := &pgx.Batch{}

	batch.Queue(`
		INSERT INTO pings (check_id, received_at, source_ip, exit_code)
		VALUES ($1, $2, $3, $4)`, checkID, now, sourceIP, exitCode)

	// Update check: set last_ping_at and transition status to 'up' if recovering.
	// This handles: new→up, down→up, alerting→up, silenced stays silenced.
	batch.Queue(`
		UPDATE checks SET
			last_ping_at = $2,
			status = CASE
				WHEN status = 'silenced' THEN 'silenced'
				ELSE 'up'
			END
		WHERE id = $1`, checkID, now)

	br := d.Pool.SendBatch(ctx, batch)
	defer br.Close()

	if _, err := br.Exec(); err != nil {
		return fmt.Errorf("inserting ping: %w", err)
	}
	if _, err := br.Exec(); err != nil {
		return fmt.Errorf("updating check: %w", err)
	}
	return nil
}

// RecordPingWithRecovery records a ping and returns true if the check recovered
// from alerting state (so a recovery notification should be sent).
func (d *DB) RecordPingWithRecovery(ctx context.Context, checkID uuid.UUID, sourceIP *string, exitCode *int) (recovered bool, err error) {
	now := time.Now().UTC()

	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	// Insert ping
	if _, err := tx.Exec(ctx, `
		INSERT INTO pings (check_id, received_at, source_ip, exit_code)
		VALUES ($1, $2, $3, $4)`, checkID, now, sourceIP, exitCode); err != nil {
		return false, fmt.Errorf("inserting ping: %w", err)
	}

	// Get current status before update
	var currentStatus CheckStatus
	if err := tx.QueryRow(ctx, `SELECT status FROM checks WHERE id = $1`, checkID).
		Scan(&currentStatus); err != nil {
		return false, fmt.Errorf("reading status: %w", err)
	}

	// Update check status
	newStatus := StatusUp
	if currentStatus == StatusSilenced {
		newStatus = StatusSilenced
	}

	if _, err := tx.Exec(ctx, `
		UPDATE checks SET last_ping_at = $2, status = $3 WHERE id = $1`,
		checkID, now, newStatus); err != nil {
		return false, fmt.Errorf("updating check: %w", err)
	}

	// If recovering from alerting, resolve the open alert
	if currentStatus == StatusAlerting {
		if _, err := tx.Exec(ctx, `
			UPDATE alerts SET resolved_at = $2
			WHERE check_id = $1 AND resolved_at IS NULL`,
			checkID, now); err != nil {
			return false, fmt.Errorf("resolving alert: %w", err)
		}
		recovered = true
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	// Notify SSE clients after the transaction commits. This must happen outside
	// the transaction: any error inside a PostgreSQL transaction aborts the whole
	// transaction, so a pg_notify failure would have silently rolled back the ping.
	if _, err := d.Pool.Exec(ctx, `
		SELECT pg_notify('check_events',
			json_build_object('check_id', $1::text, 'status', $2, 'event', 'ping_received')::text)`,
		checkID, string(newStatus)); err != nil {
		// Non-fatal: SSE clients will miss this event
	}

	return recovered, nil
}

func (d *DB) ListPings(ctx context.Context, checkID uuid.UUID, limit, offset int) ([]Ping, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT id, check_id, received_at, source_ip, exit_code
		FROM pings WHERE check_id = $1
		ORDER BY received_at DESC
		LIMIT $2 OFFSET $3`, checkID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pings []Ping
	for rows.Next() {
		var p Ping
		if err := rows.Scan(&p.ID, &p.CheckID, &p.ReceivedAt, &p.SourceIP, &p.ExitCode); err != nil {
			return nil, err
		}
		pings = append(pings, p)
	}
	return pings, rows.Err()
}

// --- Alerts ---

func (d *DB) ListAlerts(ctx context.Context, limit int) ([]Alert, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT a.id, a.check_id, a.started_at, a.resolved_at, a.alert_count, c.name
		FROM alerts a JOIN checks c ON c.id = a.check_id
		ORDER BY a.started_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.CheckID, &a.StartedAt, &a.ResolvedAt,
			&a.AlertCount, &a.CheckName); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

func (d *DB) GetAlert(ctx context.Context, id uuid.UUID) (*Alert, error) {
	var a Alert
	err := d.Pool.QueryRow(ctx, `
		SELECT a.id, a.check_id, a.started_at, a.resolved_at, a.alert_count, c.name
		FROM alerts a JOIN checks c ON c.id = a.check_id
		WHERE a.id = $1`, id).Scan(
		&a.ID, &a.CheckID, &a.StartedAt, &a.ResolvedAt, &a.AlertCount, &a.CheckName)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

// --- Silences ---

func (d *DB) CreateSilence(ctx context.Context, checkID uuid.UUID, userID *uuid.UUID, endsAt *time.Time, reason *string) (*Silence, error) {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var s Silence
	err = tx.QueryRow(ctx, `
		INSERT INTO silences (check_id, silenced_by, ends_at, reason)
		VALUES ($1, $2, $3, $4)
		RETURNING id, check_id, silenced_by, starts_at, ends_at, reason, created_at`,
		checkID, userID, endsAt, reason).Scan(
		&s.ID, &s.CheckID, &s.SilencedBy, &s.StartsAt, &s.EndsAt, &s.Reason, &s.CreatedAt)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `UPDATE checks SET status = 'silenced' WHERE id = $1`, checkID); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		SELECT pg_notify('check_events',
			json_build_object('check_id', $1, 'status', 'silenced', 'event', 'status_changed')::text)`,
		checkID); err != nil {
		// Non-fatal
	}

	return &s, tx.Commit(ctx)
}

func (d *DB) DeleteSilence(ctx context.Context, checkID uuid.UUID) error {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `DELETE FROM silences WHERE check_id = $1`, checkID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("no active silence")
	}

	// Transition back to 'down' — the poller will re-evaluate on next tick
	if _, err := tx.Exec(ctx, `
		UPDATE checks SET status = 'down' WHERE id = $1 AND status = 'silenced'`,
		checkID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		SELECT pg_notify('check_events',
			json_build_object('check_id', $1, 'status', 'down', 'event', 'status_changed')::text)`,
		checkID); err != nil {
		// Non-fatal
	}

	return tx.Commit(ctx)
}

// --- Users ---

func (d *DB) UpsertUser(ctx context.Context, email string, name *string) (*User, error) {
	var u User
	err := d.Pool.QueryRow(ctx, `
		INSERT INTO users (email, name) VALUES ($1, $2)
		ON CONFLICT (email) DO UPDATE SET name = COALESCE(EXCLUDED.name, users.name)
		RETURNING id, email, name, created_at`, email, name).Scan(
		&u.ID, &u.Email, &u.Name, &u.CreatedAt)
	return &u, err
}

func (d *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := d.Pool.QueryRow(ctx, `
		SELECT id, email, name, created_at FROM users WHERE email = $1`, email).Scan(
		&u.ID, &u.Email, &u.Name, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

// --- Notification Channels ---

func (d *DB) GetChannelsForCheck(ctx context.Context, checkID uuid.UUID) ([]NotificationChannel, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT nc.id, nc.user_id, nc.label, nc.type, nc.target, nc.enabled, nc.created_at
		FROM check_notification_channels cnc
		JOIN notification_channels nc ON nc.id = cnc.channel_id
		WHERE cnc.check_id = $1 AND nc.enabled = true`, checkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []NotificationChannel
	for rows.Next() {
		var nc NotificationChannel
		if err := rows.Scan(&nc.ID, &nc.UserID, &nc.Label, &nc.Type,
			&nc.Target, &nc.Enabled, &nc.CreatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, nc)
	}
	return channels, rows.Err()
}

func (d *DB) ListUserChannels(ctx context.Context, userID uuid.UUID) ([]NotificationChannel, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT id, user_id, label, type, target, enabled, created_at
		FROM notification_channels WHERE user_id = $1
		ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []NotificationChannel
	for rows.Next() {
		var nc NotificationChannel
		if err := rows.Scan(&nc.ID, &nc.UserID, &nc.Label, &nc.Type,
			&nc.Target, &nc.Enabled, &nc.CreatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, nc)
	}
	return channels, rows.Err()
}

func (d *DB) SetCheckChannels(ctx context.Context, checkID uuid.UUID, channelIDs []uuid.UUID) error {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM check_notification_channels WHERE check_id = $1`, checkID); err != nil {
		return err
	}
	for _, chID := range channelIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO check_notification_channels (check_id, channel_id) VALUES ($1, $2)`,
			checkID, chID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// --- Poller Queries ---

// GetMissedChecks returns checks that have missed their ping window (past grace period).
// Excludes silenced checks with an active (non-expired) silence.
func (d *DB) GetMissedChecks(ctx context.Context) ([]Check, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT c.id, c.name, c.slug, c.period_seconds, c.grace_seconds,
		       c.status, c.last_ping_at, c.last_alerted_at, c.created_at, c.created_by
		FROM checks c
		WHERE c.status IN ('up', 'down', 'alerting')
		  AND c.last_ping_at IS NOT NULL
		  AND c.last_ping_at < NOW() - (c.period_seconds || ' seconds')::interval - (c.grace_seconds || ' seconds')::interval
		  AND c.id NOT IN (
		      SELECT s.check_id FROM silences s
		      WHERE (s.ends_at IS NULL OR s.ends_at > NOW())
		  )
		ORDER BY c.last_ping_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []Check
	for rows.Next() {
		var c Check
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.PeriodSeconds,
			&c.GraceSeconds, &c.Status, &c.LastPingAt, &c.LastAlertedAt,
			&c.CreatedAt, &c.CreatedBy); err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

// TransitionToDown sets a check from 'up' to 'down'.
func (d *DB) TransitionToDown(ctx context.Context, checkID uuid.UUID) error {
	_, err := d.Pool.Exec(ctx, `
		UPDATE checks SET status = 'down'
		WHERE id = $1 AND status = 'up'`, checkID)
	return err
}

// TransitionToAlerting sets a check from 'down' to 'alerting' and creates an alert record.
func (d *DB) TransitionToAlerting(ctx context.Context, checkID uuid.UUID) error {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE checks SET status = 'alerting', last_alerted_at = $2
		WHERE id = $1`, checkID, now); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO alerts (check_id, started_at) VALUES ($1, $2)`,
		checkID, now); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		SELECT pg_notify('check_events',
			json_build_object('check_id', $1, 'status', 'alerting', 'event', 'alert_fired')::text)`,
		checkID); err != nil {
		// Non-fatal
	}

	return tx.Commit(ctx)
}

// ReAlert updates last_alerted_at and increments alert_count for an already-alerting check.
func (d *DB) ReAlert(ctx context.Context, checkID uuid.UUID) error {
	now := time.Now().UTC()

	batch := &pgx.Batch{}
	batch.Queue(`UPDATE checks SET last_alerted_at = $2 WHERE id = $1`, checkID, now)
	batch.Queue(`
		UPDATE alerts SET alert_count = alert_count + 1
		WHERE check_id = $1 AND resolved_at IS NULL`, checkID)

	br := d.Pool.SendBatch(ctx, batch)
	defer br.Close()

	if _, err := br.Exec(); err != nil {
		return err
	}
	if _, err := br.Exec(); err != nil {
		return err
	}
	return nil
}
