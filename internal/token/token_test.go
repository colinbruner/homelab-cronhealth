package token_test

import (
	"testing"

	"github.com/colinbruner/cronhealth/internal/token"
)

func TestGenerate_Length(t *testing.T) {
	raw, hash, err := token.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if len(raw) != 64 {
		t.Errorf("raw len = %d, want 64", len(raw))
	}
	if len(hash) != 64 {
		t.Errorf("hash len = %d, want 64", len(hash))
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	raw1, _, _ := token.Generate()
	raw2, _, _ := token.Generate()
	if raw1 == raw2 {
		t.Error("Generate() returned same token twice")
	}
}

func TestVerify_CorrectToken(t *testing.T) {
	raw, hash, err := token.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if !token.Verify(raw, hash) {
		t.Error("Verify(raw, hash) = false, want true")
	}
}

func TestVerify_WrongToken(t *testing.T) {
	_, hash, err := token.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if token.Verify("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", hash) {
		t.Error("Verify(wrong, hash) = true, want false")
	}
}

func TestVerify_EmptyHash(t *testing.T) {
	raw, _, err := token.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if token.Verify(raw, "") {
		t.Error("Verify(raw, \"\") = true, want false")
	}
}
