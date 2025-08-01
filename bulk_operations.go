package rbac

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// BulkPermissionResult represents the result of bulk permission checks
type BulkPermissionResult struct {
	EmployeeID uint
	Permission string
	Allowed    bool
	Error      error
}

// BulkEmployeePermission represents employee-permission pairs for bulk operations
type BulkEmployeePermission struct {
	EmployeeID       uint
	Permission       string
	DepartmentID     *uint
	TargetEmployeeID *uint
}

// CheckBulkPermissions checks multiple permissions for multiple employees efficiently
func (r *RBAC) CheckBulkPermissions(checks []BulkEmployeePermission) []BulkPermissionResult {
	results := make([]BulkPermissionResult, len(checks))

	// Use worker pool for concurrent processing
	workerCount := 10
	if len(checks) < workerCount {
		workerCount = len(checks)
	}

	// Create channels for work distribution
	jobs := make(chan int, len(checks))
	resultsChan := make(chan BulkPermissionResult, len(checks))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for jobIndex := range jobs {
				check := checks[jobIndex]
				err := r.CheckPermission(check.EmployeeID, check.Permission, check.DepartmentID, check.TargetEmployeeID)

				resultsChan <- BulkPermissionResult{
					EmployeeID: check.EmployeeID,
					Permission: check.Permission,
					Allowed:    err == nil,
					Error:      err,
				}
			}
		}()
	}

	// Send jobs
	for i := range checks {
		jobs <- i
	}
	close(jobs)

	// Wait for completion
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		for i, check := range checks {
			if check.EmployeeID == result.EmployeeID && check.Permission == result.Permission {
				results[i] = result
				break
			}
		}
	}

	return results
}

// BulkAssignRoles assigns multiple roles to multiple employees efficiently
func (r *RBAC) BulkAssignRoles(assignments map[uint][]uint) error {
	// Use transaction for consistency
	return r.db.Transaction(func(tx *gorm.DB) error {
		for employeeID, roleIDs := range assignments {
			for _, roleID := range roleIDs {
				empRole := &EmployeeRole{
					EmployeeID: employeeID,
					RoleID:     roleID,
				}

				// Use FirstOrCreate to avoid duplicates
				if err := tx.Where("employee_id = ? AND role_id = ?", employeeID, roleID).
					FirstOrCreate(empRole).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// BulkRemoveRoles removes multiple roles from multiple employees efficiently
func (r *RBAC) BulkRemoveRoles(removals map[uint][]uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for employeeID, roleIDs := range removals {
			if err := tx.Where("employee_id = ? AND role_id IN ?", employeeID, roleIDs).
				Delete(&EmployeeRole{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetEmployeePermissionsBulk efficiently retrieves permissions for multiple employees
func (r *RBAC) GetEmployeePermissionsBulk(employeeIDs []uint) map[uint][]string {
	results := make(map[uint][]string)

	// Use a single query to get all employee roles
	var empRoles []EmployeeRole
	if err := r.db.Where("employee_id IN ?", employeeIDs).Find(&empRoles).Error; err != nil {
		return results
	}

	// Group by employee
	empRoleMap := make(map[uint][]uint)
	for _, empRole := range empRoles {
		empRoleMap[empRole.EmployeeID] = append(empRoleMap[empRole.EmployeeID], empRole.RoleID)
	}

	// Get all unique role IDs
	var allRoleIDs []uint
	roleIDSet := make(map[uint]bool)
	for _, roleIDs := range empRoleMap {
		for _, roleID := range roleIDs {
			if !roleIDSet[roleID] {
				roleIDSet[roleID] = true
				allRoleIDs = append(allRoleIDs, roleID)
			}
		}
	}

	// Get all scoped permissions for these roles
	var scopedPerms []ScopedPermission
	if err := r.db.Where("role_id IN ?", allRoleIDs).Find(&scopedPerms).Error; err != nil {
		return results
	}

	// Get permission names
	var permIDs []uint
	permIDSet := make(map[uint]bool)
	for _, sp := range scopedPerms {
		if !permIDSet[sp.PermissionID] {
			permIDSet[sp.PermissionID] = true
			permIDs = append(permIDs, sp.PermissionID)
		}
	}

	var perms []Permission
	if err := r.db.Where("id IN ?", permIDs).Find(&perms).Error; err != nil {
		return results
	}

	// Create permission name map
	permNameMap := make(map[uint]string)
	for _, perm := range perms {
		permNameMap[perm.ID] = perm.Name
	}

	// Build results
	for employeeID, roleIDs := range empRoleMap {
		permSet := make(map[string]bool)
		for _, roleID := range roleIDs {
			for _, sp := range scopedPerms {
				if sp.RoleID == roleID {
					if permName, exists := permNameMap[sp.PermissionID]; exists {
						permSet[permName] = true
					}
				}
			}
		}

		var permissions []string
		for perm := range permSet {
			permissions = append(permissions, perm)
		}
		results[employeeID] = permissions
	}

	return results
}

// CacheBulkPermissions caches permission results for multiple employees
func (r *RBAC) CacheBulkPermissions(permissions map[string][]uint) error {
	if r.redis == nil {
		return nil
	}

	// Use pipeline for better performance
	pipe := r.redis.Pipeline()

	for permName, employeeIDs := range permissions {
		for _, empID := range employeeIDs {
			key := r.getCacheKey(empID, permName, nil, nil)
			// Cache as allowed (you might want to check actual permissions first)
			pipe.Set(r.ctx, key, "true", 30*time.Minute)
		}
	}

	_, err := pipe.Exec(r.ctx)
	return err
}

// InvalidateBulkCache invalidates cache for multiple employees
func (r *RBAC) InvalidateBulkCache(employeeIDs []uint) error {
	if r.redis == nil {
		return nil
	}

	var keys []string
	for _, empID := range employeeIDs {
		pattern := r.appName + ":perm:" + fmt.Sprintf("%d", empID) + ":*"
		empKeys, err := r.redis.Keys(r.ctx, pattern).Result()
		if err != nil {
			continue
		}
		keys = append(keys, empKeys...)
	}

	if len(keys) > 0 {
		return r.redis.Del(r.ctx, keys...).Err()
	}

	return nil
}
