package handler

import (
	"github.com/gin-gonic/gin"

	"simple-employees-crud/internal/dto"
	"simple-employees-crud/internal/service"
	"simple-employees-crud/pkg/apperror"
	"simple-employees-crud/pkg/response"
)

// DepartmentHandler exposes CRUD endpoints for departments.
type DepartmentHandler struct {
	deptService *service.DepartmentService
}

// NewDepartmentHandler constructs a DepartmentHandler.
func NewDepartmentHandler(deptService *service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{deptService: deptService}
}

// Create creates a new department.
//
// @Summary      Create department
// @Description  Creates a new organisational department. Requires admin role.
// @Tags         departments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.CreateDepartmentRequest  true  "Department data"
// @Success      201   {object}  response.Envelope{data=dto.DepartmentResponse}
// @Failure      400   {object}  response.Envelope
// @Failure      401   {object}  response.Envelope
// @Failure      403   {object}  response.Envelope
// @Failure      409   {object}  response.Envelope
// @Failure      422   {object}  response.Envelope
// @Router       /departments [post]
func (h *DepartmentHandler) Create(c *gin.Context) {
	var req dto.CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.NewValidation(formatBindingError(err)))
		c.Abort()
		return
	}

	result, err := h.deptService.Create(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response.Created(c, result)
}

// GetByID retrieves a single department by ID.
//
// @Summary      Get department by ID
// @Description  Returns a single department resource.
// @Tags         departments
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Department UUID"
// @Success      200  {object}  response.Envelope{data=dto.DepartmentResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /departments/{id} [get]
func (h *DepartmentHandler) GetByID(c *gin.Context) {
	id, err := parsePathUUID(c, "id")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	result, svcErr := h.deptService.GetByID(c.Request.Context(), id)
	if svcErr != nil {
		_ = c.Error(svcErr)
		c.Abort()
		return
	}

	response.OK(c, result)
}

// List returns a paged list of departments.
//
// @Summary      List departments
// @Description  Returns a paginated list of departments with optional filters.
// @Tags         departments
// @Produce      json
// @Security     BearerAuth
// @Param        page        query     int     false  "Page number (default 1)"
// @Param        limit       query     int     false  "Items per page (default 20, max 100)"
// @Param        sort_by     query     string  false  "Sort field: name | created_at | updated_at"
// @Param        sort_order  query     string  false  "Sort direction: asc or desc"
// @Param        manager_id  query     string  false  "Filter by manager employee UUID"
// @Success      200         {object}  response.Envelope{data=[]dto.DepartmentResponse}
// @Failure      401         {object}  response.Envelope
// @Router       /departments [get]
func (h *DepartmentHandler) List(c *gin.Context) {
	var req dto.DepartmentListFilter
	if err := c.ShouldBindQuery(&req); err != nil {
		_ = c.Error(apperror.NewValidation(formatBindingError(err)))
		c.Abort()
		return
	}
	req.Normalise()

	results, total, svcErr := h.deptService.List(c.Request.Context(), req)
	if svcErr != nil {
		_ = c.Error(svcErr)
		c.Abort()
		return
	}

	response.OKList(c, results, response.NewPagination(req.Page, req.Limit, total))
}

// Update applies partial updates to a department.
//
// @Summary      Update department
// @Description  Partially updates a department. Requires admin role.
// @Tags         departments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                       true  "Department UUID"
// @Param        body  body      dto.UpdateDepartmentRequest  true  "Fields to update"
// @Success      200   {object}  response.Envelope{data=dto.DepartmentResponse}
// @Failure      400   {object}  response.Envelope
// @Failure      401   {object}  response.Envelope
// @Failure      403   {object}  response.Envelope
// @Failure      404   {object}  response.Envelope
// @Failure      409   {object}  response.Envelope
// @Router       /departments/{id} [patch]
func (h *DepartmentHandler) Update(c *gin.Context) {
	id, err := parsePathUUID(c, "id")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	var req dto.UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.NewValidation(formatBindingError(err)))
		c.Abort()
		return
	}

	result, svcErr := h.deptService.Update(c.Request.Context(), id, req)
	if svcErr != nil {
		_ = c.Error(svcErr)
		c.Abort()
		return
	}

	response.OK(c, result)
}

// Delete permanently removes a department.
//
// @Summary      Delete department
// @Description  Permanently deletes a department. Employees are un-assigned (department_id set to NULL). Requires admin role.
// @Tags         departments
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Department UUID"
// @Success      204  "No Content"
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /departments/{id} [delete]
func (h *DepartmentHandler) Delete(c *gin.Context) {
	id, err := parsePathUUID(c, "id")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	if svcErr := h.deptService.Delete(c.Request.Context(), id); svcErr != nil {
		_ = c.Error(svcErr)
		c.Abort()
		return
	}

	response.NoContent(c)
}
