RBAC Framework for Go
The RBAC Framework is a high-performance, Redis-cached Role-Based Access Control (RBAC) system for Go applications. Built with GORM and Fiber, it provides robust permission management with roles, permissions, and audit logging, optimized for minimal database calls and efficient caching.
📋 Table of Contents

Features
Installation
Quick Start
API Overview
Cache Management
Audit Logging
Contributing
License

✨ Features

Role-based permission management
Redis caching with automatic invalidation
Audit logging for all operations
Bulk permission validation
Fiber middleware for route protection
GORM integration (PostgreSQL, MySQL, SQLite)
Comprehensive APIs for permission and role management

📦 Installation
Prerequisites

Go 1.19+
Redis Server
GORM-compatible database (PostgreSQL, MySQL, SQLite)
Fiber v2

Install the Module
go get github.com/your-org/rbac

Database Schema
Ensure your database has the following schema (auto-migration is supported):
CREATE TABLE permissions (
    permission_id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE roles (
    role_id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE role_permissions (
    role_id INTEGER REFERENCES roles(role_id),
    permission_id INTEGER REFERENCES permissions(permission_id),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE employee_roles (
    employee_id INTEGER NOT NULL,
    role_id INTEGER REFERENCES roles(role_id),
    PRIMARY KEY (employee_id, role_id)
);

CREATE TABLE audits (
    employee_id INTEGER,
    action VARCHAR(255),
    resource TEXT,
    success BOOLEAN,
    timestamp TIMESTAMP
);

🚀 Quick Start

Set Up DependenciesInitialize your Go project and install dependencies:
go mod init my-project
go get github.com/your-org/rbac github.com/redis/go-redis/v9 gorm.io/gorm gorm.io/driver/postgres github.com/gofiber/fiber/v2


Initialize the RBAC ServiceCreate a main.go file with the following:
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
        DB:                      db,
        RedisClient:             redisDB,
        CacheTTL:                30 * time.Minute,
        CachePrefix:             "rbac:",
        AutoMigrate:             true,
        AutoMigrateCompositeKeys: true,
        EnableAuditLogging:      true,
    }

    // Initialize RBAC service
    rbacService, err := rbac.NewRBACService(rbacConfig)
    if err != nil {
        log.Fatalf("Failed to initialize RBAC service: %v", err)
    }

    // Set up Fiber app
    app := fiber.New()

    // Mock auth middleware (set employee_id)
    app.Use(func(c *fiber.Ctx) error {
        c.Locals("employee_id", 123)
        return c.Next()
    })

    // Example protected route
    app.Get("/users", rbacService.RbacMiddleware("users.read"), func(c *fiber.Ctx) error {
        return c.SendString("User data")
    })

    // Start server
    log.Fatal(app.Listen(":8080"))
}


Run the Application
go run .


Test the RBAC Service

Add permissions: rbacService.AddNewPermission(ctx, "users.read", "Read user data", 1)
Add roles: rbacService.AddNewRole(ctx, "Admin", "Administrator role", []string{"users.read"}, 1)
Assign roles: rbacService.AssignRolesToEmployeeByNames(ctx, 123, []string{"Admin"}, 1)
Validate permissions: rbacService.ValidatePermission(ctx, 123, "users.read")



📚 API Overview
Permission Management

AddNewPermission(ctx, name, description string, employeeID int) (int, error)
DeletePermission(ctx, permissionName string, employeeID int) error
ListAllPermissions(ctx context.Context, employeeID int) ([]Permission, error)
GetAllPermissionsByRoleName(ctx, roleName string, employeeID int) ([]string, error)

Role Management

AddNewRole(ctx, name, description string, permissionNames []string, employeeID int) (int, error)
ModifyRolePermissions(ctx, roleID int, permissionNames []string, employeeID int) error
DeleteRole(ctx, roleName string, employeeID int) error
ListAllRoles(ctx context.Context, employeeID int) ([]Role, error)
GetRoleIDByRoleName(ctx, roleName string, employeeID int) (int, error)
GetRoleNameByRoleID(ctx, roleID int, employeeID int) (string, error)

Employee Role Management

AssignRolesToEmployeeByNames(ctx, employeeID int, roleNames []string, actorID int) error
AssignRolesToEmployee(ctx, employeeID int, roleIDs []int, actorID int) error
RemoveRolesFromEmployee(ctx, employeeID int, roleIDs []int, actorID int) error
GetEmployeeRoles(ctx, employeeID, actorID int) ([]int, error)
GetEmployeeRoleNames(ctx, employeeID, actorID int) ([]string, error)
GetEmployeePermissions(ctx, employeeID, actorID int) ([]string, error)
GetEmployeesByRoleName(ctx, roleName string, employeeID int) ([]int, error)

Permission Validation

ValidatePermission(ctx, employeeID int, requiredPermission string) error
ValidateAnyPermission(ctx, employeeID int, required []string) error
ValidateAllPermissions(ctx, employeeID int, required []string) error
CompareEmpByRoleName(ctx, employeeID int, roleName string) error
ValidatePermissionsBulk(ctx, employeeIDs []int, requiredPermission string, actorID int) (map[int]bool, error)

Cache Management

RefreshAllCaches(ctx context.Context, employeeID int) error
GetCacheStats(employeeID int) map[string]interface{}
GetCacheMemoryUsage(ctx context.Context, employeeID int) (map[string]int64, error)

Audit Logging

AuditLog(ctx context.Context, employeeID int, action, resource string, success bool) error
GetAuditLogs(ctx context.Context, employeeID *int, action *string, startTime, endTime *time.Time, actorID int) ([]map[string]interface{}, error)

Middleware

RbacMiddleware(permission string) fiber.Handler

🗄️ Cache Management

Keys: rbac:permissions:all, rbac:roles:all, rbac:role:<name>:permissions, rbac:employee:<id>:roles, rbac:employee:<id>:permissions
Invalidation: Automatic on create/update/delete
TTL: Configurable via Config.CacheTTL
Monitoring: Use GetCacheStats and GetCacheMemoryUsage

📝 Audit Logging

Enabled via Config.EnableAuditLogging
Logs all API operations (e.g., permission/role creation, validation)
Query logs with GetAuditLogs by employee ID, action, or time range

🤝 Contributing

Clone: git clone github.com/your-org/rbac
Install: go mod tidy
Test: go test -v ./... or go test -race ./...
Submit issues/pull requests: GitHub Issues

📄 License
MIT License. See LICENSE.