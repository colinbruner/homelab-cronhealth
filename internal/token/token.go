package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Generate creates a new random ping token.
// Returns raw (64-char hex, never stored) and its SHA-256 hex hash (stored in DB).
func Generate() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("token.Generate: %w", err)
	}
	raw = hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(sum[:])
	return raw, hash, nil
}

// Verify returns true if sha256(raw) equals the stored hash.
// Returns false immediately for an empty hash (token not yet configured).
func Verify(raw, hash string) bool {
	if hash == "" {
		return false
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:]) == hash
}
