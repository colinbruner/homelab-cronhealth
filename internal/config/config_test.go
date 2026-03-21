package config

import (
	"os"
	"testing"
)

func clearEnv() {
	for _, key := range []string{
		"DATABASE_URL", "OIDC_ISSUER", "OIDC_CLIENT_ID", "OIDC_CLIENT_SECRET",
		"OIDC_REDIRECT_URL", "SESSION_SECRET", "ALLOWED_OIDC_EMAILS",
		"AWS_REGION", "AWS_SES_FROM", "AWS_SNS_SMS_ENABLED",
		"POLL_INTERVAL_SECONDS", "ALERT_COOLDOWN_MINUTES", "PORT",
	} {
		os.Unsetenv(key)
	}
}

func TestLoad_RequiresDatabaseURL(t *testing.T) {
	clearEnv()
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is not set")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv()
	os.Setenv("DATABASE_URL", "postgres://localhost/test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.AWSRegion != "us-east-1" {
		t.Errorf("AWSRegion = %q, want %q", cfg.AWSRegion, "us-east-1")
	}
	if cfg.PollIntervalSeconds != 30 {
		t.Errorf("PollIntervalSeconds = %d, want 30", cfg.PollIntervalSeconds)
	}
	if cfg.AlertCooldownMinutes != 60 {
		t.Errorf("AlertCooldownMinutes = %d, want 60", cfg.AlertCooldownMinutes)
	}
	if cfg.SNSEnabled {
		t.Error("SNSEnabled should be false by default")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	clearEnv()
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("PORT", "9090")
	os.Setenv("AWS_REGION", "eu-west-1")
	os.Setenv("POLL_INTERVAL_SECONDS", "10")
	os.Setenv("ALERT_COOLDOWN_MINUTES", "30")
	os.Setenv("AWS_SNS_SMS_ENABLED", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.AWSRegion != "eu-west-1" {
		t.Errorf("AWSRegion = %q, want %q", cfg.AWSRegion, "eu-west-1")
	}
	if cfg.PollIntervalSeconds != 10 {
		t.Errorf("PollIntervalSeconds = %d, want 10", cfg.PollIntervalSeconds)
	}
	if cfg.AlertCooldownMinutes != 30 {
		t.Errorf("AlertCooldownMinutes = %d, want 30", cfg.AlertCooldownMinutes)
	}
	if !cfg.SNSEnabled {
		t.Error("SNSEnabled should be true")
	}
}

func TestLoad_AllowedEmails(t *testing.T) {
	clearEnv()
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("ALLOWED_OIDC_EMAILS", "alice@example.com, bob@example.com , charlie@example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"alice@example.com", "bob@example.com", "charlie@example.com"}
	if len(cfg.AllowedEmails) != len(want) {
		t.Fatalf("AllowedEmails len = %d, want %d", len(cfg.AllowedEmails), len(want))
	}
	for i, email := range want {
		if cfg.AllowedEmails[i] != email {
			t.Errorf("AllowedEmails[%d] = %q, want %q", i, cfg.AllowedEmails[i], email)
		}
	}
}

func TestLoad_EmptyAllowedEmails(t *testing.T) {
	clearEnv()
	os.Setenv("DATABASE_URL", "postgres://localhost/test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.AllowedEmails) != 0 {
		t.Errorf("AllowedEmails should be empty, got %v", cfg.AllowedEmails)
	}
}

func TestLoad_InvalidPollInterval(t *testing.T) {
	clearEnv()
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("POLL_INTERVAL_SECONDS", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid POLL_INTERVAL_SECONDS")
	}
}

func TestLoad_InvalidAlertCooldown(t *testing.T) {
	clearEnv()
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("ALERT_COOLDOWN_MINUTES", "abc")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid ALERT_COOLDOWN_MINUTES")
	}
}

func TestLoad_OIDCFields(t *testing.T) {
	clearEnv()
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("OIDC_ISSUER", "https://id.example.com")
	os.Setenv("OIDC_CLIENT_ID", "my-client")
	os.Setenv("OIDC_CLIENT_SECRET", "my-secret")
	os.Setenv("OIDC_REDIRECT_URL", "https://app.example.com/auth/callback")
	os.Setenv("SESSION_SECRET", "session-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.OIDCIssuer != "https://id.example.com" {
		t.Errorf("OIDCIssuer = %q", cfg.OIDCIssuer)
	}
	if cfg.OIDCClientID != "my-client" {
		t.Errorf("OIDCClientID = %q", cfg.OIDCClientID)
	}
	if cfg.OIDCClientSecret != "my-secret" {
		t.Errorf("OIDCClientSecret = %q", cfg.OIDCClientSecret)
	}
	if cfg.OIDCRedirectURL != "https://app.example.com/auth/callback" {
		t.Errorf("OIDCRedirectURL = %q", cfg.OIDCRedirectURL)
	}
	if cfg.SessionSecret != "session-key" {
		t.Errorf("SessionSecret = %q", cfg.SessionSecret)
	}
}
