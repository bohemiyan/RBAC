package main

import (
	"context"
	"log"
	"time"

	rbac "github.com/bohemiyan/RBAC"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Initialize database
	dsn := "host=localhost user=rbac_user password=rbac_password dbname=rbac_db port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize Redis
	redisDB := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if _, err := redisDB.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Configure RBAC
	rbacConfig := rbac.Config{
		DB:                       db,
		RedisClient:              redisDB,
		CacheTTL:                 30 * time.Minute,
		CachePrefix:              "rbac:",
		AutoMigrate:              true,
		AutoMigrateCompositeKeys: true,
		EnableAuditLogging:       true,
	}

	// Initialize RBAC service
	rbacService, err := rbac.NewRBACService(rbacConfig)
	if err != nil {
		log.Fatalf("Failed to initialize RBAC service: %v", err)
	}

	// Example usage
	_, err = rbacService.AddNewPermission("users.read", "Read user data", 1)
	if err != nil {
		log.Printf("Add permission error: %v", err)
	}

	_, err = rbacService.AddNewRole( "Admin", "Administrator role", []string{"users.read"}, 1)
	if err != nil {
		log.Printf("Add role error: %v", err)
	}

	err = rbacService.AssignRolesToEmployeeByNames(123, []string{"Admin"}, 1)
	if err != nil {
		log.Printf("Assign role error: %v", err)
	}

	// Set up Fiber app
	app := fiber.New()

	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("employee_id", 123)
		return c.Next()
	})

	// Protected route
	app.Get("/users", rbacService.RbacMiddleware("users.read"), func(c *fiber.Ctx) error {
		return c.SendString("User data")
	})

	// Start server
	log.Fatal(app.Listen(":8080"))
}
