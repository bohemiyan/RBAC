package routes

import (
	"github.com/gofiber/fiber/v2"
)

func Setup(app *fiber.App, rbacService *rbac.RBACService) {
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
