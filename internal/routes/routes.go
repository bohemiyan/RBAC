package routes

import (
	"log"

	"github.com/bohemiyan/RBAC/internal/rbac"
	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App, cnf *rbac.Config) {

	// Initialize RBAC service
	rbacService, err := rbac.NewRBACService(*cnf)
	if err != nil {
		log.Fatalf("Failed to initialize RBAC service: %v", err)
	}
	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("employee_id", 123)
		return c.Next()
	})

	api := app.Group("/api/v1")
	api.Get("/users", rbacService.RbacMiddleware("users.read"), func(c *fiber.Ctx) error {
		return c.SendString("User data")
	})
}
