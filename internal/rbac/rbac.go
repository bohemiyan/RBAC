package rbac

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Config holds the configuration for the RBAC service
type Config struct {
	DB                      *gorm.DB
	RedisClient             *redis.Client
	CacheTTL                time.Duration
	CachePrefix             string
	AutoMigrate             bool
	AutoMigrateCompositeKeys bool
	EnableAuditLogging      bool
}

// RBACService is the main service struct for the RBAC framework
type RBACService struct {
	db           *gorm.DB
	redisClient  *redis.Client
	cacheTTL     time.Duration
	cachePrefix  string
	cacheMu      sync.RWMutex
	cacheStats   map[string]interface{}
	auditEnabled bool
}

// Permission represents a permission entity
type Permission struct {
	ID          int    `gorm:"primaryKey;autoIncrement"`
	Name        string `gorm:"unique;not null"`
	Description string
}

// Role represents a role entity
type Role struct {
	ID          int    `gorm:"primaryKey;autoIncrement"`
	Name        string `gorm:"unique;not null"`
	Description string
}

// RolePermission maps roles to permissions
type RolePermission struct {
	RoleID       int `gorm:"primaryKey"`
	PermissionID int `gorm:"primaryKey"`
}

// EmployeeRole maps employees to roles
type EmployeeRole struct {
	EmployeeID int `gorm:"primaryKey"`
	RoleID     int `gorm:"primaryKey"`
}

// Audit represents an audit log entry
type Audit struct {
	EmployeeID int       `gorm:"index"`
	Action     string
	Resource   string
	Success    bool
	Timestamp  time.Time
}

// NewRBACService initializes a new RBAC service
func NewRBACService(cfg Config) (*RBACService, error) {
	if cfg.DB == nil || cfg.RedisClient == nil {
		return nil, fmt.Errorf("database and redis client are required")
	}

	if cfg.CachePrefix == "" {
		cfg.CachePrefix = "rbac:"
	}

	if cfg.AutoMigrate {
		if cfg.AutoMigrateCompositeKeys {
			err := cfg.DB.AutoMigrate(&Permission{}, &Role{}, &Audit{})
			if err != nil {
				return nil, fmt.Errorf("failed to auto-migrate core tables: %w", err)
			}
			if err := cfg.DB.Exec(`
				CREATE TABLE IF NOT EXISTS role_permissions (
					role_id INTEGER REFERENCES roles(role_id),
					permission_id INTEGER REFERENCES permissions(permission_id),
					PRIMARY KEY (role_id, permission_id)
				)
			`).Error; err != nil {
				return nil, fmt.Errorf("failed to create role_permissions table: %w", err)
			}
			if err := cfg.DB.Exec(`
				CREATE TABLE IF NOT EXISTS employee_roles (
					employee_id INTEGER NOT NULL,
					role_id INTEGER REFERENCES roles(role_id),
					PRIMARY KEY (employee_id, role_id)
				)
			`).Error; err != nil {
				return nil, fmt.Errorf("failed to create employee_roles table: %w", err)
			}
		} else {
			err := cfg.DB.AutoMigrate(&Permission{}, &Role{}, &RolePermission{}, &EmployeeRole{}, &Audit{})
			if err != nil {
				return nil, fmt.Errorf("failed to auto-migrate: %w", err)
			}
		}
	}

	return &RBACService{
		db:           cfg.DB,
		redisClient:  cfg.RedisClient,
		cacheTTL:     cfg.CacheTTL,
		cachePrefix:  cfg.CachePrefix,
		auditEnabled: cfg.EnableAuditLogging,
		cacheStats: map[string]interface{}{
			"permissions_cached":               false,
			"roles_cached":                    false,
			"role_permissions_cached":         false,
			"employee_roles_with_names_cached": false,
			"cache_ttl_minutes":               cfg.CacheTTL.Minutes(),
			"total_cache_keys":                0,
		},
	}, nil
}

// AddNewPermission creates a new permission
func (s *RBACService) AddNewPermission(ctx context.Context, name, description string, employeeID int) (int, error) {
	var existing Permission
	if err := s.db.WithContext(ctx).Where("name = ?", name).First(&existing).Error; err == nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "add_permission", fmt.Sprintf("permission:%s", name), false)
		}
		return 0, fmt.Errorf("permission %s already exists", name)
	}

	perm := Permission{Name: name, Description: description}
	if err := s.db.WithContext(ctx).Create(&perm).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "add_permission", fmt.Sprintf("permission:%s", name), false)
		}
		return 0, fmt.Errorf("failed to create permission: %w", err)
	}

	s.invalidateCache(ctx, "permissions:all")
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "add_permission", fmt.Sprintf("permission:%s", name), true)
	}
	return perm.ID, nil
}

// GetAllPermissionsByRoleName retrieves all permission names for a role
func (s *RBACService) GetAllPermissionsByRoleName(ctx context.Context, roleName string, employeeID int) ([]string, error) {
	cacheKey := fmt.Sprintf("%srole:%s:permissions", s.cachePrefix, roleName)
	var permissions []string

	if err := s.redisClient.Get(ctx, cacheKey).Scan(&permissions); err == nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "get_role_permissions", fmt.Sprintf("role:%s", roleName), true)
		}
		return permissions, nil
	}

	var role Role
	if err := s.db.WithContext(ctx).Where("name = ?", roleName).First(&role).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "get_role_permissions", fmt.Sprintf("role:%s", roleName), false)
		}
		return nil, fmt.Errorf("role name %s not found", roleName)
	}

	var rolePermissions []RolePermission
	if err := s.db.WithContext(ctx).Where("role_id = ?", role.ID).Find(&rolePermissions).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "get_role_permissions", fmt.Sprintf("role:%s", roleName), false)
		}
		return nil, fmt.Errorf("failed to fetch role permissions: %w", err)
	}

	var permIDs []int
	for _, rp := range rolePermissions {
		permIDs = append(permIDs, rp.PermissionID)
	}

	var perms []Permission
	if err := s.db.WithContext(ctx).Where("permission_id IN ?", permIDs).Find(&perms).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "get_role_permissions", fmt.Sprintf("role:%s", roleName), false)
		}
		return nil, fmt.Errorf("failed to fetch permissions: %w", err)
	}

	permissions = make([]string, 0, len(perms))
	for _, perm := range perms {
		permissions = append(permissions, perm.Name)
	}

	s.redisClient.Set(ctx, cacheKey, permissions, s.cacheTTL)
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "get_role_permissions", fmt.Sprintf("role:%s", roleName), true)
	}
	return permissions, nil
}

// AddNewRole creates a new role with permissions
func (s *RBACService) AddNewRole(ctx context.Context, name, description string, permissionNames []string, employeeID int) (int, error) {
	var existing Role
	if err := s.db.WithContext(ctx).Where("name = ?", name).First(&existing).Error; err == nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "add_role", fmt.Sprintf("role:%s", name), false)
		}
		return 0, fmt.Errorf("role %s already exists", name)
	}

	var permissionIDs []int
	for _, permName := range permissionNames {
		var perm Permission
		if err := s.db.WithContext(ctx).Where("name = ?", permName).First(&perm).Error; err != nil {
			if s.auditEnabled {
				s.AuditLog(ctx, employeeID, "add_role", fmt.Sprintf("role:%s", name), false)
			}
			return 0, fmt.Errorf("permission %s not found", permName)
		}
		permissionIDs = append(permissionIDs, perm.ID)
	}

	role := Role{Name: name, Description: description}
	if err := s.db.WithContext(ctx).Create(&role).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "add_role", fmt.Sprintf("role:%s", name), false)
		}
		return 0, fmt.Errorf("failed to create role: %w", err)
	}

	for _, permID := range permissionIDs {
		rp := RolePermission{RoleID: role.ID, PermissionID: permID}
		if err := s.db.WithContext(ctx).Create(&rp).Error; err != nil {
			if s.auditEnabled {
				s.AuditLog(ctx, employeeID, "add_role", fmt.Sprintf("role:%s", name), false)
			}
			return 0, fmt.Errorf("failed to assign permission: %w", err)
		}
	}

	s.invalidateCache(ctx, "roles:all")
	s.invalidateCache(ctx, "role_permissions:all")
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "add_role", fmt.Sprintf("role:%s", name), true)
	}
	return role.ID, nil
}

// ModifyRolePermissions updates the permissions for a role
func (s *RBACService) ModifyRolePermissions(ctx context.Context, roleID int, permissionNames []string, employeeID int) error {
	var role Role
	if err := s.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "modify_role_permissions", fmt.Sprintf("role:%d", roleID), false)
		}
		return fmt.Errorf("role ID %d not found", roleID)
	}

	var permissionIDs []int
	for _, permName := range permissionNames {
		var perm Permission
		if err := s.db.WithContext(ctx).Where("name = ?", permName).First(&perm).Error; err != nil {
			if s.auditEnabled {
				s.AuditLog(ctx, employeeID, "modify_role_permissions", fmt.Sprintf("role:%d", roleID), false)
			}
			return fmt.Errorf("permission %s not found", permName)
		}
		permissionIDs = append(permissionIDs, perm.ID)
	}

	if err := s.db.WithContext(ctx).Where("role_id = ?", roleID).Delete(&RolePermission{}).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "modify_role_permissions", fmt.Sprintf("role:%d", roleID), false)
		}
		return fmt.Errorf("failed to clear existing permissions: %w", err)
	}

	for _, permID := range permissionIDs {
		rp := RolePermission{RoleID: roleID, PermissionID: permID}
		if err := s.db.WithContext(ctx).Create(&rp).Error; err != nil {
			if s.auditEnabled {
				s.AuditLog(ctx, employeeID, "modify_role_permissions", fmt.Sprintf("role:%d", roleID), false)
			}
			return fmt.Errorf("failed to assign new permission: %w", err)
		}
	}

	s.invalidateCache(ctx, "role_permissions:all")
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "modify_role_permissions", fmt.Sprintf("role:%d", roleID), true)
	}
	return nil
}

// GetRoleIDByRoleName retrieves role ID by name
func (s *RBACService) GetRoleIDByRoleName(ctx context.Context, roleName string, employeeID int) (int, error) {
	cacheKey := fmt.Sprintf("%srole:%s:id", s.cachePrefix, roleName)
	var roleID int

	if err := s.redisClient.Get(ctx, cacheKey).Scan(&roleID); err == nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "get_role_id", fmt.Sprintf("role:%s", roleName), true)
		}
		return roleID, nil
	}

	var role Role
	if err := s.db.WithContext(ctx).Where("name = ?", roleName).First(&role).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "get_role_id", fmt.Sprintf("role:%s", roleName), false)
		}
		return 0, fmt.Errorf("role name %s not found", roleName)
	}

	s.redisClient.Set(ctx, cacheKey, role.ID, s.cacheTTL)
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "get_role_id", fmt.Sprintf("role:%s", roleName), true)
	}
	return role.ID, nil
}

// GetRoleNameByRoleID retrieves role name by ID
func (s *RBACService) GetRoleNameByRoleID(ctx context.Context, roleID int, employeeID int) (string, error) {
	cacheKey := fmt.Sprintf("%srole:%d:name", s.cachePrefix, roleID)
	var roleName string

	if err := s.redisClient.Get(ctx, cacheKey).Scan(&roleName); err == nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "get_role_name", fmt.Sprintf("role:%d", roleID), true)
		}
		return roleName, nil
	}

	var role Role
	if err := s.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "get_role_name", fmt.Sprintf("role:%d", roleID), false)
		}
		return "", fmt.Errorf("role ID %d not found", roleID)
	}

	s.redisClient.Set(ctx, cacheKey, role.Name, s.cacheTTL)
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "get_role_name", fmt.Sprintf("role:%d", roleID), true)
	}
	return role.Name, nil
}

// AssignRolesToEmployeeByNames assigns roles to an employee by role names
func (s *RBACService) AssignRolesToEmployeeByNames(ctx context.Context, employeeID int, roleNames []string, actorID int) error {
	var roleIDs []int
	for _, roleName := range roleNames {
		roleID, err := s.GetRoleIDByRoleName(ctx, roleName, actorID)
		if err != nil {
			if s.auditEnabled {
				s.AuditLog(ctx, actorID, "assign_roles_by_names", fmt.Sprintf("employee:%d", employeeID), false)
			}
			return err
		}
		roleIDs = append(roleIDs, roleID)
	}
	err := s.AssignRolesToEmployee(ctx, employeeID, roleIDs, actorID)
	if err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "assign_roles_by_names", fmt.Sprintf("employee:%d", employeeID), false)
		}
		return err
	}
	if s.auditEnabled {
		s.AuditLog(ctx, actorID, "assign_roles_by_names", fmt.Sprintf("employee:%d", employeeID), true)
	}
	return nil
}

// AssignRolesToEmployee assigns roles to an employee by role IDs
func (s *RBACService) AssignRolesToEmployee(ctx context.Context, employeeID int, roleIDs []int, actorID int) error {
	for _, roleID := range roleIDs {
		var role Role
		if err := s.db.WithContext(ctx).First(&role, roleID).Error; err != nil {
			if s.auditEnabled {
				s.AuditLog(ctx, actorID, "assign_roles", fmt.Sprintf("employee:%d", employeeID), false)
			}
			return fmt.Errorf("role ID %d not found", roleID)
		}

		er := EmployeeRole{EmployeeID: employeeID, RoleID: roleID}
		if err := s.db.WithContext(ctx).Create(&er).Error; err != nil {
			if s.auditEnabled {
				s.AuditLog(ctx, actorID, "assign_roles", fmt.Sprintf("employee:%d", employeeID), false)
			}
			return fmt.Errorf("failed to assign role: %w", err)
		}
	}

	s.invalidateCache(ctx, "employee_roles:all")
	if s.auditEnabled {
		s.AuditLog(ctx, actorID, "assign_roles", fmt.Sprintf("employee:%d", employeeID), true)
	}
	return nil
}

// RemoveRolesFromEmployee removes roles from an employee
func (s *RBACService) RemoveRolesFromEmployee(ctx context.Context, employeeID int, roleIDs []int, actorID int) error {
	if err := s.db.WithContext(ctx).Where("employee_id = ? AND role_id IN ?", employeeID, roleIDs).Delete(&EmployeeRole{}).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "remove_roles", fmt.Sprintf("employee:%d", employeeID), false)
		}
		return fmt.Errorf("failed to remove roles: %w", err)
	}

	s.invalidateCache(ctx, "employee_roles:all")
	if s.auditEnabled {
		s.AuditLog(ctx, actorID, "remove_roles", fmt.Sprintf("employee:%d", employeeID), true)
	}
	return nil
}

// GetEmployeeRoles retrieves role IDs for an employee
func (s *RBACService) GetEmployeeRoles(ctx context.Context, employeeID int, actorID int) ([]int, error) {
	cacheKey := fmt.Sprintf("%semployee:%d:roles", s.cachePrefix, employeeID)
	var roleIDs []int

	if err := s.redisClient.Get(ctx, cacheKey).Scan(&roleIDs); err == nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "get_employee_roles", fmt.Sprintf("employee:%d", employeeID), true)
		}
		return roleIDs, nil
	}

	var employeeRoles []EmployeeRole
	if err := s.db.WithContext(ctx).Where("employee_id = ?", employeeID).Find(&employeeRoles).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "get_employee_roles", fmt.Sprintf("employee:%d", employeeID), false)
		}
		return nil, fmt.Errorf("failed to fetch employee roles: %w", err)
	}

	roleIDs = make([]int, 0, len(employeeRoles))
	for _, er := range employeeRoles {
		roleIDs = append(roleIDs, er.RoleID)
	}

	s.redisClient.Set(ctx, cacheKey, roleIDs, s.cacheTTL)
	if s.auditEnabled {
		s.AuditLog(ctx, actorID, "get_employee_roles", fmt.Sprintf("employee:%d", employeeID), true)
	}
	return roleIDs, nil
}

// GetEmployeeRoleNames retrieves role names for an employee
func (s *RBACService) GetEmployeeRoleNames(ctx context.Context, employeeID int, actorID int) ([]string, error) {
	roleIDs, err := s.GetEmployeeRoles(ctx, employeeID, actorID)
	if err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "get_employee_role_names", fmt.Sprintf("employee:%d", employeeID), false)
		}
		return nil, err
	}

	roleNames := make([]string, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		name, err := s.GetRoleNameByRoleID(ctx, roleID, actorID)
		if err != nil {
			if s.auditEnabled {
				s.AuditLog(ctx, actorID, "get_employee_role_names", fmt.Sprintf("employee:%d", employeeID), false)
			}
			return nil, err
		}
		roleNames = append(roleNames, name)
	}
	if s.auditEnabled {
		s.AuditLog(ctx, actorID, "get_employee_role_names", fmt.Sprintf("employee:%d", employeeID), true)
	}
	return roleNames, nil
}

// ValidatePermission checks if an employee has a specific permission
func (s *RBACService) ValidatePermission(ctx context.Context, employeeID int, requiredPermission string) error {
	cacheKey := fmt.Sprintf("%semployee:%d:permissions", s.cachePrefix, employeeID)
	var permissions []string

	if err := s.redisClient.Get(ctx, cacheKey).Scan(&permissions); err == nil {
		for _, perm := range permissions {
			if perm == requiredPermission {
				if s.auditEnabled {
					s.AuditLog(ctx, employeeID, "validate_permission", fmt.Sprintf("permission:%s", requiredPermission), true)
				}
				return nil
			}
		}
	}

	roleIDs, err := s.GetEmployeeRoles(ctx, employeeID, employeeID)
	if err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "validate_permission", fmt.Sprintf("permission:%s", requiredPermission), false)
		}
		return err
	}

	var rolePermissions []RolePermission
	if err := s.db.WithContext(ctx).Where("role_id IN ?", roleIDs).Find(&rolePermissions).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "validate_permission", fmt.Sprintf("permission:%s", requiredPermission), false)
		}
		return fmt.Errorf("failed to fetch role permissions: %w", err)
	}

	var permIDs []int
	for _, rp := range rolePermissions {
		permIDs = append(permIDs, rp.PermissionID)
	}

	var perms []Permission
	if err := s.db.WithContext(ctx).Where("permission_id IN ?", permIDs).Find(&perms).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "validate_permission", fmt.Sprintf("permission:%s", requiredPermission), false)
		}
		return fmt.Errorf("failed to fetch permissions: %w", err)
	}

	permissions = make([]string, 0, len(perms))
	for _, perm := range perms {
		permissions = append(permissions, perm.Name)
		if perm.Name == requiredPermission {
			s.redisClient.Set(ctx, cacheKey, permissions, s.cacheTTL)
			if s.auditEnabled {
				s.AuditLog(ctx, employeeID, "validate_permission", fmt.Sprintf("permission:%s", requiredPermission), true)
			}
			return nil
		}
	}

	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "validate_permission", fmt.Sprintf("permission:%s", requiredPermission), false)
	}
	return fmt.Errorf("permission denied: missing %s", requiredPermission)
}

// ValidateAnyPermission checks if an employee has any of the required permissions
func (s *RBACService) ValidateAnyPermission(ctx context.Context, employeeID int, required []string) error {
	for _, perm := range required {
		if err := s.ValidatePermission(ctx, employeeID, perm); err == nil {
			if s.auditEnabled {
				s.AuditLog(ctx, employeeID, "validate_any_permission", fmt.Sprintf("permissions:%v", required), true)
			}
			return nil
		}
	}
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "validate_any_permission", fmt.Sprintf("permissions:%v", required), false)
	}
	return fmt.Errorf("permission denied: missing all required permissions")
}

// ValidateAllPermissions checks if an employee has all required permissions
func (s *RBACService) ValidateAllPermissions(ctx context.Context, employeeID int, required []string) error {
	for _, perm := range required {
		if err := s.ValidatePermission(ctx, employeeID, perm); err != nil {
			if s.auditEnabled {
				s.AuditLog(ctx, employeeID, "validate_all_permissions", fmt.Sprintf("permissions:%v", required), false)
			}
			return err
		}
	}
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "validate_all_permissions", fmt.Sprintf("permissions:%v", required), true)
	}
	return nil
}

// CompareEmpByRoleName checks if an employee has a specific role
func (s *RBACService) CompareEmpByRoleName(ctx context.Context, employeeID int, roleName string) error {
	roleID, err := s.GetRoleIDByRoleName(ctx, roleName, employeeID)
	if err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "compare_by_role", fmt.Sprintf("role:%s", roleName), false)
		}
		return err
	}

	roleIDs, err := s.GetEmployeeRoles(ctx, employeeID, employeeID)
	if err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "compare_by_role", fmt.Sprintf("role:%s", roleName), false)
		}
		return err
	}

	for _, id := range roleIDs {
		if id == roleID {
			if s.auditEnabled {
				s.AuditLog(ctx, employeeID, "compare_by_role", fmt.Sprintf("role:%s", roleName), true)
			}
			return nil
		}
	}
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "compare_by_role", fmt.Sprintf("role:%s", roleName), false)
	}
	return fmt.Errorf("employee does not have role %s", roleName)
}

// ValidatePermissionsBulk validates permissions for multiple employees
func (s *RBACService) ValidatePermissionsBulk(ctx context.Context, employeeIDs []int, requiredPermission string, actorID int) (map[int]bool, error) {
	results := make(map[int]bool, len(employeeIDs))

	var employeeRoles []EmployeeRole
	if err := s.db.WithContext(ctx).Where("employee_id IN ?", employeeIDs).Find(&employeeRoles).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "validate_permissions_bulk", fmt.Sprintf("permission:%s", requiredPermission), false)
		}
		return nil, fmt.Errorf("failed to fetch employee roles: %w", err)
	}

	roleIDMap := make(map[int][]int)
	for _, er := range employeeRoles {
		roleIDMap[er.EmployeeID] = append(roleIDMap[er.EmployeeID], er.RoleID)
	}

	var roleIDs []int
	for _, rids := range roleIDMap {
		roleIDs = append(roleIDs, rids...)
	}

	var rolePermissions []RolePermission
	if err := s.db.WithContext(ctx).Where("role_id IN ?", roleIDs).Find(&rolePermissions).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "validate_permissions_bulk", fmt.Sprintf("permission:%s", requiredPermission), false)
		}
		return nil, fmt.Errorf("failed to fetch role permissions: %w", err)
	}

	permIDMap := make(map[int][]int)
	for _, rp := range rolePermissions {
		permIDMap[rp.RoleID] = append(permIDMap[rp.RoleID], rp.PermissionID)
	}

	var perm Permission
	if err := s.db.WithContext(ctx).Where("name = ?", requiredPermission).First(&perm).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "validate_permissions_bulk", fmt.Sprintf("permission:%s", requiredPermission), false)
		}
		return nil, fmt.Errorf("permission %s not found", requiredPermission)
	}

	for _, empID := range employeeIDs {
		results[empID] = false
		for _, roleID := range roleIDMap[empID] {
			for _, permID := range permIDMap[roleID] {
				if permID == perm.ID {
					results[empID] = true
					cacheKey := fmt.Sprintf("%semployee:%d:permissions", s.cachePrefix, empID)
					s.redisClient.Set(ctx, cacheKey, []string{requiredPermission}, s.cacheTTL)
					break
				}
			}
		}
	}

	if s.auditEnabled {
		s.AuditLog(ctx, actorID, "validate_permissions_bulk", fmt.Sprintf("permission:%s", requiredPermission), true)
	}
	return results, nil
}

// RefreshAllCaches refreshes all caches
func (s *RBACService) RefreshAllCaches(ctx context.Context, employeeID int) error {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	keys := []string{"permissions:all", "roles:all", "role_permissions:all", "employee_roles:all"}
	for _, key := range keys {
		s.redisClient.Del(ctx, s.cachePrefix+key)
		s.cacheStats[fmt.Sprintf("%s_cached", key)] = false
	}
	s.cacheStats["total_cache_keys"] = 0

	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "refresh_caches", "all_caches", true)
	}
	return nil
}

// GetCacheStats returns cache statistics
func (s *RBACService) GetCacheStats(employeeID int) map[string]interface{} {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	if s.auditEnabled {
		s.AuditLog(context.Background(), employeeID, "get_cache_stats", "cache_stats", true)
	}
	return s.cacheStats
}

// GetCacheMemoryUsage estimates cache memory usage
func (s *RBACService) GetCacheMemoryUsage(ctx context.Context, employeeID int) (map[string]int64, error) {
	keys := []string{
		s.cachePrefix + "permissions:all",
		s.cachePrefix + "roles:all",
		s.cachePrefix + "role_permissions:all",
		s.cachePrefix + "employee_roles:all",
	}

	result := make(map[string]int64)
	for _, key := range keys {
		info, err := s.redisClient.MemoryUsage(ctx, key).Result()
		if err != nil && err != redis.Nil {
			if s.auditEnabled {
				s.AuditLog(ctx, employeeID, "get_cache_memory_usage", fmt.Sprintf("cache:%s", key), false)
			}
			return nil, fmt.Errorf("failed to get memory usage for %s: %w", key, err)
		}
		result[key] = int64(info)
	}

	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "get_cache_memory_usage", "cache_memory", true)
	}
	return result, nil
}

// RbacMiddleware provides Fiber middleware for permission checking
func (s *RBACService) RbacMiddleware(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		employeeID, ok := c.Locals("employee_id").(int)
		if !ok {
			if s.auditEnabled {
				s.AuditLog(c.Context(), 0, "middleware_check", fmt.Sprintf("permission:%s", permission), false)
			}
			return fiber.NewError(fiber.StatusUnauthorized, "employee_id not found in context")
		}
		if err := s.ValidatePermission(c.Context(), employeeID, permission); err != nil {
			if s.auditEnabled {
				s.AuditLog(c.Context(), employeeID, "middleware_check", fmt.Sprintf("permission:%s", permission), false)
			}
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}
		if s.auditEnabled {
			s.AuditLog(c.Context(), employeeID, "middleware_check", fmt.Sprintf("permission:%s", permission), true)
		}
		return c.Next()
	}
}

// GetEmployeePermissions retrieves all permissions for an employee
func (s *RBACService) GetEmployeePermissions(ctx context.Context, employeeID int, actorID int) ([]string, error) {
	cacheKey := fmt.Sprintf("%semployee:%d:permissions", s.cachePrefix, employeeID)
	var permissions []string

	if err := s.redisClient.Get(ctx, cacheKey).Scan(&permissions); err == nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "get_employee_permissions", fmt.Sprintf("employee:%d", employeeID), true)
		}
		return permissions, nil
	}

	roleIDs, err := s.GetEmployeeRoles(ctx, employeeID, actorID)
	if err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "get_employee_permissions", fmt.Sprintf("employee:%d", employeeID), false)
		}
		return nil, err
	}

	var rolePermissions []RolePermission
	if err := s.db.WithContext(ctx).Where("role_id IN ?", roleIDs).Find(&rolePermissions).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "get_employee_permissions", fmt.Sprintf("employee:%d", employeeID), false)
		}
		return nil, fmt.Errorf("failed to fetch role permissions: %w", err)
	}

	var permIDs []int
	for _, rp := range rolePermissions {
		permIDs = append(permIDs, rp.PermissionID)
	}

	var perms []Permission
	if err := s.db.WithContext(ctx).Where("permission_id IN ?", permIDs).Find(&perms).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "get_employee_permissions", fmt.Sprintf("employee:%d", employeeID), false)
		}
		return nil, fmt.Errorf("failed to fetch permissions: %w", err)
	}

	permissions = make([]string, 0, len(perms))
	permSet := make(map[string]struct{})
	for _, perm := range perms {
		if _, exists := permSet[perm.Name]; !exists {
			permSet[perm.Name] = struct{}{}
			permissions = append(permissions, perm.Name)
		}
	}

	s.redisClient.Set(ctx, cacheKey, permissions, s.cacheTTL)
	if s.auditEnabled {
		s.AuditLog(ctx, actorID, "get_employee_permissions", fmt.Sprintf("employee:%d", employeeID), true)
	}
	return permissions, nil
}

// DeletePermission removes a permission and updates related mappings
func (s *RBACService) DeletePermission(ctx context.Context, permissionName string, employeeID int) error {
	var perm Permission
	if err := s.db.WithContext(ctx).Where("name = ?", permissionName).First(&perm).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "delete_permission", fmt.Sprintf("permission:%s", permissionName), false)
		}
		return fmt.Errorf("permission %s not found", permissionName)
	}

	if err := s.db.WithContext(ctx).Where("permission_id = ?", perm.ID).Delete(&RolePermission{}).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "delete_permission", fmt.Sprintf("permission:%s", permissionName), false)
		}
		return fmt.Errorf("failed to delete role-permission mappings: %w", err)
	}

	if err := s.db.WithContext(ctx).Delete(&perm).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "delete_permission", fmt.Sprintf("permission:%s", permissionName), false)
		}
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	s.invalidateCache(ctx, "permissions:all")
	s.invalidateCache(ctx, "role_permissions:all")
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "delete_permission", fmt.Sprintf("permission:%s", permissionName), true)
	}
	return nil
}

// DeleteRole removes a role and updates related mappings
func (s *RBACService) DeleteRole(ctx context.Context, roleName string, employeeID int) error {
	var role Role
	if err := s.db.WithContext(ctx).Where("name = ?", roleName).First(&role).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "delete_role", fmt.Sprintf("role:%s", roleName), false)
		}
		return fmt.Errorf("role %s not found", roleName)
	}

	if err := s.db.WithContext(ctx).Where("role_id = ?", role.ID).Delete(&RolePermission{}).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "delete_role", fmt.Sprintf("role:%s", roleName), false)
		}
		return fmt.Errorf("failed to delete role-permission mappings: %w", err)
	}

	if err := s.db.WithContext(ctx).Where("role_id = ?", role.ID).Delete(&EmployeeRole{}).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "delete_role", fmt.Sprintf("role:%s", roleName), false)
		}
		return fmt.Errorf("failed to delete employee-role mappings: %w", err)
	}

	if err := s.db.WithContext(ctx).Delete(&role).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "delete_role", fmt.Sprintf("role:%s", roleName), false)
		}
		return fmt.Errorf("failed to delete role: %w", err)
	}

	s.invalidateCache(ctx, "roles:all")
	s.invalidateCache(ctx, "role_permissions:all")
	s.invalidateCache(ctx, "employee_roles:all")
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "delete_role", fmt.Sprintf("role:%s", roleName), true)
	}
	return nil
}

// ListAllPermissions retrieves all permissions in the system
func (s *RBACService) ListAllPermissions(ctx context.Context, employeeID int) ([]Permission, error) {
	cacheKey := fmt.Sprintf("%spermissions:all", s.cachePrefix)
	var permissions []Permission

	if err := s.redisClient.Get(ctx, cacheKey).Scan(&permissions); err == nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "list_permissions", "all_permissions", true)
		}
		return permissions, nil
	}

	if err := s.db.WithContext(ctx).Find(&permissions).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "list_permissions", "all_permissions", false)
		}
		return nil, fmt.Errorf("failed to fetch permissions: %w", err)
	}

	s.redisClient.Set(ctx, cacheKey, permissions, s.cacheTTL)
	s.cacheMu.Lock()
	s.cacheStats["permissions_cached"] = true
	s.cacheMu.Unlock()
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "list_permissions", "all_permissions", true)
	}
	return permissions, nil
}

// ListAllRoles retrieves all roles in the system
func (s *RBACService) ListAllRoles(ctx context.Context, employeeID int) ([]Role, error) {
	cacheKey := fmt.Sprintf("%sroles:all", s.cachePrefix)
	var roles []Role

	if err := s.redisClient.Get(ctx, cacheKey).Scan(&roles); err == nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "list_roles", "all_roles", true)
		}
		return roles, nil
	}

	if err := s.db.WithContext(ctx).Find(&roles).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "list_roles", "all_roles", false)
		}
		return nil, fmt.Errorf("failed to fetch roles: %w", err)
	}

	s.redisClient.Set(ctx, cacheKey, roles, s.cacheTTL)
	s.cacheMu.Lock()
	s.cacheStats["roles_cached"] = true
	s.cacheMu.Unlock()
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "list_roles", "all_roles", true)
	}
	return roles, nil
}

// GetEmployeesByRoleName retrieves all employee IDs with a specific role
func (s *RBACService) GetEmployeesByRoleName(ctx context.Context, roleName string, employeeID int) ([]int, error) {
	roleID, err := s.GetRoleIDByRoleName(ctx, roleName, employeeID)
	if err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "get_employees_by_role", fmt.Sprintf("role:%s", roleName), false)
		}
		return nil, err
	}

	cacheKey := fmt.Sprintf("%srole:%s:employees", s.cachePrefix, roleName)
	var employeeIDs []int

	if err := s.redisClient.Get(ctx, cacheKey).Scan(&employeeIDs); err == nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "get_employees_by_role", fmt.Sprintf("role:%s", roleName), true)
		}
		return employeeIDs, nil
	}

	var employeeRoles []EmployeeRole
	if err := s.db.WithContext(ctx).Where("role_id = ?", roleID).Find(&employeeRoles).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, employeeID, "get_employees_by_role", fmt.Sprintf("role:%s", roleName), false)
		}
		return nil, fmt.Errorf("failed to fetch employee roles: %w", err)
	}

	employeeIDs = make([]int, 0, len(employeeRoles))
	for _, er := range employeeRoles {
		employeeIDs = append(employeeIDs, er.EmployeeID)
	}

	s.redisClient.Set(ctx, cacheKey, employeeIDs, s.cacheTTL)
	if s.auditEnabled {
		s.AuditLog(ctx, employeeID, "get_employees_by_role", fmt.Sprintf("role:%s", roleName), true)
	}
	return employeeIDs, nil
}

// AuditLog records an RBAC-related action for auditing
func (s *RBACService) AuditLog(ctx context.Context, employeeID int, action, resource string, success bool) error {
	if !s.auditEnabled {
		return nil
	}

	audit := Audit{
		EmployeeID: employeeID,
		Action:     action,
		Resource:   resource,
		Success:    success,
		Timestamp:  time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(&audit).Error; err != nil {
		return fmt.Errorf("failed to record audit log: %w", err)
	}
	return nil
}

// GetAuditLogs retrieves audit logs with optional filters
func (s *RBACService) GetAuditLogs(ctx context.Context, employeeID *int, action *string, startTime, endTime *time.Time, actorID int) ([]map[string]interface{}, error) {
	query := s.db.WithContext(ctx).Model(&Audit{})
	if employeeID != nil {
		query = query.Where("employee_id = ?", *employeeID)
	}
	if action != nil {
		query = query.Where("action = ?", *action)
	}
	if startTime != nil {
		query = query.Where("timestamp >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("timestamp <= ?", *endTime)
	}

	var audits []Audit
	if err := query.Find(&audits).Error; err != nil {
		if s.auditEnabled {
			s.AuditLog(ctx, actorID, "get_audit_logs", "audit_logs", false)
		}
		return nil, fmt.Errorf("failed to fetch audit logs: %w", err)
	}

	results := make([]map[string]interface{}, len(audits))
	for i, audit := range audits {
		results[i] = map[string]interface{}{
			"employee_id": audit.EmployeeID,
			"action":      audit.Action,
			"resource":    audit.Resource,
			"success":     audit.Success,
			"timestamp":   audit.Timestamp,
		}
	}

	if s.auditEnabled {
		s.AuditLog(ctx, actorID, "get_audit_logs", "audit_logs", true)
	}
	return results, nil
}

// invalidateCache invalidates a specific cache key
func (s *RBACService) invalidateCache(ctx context.Context, key string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.redisClient.Del(ctx, s.cachePrefix+key)
	s.cacheStats[fmt.Sprintf("%s_cached", key)] = false
}