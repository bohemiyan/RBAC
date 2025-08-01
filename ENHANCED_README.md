# Enhanced RBAC System for Go

A high-performance, Redis-cached Role-Based Access Control (RBAC) system for Go applications with PostgreSQL backend and enhanced features.

## 🚀 Key Features

### ✅ **Simple Integration**
- Just pass your existing DB and Redis instances
- No new connections required
- Default context handling (no need to pass context everywhere)

### ✅ **Enhanced Caching**
- Multi-level caching (local + Redis)
- Automatic cache invalidation
- Cache warming capabilities
- Performance monitoring

### ✅ **Bulk Operations**
- Bulk permission checks with worker pools
- Bulk role assignments
- Bulk cache operations
- Optimized for high-throughput scenarios

### ✅ **Simple API**
- Easy-to-use wrapper methods
- No complex parameters
- Intuitive method names

### ✅ **Performance Optimizations**
- Concurrent processing
- Database query optimization
- Redis pipelining
- Connection pooling

## 📦 Installation

```bash
go get github.com/bohemiyan/RBAC
```

## 🚀 Quick Start

### 1. Basic Setup

```go
package main

import (
    "github.com/bohemiyan/RBAC/akarbac"
    "github.com/redis/go-redis/v9"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    // Your existing database connection
    db, _ := gorm.Open(postgres.Open("your-dsn"), &gorm.Config{})
    
    // Your existing Redis connection
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    // Initialize RBAC with your connections
    config := akarbac.Config{
        DB:      db,
        Redis:   redisClient,
        AppName: "myapp",
    }
    
    rbac := akarbac.Init(config)
    defer rbac.Close()
    
    // Use simple API for easy operations
    api := akarbac.NewSimpleAPI(rbac)
    
    // Ready to use!
}
```

### 2. Basic Usage

```go
// Create permissions
api.CreatePermission("users.read")
api.CreatePermission("users.write")

// Create roles
api.CreateRole("Manager", departmentID)

// Grant permissions to roles
api.GrantPermission("Manager", "users.read")
api.GrantPermission("Manager", "users.write")

// Assign roles to employees
api.AssignRole(employeeID, "Manager")

// Check permissions (no context needed!)
if api.HasPermission(employeeID, "users.read") {
    // Allow access
}
```

## 📚 API Reference

### Simple API Methods

#### Permission Checks
```go
// Basic permission check
api.HasPermission(employeeID, "users.read")

// Department-specific permission
api.HasPermissionInDepartment(employeeID, "users.read", departmentID)

// Employee-specific access
api.CanAccessEmployee(employeeID, "users.read", targetEmployeeID)
```

#### Role Management
```go
// Assign role to employee
api.AssignRole(employeeID, "Manager")

// Remove role from employee
api.RemoveRole(employeeID, "Manager")

// Get employee roles
roles := api.GetEmployeeRoles(employeeID)
```

#### Permission Management
```go
// Create permission
api.CreatePermission("users.read")

// Grant permission to role
api.GrantPermission("Manager", "users.read")

// Revoke permission from role
api.RevokePermission("Manager", "users.read")

// Get employee permissions
permissions := api.GetEmployeePermissions(employeeID)
```

#### Cache Management
```go
// Get cache statistics
stats := api.GetCacheStats()

// Clear all cache
api.ClearCache()

// Warm cache with frequently accessed data
api.WarmCache()
```

### Advanced API Methods

#### Bulk Operations
```go
// Bulk permission checks
checks := []akarbac.BulkEmployeePermission{
    {EmployeeID: 101, Permission: "users.read"},
    {EmployeeID: 102, Permission: "users.write"},
}
results := rbac.CheckBulkPermissions(checks)

// Bulk role assignments
assignments := map[uint][]uint{
    101: {1, 2}, // Employee 101 gets roles 1 and 2
    102: {1},    // Employee 102 gets role 1
}
rbac.BulkAssignRoles(assignments)

// Get permissions for multiple employees
employeeIDs := []uint{101, 102, 103}
permissions := rbac.GetEmployeePermissionsBulk(employeeIDs)
```

#### Cache Operations
```go
// Cache bulk permissions
permissions := map[string][]uint{
    "users.read": {101, 102, 103},
    "users.write": {101, 102},
}
rbac.CacheBulkPermissions(permissions)

// Invalidate cache for multiple employees
employeeIDs := []uint{101, 102, 103}
rbac.InvalidateBulkCache(employeeIDs)
```

## 🏗️ Database Schema

The system automatically creates these tables:

```sql
-- Departments
CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Roles
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    department_id INTEGER NOT NULL,
    parent_role_id INTEGER,
    is_global BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Permissions
CREATE TABLE permissions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    is_global BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Employee Roles
CREATE TABLE employee_roles (
    employee_id INTEGER NOT NULL,
    role_id INTEGER NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP,
    PRIMARY KEY (employee_id, role_id)
);

-- Scoped Permissions
CREATE TABLE scoped_permissions (
    id SERIAL PRIMARY KEY,
    role_id INTEGER NOT NULL,
    permission_id INTEGER NOT NULL,
    department_id INTEGER,
    employee_id INTEGER,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Audit Logs
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    actor_emp_id INTEGER NOT NULL,
    action VARCHAR(255) NOT NULL,
    target_type VARCHAR(255) NOT NULL,
    target_id INTEGER NOT NULL,
    details TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);
```

## 🔧 Configuration

### Basic Configuration
```go
config := akarbac.Config{
    DB:      yourDB,           // Your GORM DB instance
    Redis:   yourRedis,        // Your Redis client (optional)
    AppName: "myapp",          // App name for cache prefixing
}
```

### Advanced Configuration (Optional)
```go
config := akarbac.Config{
    DB:      yourDB,
    Redis:   yourRedis,
    AppName: "myapp",
    
    // Custom context (optional)
    // Context: yourContext,
    
    // Cache settings (optional)
    // CacheTTL: 30 * time.Minute,
    // CachePrefix: "rbac:",
}
```

## 📊 Performance Features

### 1. **Multi-Level Caching**
- Local in-memory cache for fastest access
- Redis cache for distributed systems
- Automatic cache invalidation

### 2. **Bulk Operations**
- Worker pools for concurrent processing
- Database transactions for consistency
- Optimized queries for multiple records

### 3. **Cache Warming**
- Preload frequently accessed data
- Reduce cold start latency
- Background cache population

### 4. **Performance Monitoring**
- Cache hit/miss statistics
- Query performance metrics
- Memory usage tracking

## 🔒 Security Features

### 1. **Role Hierarchy**
- Parent-child role relationships
- Inherited permissions
- Global vs department-specific roles

### 2. **Scoped Permissions**
- Department-level scoping
- Employee-level scoping
- Flexible permission granularity

### 3. **Audit Logging**
- All operations logged
- Actor tracking
- Change history

## 🚀 Performance Tips

### 1. **Use Bulk Operations**
```go
// Instead of individual checks
for _, empID := range employeeIDs {
    api.HasPermission(empID, "users.read")
}

// Use bulk check
checks := make([]akarbac.BulkEmployeePermission, len(employeeIDs))
for i, empID := range employeeIDs {
    checks[i] = akarbac.BulkEmployeePermission{
        EmployeeID: empID,
        Permission: "users.read",
    }
}
results := rbac.CheckBulkPermissions(checks)
```

### 2. **Warm Cache on Startup**
```go
// Warm cache with frequently accessed data
err := api.WarmCache()
if err != nil {
    log.Printf("Cache warming failed: %v", err)
}
```

### 3. **Monitor Cache Performance**
```go
// Check cache statistics
stats := api.GetCacheStats()
fmt.Printf("Cache hit rate: %.2f%%\n", stats["hit_rate"])
```

## 🔧 Error Handling

The system provides clear error types:

```go
// Check for specific errors
if err := api.AssignRole(empID, "Manager"); err != nil {
    if err == akarbac.ErrNotFound {
        // Role not found
    } else if err == akarbac.ErrInvalidInput {
        // Invalid input
    }
}
```

## 📈 Monitoring and Metrics

### Cache Statistics
```go
stats := api.GetCacheStats()
// Returns:
// - app_name: Application name
// - redis_enabled: Whether Redis is enabled
// - cache_keys_count: Number of cached keys
// - redis_memory: Redis memory usage info
```

### Performance Metrics
```go
// The system automatically tracks:
// - Cache hits and misses
// - Database query performance
// - Operation latency
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details.

## 🆘 Support

For issues and questions:
- Create an issue on GitHub
- Check the example code in `example/main.go`
- Review the API documentation above

---

**Happy coding! 🎉** 