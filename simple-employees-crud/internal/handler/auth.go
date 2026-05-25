// Package handler contains the HTTP handler layer.
// Handlers are thin: they parse/validate DTOs, delegate to a service,
// then write a success response.  All error responses flow through
// the Exception middleware — handlers never call c.JSON with an error body.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simple-employees-crud/internal/dto"
	"simple-employees-crud/internal/middleware"
	"simple-employees-crud/internal/service"
	"simple-employees-crud/pkg/apperror"
	"simple-employees-crud/pkg/response"
)

// AuthHandler exposes authentication endpoints.
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login authenticates an employee and issues a JWT access token.
//
// @Summary      Employee login
// @Description  Authenticates an employee with email and password and returns a signed JWT access token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LoginRequest   true  "Login credentials"
// @Success      200   {object}  response.Envelope{data=dto.TokenResponse}
// @Failure      400   {object}  response.Envelope
// @Failure      401   {object}  response.Envelope
// @Failure      422   {object}  response.Envelope
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.NewValidation(formatBindingError(err)))
		c.Abort()
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response.OK(c, result)
}

// Me returns the authenticated employee's profile from the JWT claims.
//
// @Summary      Get current employee
// @Description  Returns the profile of the currently authenticated employee.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Envelope{data=dto.MeResponse}
// @Failure      401  {object}  response.Envelope
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	employeeID, exists := c.Get(middleware.CtxEmployeeID)
	if !exists {
		_ = c.Error(apperror.NewUnauthorized("not authenticated"))
		c.Abort()
		return
	}

	result, err := h.authService.GetMe(c.Request.Context(), employeeID.(string))
	if err != nil {
		_ = c.Error(err)
		c.Abort()
		return
	}

	response.OK(c, result)
}

// HealthCheck is a simple liveness probe endpoint.
//
// @Summary      Health check
// @Description  Returns 200 OK when the server is running.
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
