package config

import (
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/your-org/rbac"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	RBAC *rbac.Config
}

func Load() (*Config, error) {
	db, err := gorm.Open(postgres.Open(os.Getenv("DB_DSN")), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR")})

	return &Config{
		RBAC: &rbac.Config{
			DB:                      db,
			RedisClient:             rdb,
			CacheTTL:                30 * time.Minute,
			CachePrefix:             "rbac:",
			AutoMigrate:             true,
			AutoMigrateCompositeKeys: true,
			EnableAuditLogging:      true,
		},
	}, nil
}