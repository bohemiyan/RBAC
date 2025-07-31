package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/your-org/rbac"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Initialize database
	dsn := "host=localhost user=postgres password=your-password dbname=rbac_db port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize Redis
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Configure RBAC
	cfg := rbac.Config{
		DB:                       db,
		RedisClient:              rdb,
		CacheTTL:                 30 * time.Minute,
		CachePrefix:              "rbac:",
		AutoMigrate:              true,
		AutoMigrateCompositeKeys: true,
		EnableAuditLogging:       true,
	}

	// Initialize RBAC service
	rbacService, err := rbac.NewRBACService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize RBAC service: %v", err)
	}

	// Set up Fiber app
	app := fiber.New()

	// Example route with RBAC middleware
	app.Get("/users", rbacService.RbacMiddleware("users.read"), func(c *fiber.Ctx) error {
		return c.SendString("User data")
	})

	// Start server
	log.Fatal(app.Listen(":8080"))
}
