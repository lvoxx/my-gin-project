package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simple-employees-crud/internal/dto"
	"simple-employees-crud/internal/service"
	"simple-employees-crud/pkg/apperror"
	"simple-employees-crud/pkg/response"
)

// EmployeeHandler exposes CRUD, search, and avatar-upload endpoints for employees.
type EmployeeHandler struct {
	employeeService *service.EmployeeService
}

// NewEmployeeHandler constructs an EmployeeHandler.
func NewEmployeeHandler(employeeService *service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{employeeService: employeeService}
}

// Create creates a new employee record.
//
// @Summary      Create employee
// @Description  Creates a new employee. Requires admin or manager role.
// @Tags         employees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.CreateEmployeeRequest  true  "Employee data"
// @Success      201   {object}  response.Envelope{data=dto.EmployeeResponse}
// @Failure      400   {object}  response.Envelope
// @Failure      401   {object}  response.Envelope
// @Failure      403   {object}  response.Envelope
// @Failure      409   {object}  response.Envelope
// @Failure      422   {object}  response.Envelope
// @Router       /employees [post]
func (h *EmployeeHandler) Create(c *gin.Context) {
	var req dto.CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.NewValidation(formatBindingError(err)))
		c.Abort()
		return
	}

	result, err := h.employeeService.Create(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response.Created(c, result)
}

// GetByID retrieves a single employee by ID.
//
// @Summary      Get employee by ID
// @Description  Returns a single employee resource.
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Employee UUID"
// @Success      200  {object}  response.Envelope{data=dto.EmployeeResponse}
// @Failure      400  {object}  response.Envelope
// @Failure      401  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /employees/{id} [get]
func (h *EmployeeHandler) GetByID(c *gin.Context) {
	id, err := parsePathUUID(c, "id")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	result, svcErr := h.employeeService.GetByID(c.Request.Context(), id)
	if svcErr != nil {
		_ = c.Error(svcErr)
		c.Abort()
		return
	}

	response.OK(c, result)
}

// List returns a paged, filterable list of employees.
//
// @Summary      List employees
// @Description  Returns a paginated list of employees with optional filters.
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        page           query     int     false  "Page number (default 1)"
// @Param        limit          query     int     false  "Items per page (default 20, max 100)"
// @Param        sort_by        query     string  false  "Sort field"
// @Param        sort_order     query     string  false  "Sort direction: asc or desc"
// @Param        department_id  query     string  false  "Filter by department UUID"
// @Param        role           query     string  false  "Filter by role: admin | manager | employee"
// @Param        is_active      query     bool    false  "Filter by active status"
// @Success      200            {object}  response.Envelope{data=[]dto.EmployeeResponse}
// @Failure      401            {object}  response.Envelope
// @Router       /employees [get]
func (h *EmployeeHandler) List(c *gin.Context) {
	var req dto.EmployeeListFilter
	if err := c.ShouldBindQuery(&req); err != nil {
		_ = c.Error(apperror.NewValidation(formatBindingError(err)))
		c.Abort()
		return
	}
	req.Normalise()

	results, total, svcErr := h.employeeService.List(c.Request.Context(), req)
	if svcErr != nil {
		_ = c.Error(svcErr)
		c.Abort()
		return
	}

	response.OKList(c, results, response.NewPagination(req.Page, req.Limit, total))
}

// Search performs full-text search across employee name and email fields.
//
// @Summary      Search employees
// @Description  Full-text search on employee first name, last name, and email.
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        q              query     string  true   "Search query (min 1 char)"
// @Param        page           query     int     false  "Page number"
// @Param        limit          query     int     false  "Items per page"
// @Param        department_id  query     string  false  "Filter by department UUID"
// @Param        is_active      query     bool    false  "Filter by active status"
// @Success      200            {object}  response.Envelope{data=[]dto.EmployeeResponse}
// @Failure      400            {object}  response.Envelope
// @Failure      401            {object}  response.Envelope
// @Router       /employees/search [get]
func (h *EmployeeHandler) Search(c *gin.Context) {
	var req dto.EmployeeSearchFilter
	if err := c.ShouldBindQuery(&req); err != nil {
		_ = c.Error(apperror.NewValidation(formatBindingError(err)))
		c.Abort()
		return
	}
	req.Normalise()

	results, total, svcErr := h.employeeService.Search(c.Request.Context(), req)
	if svcErr != nil {
		_ = c.Error(svcErr)
		c.Abort()
		return
	}

	response.OKList(c, results, response.NewPagination(req.Page, req.Limit, total))
}

// Update applies partial updates to an employee record.
//
// @Summary      Update employee
// @Description  Partially updates an employee. Requires admin or manager role.
// @Tags         employees
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                    true  "Employee UUID"
// @Param        body  body      dto.UpdateEmployeeRequest true  "Fields to update"
// @Success      200   {object}  response.Envelope{data=dto.EmployeeResponse}
// @Failure      400   {object}  response.Envelope
// @Failure      401   {object}  response.Envelope
// @Failure      403   {object}  response.Envelope
// @Failure      404   {object}  response.Envelope
// @Failure      409   {object}  response.Envelope
// @Router       /employees/{id} [patch]
func (h *EmployeeHandler) Update(c *gin.Context) {
	id, err := parsePathUUID(c, "id")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	var req dto.UpdateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.NewValidation(formatBindingError(err)))
		c.Abort()
		return
	}

	result, svcErr := h.employeeService.Update(c.Request.Context(), id, req)
	if svcErr != nil {
		_ = c.Error(svcErr)
		c.Abort()
		return
	}

	response.OK(c, result)
}

// Delete permanently removes an employee.
//
// @Summary      Delete employee
// @Description  Permanently deletes an employee record. Requires admin role.
// @Tags         employees
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Employee UUID"
// @Success      204  "No Content"
// @Failure      401  {object}  response.Envelope
// @Failure      403  {object}  response.Envelope
// @Failure      404  {object}  response.Envelope
// @Router       /employees/{id} [delete]
func (h *EmployeeHandler) Delete(c *gin.Context) {
	id, err := parsePathUUID(c, "id")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	if svcErr := h.employeeService.Delete(c.Request.Context(), id); svcErr != nil {
		_ = c.Error(svcErr)
		c.Abort()
		return
	}

	response.NoContent(c)
}

// UploadAvatar uploads a new avatar image for an employee.
//
// @Summary      Upload employee avatar
// @Description  Accepts a multipart/form-data image and stores it via Cloudinary. Replaces any existing avatar.
// @Tags         employees
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      string  true  "Employee UUID"
// @Param        avatar  formData  file    true  "Avatar image file (JPEG / PNG, max 5 MB)"
// @Success      200     {object}  response.Envelope{data=dto.AvatarUploadResponse}
// @Failure      400     {object}  response.Envelope
// @Failure      401     {object}  response.Envelope
// @Failure      404     {object}  response.Envelope
// @Router       /employees/{id}/avatar [post]
func (h *EmployeeHandler) UploadAvatar(c *gin.Context) {
	id, err := parsePathUUID(c, "id")
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	fileHeader, err2 := c.FormFile("avatar")
	if err2 != nil {
		_ = c.Error(apperror.NewBadRequest("avatar file is required (field name: avatar)"))
		c.Abort()
		return
	}

	// Validate MIME type.
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		_ = c.Error(apperror.NewBadRequest("avatar must be a JPEG, PNG, or WebP image"))
		c.Abort()
		return
	}

	file, err2 := fileHeader.Open()
	if err2 != nil {
		_ = c.Error(apperror.NewInternal("failed to read uploaded file"))
		c.Abort()
		return
	}
	defer file.Close()

	result, svcErr := h.employeeService.UploadAvatar(c.Request.Context(), id, file)
	if svcErr != nil {
		_ = c.Error(svcErr)
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// ─── Path parameter helpers ───────────────────────────────────────────────────

// parsePathUUID extracts a named path parameter and parses it as a UUID.
func parsePathUUID(c *gin.Context, param string) (uuid.UUID, *apperror.AppError) {
	raw := c.Param(param)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperror.NewBadRequest("invalid " + param + " — must be a valid UUID")
	}
	return id, nil
}
