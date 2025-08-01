package rbac

// AssignRole creates a new employee-role mapping.
func (r *RBAC) AssignRole(empID, roleID uint) error {
	if empID == 0 || roleID == 0 {
		return ErrInvalidInput
	}

	// Validate role exists
	var role Role
	if err := r.db.First(&role, roleID).Error; err != nil {
		return ErrNotFound
	}

	empRole := &EmployeeRole{EmployeeID: empID, RoleID: roleID}
	if err := r.db.Create(empRole).Error; err != nil {
		return err
	}

	r.invalidateCache(empID)
	r.logAudit(empID, "assign_role", "employee_role", roleID, "Assigned role to employee")
	return nil
}

// UpdateEmployeeRole updates an employee-role mapping (reassigns role).
func (r *RBAC) UpdateEmployeeRole(empID, oldRoleID, newRoleID uint) error {
	if empID == 0 || oldRoleID == 0 || newRoleID == 0 {
		return ErrInvalidInput
	}

	var empRole EmployeeRole
	if err := r.db.Where("employee_id = ? AND role_id = ?", empID, oldRoleID).First(&empRole).Error; err != nil {
		return ErrNotFound
	}

	// Validate new role exists
	var role Role
	if err := r.db.First(&role, newRoleID).Error; err != nil {
		return ErrNotFound
	}

	empRole.RoleID = newRoleID
	if err := r.db.Save(&empRole).Error; err != nil {
		return err
	}

	r.invalidateCache(empID)
	r.logAudit(empID, "update_employee_role", "employee_role", newRoleID, "Updated role assignment")
	return nil
}

// GetEmployeeRole retrieves an employee-role mapping.
func (r *RBAC) GetEmployeeRole(empID, roleID uint) (*EmployeeRole, error) {
	if empID == 0 || roleID == 0 {
		return nil, ErrInvalidInput
	}

	var empRole EmployeeRole
	if err := r.db.Where("employee_id = ? AND role_id = ?", empID, roleID).First(&empRole).Error; err != nil {
		return nil, ErrNotFound
	}

	return &empRole, nil
}

// DeleteEmployeeRole soft-deletes an employee-role mapping.
func (r *RBAC) DeleteEmployeeRole(empID, roleID uint) error {
	if empID == 0 || roleID == 0 {
		return ErrInvalidInput
	}

	var empRole EmployeeRole
	if err := r.db.Where("employee_id = ? AND role_id = ?", empID, roleID).First(&empRole).Error; err != nil {
		return ErrNotFound
	}

	if err := r.db.Delete(&empRole).Error; err != nil {
		return err
	}

	r.invalidateCache(empID)
	r.logAudit(empID, "delete_employee_role", "employee_role", roleID, "Removed role from employee")
	return nil
}

// ListEmployeeRoles retrieves all roles for an employee.
func (r *RBAC) ListEmployeeRoles(empID uint) ([]EmployeeRole, error) {
	if empID == 0 {
		return nil, ErrInvalidInput
	}

	var empRoles []EmployeeRole
	if err := r.db.Where("employee_id = ?", empID).Find(&empRoles).Error; err != nil {
		return nil, err
	}
	return empRoles, nil
}
