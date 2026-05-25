// Package service contains the application's business logic.
// Services orchestrate one or more repositories, apply domain rules, manage
// caching, and return typed errors from the apperror package.
// They never import Gin or write HTTP responses.
package service

import (
	"context"
	"time"

	"simple-employees-crud/internal/domain"
	"simple-employees-crud/internal/dto"
	"simple-employees-crud/internal/repository"
	"simple-employees-crud/pkg/apperror"
	"simple-employees-crud/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication concerns: credential validation and
// JWT issuance.  It depends on the EmployeeRepository and a JWT manager.
type AuthService struct {
	employeeRepo   repository.EmployeeRepository
	jwtManager     *jwt.Manager
	accessTokenExp time.Duration
}

// NewAuthService constructs an AuthService.
func NewAuthService(
	employeeRepo repository.EmployeeRepository,
	jwtManager *jwt.Manager,
	accessTokenExp time.Duration,
) *AuthService {
	return &AuthService{
		employeeRepo:   employeeRepo,
		jwtManager:     jwtManager,
		accessTokenExp: accessTokenExp,
	}
}

// Login validates credentials and returns a signed access token together with
// the authenticated employee's public profile.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenResponse, error) {
	// 1. Locate the employee by email; treat both "not found" and "wrong
	//    password" as the same generic error to avoid email enumeration.
	employee, err := s.employeeRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if apperror.IsNotFound(err) {
			return nil, apperror.NewUnauthorized("invalid email or password")
		}
		return nil, err
	}

	// 2. Verify the plaintext password against the stored bcrypt hash.
	if err := bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperror.NewUnauthorized("invalid email or password")
	}

	// 3. Reject disabled accounts.
	if !employee.IsActive {
		return nil, apperror.NewForbidden("account is disabled; contact an administrator")
	}

	// 4. Issue a signed access token.
	token, err := s.jwtManager.Generate(employee.ID, employee.Email, string(employee.Role), s.accessTokenExp)
	if err != nil {
		return nil, apperror.NewInternal("failed to generate access token")
	}

	return &dto.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.accessTokenExp.Seconds()),
		Employee:    mapEmployeeToResponse(employee),
	}, nil
}

// GetMe returns the authenticated employee's profile by ID (read from JWT claims).
func (s *AuthService) GetMe(ctx context.Context, employeeID string) (*dto.MeResponse, error) {
	uid, err := parseUUID(employeeID)
	if err != nil {
		return nil, apperror.NewBadRequest("invalid employee ID in token")
	}

	employee, err := s.employeeRepo.FindByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	return &dto.MeResponse{
		ID:           employee.ID,
		FirstName:    employee.FirstName,
		LastName:     employee.LastName,
		Email:        employee.Email,
		Role:         string(employee.Role),
		DepartmentID: employee.DepartmentID,
		AvatarURL:    employee.AvatarURL,
		IsActive:     employee.IsActive,
	}, nil
}

// HashPassword produces a bcrypt hash suitable for storing in the database.
// Exposed as a package-level helper so EmployeeService can reuse it.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", apperror.NewInternal("failed to hash password")
	}
	return string(hash), nil
}

// mapEmployeeToResponse converts a domain.Employee to a dto.EmployeeResponse.
// Centralised here so all services produce consistent output shapes.
func mapEmployeeToResponse(e *domain.Employee) dto.EmployeeResponse {
	return dto.EmployeeResponse{
		ID:           e.ID,
		FirstName:    e.FirstName,
		LastName:     e.LastName,
		FullName:     e.FullName(),
		Email:        e.Email,
		Role:         string(e.Role),
		DepartmentID: e.DepartmentID,
		AvatarURL:    e.AvatarURL,
		IsActive:     e.IsActive,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}
