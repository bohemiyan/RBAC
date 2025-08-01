package rbac

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Config holds configuration for initializing the RBAC system.
type Config struct {
	DB      *gorm.DB
	Redis   *redis.Client // Optional; nil disables caching
	AppName string        // For Redis key prefixing
}

// RBAC is the main struct for the RBAC system.
type RBAC struct {
	db      *gorm.DB
	redis   *redis.Client
	appName string
	ctx     context.Context
	cancel  context.CancelFunc
}

// Init initializes the RBAC system with the provided configuration.
func Init(config Config) *RBAC {
	// Create a default context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	rbac := &RBAC{
		db:      config.DB,
		redis:   config.Redis,
		appName: config.AppName,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Ensure PostgreSQL-specific settings (if DB not already initialized)
	if config.DB.Dialector.Name() == "postgres" {
		// GORM automatically handles PostgreSQL schema creation
		// Run migrations for all entities
		err := rbac.db.AutoMigrate(
			&Department{},
			&Role{},
			&Permission{},
			&EmployeeRole{},
			&ScopedPermission{},
			&AuditLog{},
		)
		if err != nil {
			panic("failed to run migrations: " + err.Error())
		}
	}

	return rbac
}

// Close cleans up resources
func (r *RBAC) Close() {
	if r.cancel != nil {
		r.cancel()
	}
}

// SetContext allows setting a custom context
func (r *RBAC) SetContext(ctx context.Context) {
	if r.cancel != nil {
		r.cancel()
	}
	r.ctx = ctx
}

// GetContext returns the current context
func (r *RBAC) GetContext() context.Context {
	return r.ctx
}
