// Package jwt wraps github.com/golang-jwt/jwt/v5 to provide a thin,
// typed API for generating and validating signed access tokens.
// The token payload (Claims) carries the employee ID, email, and role
// so downstream middleware can authorise requests without a DB round-trip.
package jwt

import (
	"errors"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is the custom JWT payload embedded inside a signed token.
type Claims struct {
	EmployeeID uuid.UUID `json:"employee_id"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	gojwt.RegisteredClaims
}

// Manager handles token generation and validation for a given signing secret.
type Manager struct {
	secret []byte
}

// NewManager creates a Manager. secret must not be empty.
func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

// Generate signs a new access token for the given employee attributes.
// exp controls the token lifetime (e.g. 60*time.Minute for 1 hour).
func (m *Manager) Generate(employeeID uuid.UUID, email, role string, exp time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		EmployeeID: employeeID,
		Email:      email,
		Role:       role,
		RegisteredClaims: gojwt.RegisteredClaims{
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(exp)),
			Issuer:    "employee-management",
		},
	}

	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", errors.New("jwt: failed to sign token: " + err.Error())
	}
	return signed, nil
}

// Validate parses and verifies a signed token string.
// Returns the decoded Claims on success, or an error if the token is
// expired, tampered with, or otherwise invalid.
func (m *Manager) Validate(tokenString string) (*Claims, error) {
	token, err := gojwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(t *gojwt.Token) (any, error) {
			// Guard against algorithm substitution attacks.
			if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
				return nil, errors.New("jwt: unexpected signing method")
			}
			return m.secret, nil
		},
		gojwt.WithIssuedAt(),
		gojwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("jwt: invalid token claims")
	}
	return claims, nil
}
