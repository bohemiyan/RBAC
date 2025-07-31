RBAC Framework for Go
The RBAC Framework is a high-performance, Redis-cached Role-Based Access Control (RBAC) system designed for Go applications. Built with GORM and Fiber, it provides robust permission management with roles, permissions, and audit logging, optimized for minimal database calls and efficient caching.
📋 Table of Contents

Features
Installation
Usage
API Overview
Cache Management
Audit Logging
Contributing
License

✨ Features

Role-Based Access Control: Manage permissions through roles assigned to employees.
Redis Caching: Cache-first strategy with automatic invalidation for high performance.
Audit Logging: Track all RBAC operations with configurable logging.
Bulk Operations: Efficiently validate permissions for multiple employees.
Fiber Middleware: Seamless integration for permission-based route protection.
GORM Integration: Supports PostgreSQL, MySQL, and SQLite.
Comprehensive APIs: Manage permissions, roles, and employee assignments with ease.

📦 Installation
Prerequisites

Go 1.19+
Redis Server
GORM-compatible database (PostgreSQL, MySQL, SQLite)
Fiber v2

Install the Module
go get github.com/your-org/rbac

Database Schema
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

🚀 Usage
Initialize the RBAC Service
package main

import (
    "github.com/redis/go-redis/v9"
    "github.com/your-org/rbac"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "time"
)

func main() {
    // Initialize database
    db, _ := gorm.Open(postgres.Open("your-dsn"), &gorm.Config{})
    
    // Initialize Redis
    rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    
    // Configure RBAC
    cfg := rbac.Config{
        DB:                      db,
        RedisClient:             rdb,
        CacheTTL:                30 * time.Minute,
        CachePrefix:             "rbac:",
        AutoMigrate:             true,
        AutoMigrateCompositeKeys: true,
        EnableAuditLogging:      true,
    }
    
    // Create RBAC service
    rbacService, _ := rbac.NewRBACService(cfg)
}

Example: Managing Permissions and Roles
// Add a permission
permID, _ := rbacService.AddNewPermission(ctx, "users.read", "Read user data", 1)

// Add a role with permissions
roleID, _ := rbacService.AddNewRole(ctx, "Admin", "Administrator role", []string{"users.read", "users.write"}, 1)

// Assign role to employee
rbacService.AssignRolesToEmployeeByNames(ctx, 123, []string{"Admin"}, 1)

Example: Permission Validation
// Check single permission
err := rbacService.ValidatePermission(ctx, 123, "users.read")
if err != nil {
    // Handle permission denied
}

// Check any permission (OR logic)
err = rbacService.ValidateAnyPermission(ctx, 123, []string{"users.read", "users.edit"})

// Check all permissions (AND logic)
err = rbacService.ValidateAllPermissions(ctx, 123, []string{"users.read", "users.edit"})

// Check role assignment
err = rbacService.CompareEmpByRoleName(ctx, 123, "Admin")

Example: Fiber Middleware
import "github.com/gofiber/fiber/v2"

func setupRoutes(app *fiber.App, rbacService *rbac.RBACService) {
    api := app.Group("/api/v1")
    
    // Protected route
    api.Get("/users", rbacService.RbacMiddleware("users.read"), func(c *fiber.Ctx) error {
        return c.SendString("User data")
    })
}

Example: Audit Logging
// Retrieve audit logs
logs, _ := rbacService.GetAuditLogs(ctx, nil, nil, nil, nil, 1)
for _, log := range logs {
    fmt.Printf("Action: %s, Resource: %s, Success: %v, Time: %v\n",
        log["action"], log["resource"], log["success"], log["timestamp"])
}

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

Cache Keys:
rbac:permissions:all: All permissions
rbac:roles:all: All roles
rbac:role:<name>:permissions: Permissions for a role
rbac:employee:<id>:roles: Roles for an employee
rbac:employee:<id>:permissions: Permissions for an employee


Invalidation: Automatic on create, update, or delete operations.
TTL: Configurable via Config.CacheTTL.
Stats: Monitor hit rates and memory usage with GetCacheStats and GetCacheMemoryUsage.

📝 Audit Logging

Enabled: Configurable via Config.EnableAuditLogging.
Logged Actions: All API calls (e.g., add/delete permissions, roles, validations).
Querying: Filter by employee ID, action, time range with GetAuditLogs.
Storage: Persisted in the audits table.

🤝 Contributing

Clone: git clone github.com/your-org/rbac
Install: go mod tidy
Test: go test -v ./... or go test -race ./...
Style: Follow Go best practices and use gofmt.

Submit issues and pull requests at GitHub Issues.
📄 License
MIT License. See LICENSE for details.

Built with ❤️ using Go, GORM, Redis, and Fiber