package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"simple-employees-crud/internal/domain"
	"simple-employees-crud/internal/dto"
	"simple-employees-crud/internal/repository"
	"simple-employees-crud/pkg/apperror"
	"simple-employees-crud/pkg/database"
	"simple-employees-crud/pkg/logger"
)

const (
	deptCacheKeyFmt  = "department:%s"
	deptListCacheKey = "department:list:*"
)

// DepartmentService handles all department business logic.
type DepartmentService struct {
	repo         repository.DepartmentRepository
	employeeRepo repository.EmployeeRepository
	redis        *redis.Client
	cacheTTL     time.Duration
}

// NewDepartmentService constructs a DepartmentService.
func NewDepartmentService(
	repo repository.DepartmentRepository,
	employeeRepo repository.EmployeeRepository,
	redisClient *redis.Client,
	cacheTTL time.Duration,
) *DepartmentService {
	return &DepartmentService{
		repo:         repo,
		employeeRepo: employeeRepo,
		redis:        redisClient,
		cacheTTL:     cacheTTL,
	}
}

// Create validates uniqueness and persists a new department.
func (s *DepartmentService) Create(ctx context.Context, req dto.CreateDepartmentRequest) (*dto.DepartmentResponse, error) {
	exists, err := s.repo.ExistsByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflict("a department with this name already exists")
	}

	dept := &domain.Department{
		Name:        req.Name,
		Description: req.Description,
	}

	if req.ManagerID != nil {
		uid, err := parseUUID(*req.ManagerID)
		if err != nil {
			return nil, apperror.NewBadRequest("invalid manager_id format")
		}
		// Verify the manager employee exists.
		if _, err := s.employeeRepo.FindByID(ctx, uid); err != nil {
			if apperror.IsNotFound(err) {
				return nil, apperror.NewBadRequest("the specified manager employee does not exist")
			}
			return nil, err
		}
		dept.ManagerID = &uid
	}

	if err := s.repo.Create(ctx, dept); err != nil {
		return nil, err
	}

	s.invalidateListCache(ctx)

	resp := mapDeptToResponse(dept)
	return &resp, nil
}

// GetByID returns a single department, using Redis as a read-through cache.
func (s *DepartmentService) GetByID(ctx context.Context, id uuid.UUID) (*dto.DepartmentResponse, error) {
	cacheKey := fmt.Sprintf(deptCacheKeyFmt, id.String())

	var cached dto.DepartmentResponse
	if err := database.CacheGet(ctx, s.redis, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	dept, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := mapDeptToResponse(dept)
	if err := database.CacheSet(ctx, s.redis, cacheKey, resp, s.cacheTTL); err != nil {
		logger.S().Warnw("failed to cache department", "id", id, "error", err)
	}

	return &resp, nil
}

// List returns a paged, optionally-filtered department list.
func (s *DepartmentService) List(ctx context.Context, req dto.DepartmentListFilter) ([]*dto.DepartmentResponse, int64, error) {
	req.Normalise()

	filter := domain.DepartmentFilter{
		Page:      req.Page,
		Limit:     req.Limit,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	if req.ManagerID != nil {
		uid, err := parseUUID(*req.ManagerID)
		if err != nil {
			return nil, 0, apperror.NewBadRequest("invalid manager_id format")
		}
		filter.ManagerID = &uid
	}

	depts, total, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.DepartmentResponse, len(depts))
	for i, d := range depts {
		resp := mapDeptToResponse(d)
		responses[i] = &resp
	}
	return responses, total, nil
}

// Update applies partial updates to a department record.
func (s *DepartmentService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateDepartmentRequest) (*dto.DepartmentResponse, error) {
	dept, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		// Check name uniqueness only when the name is actually changing.
		if *req.Name != dept.Name {
			exists, err := s.repo.ExistsByName(ctx, *req.Name)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, apperror.NewConflict("a department with this name already exists")
			}
		}
		dept.Name = *req.Name
	}

	if req.Description != nil {
		dept.Description = *req.Description
	}

	if req.ManagerID != nil {
		uid, err := parseUUID(*req.ManagerID)
		if err != nil {
			return nil, apperror.NewBadRequest("invalid manager_id format")
		}
		// Verify manager exists.
		if _, err := s.employeeRepo.FindByID(ctx, uid); err != nil {
			if apperror.IsNotFound(err) {
				return nil, apperror.NewBadRequest("the specified manager employee does not exist")
			}
			return nil, err
		}
		dept.ManagerID = &uid
	}

	if err := s.repo.Update(ctx, dept); err != nil {
		return nil, err
	}

	s.invalidateDeptCache(ctx, id)
	s.invalidateListCache(ctx)

	resp := mapDeptToResponse(dept)
	return &resp, nil
}

// Delete removes a department. Employees within the department are un-assigned
// by the ON DELETE SET NULL FK constraint in PostgreSQL.
func (s *DepartmentService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidateDeptCache(ctx, id)
	s.invalidateListCache(ctx)
	return nil
}

// ─── Mapping & cache helpers ──────────────────────────────────────────────────

func mapDeptToResponse(d *domain.Department) dto.DepartmentResponse {
	return dto.DepartmentResponse{
		ID:          d.ID,
		Name:        d.Name,
		Description: d.Description,
		ManagerID:   d.ManagerID,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

func (s *DepartmentService) invalidateDeptCache(ctx context.Context, id uuid.UUID) {
	key := fmt.Sprintf(deptCacheKeyFmt, id.String())
	if err := database.CacheDel(ctx, s.redis, key); err != nil {
		logger.S().Warnw("failed to invalidate dept cache", "key", key, "error", err)
	}
}

func (s *DepartmentService) invalidateListCache(ctx context.Context) {
	if err := database.CacheDelPattern(ctx, s.redis, deptListCacheKey); err != nil {
		logger.S().Warnw("failed to invalidate dept list cache", "error", err)
	}
}
