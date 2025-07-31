package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bohemiyan/RBAC/internal/config"
	"github.com/bohemiyan/RBAC/internal/db"
	"github.com/bohemiyan/RBAC/internal/rbac"
	"github.com/bohemiyan/RBAC/internal/routes"
	"github.com/bohemiyan/RBAC/zapLogger"
	"github.com/gofiber/fiber/v2"
)

func main() {
	// Initialize zapLogger
	logFile := zapLogger.Init()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		zapLogger.Log.Fatalf("Failed to load config: %v", err)
	}

	pgDB, err := db.NewPostgresDB(cfg)
	if err != nil {
		zapLogger.Log.Fatalf("Failed to initialize PostgreSQL: %v", err)
	}
	zapLogger.Log.Info("Successfully connected to PostgreSQL database")
	defer pgDB.Close()

	redisDB, err := db.NewRedisClient(cfg)
	if err != nil {
		zapLogger.Log.Fatalf("Failed to initialize Redis: %v", err)
	}
	zapLogger.Log.Info("Successfully connected to Redis")
	defer redisDB.Close()

	// Configure RBAC
	rbacconfig := rbac.Config{
		DB:                       pgDB.GormDB,
		RedisClient:              redisDB,
		CacheTTL:                 30 * time.Minute,
		CachePrefix:              "rbac:",
		AutoMigrate:              true,
		AutoMigrateCompositeKeys: true,
		EnableAuditLogging:       true,
	}

	// Set up Fiber app
	app := fiber.New()

	// Middleware
	app.Use(zapLogger.FiberLoggingMiddleware(logFile))

	// // Set up routes
	routes.Setup(app, &rbacconfig)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.AppPort)
	zapLogger.Log.Infof("Server started on port %d", cfg.AppPort)
	log.Fatal(app.Listen(addr))
}
