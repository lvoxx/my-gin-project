// Package repository defines the repository interfaces (ports in hexagonal
// architecture).  Concrete PostgreSQL implementations live in the same package
// but depend only on the DBTX abstraction, allowing easy substitution of
// in-memory fakes in unit tests.
package repository

import (
	"context"

	"github.com/google/uuid"

	"simple-employees-crud/internal/domain"
)

// EmployeeRepository is the data-access contract for the employees aggregate.
type EmployeeRepository interface {
	// Create persists a new employee and populates e.ID, e.CreatedAt, e.UpdatedAt.
	Create(ctx context.Context, e *domain.Employee) error

	// FindByID returns a single employee or *apperror.AppError(NOT_FOUND).
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error)

	// FindByEmail returns a single employee by unique email address.
	FindByEmail(ctx context.Context, email string) (*domain.Employee, error)

	// FindAll returns a paged, optionally-filtered list and the total row count.
	FindAll(ctx context.Context, f domain.EmployeeFilter) ([]*domain.Employee, int64, error)

	// Search performs PostgreSQL full-text search across name and email columns.
	// Results are ordered by ts_rank descending.
	Search(ctx context.Context, query string, f domain.EmployeeFilter) ([]*domain.Employee, int64, error)

	// Update persists a full employee record overwrite (caller manages partial updates).
	Update(ctx context.Context, e *domain.Employee) error

	// UpdateAvatar persists only the avatar URL for an employee.
	UpdateAvatar(ctx context.Context, id uuid.UUID, avatarURL string) error

	// Delete permanently removes an employee record.
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsByEmail reports whether an active employee with email exists.
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// DepartmentRepository is the data-access contract for the departments aggregate.
type DepartmentRepository interface {
	// Create persists a new department and populates d.ID, d.CreatedAt, d.UpdatedAt.
	Create(ctx context.Context, d *domain.Department) error

	// FindByID returns a single department or *apperror.AppError(NOT_FOUND).
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Department, error)

	// FindAll returns a paged list and the total row count.
	FindAll(ctx context.Context, f domain.DepartmentFilter) ([]*domain.Department, int64, error)

	// Update persists a full department record overwrite.
	Update(ctx context.Context, d *domain.Department) error

	// Delete permanently removes a department.
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsByName reports whether a department with name already exists.
	ExistsByName(ctx context.Context, name string) (bool, error)
}
