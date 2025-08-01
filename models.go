package rbac

import (
	"time"

	"gorm.io/gorm"
)

// Department represents a logical group (e.g., Sales, HR).
type Department struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"unique;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Role represents a hierarchical position within a department.
type Role struct {
	ID           uint   `gorm:"primaryKey"`
	Name         string `gorm:"not null"`
	DepartmentID uint   `gorm:"not null;index"`
	ParentRoleID *uint  `gorm:"index"` // For role inheritance
	IsGlobal     bool   `gorm:"default:false"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// Permission represents a named access action.
type Permission struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"unique;not null"`
	IsGlobal  bool   `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// EmployeeRole maps an employee to a role.
type EmployeeRole struct {
	EmployeeID uint `gorm:"primaryKey;autoIncrement:false"`
	RoleID     uint `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// ScopedPermission grants a permission to a role with optional scoping.
type ScopedPermission struct {
	ID           uint  `gorm:"primaryKey"`
	RoleID       uint  `gorm:"index;not null"`
	PermissionID uint  `gorm:"index;not null"`
	DepartmentID *uint `gorm:"index"` // Optional department scope
	EmployeeID   *uint `gorm:"index"` // Optional employee scope
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// AuditLog tracks permission/role-related events.
type AuditLog struct {
	ID         uint   `gorm:"primaryKey"`
	ActorEmpID uint   `gorm:"index;not null"`
	Action     string `gorm:"not null"`
	TargetType string `gorm:"not null"`
	TargetID   uint   `gorm:"index;not null"`
	Details    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}
