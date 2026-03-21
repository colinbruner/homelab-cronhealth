package notify

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/colinbruner/cronhealth/internal/db"
	"github.com/google/uuid"
)

func makeCheck(name string, status db.CheckStatus, lastPing *time.Time) db.Check {
	return db.Check{
		ID:         uuid.New(),
		Name:       name,
		Slug:       "test-slug",
		Status:     status,
		LastPingAt: lastPing,
	}
}

func TestFormatAlertBody_WithLastPing(t *testing.T) {
	lastPing := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	check := makeCheck("nightly-backup", db.StatusAlerting, &lastPing)

	body := formatAlertBody(check)

	if !strings.Contains(body, "nightly-backup") {
		t.Error("body should contain check name")
	}
	if !strings.Contains(body, "alerting") {
		t.Error("body should contain status")
	}
	if !strings.Contains(body, "2026-03-20T10:00:00Z") {
		t.Error("body should contain last ping time in RFC3339")
	}
	if !strings.Contains(body, "missed its expected ping window") {
		t.Error("body should contain alert explanation")
	}
}

func TestFormatAlertBody_NeverPinged(t *testing.T) {
	check := makeCheck("never-pinged", db.StatusAlerting, nil)
	body := formatAlertBody(check)

	if !strings.Contains(body, "never") {
		t.Error("body should say 'never' when no last ping")
	}
}

func TestFormatRecoveryBody(t *testing.T) {
	lastPing := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	check := makeCheck("db-backup", db.StatusUp, &lastPing)

	body := formatRecoveryBody(check)

	if !strings.Contains(body, "db-backup") {
		t.Error("body should contain check name")
	}
	if !strings.Contains(body, "recovered") {
		t.Error("body should mention recovery")
	}
}

func TestFormatAlertSMS(t *testing.T) {
	lastPing := time.Now().Add(-2 * time.Hour)
	check := makeCheck("hourly-job", db.StatusAlerting, &lastPing)

	msg := formatAlertSMS(check)

	if !strings.Contains(msg, "[cronhealth]") {
		t.Error("SMS should start with [cronhealth] prefix")
	}
	if !strings.Contains(msg, "hourly-job") {
		t.Error("SMS should contain check name")
	}
	if !strings.Contains(msg, "missed ping") {
		t.Error("SMS should say 'missed ping'")
	}
}

func TestFormatRecoverySMS(t *testing.T) {
	check := makeCheck("daily-sync", db.StatusUp, nil)
	msg := formatRecoverySMS(check)

	if msg != "[cronhealth] daily-sync recovered." {
		t.Errorf("unexpected SMS: %q", msg)
	}
}

func TestTimeSince_Seconds(t *testing.T) {
	result := timeSince(time.Now().Add(-30 * time.Second))
	if !strings.HasSuffix(result, "s ago") {
		t.Errorf("expected seconds ago, got %q", result)
	}
}

func TestTimeSince_Minutes(t *testing.T) {
	result := timeSince(time.Now().Add(-5 * time.Minute))
	if !strings.HasSuffix(result, "m ago") {
		t.Errorf("expected minutes ago, got %q", result)
	}
}

func TestTimeSince_Hours(t *testing.T) {
	result := timeSince(time.Now().Add(-3 * time.Hour))
	if !strings.Contains(result, "h") || !strings.Contains(result, "m ago") {
		t.Errorf("expected hours+minutes ago, got %q", result)
	}
}

func TestTimeSince_Days(t *testing.T) {
	result := timeSince(time.Now().Add(-48 * time.Hour))
	if !strings.Contains(result, "d") || !strings.Contains(result, "h ago") {
		t.Errorf("expected days+hours ago, got %q", result)
	}
}

func TestNoopNotifier_SendAlert(t *testing.T) {
	n := &NoopNotifier{}
	check := makeCheck("test", db.StatusAlerting, nil)
	channels := []db.NotificationChannel{
		{ID: uuid.New(), Type: "email", Target: "test@example.com", Enabled: true},
	}

	err := n.SendAlert(context.Background(), check, channels)
	if err != nil {
		t.Errorf("NoopNotifier.SendAlert should not return error, got %v", err)
	}
}

func TestNoopNotifier_SendRecovery(t *testing.T) {
	n := &NoopNotifier{}
	check := makeCheck("test", db.StatusUp, nil)

	err := n.SendRecovery(context.Background(), check, nil)
	if err != nil {
		t.Errorf("NoopNotifier.SendRecovery should not return error, got %v", err)
	}
}
