// @title           Employee Management System API
// @version         1.0.0
// @description     Production-grade REST API for managing employees and departments.
// @termsOfService  http://example.com/terms/

// @contact.name   API Support
// @contact.email  support@company.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and the JWT token.

// Package main is the application entry point.
// It wires the dependency graph, initialises infrastructure, and starts the
// HTTP server.  All construction happens here so every other package is free
// of global state and is trivially testable with constructor injection.
package main

import (
	"context"
	"os"
	"time"

	"simple-employees-crud/internal/config"
	"simple-employees-crud/internal/handler"
	"simple-employees-crud/internal/repository"
	"simple-employees-crud/internal/server"
	"simple-employees-crud/internal/service"
	"simple-employees-crud/pkg/database"
	jwtpkg "simple-employees-crud/pkg/jwt"
	"simple-employees-crud/pkg/logger"
	"simple-employees-crud/pkg/storage"
)

func main() {
	// ── 1. Load configuration ──────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		// Logger isn't initialised yet; write directly to stderr.
		os.Stderr.WriteString("FATAL config load failed: " + err.Error() + "\n")
		os.Exit(1)
	}

	// ── 2. Initialise logger (must happen before any other package logs) ───
	logger.Init(logger.Options{
		LogDir:      cfg.Log.Dir,
		MaxSizeMB:   cfg.Log.MaxSizeMB,
		MaxBackups:  cfg.Log.MaxBackups,
		MaxAgeDays:  cfg.Log.MaxAgeDays,
		Development: cfg.IsDevelopment(),
	})
	defer logger.Sync()

	logger.S().Infow("starting Employee Management System",
		"env", cfg.App.Env,
		"port", cfg.Server.Port,
	)

	// ── 3. Connect to infrastructure ──────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// PostgreSQL
	pool, err := database.NewPool(ctx, database.PoolConfig{
		DSN:             cfg.Database.DSN,
		MaxConns:        cfg.Database.MaxConns,
		MinConns:        cfg.Database.MinConns,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
	})
	if err != nil {
		logger.S().Fatalw("PostgreSQL connection failed", "error", err)
	}
	defer pool.Close()

	// Redis
	redisClient, err := database.NewRedisClient(ctx, database.RedisConfig{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		logger.S().Fatalw("Redis connection failed", "error", err)
	}
	defer redisClient.Close()

	// Cloudinary (optional; nil in local dev when credentials are absent)
	var cloudinaryClient *storage.CloudinaryClient
	if cfg.Cloudinary.CloudName != "" && cfg.Cloudinary.CloudName != "REPLACE-ME" {
		cloudinaryClient, err = storage.NewCloudinaryClient(storage.CloudinaryConfig{
			CloudName: cfg.Cloudinary.CloudName,
			APIKey:    cfg.Cloudinary.APIKey,
			APISecret: cfg.Cloudinary.APISecret,
			Folder:    cfg.Cloudinary.Folder,
		})
		if err != nil {
			logger.S().Warnw("Cloudinary init failed — avatar upload disabled", "error", err)
		}
	} else {
		logger.S().Warn("Cloudinary not configured — avatar uploads will use local fallback")
	}

	// ── 4. JWT manager ─────────────────────────────────────────────────────
	jwtManager := jwtpkg.NewManager(cfg.JWT.Secret)

	// ── 5. Repositories ────────────────────────────────────────────────────
	employeeRepo := repository.NewEmployeeRepository(pool)
	deptRepo := repository.NewDepartmentRepository(pool)

	// ── 6. Services ────────────────────────────────────────────────────────
	authService := service.NewAuthService(
		employeeRepo,
		jwtManager,
		cfg.JWT.AccessTokenExp,
	)

	employeeService := service.NewEmployeeService(
		employeeRepo,
		redisClient,
		cloudinaryClient,
		cfg.Redis.CacheTTL,
	)

	deptService := service.NewDepartmentService(
		deptRepo,
		employeeRepo,
		redisClient,
		cfg.Redis.CacheTTL,
	)

	// ── 7. Handlers ────────────────────────────────────────────────────────
	authHandler := handler.NewAuthHandler(authService)
	employeeHandler := handler.NewEmployeeHandler(employeeService)
	deptHandler := handler.NewDepartmentHandler(deptService)

	// ── 8. HTTP server ─────────────────────────────────────────────────────
	srv := server.New(cfg, authHandler, employeeHandler, deptHandler, jwtManager)

	if err := srv.Start(); err != nil {
		logger.S().Fatalw("server exited with error", "error", err)
	}
}
