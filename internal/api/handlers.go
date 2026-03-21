package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/colinbruner/cronhealth/internal/db"
	"github.com/colinbruner/cronhealth/internal/sse"
)

type Handlers struct {
	DB  *db.DB
	Hub *sse.Hub
}

// --- Ping (unauthenticated) ---

func (h *Handlers) Ping(c *gin.Context) {
	slug := c.Param("slug")

	check, err := h.DB.GetCheckBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}
	if check == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "check not found"})
		return
	}

	var sourceIP *string
	if ip := c.ClientIP(); ip != "" {
		sourceIP = &ip
	}

	var exitCode *int
	if ecStr := c.Query("exit_code"); ecStr != "" {
		if ec, err := strconv.Atoi(ecStr); err == nil {
			exitCode = &ec
		}
	}

	recovered, err := h.DB.RecordPingWithRecovery(c.Request.Context(), check.ID, sourceIP, exitCode)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to record ping"})
		return
	}

	resp := gin.H{"ok": true, "check": check.Name}
	if recovered {
		resp["recovered"] = true
	}
	c.JSON(http.StatusOK, resp)
}

// --- Checks (authenticated) ---

func (h *Handlers) ListChecks(c *gin.Context) {
	checks, err := h.DB.ListChecks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list checks"})
		return
	}
	if checks == nil {
		checks = []db.Check{}
	}
	c.JSON(http.StatusOK, checks)
}

type createCheckRequest struct {
	Name          string      `json:"name" binding:"required"`
	PeriodSeconds int         `json:"period_seconds" binding:"required,min=1"`
	GraceSeconds  int         `json:"grace_seconds"`
	ChannelIDs    []uuid.UUID `json:"channel_ids"`
}

func (h *Handlers) CreateCheck(c *gin.Context) {
	var req createCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.GraceSeconds <= 0 {
		req.GraceSeconds = 300
	}

	slug := uuid.New().String()
	userID := getUserID(c)

	check, err := h.DB.CreateCheck(c.Request.Context(), db.CreateCheckParams{
		Name:          req.Name,
		Slug:          slug,
		PeriodSeconds: req.PeriodSeconds,
		GraceSeconds:  req.GraceSeconds,
		CreatedBy:     userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create check"})
		return
	}

	if len(req.ChannelIDs) > 0 {
		if err := h.DB.SetCheckChannels(c.Request.Context(), check.ID, req.ChannelIDs); err != nil {
			// Non-fatal: check was created, channels just weren't linked
		}
	}

	c.JSON(http.StatusCreated, check)
}

func (h *Handlers) GetCheck(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid check ID"})
		return
	}

	check, err := h.DB.GetCheck(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get check"})
		return
	}
	if check == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "check not found"})
		return
	}

	c.JSON(http.StatusOK, check)
}

type updateCheckRequest struct {
	Name          string `json:"name" binding:"required"`
	PeriodSeconds int    `json:"period_seconds" binding:"required,min=1"`
	GraceSeconds  int    `json:"grace_seconds"`
}

func (h *Handlers) UpdateCheck(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid check ID"})
		return
	}

	var req updateCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.GraceSeconds <= 0 {
		req.GraceSeconds = 300
	}

	check, err := h.DB.UpdateCheck(c.Request.Context(), db.UpdateCheckParams{
		ID:            id,
		Name:          req.Name,
		PeriodSeconds: req.PeriodSeconds,
		GraceSeconds:  req.GraceSeconds,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update check"})
		return
	}
	if check == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "check not found"})
		return
	}

	c.JSON(http.StatusOK, check)
}

func (h *Handlers) DeleteCheck(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid check ID"})
		return
	}

	if err := h.DB.DeleteCheck(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "check not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handlers) ListPings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid check ID"})
		return
	}

	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	pings, err := h.DB.ListPings(c.Request.Context(), id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list pings"})
		return
	}
	if pings == nil {
		pings = []db.Ping{}
	}
	c.JSON(http.StatusOK, pings)
}

// --- Snooze / Silence ---

type snoozeRequest struct {
	DurationMinutes int `json:"duration_minutes" binding:"required,min=1"`
}

func (h *Handlers) SnoozeCheck(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid check ID"})
		return
	}

	var req snoozeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	endsAt := time.Now().UTC().Add(time.Duration(req.DurationMinutes) * time.Minute)
	userID := getUserID(c)
	reason := "snoozed"

	silence, err := h.DB.CreateSilence(c.Request.Context(), id, userID, &endsAt, &reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to snooze"})
		return
	}

	c.JSON(http.StatusOK, silence)
}

type silenceRequest struct {
	Reason string     `json:"reason"`
	EndsAt *time.Time `json:"ends_at"`
}

func (h *Handlers) SilenceCheck(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid check ID"})
		return
	}

	var req silenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	var reason *string
	if req.Reason != "" {
		reason = &req.Reason
	}

	silence, err := h.DB.CreateSilence(c.Request.Context(), id, userID, req.EndsAt, reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to silence"})
		return
	}

	c.JSON(http.StatusOK, silence)
}

func (h *Handlers) RemoveSilence(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid check ID"})
		return
	}

	if err := h.DB.DeleteSilence(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active silence"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Alerts ---

func (h *Handlers) ListAlerts(c *gin.Context) {
	alerts, err := h.DB.ListAlerts(c.Request.Context(), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list alerts"})
		return
	}
	if alerts == nil {
		alerts = []db.Alert{}
	}
	c.JSON(http.StatusOK, alerts)
}

func (h *Handlers) GetAlert(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	alert, err := h.DB.GetAlert(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get alert"})
		return
	}
	if alert == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// --- SSE ---

func (h *Handlers) Events(c *gin.Context) {
	ch, unregister := h.Hub.Register()
	defer unregister()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			c.SSEvent(event.Type, event.Data)
			c.Writer.Flush()
		}
	}
}

// --- Health ---

func (h *Handlers) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) Ready(c *gin.Context) {
	if err := h.DB.Pool.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// --- Helpers ---

func getUserID(c *gin.Context) *uuid.UUID {
	if idStr, exists := c.Get("user_id"); exists {
		if s, ok := idStr.(string); ok {
			if id, err := uuid.Parse(s); err == nil {
				return &id
			}
		}
	}
	return nil
}
