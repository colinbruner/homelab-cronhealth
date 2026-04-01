package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/colinbruner/cronhealth/internal/api"
	"github.com/colinbruner/cronhealth/internal/auth"
	"github.com/colinbruner/cronhealth/internal/config"
	"github.com/colinbruner/cronhealth/internal/db"
	"github.com/colinbruner/cronhealth/internal/sse"
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

	hub := sse.NewHub()

	// Start SSE listener (Postgres LISTEN/NOTIFY → hub broadcast)
	go sse.StartListener(ctx, cfg.DatabaseURL, hub)

	// Set up auth
	var authenticator *auth.Auth
	if cfg.DevAuthBypass {
		log.Println("WARNING: DEV_AUTH_BYPASS=true — authentication is disabled, do not use in production")
		authenticator = auth.NewDev()
	} else {
		authenticator, err = auth.New(
			ctx,
			cfg.OIDCIssuer,
			cfg.OIDCClientID,
			cfg.OIDCClientSecret,
			cfg.OIDCRedirectURL,
			cfg.SessionSecret,
			database,
		)
		if err != nil {
			log.Fatalf("failed to initialize auth: %v", err)
		}
	}

	handlers := &api.Handlers{DB: database, Hub: hub}

	router := gin.Default()

	// Health probes (unauthenticated)
	router.GET("/health", handlers.Health)
	router.GET("/ready", handlers.Ready)

	// Ping endpoint (unauthenticated)
	router.POST("/ping/:slug", handlers.Ping)

	// Auth routes (unauthenticated)
	router.GET("/auth/login", authenticator.LoginHandler)
	router.GET("/auth/callback", authenticator.CallbackHandler)
	router.POST("/auth/logout", authenticator.LogoutHandler)

	// Authenticated API routes
	authorized := router.Group("/api")
	authorized.Use(authenticator.Middleware())
	{
		authorized.GET("/checks", handlers.ListChecks)
		authorized.POST("/checks", handlers.CreateCheck)
		authorized.GET("/checks/:id", handlers.GetCheck)
		authorized.PUT("/checks/:id", handlers.UpdateCheck)
		authorized.DELETE("/checks/:id", handlers.DeleteCheck)
		authorized.GET("/checks/:id/pings", handlers.ListPings)
		authorized.POST("/checks/:id/snooze", handlers.SnoozeCheck)
		authorized.POST("/checks/:id/silence", handlers.SilenceCheck)
		authorized.DELETE("/checks/:id/silence", handlers.RemoveSilence)

		authorized.GET("/alerts", handlers.ListAlerts)
		authorized.GET("/alerts/:id", handlers.GetAlert)

		authorized.GET("/events", handlers.Events)

		authorized.GET("/me", authenticator.MeHandler)
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down API server...")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("cronhealth-api listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
