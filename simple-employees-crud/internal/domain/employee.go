// Package domain contains the core business entities.
// These structs are plain Go — no framework or persistence tags —
// so the domain layer stays independent of any infrastructure concern.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// EmployeeRole enumerates the access roles available within the system.
type EmployeeRole string

const (
	RoleAdmin    EmployeeRole = "admin"
	RoleManager  EmployeeRole = "manager"
	RoleEmployee EmployeeRole = "employee"
)

// IsValid reports whether r is a recognised role.
func (r EmployeeRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleManager, RoleEmployee:
		return true
	}
	return false
}

// hierarchy maps roles to numeric privilege levels for comparison.
var hierarchy = map[EmployeeRole]int{
	RoleEmployee: 1,
	RoleManager:  2,
	RoleAdmin:    3,
}

// Employee is the core domain entity representing a member of staff.
type Employee struct {
	ID           uuid.UUID
	FirstName    string
	LastName     string
	Email        string
	PasswordHash string // bcrypt hash; never serialised to JSON
	Role         EmployeeRole
	DepartmentID *uuid.UUID // nil when unassigned
	AvatarURL    string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// FullName returns the employee's display name.
func (e *Employee) FullName() string {
	return e.FirstName + " " + e.LastName
}

// HasRole reports whether the employee holds exactly the given role.
func (e *Employee) HasRole(role EmployeeRole) bool {
	return e.Role == role
}

// IsAtLeast reports whether the employee's privilege level is greater than
// or equal to that of role.  Useful for "admin and above" checks.
func (e *Employee) IsAtLeast(role EmployeeRole) bool {
	return hierarchy[e.Role] >= hierarchy[role]
}

// Normalise applies default pagination constraints.
func (f *EmployeeFilter) Normalise() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
}

// Offset returns SQL OFFSET value.
func (f *EmployeeFilter) Offset() int {
	return (f.Page - 1) * f.Limit
}

// EmployeeFilter carries optional predicates for listing employees.
// nil pointer fields mean "no filter on this column".
type EmployeeFilter struct {
	DepartmentID *uuid.UUID
	Role         *EmployeeRole
	IsActive     *bool
	Page         int
	Limit        int
	SortBy       string // column name, validated in repository
	SortOrder    string // "asc" | "desc"
}
