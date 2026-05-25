// Package middleware contains Gin middleware functions.
// Middleware is responsible for cross-cutting concerns: authentication,
// exception handling, request logging, and CORS.
// Business logic never lives here.
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"simple-employees-crud/pkg/apperror"
	jwtpkg "simple-employees-crud/pkg/jwt"
	"simple-employees-crud/pkg/response"
)

// Context keys for values stored by Auth middleware.
const (
	// CtxEmployeeID is the key under which the authenticated employee's UUID is stored.
	CtxEmployeeID = "employee_id"
	// CtxEmployeeEmail is the key for the authenticated employee's email.
	CtxEmployeeEmail = "employee_email"
	// CtxEmployeeRole is the key for the authenticated employee's role string.
	CtxEmployeeRole = "employee_role"
)

// Auth validates the Authorization: Bearer <token> header, parses the JWT,
// and injects the employee's ID, email, and role into the Gin context.
// If the token is missing or invalid, the request is aborted with 401.
func Auth(jwtManager *jwtpkg.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Err(c, apperror.NewUnauthorized("Authorization header is required"))
			c.Abort()
			return
		}

		// Expect "Bearer <token>".
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			response.Err(c, apperror.NewUnauthorized("Authorization header format must be: Bearer <token>"))
			c.Abort()
			return
		}

		claims, err := jwtManager.Validate(parts[1])
		if err != nil {
			response.Err(c, apperror.NewUnauthorized("token is invalid or expired"))
			c.Abort()
			return
		}

		// Inject claims into the Gin context for downstream handlers.
		c.Set(CtxEmployeeID, claims.EmployeeID.String())
		c.Set(CtxEmployeeEmail, claims.Email)
		c.Set(CtxEmployeeRole, claims.Role)

		c.Next()
	}
}

// RequireRole returns middleware that aborts with 403 if the authenticated
// employee's role is not one of the allowed roles.
// Auth middleware must run before RequireRole.
func RequireRole(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}

	return func(c *gin.Context) {
		role, exists := c.Get(CtxEmployeeRole)
		if !exists {
			response.Err(c, apperror.NewUnauthorized("not authenticated"))
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			response.Err(c, apperror.NewInternal("malformed role claim"))
			c.Abort()
			return
		}

		if _, ok := allowedSet[roleStr]; !ok {
			response.Err(c, apperror.NewForbidden("you do not have permission to perform this action"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// ─── Context accessors ────────────────────────────────────────────────────────

// GetEmployeeID extracts and parses the authenticated employee's UUID from ctx.
// Returns an error if the middleware has not run or the value is malformed.
func GetEmployeeID(c *gin.Context) (uuid.UUID, error) {
	raw, exists := c.Get(CtxEmployeeID)
	if !exists {
		return uuid.Nil, apperror.NewUnauthorized("not authenticated")
	}
	id, err := uuid.Parse(raw.(string))
	if err != nil {
		return uuid.Nil, apperror.NewInternal("malformed employee ID in context")
	}
	return id, nil
}

// GetEmployeeRole extracts the authenticated employee's role string from ctx.
func GetEmployeeRole(c *gin.Context) string {
	role, _ := c.Get(CtxEmployeeRole)
	if r, ok := role.(string); ok {
		return r
	}
	return ""
}
