package rbac

// CreateRole creates a new role in a department with optional parent role.
func (r *RBAC) CreateRole(name string, deptID uint, parentRoleID *uint, isGlobal bool) (*Role, error) {
	if name == "" || deptID == 0 {
		return nil, ErrInvalidInput
	}

	// Validate department exists
	var dept Department
	if err := r.db.First(&dept, deptID).Error; err != nil {
		return nil, ErrNotFound
	}

	// Validate parent role if provided
	if parentRoleID != nil {
		var parent Role
		if err := r.db.First(&parent, *parentRoleID).Error; err != nil {
			return nil, ErrNotFound
		}
	}

	role := &Role{
		Name:         name,
		DepartmentID: deptID,
		ParentRoleID: parentRoleID,
		IsGlobal:     isGlobal,
	}

	if err := r.db.Create(role).Error; err != nil {
		return nil, err
	}

	r.logAudit(0, "create_role", "role", role.ID, "Created role: "+name)
	return role, nil
}

// UpdateRole updates a role's details.
func (r *RBAC) UpdateRole(id uint, name string, deptID uint, parentRoleID *uint, isGlobal bool) (*Role, error) {
	if id == 0 || name == "" || deptID == 0 {
		return nil, ErrInvalidInput
	}

	var role Role
	if err := r.db.First(&role, id).Error; err != nil {
		return nil, ErrNotFound
	}

	// Validate department
	var dept Department
	if err := r.db.First(&dept, deptID).Error; err != nil {
		return nil, ErrNotFound
	}

	// Validate parent role if provided
	if parentRoleID != nil {
		var parent Role
		if err := r.db.First(&parent, *parentRoleID).Error; err != nil {
			return nil, ErrNotFound
		}
	}

	role.Name = name
	role.DepartmentID = deptID
	role.ParentRoleID = parentRoleID
	role.IsGlobal = isGlobal

	if err := r.db.Save(&role).Error; err != nil {
		return nil, err
	}

	r.invalidateCache(0) // Invalidate cache as role changes affect permissions
	r.logAudit(0, "update_role", "role", role.ID, "Updated role: "+name)
	return &role, nil
}

// GetRole retrieves a role by ID.
func (r *RBAC) GetRole(id uint) (*Role, error) {
	if id == 0 {
		return nil, ErrInvalidInput
	}

	var role Role
	if err := r.db.First(&role, id).Error; err != nil {
		return nil, ErrNotFound
	}

	return &role, nil
}

// DeleteRole soft-deletes a role by ID.
func (r *RBAC) DeleteRole(id uint) error {
	if id == 0 {
		return ErrInvalidInput
	}

	var role Role
	if err := r.db.First(&role, id).Error; err != nil {
		return ErrNotFound
	}

	if err := r.db.Delete(&role).Error; err != nil {
		return err
	}

	r.invalidateCache(0) // Invalidate cache as role deletion affects permissions
	r.logAudit(0, "delete_role", "role", id, "Deleted role")
	return nil
}

// ListRoles retrieves all roles, optionally filtered by department.
func (r *RBAC) ListRoles(deptID *uint) ([]Role, error) {
	var roles []Role
	query := r.db
	if deptID != nil {
		query = query.Where("department_id = ?", *deptID)
	}
	if err := query.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}
