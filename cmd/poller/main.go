package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/colinbruner/cronhealth/internal/config"
	"github.com/colinbruner/cronhealth/internal/db"
	"github.com/colinbruner/cronhealth/internal/notify"
	"github.com/colinbruner/cronhealth/internal/poller"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	var notifier notify.Notifier
	if cfg.SESFrom != "" {
		n, err := notify.NewAWSNotifier(ctx, notify.Config{
			AWSRegion:  cfg.AWSRegion,
			SESFrom:    cfg.SESFrom,
			SNSEnabled: cfg.SNSEnabled,
		})
		if err != nil {
			log.Printf("[poller] warning: failed to initialize AWS notifier, using noop: %v", err)
			notifier = &notify.NoopNotifier{}
		} else {
			notifier = n
		}
	} else {
		log.Println("[poller] AWS_SES_FROM not set, using noop notifier")
		notifier = &notify.NoopNotifier{}
	}

	p := &poller.Poller{
		DB:              database,
		Notifier:        notifier,
		IntervalSeconds: cfg.PollIntervalSeconds,
		CooldownMinutes: cfg.AlertCooldownMinutes,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down poller...")
		cancel()
	}()

	p.Run(ctx)
}
