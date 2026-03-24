package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL string

	DevAuthBypass bool

	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	SessionSecret    string
	AllowedEmails    []string

	AWSRegion     string
	SESFrom       string
	SNSEnabled    bool

	PollIntervalSeconds  int
	AlertCooldownMinutes int

	Port string
}

func Load() (*Config, error) {
	c := &Config{
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		DevAuthBypass:    os.Getenv("DEV_AUTH_BYPASS") == "true",
		OIDCIssuer:       os.Getenv("OIDC_ISSUER"),
		OIDCClientID:     os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		OIDCRedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
		SessionSecret:    os.Getenv("SESSION_SECRET"),
		AWSRegion:        getEnvDefault("AWS_REGION", "us-east-1"),
		SESFrom:          os.Getenv("AWS_SES_FROM"),
		Port:             getEnvDefault("PORT", "8080"),
	}

	if emails := os.Getenv("ALLOWED_OIDC_EMAILS"); emails != "" {
		for _, e := range strings.Split(emails, ",") {
			if trimmed := strings.TrimSpace(e); trimmed != "" {
				c.AllowedEmails = append(c.AllowedEmails, trimmed)
			}
		}
	}

	c.SNSEnabled = os.Getenv("AWS_SNS_SMS_ENABLED") == "true"

	var err error
	c.PollIntervalSeconds, err = getEnvInt("POLL_INTERVAL_SECONDS", 30)
	if err != nil {
		return nil, fmt.Errorf("POLL_INTERVAL_SECONDS: %w", err)
	}
	c.AlertCooldownMinutes, err = getEnvInt("ALERT_COOLDOWN_MINUTES", 60)
	if err != nil {
		return nil, fmt.Errorf("ALERT_COOLDOWN_MINUTES: %w", err)
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return c, nil
}

func getEnvDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	return strconv.Atoi(v)
}
