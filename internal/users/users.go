package users

import (
	"fmt"
	"strings"

	"github.com/thatcatcamp/stinkykitty/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// CreateUser creates a new user with hashed password
// If a soft-deleted user exists with this email, it will be restored
func CreateUser(db *gorm.DB, email, password string) (*models.User, error) {
	// Normalize email to lowercase
	email = strings.ToLower(strings.TrimSpace(email))

	// Check if active user already exists
	var existing models.User
	result := db.Where("email = ?", email).First(&existing)
	if result.Error == nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Check for soft-deleted user with this email
	var deletedUser models.User
	deletedResult := db.Unscoped().Where("email = ? AND deleted_at IS NOT NULL", email).First(&deletedUser)

	var user *models.User
	if deletedResult.Error == nil {
		// Found a soft-deleted user - restore them with new password
		user = &deletedUser
		if err := db.Unscoped().Model(user).Updates(map[string]interface{}{
			"deleted_at":    nil,
			"password_hash": string(hashedPassword),
		}).Error; err != nil {
			return nil, fmt.Errorf("failed to restore user: %w", err)
		}
	} else {
		// No deleted user found - create new user
		user = &models.User{
			Email:        email,
			PasswordHash: string(hashedPassword),
		}

		if err := db.Create(user).Error; err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email address
func GetUserByEmail(db *gorm.DB, email string) (*models.User, error) {
	// Normalize email to lowercase
	email = strings.ToLower(strings.TrimSpace(email))

	var user models.User
	result := db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found: %w", result.Error)
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func GetUserByID(db *gorm.DB, id uint) (*models.User, error) {
	var user models.User
	result := db.First(&user, id)
	if result.Error != nil {
		return nil, fmt.Errorf("user not found: %w", result.Error)
	}
	return &user, nil
}

// ListUsers returns all users
func ListUsers(db *gorm.DB) ([]models.User, error) {
	var users []models.User
	result := db.Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list users: %w", result.Error)
	}
	return users, nil
}

// DeleteUser soft-deletes a user
func DeleteUser(db *gorm.DB, id uint) error {
	result := db.Delete(&models.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// ValidatePassword checks if a password matches the user's hash
func ValidatePassword(user *models.User, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
}
