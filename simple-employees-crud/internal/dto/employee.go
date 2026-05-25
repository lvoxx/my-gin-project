package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateEmployeeRequest is the body for POST /employees.
//
// @Description Create employee request body
type CreateEmployeeRequest struct {
	FirstName    string  `json:"first_name"    binding:"required,min=1,max=50"`
	LastName     string  `json:"last_name"     binding:"required,min=1,max=50"`
	Email        string  `json:"email"         binding:"required,email"`
	Password     string  `json:"password"      binding:"required,min=8,max=128"`
	Role         string  `json:"role"          binding:"required,oneof=admin manager employee"`
	DepartmentID *string `json:"department_id" binding:"omitempty,uuid"`
}

// UpdateEmployeeRequest is the body for PATCH /employees/:id.
// All fields are optional; only non-zero values are applied.
//
// @Description Update employee request body
type UpdateEmployeeRequest struct {
	FirstName    *string `json:"first_name"    binding:"omitempty,min=1,max=50"`
	LastName     *string `json:"last_name"     binding:"omitempty,min=1,max=50"`
	Role         *string `json:"role"          binding:"omitempty,oneof=admin manager employee"`
	DepartmentID *string `json:"department_id" binding:"omitempty,uuid"`
	IsActive     *bool   `json:"is_active"`
}

// EmployeeResponse is the public representation of an employee.
// The password hash is never included.
//
// @Description Employee resource
type EmployeeResponse struct {
	ID           uuid.UUID  `json:"id"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	FullName     string     `json:"full_name"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	AvatarURL    string     `json:"avatar_url,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// EmployeeListFilter are query parameters for GET /employees.
//
// @Description Employee list filter parameters
type EmployeeListFilter struct {
	PaginationRequest
	DepartmentID *string `form:"department_id" binding:"omitempty,uuid"`
	Role         *string `form:"role"          binding:"omitempty,oneof=admin manager employee"`
	IsActive     *bool   `form:"is_active"`
}

// EmployeeSearchFilter adds a required search query on top of pagination.
//
// @Description Employee search filter parameters
type EmployeeSearchFilter struct {
	SearchRequest
	DepartmentID *string `form:"department_id" binding:"omitempty,uuid"`
	IsActive     *bool   `form:"is_active"`
}

// AvatarUploadResponse is returned after a successful avatar upload.
//
// @Description Avatar upload response
type AvatarUploadResponse struct {
	AvatarURL string `json:"avatar_url"`
}
