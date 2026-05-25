package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateDepartmentRequest is the body for POST /departments.
//
// @Description Create department request body
type CreateDepartmentRequest struct {
	Name        string  `json:"name"        binding:"required,min=1,max=100"`
	Description string  `json:"description" binding:"omitempty,max=500"`
	ManagerID   *string `json:"manager_id"  binding:"omitempty,uuid"`
}

// UpdateDepartmentRequest is the body for PATCH /departments/:id.
//
// @Description Update department request body
type UpdateDepartmentRequest struct {
	Name        *string `json:"name"        binding:"omitempty,min=1,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
	ManagerID   *string `json:"manager_id"  binding:"omitempty,uuid"`
}

// DepartmentResponse is the public representation of a department.
//
// @Description Department resource
type DepartmentResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	ManagerID   *uuid.UUID `json:"manager_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// DepartmentListFilter are query parameters for GET /departments.
type DepartmentListFilter struct {
	PaginationRequest
	ManagerID *string `form:"manager_id" binding:"omitempty,uuid"`
}
