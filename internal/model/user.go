package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents the core domain entity.
type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Name      string    `json:"name" db:"name"`
	Password  string    `json:"-" db:"password_hash"` // Never serialized to JSON
	Role      Role      `json:"role" db:"role"`
	Active    bool      `json:"active" db:"active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Role defines user authorization levels.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)
