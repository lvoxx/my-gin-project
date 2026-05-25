package domain

import (
	"time"

	"github.com/google/uuid"
)

// Department is the organisational unit that groups employees.
type Department struct {
	ID          uuid.UUID
	Name        string
	Description string
	ManagerID   *uuid.UUID // references employees.id; nil when unset
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DepartmentFilter carries optional predicates for listing departments.
type DepartmentFilter struct {
	ManagerID *uuid.UUID
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
}
