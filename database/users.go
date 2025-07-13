package db

import (
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type User struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"user_name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type UserModelInterface interface {
	Insert(user *User) error
	// Authenticate(email, password string) (int, error)
	// Exists(id int) (bool, error)
}

// Define a new UserModel type which wraps a database connection pool.
type UserModel struct {
	DB     *sql.DB
	Logger *zap.Logger
}

// CreateUser creates a new user
func (m *UserModel) Insert(user *User) error {
	query := `
	INSERT INTO users (user_id, user_name, email) 
	VALUES (?, ?, ?) 
	RETURNING id, created_at, updated_at`

	m.logger.Info("Creating new user",
		zap.Int("user_id", user.UserID),
		zap.String("username", user.Username),
		zap.String("email", user.Email))

	start := time.Now()
	err := m.DB.QueryRow(query, user.UserID, user.Username, user.Email).Scan(&user.CreatedAt, &user.UpdatedAt)

	duration := time.Since(start)

	if err != nil {
		m.logger.Error("Failed to create user",
			zap.Int("user_id", user.UserID),
			zap.String("username", user.Username),
			zap.String("email", user.Email),
			zap.Duration("duration", duration),
			zap.Error(err))
		return fmt.Errorf("failed to create user: %w", err)
	}

	m.logger.Info("User created successfully",
		zap.Int("user_id", user.UserID),
		zap.String("username", user.Username),
		zap.Duration("duration", duration))

	return nil
}
