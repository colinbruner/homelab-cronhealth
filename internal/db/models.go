package db

import (
	"time"

	"github.com/google/uuid"
)

type CheckStatus string

const (
	StatusNew      CheckStatus = "new"
	StatusUp       CheckStatus = "up"
	StatusDown     CheckStatus = "down"
	StatusAlerting CheckStatus = "alerting"
	StatusSilenced CheckStatus = "silenced"
)

type Check struct {
	ID              uuid.UUID    `json:"id"`
	Name            string       `json:"name"`
	Slug            string       `json:"slug"`
	PeriodSeconds   int          `json:"period_seconds"`
	GraceSeconds    int          `json:"grace_seconds"`
	Status          CheckStatus  `json:"status"`
	LastPingAt      *time.Time   `json:"last_ping_at"`
	LastAlertedAt   *time.Time   `json:"last_alerted_at"`
	CreatedAt       time.Time    `json:"created_at"`
	CreatedBy       *uuid.UUID   `json:"created_by"`
}

type Ping struct {
	ID         uuid.UUID  `json:"id"`
	CheckID    uuid.UUID  `json:"check_id"`
	ReceivedAt time.Time  `json:"received_at"`
	SourceIP   *string    `json:"source_ip"`
	ExitCode   *int       `json:"exit_code"`
}

type Alert struct {
	ID         uuid.UUID  `json:"id"`
	CheckID    uuid.UUID  `json:"check_id"`
	StartedAt  time.Time  `json:"started_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	AlertCount int        `json:"alert_count"`
	// Joined from checks for API convenience
	CheckName string     `json:"check_name,omitempty"`
}

type Silence struct {
	ID         uuid.UUID  `json:"id"`
	CheckID    uuid.UUID  `json:"check_id"`
	SilencedBy *uuid.UUID `json:"silenced_by"`
	StartsAt   time.Time  `json:"starts_at"`
	EndsAt     *time.Time `json:"ends_at"`
	Reason     *string    `json:"reason"`
	CreatedAt  time.Time  `json:"created_at"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      *string   `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationChannel struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Label     string    `json:"label"`
	Type      string    `json:"type"`   // email | sms
	Target    string    `json:"target"` // email address or E.164 phone
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// CheckWithChannels is used by the poller to fire notifications.
type CheckWithChannels struct {
	Check    Check
	Channels []NotificationChannel
}
