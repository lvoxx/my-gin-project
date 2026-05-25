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

// allowedEmployeeSortCols whitelists column names to prevent SQL injection
// in the dynamic ORDER BY clause.
var allowedEmployeeSortCols = map[string]struct{}{
	"first_name": {}, "last_name": {}, "email": {},
	"role": {}, "created_at": {}, "updated_at": {},
}

// PostgresEmployeeRepository implements EmployeeRepository against PostgreSQL.
type PostgresEmployeeRepository struct {
	pool *pgxpool.Pool
}

// NewEmployeeRepository creates a PostgresEmployeeRepository.
func NewEmployeeRepository(pool *pgxpool.Pool) EmployeeRepository {
	return &PostgresEmployeeRepository{pool: pool}
}

// db returns the active DBTX — a transaction if one exists in ctx, otherwise
// the connection pool.  All repository methods call this so they participate
// in service-level transactions automatically.
func (r *PostgresEmployeeRepository) db(ctx context.Context) database.DBTX {
	return database.FromContext(ctx, r.pool)
}

// Create inserts a new employee row.
func (r *PostgresEmployeeRepository) Create(ctx context.Context, e *domain.Employee) error {
	const sql = `
		INSERT INTO employees
			(first_name, last_name, email, password_hash, role, department_id, avatar_url, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	row := r.db(ctx).QueryRow(ctx, sql,
		e.FirstName, e.LastName, e.Email, e.PasswordHash,
		e.Role, e.DepartmentID, e.AvatarURL, e.IsActive,
	)
	if err := row.Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return apperror.NewConflict("an employee with this email already exists")
		}
		return apperror.NewInternal("failed to create employee: " + err.Error())
	}
	return nil
}

// FindByID fetches a single employee by primary key.
func (r *PostgresEmployeeRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error) {
	const sql = `
		SELECT id, first_name, last_name, email, password_hash, role,
		       department_id, avatar_url, is_active, created_at, updated_at
		FROM employees WHERE id = $1`

	e := &domain.Employee{}
	err := r.db(ctx).QueryRow(ctx, sql, id).Scan(
		&e.ID, &e.FirstName, &e.LastName, &e.Email, &e.PasswordHash,
		&e.Role, &e.DepartmentID, &e.AvatarURL, &e.IsActive,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound("Employee")
		}
		return nil, apperror.NewInternal("failed to fetch employee: " + err.Error())
	}
	return e, nil
}

// FindByEmail fetches an employee by their unique email address.
func (r *PostgresEmployeeRepository) FindByEmail(ctx context.Context, email string) (*domain.Employee, error) {
	const sql = `
		SELECT id, first_name, last_name, email, password_hash, role,
		       department_id, avatar_url, is_active, created_at, updated_at
		FROM employees WHERE email = $1`

	e := &domain.Employee{}
	err := r.db(ctx).QueryRow(ctx, sql, email).Scan(
		&e.ID, &e.FirstName, &e.LastName, &e.Email, &e.PasswordHash,
		&e.Role, &e.DepartmentID, &e.AvatarURL, &e.IsActive,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound("Employee")
		}
		return nil, apperror.NewInternal("failed to fetch employee: " + err.Error())
	}
	return e, nil
}

// FindAll returns a paged, filtered list of employees plus the total row count.
func (r *PostgresEmployeeRepository) FindAll(ctx context.Context, f domain.EmployeeFilter) ([]*domain.Employee, int64, error) {
	f.Normalise()

	where, args := buildEmployeeWhere(f)
	orderBy := buildOrderBy(f.SortBy, f.SortOrder, allowedEmployeeSortCols, "created_at")

	countSQL := "SELECT COUNT(*) FROM employees" + where
	var total int64
	if err := r.db(ctx).QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperror.NewInternal("failed to count employees: " + err.Error())
	}

	args = append(args, f.Limit, f.Offset())
	listSQL := fmt.Sprintf(`
		SELECT id, first_name, last_name, email, password_hash, role,
		       department_id, avatar_url, is_active, created_at, updated_at
		FROM employees%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, where, orderBy, len(args)-1, len(args))

	rows, err := r.db(ctx).Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, apperror.NewInternal("failed to list employees: " + err.Error())
	}
	defer rows.Close()

	return scanEmployeeRows(rows)
}

// Search performs full-text search using PostgreSQL tsvector/tsquery.
func (r *PostgresEmployeeRepository) Search(ctx context.Context, query string, f domain.EmployeeFilter) ([]*domain.Employee, int64, error) {
	f.Normalise()

	// Build optional filters beyond the FTS predicate.
	var filters []string
	args := []any{query} // $1 = tsquery
	argIdx := 2

	if f.IsActive != nil {
		filters = append(filters, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *f.IsActive)
		argIdx++
	}
	if f.DepartmentID != nil {
		filters = append(filters, fmt.Sprintf("department_id = $%d", argIdx))
		args = append(args, *f.DepartmentID)
		argIdx++
	}

	extraWhere := ""
	if len(filters) > 0 {
		extraWhere = " AND " + strings.Join(filters, " AND ")
	}

	countSQL := `
		SELECT COUNT(*) FROM employees
		WHERE search_vector @@ plainto_tsquery('english', $1)` + extraWhere

	var total int64
	if err := r.db(ctx).QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperror.NewInternal("failed to count search results: " + err.Error())
	}

	args = append(args, f.Limit, f.Offset())
	listSQL := fmt.Sprintf(`
		SELECT id, first_name, last_name, email, password_hash, role,
		       department_id, avatar_url, is_active, created_at, updated_at
		FROM employees
		WHERE search_vector @@ plainto_tsquery('english', $1)%s
		ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC
		LIMIT $%d OFFSET $%d`, extraWhere, argIdx, argIdx+1)

	rows, err := r.db(ctx).Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, apperror.NewInternal("failed to search employees: " + err.Error())
	}
	defer rows.Close()

	return scanEmployeeRows(rows)
}

// Update overwrites all mutable employee columns.
func (r *PostgresEmployeeRepository) Update(ctx context.Context, e *domain.Employee) error {
	const sql = `
		UPDATE employees
		SET first_name    = $1,
		    last_name     = $2,
		    email         = $3,
		    password_hash = $4,
		    role          = $5,
		    department_id = $6,
		    avatar_url    = $7,
		    is_active     = $8
		WHERE id = $9
		RETURNING updated_at`

	err := r.db(ctx).QueryRow(ctx, sql,
		e.FirstName, e.LastName, e.Email, e.PasswordHash,
		e.Role, e.DepartmentID, e.AvatarURL, e.IsActive, e.ID,
	).Scan(&e.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.NewNotFound("Employee")
		}
		if isUniqueViolation(err) {
			return apperror.NewConflict("email address is already taken")
		}
		return apperror.NewInternal("failed to update employee: " + err.Error())
	}
	return nil
}

// UpdateAvatar updates only the avatar_url column for an employee.
func (r *PostgresEmployeeRepository) UpdateAvatar(ctx context.Context, id uuid.UUID, avatarURL string) error {
	const sql = `UPDATE employees SET avatar_url = $1 WHERE id = $2`
	tag, err := r.db(ctx).Exec(ctx, sql, avatarURL, id)
	if err != nil {
		return apperror.NewInternal("failed to update avatar: " + err.Error())
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound("Employee")
	}
	return nil
}

// Delete removes an employee permanently.
func (r *PostgresEmployeeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const sql = `DELETE FROM employees WHERE id = $1`
	tag, err := r.db(ctx).Exec(ctx, sql, id)
	if err != nil {
		return apperror.NewInternal("failed to delete employee: " + err.Error())
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound("Employee")
	}
	return nil
}

// ExistsByEmail reports whether an employee with the given email exists.
func (r *PostgresEmployeeRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const sql = `SELECT EXISTS(SELECT 1 FROM employees WHERE email = $1)`
	var exists bool
	if err := r.db(ctx).QueryRow(ctx, sql, email).Scan(&exists); err != nil {
		return false, apperror.NewInternal("failed to check employee email: " + err.Error())
	}
	return exists, nil
}

// ─── Filter / scan helpers ────────────────────────────────────────────────────

// Normalise applies EmployeeFilter defaults (delegates to EmployeeFilter method).
func (f *domain.EmployeeFilter) Normalise() {
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

// Offset returns the SQL OFFSET for f.
func (f *domain.EmployeeFilter) Offset() int {
	return (f.Page - 1) * f.Limit
}

func buildEmployeeWhere(f domain.EmployeeFilter) (string, []any) {
	var clauses []string
	var args []any
	idx := 1

	if f.DepartmentID != nil {
		clauses = append(clauses, fmt.Sprintf("department_id = $%d", idx))
		args = append(args, *f.DepartmentID)
		idx++
	}
	if f.Role != nil {
		clauses = append(clauses, fmt.Sprintf("role = $%d", idx))
		args = append(args, *f.Role)
		idx++
	}
	if f.IsActive != nil {
		clauses = append(clauses, fmt.Sprintf("is_active = $%d", idx))
		args = append(args, *f.IsActive)
	}

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func buildOrderBy(sortBy, sortOrder string, allowed map[string]struct{}, defaultCol string) string {
	col := defaultCol
	if _, ok := allowed[sortBy]; ok {
		col = sortBy
	}
	order := "ASC"
	if strings.EqualFold(sortOrder, "desc") {
		order = "DESC"
	}
	return col + " " + order
}

func scanEmployeeRows(rows pgx.Rows) ([]*domain.Employee, int64, error) {
	var employees []*domain.Employee
	for rows.Next() {
		e := &domain.Employee{}
		if err := rows.Scan(
			&e.ID, &e.FirstName, &e.LastName, &e.Email, &e.PasswordHash,
			&e.Role, &e.DepartmentID, &e.AvatarURL, &e.IsActive,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, 0, apperror.NewInternal("failed to scan employee row: " + err.Error())
		}
		employees = append(employees, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperror.NewInternal("row iteration error: " + err.Error())
	}
	return employees, int64(len(employees)), nil
}

// isUniqueViolation detects PostgreSQL error code 23505 (unique_violation).
func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "23505") ||
		strings.Contains(err.Error(), "unique_violation")
}
