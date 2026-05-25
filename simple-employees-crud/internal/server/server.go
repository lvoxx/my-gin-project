// Package server wires together the Gin engine, middleware stack, and all
// route groups.  It is the only place that knows about URL paths.
package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"simple-employees-crud/internal/config"
	"simple-employees-crud/internal/handler"
	"simple-employees-crud/internal/middleware"
	"simple-employees-crud/pkg/jwt"
	"simple-employees-crud/pkg/logger"
)

// Server wraps the Gin engine and all handler dependencies.
type Server struct {
	engine   *gin.Engine
	cfg      *config.Config
	auth     *handler.AuthHandler
	employee *handler.EmployeeHandler
	dept     *handler.DepartmentHandler
	jwt      *jwt.Manager
}

// New creates a configured Gin engine and registers all routes.
func New(
	cfg *config.Config,
	authHandler *handler.AuthHandler,
	employeeHandler *handler.EmployeeHandler,
	deptHandler *handler.DepartmentHandler,
	jwtManager *jwt.Manager,
) *Server {
	gin.SetMode(cfg.Server.GinMode)

	engine := gin.New() // no default middleware — we register our own below

	s := &Server{
		engine:   engine,
		cfg:      cfg,
		auth:     authHandler,
		employee: employeeHandler,
		dept:     deptHandler,
		jwt:      jwtManager,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware registers the global middleware stack in order.
// Recovery must be first so panics are caught before anything else runs.
func (s *Server) setupMiddleware() {
	s.engine.Use(
		middleware.Recovery(),      // panic → *AppError, abort
		middleware.Exception(),     // *AppError → JSON response
		middleware.RequestLogger(), // structured request/response log
		corsMiddleware(),           // permissive CORS for APIs
	)

	// Apply max upload size globally.
	s.engine.MaxMultipartMemory = s.cfg.App.MaxFileSizeBytes
}

// setupRoutes declares all API routes grouped by domain.
func (s *Server) setupRoutes() {
	// Liveness probe — no auth required.
	s.engine.GET("/health", handler.HealthCheck)

	// Swagger UI — only mounted in non-production environments.
	if s.cfg.IsDevelopment() {
		s.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	api := s.engine.Group("/api/v1")

	// ── Public routes ──────────────────────────────────────────────────────
	authGroup := api.Group("/auth")
	{
		authGroup.POST("/login", s.auth.Login)
	}

	// ── Protected routes — JWT required ────────────────────────────────────
	authRequired := api.Group("")
	authRequired.Use(middleware.Auth(s.jwt))
	{
		authRequired.GET("/auth/me", s.auth.Me)

		// Employee routes
		emp := authRequired.Group("/employees")
		{
			// Any authenticated user can read.
			emp.GET("", s.employee.List)
			emp.GET("/search", s.employee.Search)
			emp.GET("/:id", s.employee.GetByID)

			// Only admin or manager can write.
			empWrite := emp.Group("")
			empWrite.Use(middleware.RequireRole("admin", "manager"))
			{
				empWrite.POST("", s.employee.Create)
				empWrite.PATCH("/:id", s.employee.Update)
				empWrite.POST("/:id/avatar", s.employee.UploadAvatar)
			}

			// Only admin can delete.
			empAdmin := emp.Group("")
			empAdmin.Use(middleware.RequireRole("admin"))
			{
				empAdmin.DELETE("/:id", s.employee.Delete)
			}
		}

		// Department routes
		dept := authRequired.Group("/departments")
		{
			// Any authenticated user can read departments.
			dept.GET("", s.dept.List)
			dept.GET("/:id", s.dept.GetByID)

			// Only admin can manage departments.
			deptAdmin := dept.Group("")
			deptAdmin.Use(middleware.RequireRole("admin"))
			{
				deptAdmin.POST("", s.dept.Create)
				deptAdmin.PATCH("/:id", s.dept.Update)
				deptAdmin.DELETE("/:id", s.dept.Delete)
			}
		}
	}
}

// Start begins serving HTTP requests and blocks until the process receives
// SIGINT or SIGTERM, after which it performs a graceful shutdown.
func (s *Server) Start() error {
	addr := ":" + s.cfg.Server.Port

	srv := &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start listener in background.
	errCh := make(chan error, 1)
	go func() {
		logger.S().Infow("HTTP server starting", "addr", addr, "env", s.cfg.App.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Wait for OS signal or a fatal listener error.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		logger.S().Infow("shutdown signal received", "signal", sig.String())
	}

	// Graceful shutdown: give in-flight requests up to 10 seconds to finish.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.S().Errorw("graceful shutdown failed", "error", err)
		return err
	}

	logger.S().Info("server shut down cleanly")
	return nil
}

// corsMiddleware returns a simple CORS middleware suitable for REST APIs.
// Tighten the AllowOrigins list for production deployments.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Accept")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
