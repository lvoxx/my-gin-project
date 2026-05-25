-- =============================================================================
-- Employee Management System — Initial Schema
-- Version: 001
-- Applied automatically by the PostgreSQL container via the init directory.
-- Do NOT apply through the Go application; keep schema and app concerns separate.
-- =============================================================================

-- ─── Extensions ──────────────────────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";    -- uuid_generate_v4()
CREATE EXTENSION IF NOT EXISTS "pg_trgm";      -- optional: trigram similarity

-- ─── Departments ─────────────────────────────────────────────────────────────
-- Created first; employees reference it via department_id.
-- The manager_id circular FK is added after employees is created.
CREATE TABLE IF NOT EXISTS departments (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT         DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ─── Employees ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS employees (
    id            UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    first_name    VARCHAR(50)  NOT NULL,
    last_name     VARCHAR(50)  NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT         NOT NULL,
    role          VARCHAR(20)  NOT NULL DEFAULT 'employee'
                               CHECK (role IN ('admin', 'manager', 'employee')),
    department_id UUID         REFERENCES departments(id) ON DELETE SET NULL,
    avatar_url    TEXT         DEFAULT '',
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    -- Populated and kept current by the trigger below.
    search_vector TSVECTOR,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Resolve circular FK: department manager → employee
ALTER TABLE departments
    ADD COLUMN IF NOT EXISTS manager_id UUID REFERENCES employees(id) ON DELETE SET NULL;

-- ─── Indexes ─────────────────────────────────────────────────────────────────

-- Primary lookup paths
CREATE INDEX IF NOT EXISTS idx_employees_email         ON employees(email);
CREATE INDEX IF NOT EXISTS idx_employees_department_id ON employees(department_id);
CREATE INDEX IF NOT EXISTS idx_employees_role          ON employees(role);
CREATE INDEX IF NOT EXISTS idx_employees_is_active     ON employees(is_active);

-- Full-text search — GIN index on the tsvector column
CREATE INDEX IF NOT EXISTS idx_employees_search        ON employees USING GIN(search_vector);

-- Departments
CREATE INDEX IF NOT EXISTS idx_departments_manager_id ON departments(manager_id);

-- ─── Functions ───────────────────────────────────────────────────────────────

-- Rebuild the search_vector on every INSERT or UPDATE to employees.
-- Covers: first_name, last_name, email.  Extend the concatenation to add
-- other searchable fields (e.g. job title) in the future.
CREATE OR REPLACE FUNCTION fn_employee_search_vector()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.search_vector :=
        to_tsvector('english',
            COALESCE(NEW.first_name, '') || ' ' ||
            COALESCE(NEW.last_name,  '') || ' ' ||
            COALESCE(NEW.email,      '')
        );
    RETURN NEW;
END;
$$;

-- Generic updated_at refresh function reused by all tables.
CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- ─── Triggers ────────────────────────────────────────────────────────────────

DROP TRIGGER IF EXISTS trg_employee_search_vector ON employees;
CREATE TRIGGER trg_employee_search_vector
    BEFORE INSERT OR UPDATE ON employees
    FOR EACH ROW EXECUTE FUNCTION fn_employee_search_vector();

DROP TRIGGER IF EXISTS trg_employees_updated_at ON employees;
CREATE TRIGGER trg_employees_updated_at
    BEFORE UPDATE ON employees
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

DROP TRIGGER IF EXISTS trg_departments_updated_at ON departments;
CREATE TRIGGER trg_departments_updated_at
    BEFORE UPDATE ON departments
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- ─── Seed data ────────────────────────────────────────────────────────────────
-- Default system-admin account.
-- IMPORTANT: Replace the password_hash below with a fresh bcrypt hash.
--
-- Generate one with:
--   htpasswd -bnBC 12 "" 'YourNewPassword' | tr -d ':\n'
-- OR in Go:
--   hash, _ := bcrypt.GenerateFromPassword([]byte("YourNewPassword"), 12)
--   fmt.Println(string(hash))
--
-- Default credentials (CHANGE BEFORE GOING TO PRODUCTION):
--   email:    admin@company.com
--   password: Admin@123
INSERT INTO employees (first_name, last_name, email, password_hash, role, is_active)
VALUES (
    'System',
    'Admin',
    'admin@company.com',
    '$2a$12$REPLACETHISWITHAREALBCRYPTHASH00000000000000000000000000',
    'admin',
    true
)
ON CONFLICT (email) DO NOTHING;
