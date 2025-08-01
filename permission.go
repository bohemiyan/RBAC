package rbac

// CreatePermission creates a new permission.
func (r *RBAC) CreatePermission(name string, isGlobal bool) (*Permission, error) {
	if name == "" {
		return nil, ErrInvalidInput
	}

	perm := &Permission{Name: name, IsGlobal: isGlobal}
	if err := r.db.Create(perm).Error; err != nil {
		return nil, err
	}

	r.logAudit(0, "create_permission", "permission", perm.ID, "Created permission: "+name)
	return perm, nil
}

// UpdatePermission updates a permission's details.
func (r *RBAC) UpdatePermission(id uint, name string, isGlobal bool) (*Permission, error) {
	if id == 0 || name == "" {
		return nil, ErrInvalidInput
	}

	var perm Permission
	if err := r.db.First(&perm, id).Error; err != nil {
		return nil, ErrNotFound
	}

	perm.Name = name
	perm.IsGlobal = isGlobal
	if err := r.db.Save(&perm).Error; err != nil {
		return nil, err
	}

	r.invalidateCache(0) // Invalidate cache as permission changes affect checks
	r.logAudit(0, "update_permission", "permission", perm.ID, "Updated permission: "+name)
	return &perm, nil
}

// GetPermission retrieves a permission by ID.
func (r *RBAC) GetPermission(id uint) (*Permission, error) {
	if id == 0 {
		return nil, ErrInvalidInput
	}

	var perm Permission
	if err := r.db.First(&perm, id).Error; err != nil {
		return nil, ErrNotFound
	}

	return &perm, nil
}

// DeletePermission soft-deletes a permission by ID.
func (r *RBAC) DeletePermission(id uint) error {
	if id == 0 {
		return ErrInvalidInput
	}

	var perm Permission
	if err := r.db.First(&perm, id).Error; err != nil {
		return ErrNotFound
	}

	if err := r.db.Delete(&perm).Error; err != nil {
		return err
	}

	r.invalidateCache(0) // Invalidate cache as permission deletion affects checks
	r.logAudit(0, "delete_permission", "permission", id, "Deleted permission")
	return nil
}

// ListPermissions retrieves all permissions.
func (r *RBAC) ListPermissions() ([]Permission, error) {
	var perms []Permission
	if err := r.db.Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}
