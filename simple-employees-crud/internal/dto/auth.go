package dto

import "github.com/google/uuid"

// LoginRequest carries the credentials submitted by the client.
//
// @Description Login credentials
type LoginRequest struct {
	// Email is the employee's registered email address.
	Email string `json:"email" binding:"required,email"`
	// Password is the plaintext password (transmitted over TLS).
	Password string `json:"password" binding:"required,min=6,max=128"`
}

// TokenResponse is returned on a successful login.
//
// @Description Successful authentication response
type TokenResponse struct {
	// AccessToken is a short-lived JWT Bearer token.
	AccessToken string `json:"access_token"`
	// TokenType is always "Bearer".
	TokenType string `json:"token_type"`
	// ExpiresIn is the token lifetime in seconds.
	ExpiresIn int64 `json:"expires_in"`
	// Employee is the authenticated employee's public profile.
	Employee EmployeeResponse `json:"employee"`
}

// MeResponse is returned by GET /auth/me.
//
// @Description Authenticated employee profile
type MeResponse struct {
	ID           uuid.UUID  `json:"id"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	AvatarURL    string     `json:"avatar_url,omitempty"`
	IsActive     bool       `json:"is_active"`
}
