// SPDX-License-Identifier: MIT
package users

import (
	"testing"

	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Site{}, &models.SiteUser{}, &models.Page{}, &models.Block{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}
	return db
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)

	user, err := CreateUser(db, "test@example.com", "password123")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", user.Email)
	}

	if user.PasswordHash == "password123" {
		t.Error("Password should be hashed, not stored in plain text")
	}
}

func TestGetUserByEmail(t *testing.T) {
	db := setupTestDB(t)

	if _, err := CreateUser(db, "find@example.com", "password"); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	user, err := GetUserByEmail(db, "find@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}

	if user.Email != "find@example.com" {
		t.Errorf("Expected email find@example.com, got %s", user.Email)
	}
}

func TestListUsers(t *testing.T) {
	db := setupTestDB(t)

	if _, err := CreateUser(db, "user1@example.com", "pass1"); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if _, err := CreateUser(db, "user2@example.com", "pass2"); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	users, err := ListUsers(db)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestDeleteUser(t *testing.T) {
	db := setupTestDB(t)

	user, _ := CreateUser(db, "delete@example.com", "password")

	err := DeleteUser(db, user.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// Verify user is deleted
	_, err = GetUserByEmail(db, "delete@example.com")
	if err == nil {
		t.Error("Expected error when getting deleted user, got nil")
	}
}
