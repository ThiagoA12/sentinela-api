package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleAnalista Role = "analista"
	RoleLeitor   Role = "leitor"
)

func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleAnalista, RoleLeitor:
		return true
	}
	return false
}

type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	Role         Role
	Active       bool
	FailedLogins int16
	LockedUntil  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Version      int32
}

func (u User) CanOwnAssets() bool {
	return u.Active && (u.Role == RoleAdmin || u.Role == RoleAnalista)
}
