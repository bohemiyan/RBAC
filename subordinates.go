package rbac

// GetSubordinateIDs fetches IDs of employees whose roles are descendants of the caller's roles.
func (r *RBAC) GetSubordinateIDs(empID uint) ([]uint, error) {
	if empID == 0 {
		return nil, ErrInvalidInput
	}

	// Get employee's roles
	var empRoles []EmployeeRole
	if err := r.db.Where("employee_id = ?", empID).Find(&empRoles).Error; err != nil {
		return nil, err
	}

	var subordinateRoleIDs []uint
	for _, empRole := range empRoles {
		roleIDs, err := r.getDescendantRoleIDs(empRole.RoleID)
		if err != nil {
			return nil, err
		}
		subordinateRoleIDs = append(subordinateRoleIDs, roleIDs...)
	}

	// Get employees with these roles
	var empIDs []uint
	if err := r.db.Model(&EmployeeRole{}).
		Where("role_id IN ?", subordinateRoleIDs).
		Distinct("employee_id").
		Pluck("employee_id", &empIDs).Error; err != nil {
		return nil, err
	}

	return empIDs, nil
}

// getDescendantRoleIDs recursively fetches all descendant role IDs.
func (r *RBAC) getDescendantRoleIDs(roleID uint) ([]uint, error) {
	var roleIDs []uint
	roleIDs = append(roleIDs, roleID)

	var roles []Role
	if err := r.db.Where("parent_role_id = ?", roleID).Find(&roles).Error; err != nil {
		return nil, err
	}

	for _, role := range roles {
		childIDs, err := r.getDescendantRoleIDs(role.ID)
		if err != nil {
			return nil, err
		}
		roleIDs = append(roleIDs, childIDs...)
	}

	return roleIDs, nil
}
