package rbac

// CheckPermission verifies if an employee has a specific permission.
func (r *RBAC) CheckPermission(empID uint, permName string, deptID, targetEmpID *uint) error {
	if empID == 0 || permName == "" {
		return ErrInvalidInput
	}

	// Check cache
	if allowed, err := r.checkCache(empID, permName, deptID, targetEmpID); err == nil && allowed {
		return nil
	}

	// Get employee roles
	var empRoles []EmployeeRole
	if err := r.db.Where("employee_id = ?", empID).Find(&empRoles).Error; err != nil {
		return err
	}

	// Get permission
	var perm Permission
	if err := r.db.Where("name = ?", permName).First(&perm).Error; err != nil {
		return ErrNotFound
	}

	// Check permissions for each role and its parents
	for _, empRole := range empRoles {
		if r.checkRolePermission(empRole.RoleID, perm.ID, deptID, targetEmpID) {
			r.setCache(empID, permName, deptID, targetEmpID, true)
			return nil
		}
	}

	r.setCache(empID, permName, deptID, targetEmpID, false)
	return ErrPermissionDenied
}

// checkRolePermission checks if a role or its parents have the permission.
func (r *RBAC) checkRolePermission(roleID, permID uint, deptID, targetEmpID *uint) bool {
	var role Role
	if err := r.db.First(&role, roleID).Error; err != nil {
		return false
	}

	// Check if role is global
	if role.IsGlobal {
		var count int64
		r.db.Model(&ScopedPermission{}).
			Where("role_id = ? AND permission_id = ?", roleID, permID).
			Count(&count)
		if count > 0 {
			return true
		}
	}

	// Check direct scoped permission
	var count int64
	query := r.db.Model(&ScopedPermission{}).
		Where("role_id = ? AND permission_id = ?", roleID, permID)
	if deptID != nil {
		query = query.Where("department_id = ? OR department_id IS NULL", *deptID)
	}
	if targetEmpID != nil {
		query = query.Where("employee_id = ? OR employee_id IS NULL", *targetEmpID)
	}
	query.Count(&count)
	if count > 0 {
		return true
	}

	// Check parent roles recursively
	if role.ParentRoleID != nil {
		return r.checkRolePermission(*role.ParentRoleID, permID, deptID, targetEmpID)
	}

	return false
}
