package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"simple-employees-crud/internal/domain"
	"simple-employees-crud/internal/dto"
	"simple-employees-crud/internal/repository"
	"simple-employees-crud/pkg/apperror"
	"simple-employees-crud/pkg/database"
	"simple-employees-crud/pkg/logger"
	"simple-employees-crud/pkg/storage"
)

const (
	employeeCacheKeyFmt  = "employee:%s"     // single record cache key
	employeeListCacheKey = "employee:list:*" // pattern for list invalidation
)

// EmployeeService handles all employee business logic.
type EmployeeService struct {
	repo       repository.EmployeeRepository
	redis      *redis.Client
	cloudinary *storage.CloudinaryClient
	cacheTTL   time.Duration
}

// NewEmployeeService constructs an EmployeeService.
// cloudinary may be nil in local development; avatar uploads will be skipped.
func NewEmployeeService(
	repo repository.EmployeeRepository,
	redisClient *redis.Client,
	cloudinary *storage.CloudinaryClient,
	cacheTTL time.Duration,
) *EmployeeService {
	return &EmployeeService{
		repo:       repo,
		redis:      redisClient,
		cloudinary: cloudinary,
		cacheTTL:   cacheTTL,
	}
}

// Create validates uniqueness, hashes the password, persists the employee, and
// invalidates relevant cache entries.
func (s *EmployeeService) Create(ctx context.Context, req dto.CreateEmployeeRequest) (*dto.EmployeeResponse, error) {
	// Verify email uniqueness before hashing (fast fail).
	exists, err := s.repo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflict("an employee with this email already exists")
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	employee := &domain.Employee{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         domain.EmployeeRole(req.Role),
		IsActive:     true,
	}

	if req.DepartmentID != nil {
		uid, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			return nil, apperror.NewBadRequest("invalid department_id format")
		}
		employee.DepartmentID = &uid
	}

	if err := s.repo.Create(ctx, employee); err != nil {
		return nil, err
	}

	s.invalidateListCache(ctx)

	resp := mapEmployeeToResponse(employee)
	return &resp, nil
}

// GetByID returns a single employee, using Redis as a read-through cache.
func (s *EmployeeService) GetByID(ctx context.Context, id uuid.UUID) (*dto.EmployeeResponse, error) {
	cacheKey := fmt.Sprintf(employeeCacheKeyFmt, id.String())

	// Cache read.
	var cached dto.EmployeeResponse
	if err := database.CacheGet(ctx, s.redis, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	employee, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := mapEmployeeToResponse(employee)

	// Write to cache; non-fatal on failure.
	if err := database.CacheSet(ctx, s.redis, cacheKey, resp, s.cacheTTL); err != nil {
		logger.S().Warnw("failed to cache employee", "id", id, "error", err)
	}

	return &resp, nil
}

// List returns a paged, optionally-filtered employee list.
func (s *EmployeeService) List(ctx context.Context, req dto.EmployeeListFilter) ([]*dto.EmployeeResponse, int64, error) {
	req.Normalise()

	filter := domain.EmployeeFilter{
		Page:      req.Page,
		Limit:     req.Limit,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	if req.IsActive != nil {
		filter.IsActive = req.IsActive
	}
	if req.Role != nil {
		role := domain.EmployeeRole(*req.Role)
		filter.Role = &role
	}
	if req.DepartmentID != nil {
		uid, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			return nil, 0, apperror.NewBadRequest("invalid department_id format")
		}
		filter.DepartmentID = &uid
	}

	employees, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.EmployeeResponse, len(employees))
	for i, e := range employees {
		resp := mapEmployeeToResponse(e)
		responses[i] = &resp
	}
	return responses, total, nil
}

// Search performs full-text search and returns matching employees.
func (s *EmployeeService) Search(ctx context.Context, req dto.EmployeeSearchFilter) ([]*dto.EmployeeResponse, int64, error) {
	req.Normalise()

	filter := domain.EmployeeFilter{
		Page:      req.Page,
		Limit:     req.Limit,
		SortOrder: req.SortOrder,
	}
	if req.IsActive != nil {
		filter.IsActive = req.IsActive
	}
	if req.DepartmentID != nil {
		uid, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			return nil, 0, apperror.NewBadRequest("invalid department_id format")
		}
		filter.DepartmentID = &uid
	}

	employees, total, err := s.repo.Search(ctx, req.Query, filter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.EmployeeResponse, len(employees))
	for i, e := range employees {
		resp := mapEmployeeToResponse(e)
		responses[i] = &resp
	}
	return responses, total, nil
}

// Update applies partial updates to an employee record.
func (s *EmployeeService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateEmployeeRequest) (*dto.EmployeeResponse, error) {
	employee, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply only provided fields.
	if req.FirstName != nil {
		employee.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		employee.LastName = *req.LastName
	}
	if req.Role != nil {
		employee.Role = domain.EmployeeRole(*req.Role)
	}
	if req.IsActive != nil {
		employee.IsActive = *req.IsActive
	}
	if req.DepartmentID != nil {
		uid, err := uuid.Parse(*req.DepartmentID)
		if err != nil {
			return nil, apperror.NewBadRequest("invalid department_id format")
		}
		employee.DepartmentID = &uid
	}

	if err := s.repo.Update(ctx, employee); err != nil {
		return nil, err
	}

	// Purge stale cache entries.
	s.invalidateEmployeeCache(ctx, id)
	s.invalidateListCache(ctx)

	resp := mapEmployeeToResponse(employee)
	return &resp, nil
}

// Delete removes an employee and purges related cache entries.
func (s *EmployeeService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidateEmployeeCache(ctx, id)
	s.invalidateListCache(ctx)
	return nil
}

// UploadAvatar stores an avatar in Cloudinary and updates the employee record.
// Falls back to a no-op when Cloudinary is not configured.
func (s *EmployeeService) UploadAvatar(ctx context.Context, id uuid.UUID, file multipart.File) (*dto.AvatarUploadResponse, error) {
	// Ensure the employee exists before uploading.
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return nil, err
	}

	var avatarURL string

	if s.cloudinary != nil {
		result, err := s.cloudinary.UploadAvatar(ctx, file, id.String())
		if err != nil {
			logger.S().Errorw("cloudinary upload failed", "employee_id", id, "error", err)
			return nil, apperror.NewInternal("avatar upload failed; please try again")
		}
		avatarURL = result.URL
	} else {
		// Local development fallback.
		avatarURL = fmt.Sprintf("/uploads/avatars/%s.jpg", id.String())
		logger.S().Warnw("Cloudinary not configured; using local fallback URL", "url", avatarURL)
	}

	if err := s.repo.UpdateAvatar(ctx, id, avatarURL); err != nil {
		return nil, err
	}

	s.invalidateEmployeeCache(ctx, id)

	return &dto.AvatarUploadResponse{AvatarURL: avatarURL}, nil
}

// ─── Cache helpers ────────────────────────────────────────────────────────────

func (s *EmployeeService) invalidateEmployeeCache(ctx context.Context, id uuid.UUID) {
	key := fmt.Sprintf(employeeCacheKeyFmt, id.String())
	if err := database.CacheDel(ctx, s.redis, key); err != nil {
		logger.S().Warnw("failed to invalidate employee cache", "key", key, "error", err)
	}
}

func (s *EmployeeService) invalidateListCache(ctx context.Context) {
	if err := database.CacheDelPattern(ctx, s.redis, employeeListCacheKey); err != nil {
		logger.S().Warnw("failed to invalidate employee list cache", "error", err)
	}
}
