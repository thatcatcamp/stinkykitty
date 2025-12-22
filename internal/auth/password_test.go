package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "test-password-123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if hash == password {
		t.Error("Hash should not equal plain password")
	}

	// Hash should start with bcrypt prefix
	if len(hash) < 60 {
		t.Error("Bcrypt hash should be at least 60 characters")
	}
}

func TestCheckPasswordCorrect(t *testing.T) {
	password := "test-password-123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should return true for correct password")
	}
}

func TestCheckPasswordIncorrect(t *testing.T) {
	password := "test-password-123"
	wrongPassword := "wrong-password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if CheckPassword(wrongPassword, hash) {
		t.Error("CheckPassword should return false for incorrect password")
	}
}

func TestCheckPasswordEmpty(t *testing.T) {
	hash, _ := HashPassword("test")

	if CheckPassword("", hash) {
		t.Error("CheckPassword should return false for empty password")
	}
}

func TestHashPasswordEmpty(t *testing.T) {
	_, err := HashPassword("")
	if err == nil {
		t.Error("HashPassword should return error for empty password")
	}
}
