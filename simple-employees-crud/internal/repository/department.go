package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"simple-employees-crud/internal/domain"
	"simple-employees-crud/pkg/apperror"
	"simple-employees-crud/pkg/database"
)

// allowedDeptSortCols whitelists sortable columns for departments.
var allowedDeptSortCols = map[string]struct{}{
	"name": {}, "created_at": {}, "updated_at": {},
}

// PostgresDepartmentRepository implements DepartmentRepository.
type PostgresDepartmentRepository struct {
	pool *pgxpool.Pool
}

// NewDepartmentRepository creates a PostgresDepartmentRepository.
func NewDepartmentRepository(pool *pgxpool.Pool) DepartmentRepository {
	return &PostgresDepartmentRepository{pool: pool}
}

func (r *PostgresDepartmentRepository) db(ctx context.Context) database.DBTX {
	return database.FromContext(ctx, r.pool)
}

// Create inserts a new department row.
func (r *PostgresDepartmentRepository) Create(ctx context.Context, d *domain.Department) error {
	const sql = `
		INSERT INTO departments (name, description, manager_id)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`

	err := r.db(ctx).QueryRow(ctx, sql, d.Name, d.Description, d.ManagerID).
		Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return apperror.NewConflict("a department with this name already exists")
		}
		return apperror.NewInternal("failed to create department: " + err.Error())
	}
	return nil
}

// FindByID fetches a single department by primary key.
func (r *PostgresDepartmentRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Department, error) {
	const sql = `
		SELECT id, name, description, manager_id, created_at, updated_at
		FROM departments WHERE id = $1`

	d := &domain.Department{}
	err := r.db(ctx).QueryRow(ctx, sql, id).Scan(
		&d.ID, &d.Name, &d.Description, &d.ManagerID, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound("Department")
		}
		return nil, apperror.NewInternal("failed to fetch department: " + err.Error())
	}
	return d, nil
}

// FindAll returns a paged list of departments and the total count.
func (r *PostgresDepartmentRepository) FindAll(ctx context.Context, f domain.DepartmentFilter) ([]*domain.Department, int64, error) {
	// Normalise pagination.
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}

	var clauses []string
	var args []any
	idx := 1

	if f.ManagerID != nil {
		clauses = append(clauses, fmt.Sprintf("manager_id = $%d", idx))
		args = append(args, *f.ManagerID)
		idx++
	}

	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}

	countSQL := "SELECT COUNT(*) FROM departments" + where
	var total int64
	if err := r.db(ctx).QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperror.NewInternal("failed to count departments: " + err.Error())
	}

	orderBy := buildOrderBy(f.SortBy, f.SortOrder, allowedDeptSortCols, "name")
	offset := (f.Page - 1) * f.Limit
	args = append(args, f.Limit, offset)

	listSQL := fmt.Sprintf(`
		SELECT id, name, description, manager_id, created_at, updated_at
		FROM departments%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, where, orderBy, idx, idx+1)

	rows, err := r.db(ctx).Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, apperror.NewInternal("failed to list departments: " + err.Error())
	}
	defer rows.Close()

	var depts []*domain.Department
	for rows.Next() {
		d := &domain.Department{}
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.ManagerID, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, apperror.NewInternal("failed to scan department: " + err.Error())
		}
		depts = append(depts, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperror.NewInternal("row iteration error: " + err.Error())
	}

	return depts, total, nil
}

// Update overwrites all mutable department columns.
func (r *PostgresDepartmentRepository) Update(ctx context.Context, d *domain.Department) error {
	const sql = `
		UPDATE departments
		SET name = $1, description = $2, manager_id = $3
		WHERE id = $4
		RETURNING updated_at`

	err := r.db(ctx).QueryRow(ctx, sql, d.Name, d.Description, d.ManagerID, d.ID).Scan(&d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.NewNotFound("Department")
		}
		if isUniqueViolation(err) {
			return apperror.NewConflict("department name is already taken")
		}
		return apperror.NewInternal("failed to update department: " + err.Error())
	}
	return nil
}

// Delete removes a department permanently.
func (r *PostgresDepartmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const sql = `DELETE FROM departments WHERE id = $1`
	tag, err := r.db(ctx).Exec(ctx, sql, id)
	if err != nil {
		return apperror.NewInternal("failed to delete department: " + err.Error())
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound("Department")
	}
	return nil
}

// ExistsByName reports whether a department with name already exists.
func (r *PostgresDepartmentRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	const sql = `SELECT EXISTS(SELECT 1 FROM departments WHERE name = $1)`
	var exists bool
	if err := r.db(ctx).QueryRow(ctx, sql, name).Scan(&exists); err != nil {
		return false, apperror.NewInternal("failed to check department name: " + err.Error())
	}
	return exists, nil
}
