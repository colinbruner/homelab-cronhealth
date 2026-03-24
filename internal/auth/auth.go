package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/colinbruner/cronhealth/internal/db"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

const (
	sessionCookieName = "cronhealth_session"
	stateCookieName   = "oidc_state"
	sessionMaxAge     = 86400 // 24 hours
	stateMaxAge       = 300   // 5 minutes
)

// UserStore is the interface required for user persistence.
type UserStore interface {
	UpsertUser(ctx context.Context, email string, name *string) (*db.User, error)
}

// Auth holds OIDC provider configuration and session management state.
type Auth struct {
	devMode       bool
	provider      *oidc.Provider
	oauth2Config  oauth2.Config
	verifier      *oidc.IDTokenVerifier
	sessionSecret []byte
	allowedEmails []string
	db            UserStore
}

// sessionClaims defines the JWT claims stored in the session cookie.
type sessionClaims struct {
	jwt.RegisteredClaims
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
}

// NewDev returns an Auth instance that bypasses OIDC for local development.
// All requests are treated as authenticated with a fixed dev user identity.
// Never use this in production.
func NewDev() *Auth {
	return &Auth{devMode: true}
}

// New discovers the OIDC provider and returns a configured Auth instance.
func New(ctx context.Context, issuer, clientID, clientSecret, redirectURL, sessionSecret string, allowedEmails []string, store UserStore) (*Auth, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discovering OIDC provider: %w", err)
	}

	oauth2Config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	return &Auth{
		provider:      provider,
		oauth2Config:  oauth2Config,
		verifier:      verifier,
		sessionSecret: []byte(sessionSecret),
		allowedEmails: allowedEmails,
		db:            store,
	}, nil
}

// LoginHandler generates a random state, stores it in a secure cookie, and
// redirects the user to the OIDC authorization URL.
// In dev bypass mode it redirects directly to / without OIDC.
func (a *Auth) LoginHandler(c *gin.Context) {
	if a.devMode {
		c.Redirect(http.StatusFound, "/")
		return
	}
	state, err := randomState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(stateCookieName, state, stateMaxAge, "/", "", true, true)

	c.Redirect(http.StatusFound, a.oauth2Config.AuthCodeURL(state))
}

// CallbackHandler handles the OIDC callback: validates state, exchanges the
// authorization code for tokens, verifies the ID token, checks the email
// against the allow-list, upserts the user, and issues a signed JWT session
// cookie.
// In dev bypass mode it is unreachable (login never redirects to the provider).
func (a *Auth) CallbackHandler(c *gin.Context) {
	if a.devMode {
		c.Redirect(http.StatusFound, "/")
		return
	}
	// Validate state cookie
	storedState, err := c.Cookie(stateCookieName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing state cookie"})
		return
	}

	if c.Query("state") != storedState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state mismatch"})
		return
	}

	// Clear the state cookie
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(stateCookieName, "", -1, "/", "", true, true)

	// Exchange authorization code for tokens
	oauth2Token, err := a.oauth2Config.Exchange(c.Request.Context(), c.Query("code"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token exchange failed"})
		return
	}

	// Extract and verify the ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing id_token"})
		return
	}

	idToken, err := a.verifier.Verify(c.Request.Context(), rawIDToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "id_token verification failed"})
		return
	}

	// Extract claims
	var claims struct {
		Email string  `json:"email"`
		Name  *string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse claims"})
		return
	}

	if claims.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email not provided by identity provider"})
		return
	}

	// Check against allowed emails (if the list is non-empty)
	if len(a.allowedEmails) > 0 {
		allowed := false
		for _, e := range a.allowedEmails {
			if e == claims.Email {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "email not authorized"})
			return
		}
	}

	// Upsert user in database
	user, err := a.db.UpsertUser(c.Request.Context(), claims.Email, claims.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upsert user"})
		return
	}

	// Create signed JWT session cookie
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, sessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
		},
		UserID:    user.ID.String(),
		UserEmail: user.Email,
	})

	signed, err := token.SignedString(a.sessionSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign session token"})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, signed, sessionMaxAge, "/", "", true, true)

	c.Redirect(http.StatusFound, "/")
}

// LogoutHandler clears the session cookie.
func (a *Auth) LogoutHandler(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, "", -1, "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// MeHandler returns the current user info from the session.
func (a *Auth) MeHandler(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userEmail, _ := c.Get("user_email")

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"email":   userEmail,
	})
}

// Middleware returns a gin middleware that validates the session cookie JWT and
// sets user_id and user_email in the gin context. Returns 401 if the cookie is
// missing or invalid.
// In dev bypass mode all requests are passed through as a fixed dev user.
func (a *Auth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.devMode {
			c.Set("user_id", "dev-user")
			c.Set("user_email", "dev@localhost")
			c.Next()
			return
		}
		cookie, err := c.Cookie(sessionCookieName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		token, err := jwt.ParseWithClaims(cookie, &sessionClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return a.sessionSecret, nil
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}

		claims, ok := token.Claims.(*sessionClaims)
		if !ok || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session claims"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.UserEmail)
		c.Next()
	}
}

// randomState generates a cryptographically random hex string for OIDC state.
func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
