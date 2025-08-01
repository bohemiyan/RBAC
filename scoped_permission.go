package rbac

// AddScopedPermission grants a permission to a role with optional scoping.
func (r *RBAC) AddScopedPermission(roleID, permID uint, deptID, targetEmpID *uint) error {
	if roleID == 0 || permID == 0 {
		return ErrInvalidInput
	}

	// Validate role and permission
	var role Role
	if err := r.db.First(&role, roleID).Error; err != nil {
		return ErrNotFound
	}
	var perm Permission
	if err := r.db.First(&perm, permID).Error; err != nil {
		return ErrNotFound
	}

	// Validate department if provided
	if deptID != nil {
		var dept Department
		if err := r.db.First(&dept, *deptID).Error; err != nil {
			return ErrNotFound
		}
	}

	scopedPerm := &ScopedPermission{
		RoleID:       roleID,
		PermissionID: permID,
		DepartmentID: deptID,
		EmployeeID:   targetEmpID,
	}

	if err := r.db.Create(scopedPerm).Error; err != nil {
		return err
	}

	r.invalidateCache(0)
	details := "Granted permission to role"
	if deptID != nil {
		details += " in department"
	}
	if targetEmpID != nil {
		details += " for employee"
	}
	r.logAudit(0, "add_scoped_permission", "scoped_permission", scopedPerm.ID, details)
	return nil
}

// UpdateScopedPermission updates a scoped permission's details.
func (r *RBAC) UpdateScopedPermission(id, roleID, permID uint, deptID, targetEmpID *uint) error {
	if id == 0 || roleID == 0 || permID == 0 {
		return ErrInvalidInput
	}

	var scopedPerm ScopedPermission
	if err := r.db.First(&scopedPerm, id).Error; err != nil {
		return ErrNotFound
	}

	// Validate role and permission
	var role Role
	if err := r.db.First(&role, roleID).Error; err != nil {
		return ErrNotFound
	}
	var perm Permission
	if err := r.db.First(&perm, permID).Error; err != nil {
		return ErrNotFound
	}

	// Validate department if provided
	if deptID != nil {
		var dept Department
		if err := r.db.First(&dept, *deptID).Error; err != nil {
			return ErrNotFound
		}
	}

	scopedPerm.RoleID = roleID
	scopedPerm.PermissionID = permID
	scopedPerm.DepartmentID = deptID
	scopedPerm.EmployeeID = targetEmpID

	if err := r.db.Save(&scopedPerm).Error; err != nil {
		return err
	}

	r.invalidateCache(0)
	details := "Updated scoped permission"
	if deptID != nil {
		details += " in department"
	}
	if targetEmpID != nil {
		details += " for employee"
	}
	r.logAudit(0, "update_scoped_permission", "scoped_permission", scopedPerm.ID, details)
	return nil
}

// GetScopedPermission retrieves a scoped permission by ID.
func (r *RBAC) GetScopedPermission(id uint) (*ScopedPermission, error) {
	if id == 0 {
		return nil, ErrInvalidInput
	}

	var scopedPerm ScopedPermission
	if err := r.db.First(&scopedPerm, id).Error; err != nil {
		return nil, ErrNotFound
	}

	return &scopedPerm, nil
}

// DeleteScopedPermission soft-deletes a scoped permission by ID.
func (r *RBAC) DeleteScopedPermission(id uint) error {
	if id == 0 {
		return ErrInvalidInput
	}

	var scopedPerm ScopedPermission
	if err := r.db.First(&scopedPerm, id).Error; err != nil {
		return ErrNotFound
	}

	if err := r.db.Delete(&scopedPerm).Error; err != nil {
		return err
	}

	r.invalidateCache(0)
	r.logAudit(0, "delete_scoped_permission", "scoped_permission", id, "Deleted scoped permission")
	return nil
}

// ListScopedPermissions retrieves all scoped permissions, optionally filtered by role.
func (r *RBAC) ListScopedPermissions(roleID *uint) ([]ScopedPermission, error) {
	var scopedPerms []ScopedPermission
	query := r.db
	if roleID != nil {
		query = query.Where("role_id = ?", *roleID)
	}
	if err := query.Find(&scopedPerms).Error; err != nil {
		return nil, err
	}
	return scopedPerms, nil
}
