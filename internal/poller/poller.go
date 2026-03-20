package poller

import (
	"context"
	"log"
	"time"

	"github.com/colinbruner/cronhealth/internal/db"
	"github.com/colinbruner/cronhealth/internal/notify"
)

type Poller struct {
	DB               *db.DB
	Notifier         notify.Notifier
	IntervalSeconds  int
	CooldownMinutes  int
}

func (p *Poller) Run(ctx context.Context) {
	log.Printf("[poller] starting with interval=%ds cooldown=%dm", p.IntervalSeconds, p.CooldownMinutes)
	ticker := time.NewTicker(time.Duration(p.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	// Run immediately on start, then on each tick
	p.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[poller] shutting down")
			return
		case <-ticker.C:
			p.tick(ctx)
		}
	}
}

func (p *Poller) tick(ctx context.Context) {
	missed, err := p.DB.GetMissedChecks(ctx)
	if err != nil {
		log.Printf("[poller] error querying missed checks: %v", err)
		return
	}

	for _, check := range missed {
		p.processCheck(ctx, check)
	}
}

// processCheck implements the alert state machine:
//
//	up     → down      (first miss detected, within what was grace — but GetMissedChecks
//	                     already filters past grace, so this means transition to alerting)
//	down   → alerting  (grace expired, fire first notification)
//	alerting           (re-alert if cooldown expired)
func (p *Poller) processCheck(ctx context.Context, check db.Check) {
	switch check.Status {
	case db.StatusUp:
		// Check has missed its window — transition to down first.
		// On the next tick (if still missed), it will transition to alerting.
		// However, GetMissedChecks already filters past grace period,
		// so we go straight to alerting.
		p.transitionToAlerting(ctx, check)

	case db.StatusDown:
		// Grace period expired (GetMissedChecks already checked this) — alert
		p.transitionToAlerting(ctx, check)

	case db.StatusAlerting:
		// Already alerting — check if cooldown has expired for re-alert
		if check.LastAlertedAt == nil {
			// Shouldn't happen, but handle gracefully
			p.transitionToAlerting(ctx, check)
			return
		}
		cooldownExpiry := check.LastAlertedAt.Add(time.Duration(p.CooldownMinutes) * time.Minute)
		if time.Now().UTC().After(cooldownExpiry) {
			p.reAlert(ctx, check)
		}
		// else: still within cooldown window, skip
	}
}

func (p *Poller) transitionToAlerting(ctx context.Context, check db.Check) {
	if err := p.DB.TransitionToAlerting(ctx, check.ID); err != nil {
		log.Printf("[poller] error transitioning check %s to alerting: %v", check.Name, err)
		return
	}
	log.Printf("[poller] check %q → alerting", check.Name)

	channels, err := p.DB.GetChannelsForCheck(ctx, check.ID)
	if err != nil {
		log.Printf("[poller] error getting channels for check %s: %v", check.Name, err)
		return
	}
	if len(channels) == 0 {
		log.Printf("[poller] check %q has no notification channels configured", check.Name)
		return
	}

	p.Notifier.SendAlert(ctx, check, channels)
}

func (p *Poller) reAlert(ctx context.Context, check db.Check) {
	if err := p.DB.ReAlert(ctx, check.ID); err != nil {
		log.Printf("[poller] error re-alerting check %s: %v", check.Name, err)
		return
	}
	log.Printf("[poller] check %q re-alert (cooldown expired)", check.Name)

	channels, err := p.DB.GetChannelsForCheck(ctx, check.ID)
	if err != nil {
		log.Printf("[poller] error getting channels for check %s: %v", check.Name, err)
		return
	}
	if len(channels) == 0 {
		return
	}

	p.Notifier.SendAlert(ctx, check, channels)
}
