package rbac

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// getCacheKey generates a Redis cache key for permission checks.
func (r *RBAC) getCacheKey(empID uint, permName string, deptID, targetEmpID *uint) string {
	key := fmt.Sprintf("%s:perm:%d:%s", r.appName, empID, permName)
	if deptID != nil {
		key += fmt.Sprintf(":%d", *deptID)
	}
	if targetEmpID != nil {
		key += fmt.Sprintf(":%d", *targetEmpID)
	}
	return key
}

// checkCache checks if a permission result is cached.
func (r *RBAC) checkCache(empID uint, permName string, deptID, targetEmpID *uint) (bool, error) {
	if r.redis == nil {
		return false, nil
	}

	key := r.getCacheKey(empID, permName, deptID, targetEmpID)
	val, err := r.redis.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "true", nil
}

// setCache caches a permission check result.
func (r *RBAC) setCache(empID uint, permName string, deptID, targetEmpID *uint, allowed bool) error {
	if r.redis == nil {
		return nil
	}

	key := r.getCacheKey(empID, permName, deptID, targetEmpID)
	return r.redis.Set(r.ctx, key, allowed, 24*time.Hour).Err()
}

// invalidateCache invalidates cache entries for an employee or all.
func (r *RBAC) invalidateCache(empID uint) error {
	if r.redis == nil {
		return nil
	}

	pattern := r.appName + ":perm:*"
	if empID != 0 {
		pattern = fmt.Sprintf("%s:perm:%d:*", r.appName, empID)
	}
	keys, err := r.redis.Keys(r.ctx, pattern).Result()
	if err != nil {
		return err
	}
	for _, key := range keys {
		r.redis.Del(r.ctx, key)
	}
	return nil
}

// GetCacheStats returns cache statistics
func (r *RBAC) GetCacheStats() map[string]interface{} {
	stats := map[string]interface{}{
		"app_name":      r.appName,
		"redis_enabled": r.redis != nil,
	}

	if r.redis != nil {
		info, err := r.redis.Info(r.ctx, "memory").Result()
		if err == nil {
			stats["redis_memory"] = info
		}

		// Get cache keys count
		keys, err := r.redis.Keys(r.ctx, r.appName+":*").Result()
		if err == nil {
			stats["cache_keys_count"] = len(keys)
		}
	}

	return stats
}

// ClearAllCache clears all cache entries
func (r *RBAC) ClearAllCache() error {
	if r.redis == nil {
		return nil
	}

	pattern := r.appName + ":*"
	keys, err := r.redis.Keys(r.ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return r.redis.Del(r.ctx, keys...).Err()
	}

	return nil
}

// WarmCache preloads frequently accessed data into cache
func (r *RBAC) WarmCache() error {
	if r.redis == nil {
		return nil
	}

	// Cache all permissions
	var perms []Permission
	if err := r.db.Find(&perms).Error; err != nil {
		return err
	}

	pipe := r.redis.Pipeline()
	for _, perm := range perms {
		key := fmt.Sprintf("%s:permission:%s", r.appName, perm.Name)
		pipe.Set(r.ctx, key, perm.ID, 1*time.Hour)
	}

	_, err := pipe.Exec(r.ctx)
	return err
}
