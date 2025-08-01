package rbac

// CreateDepartment creates a new department.
func (r *RBAC) CreateDepartment(name string) (*Department, error) {
	if name == "" {
		return nil, ErrInvalidInput
	}

	dept := &Department{Name: name}
	if err := r.db.Create(dept).Error; err != nil {
		return nil, err
	}

	r.logAudit(0, "create_department", "department", dept.ID, "Created department: "+name)
	return dept, nil
}

// UpdateDepartment updates a department's name.
func (r *RBAC) UpdateDepartment(id uint, name string) (*Department, error) {
	if id == 0 || name == "" {
		return nil, ErrInvalidInput
	}

	var dept Department
	if err := r.db.First(&dept, id).Error; err != nil {
		return nil, ErrNotFound
	}

	dept.Name = name
	if err := r.db.Save(&dept).Error; err != nil {
		return nil, err
	}

	r.logAudit(0, "update_department", "department", dept.ID, "Updated department name to: "+name)
	return &dept, nil
}

// GetDepartment retrieves a department by ID.
func (r *RBAC) GetDepartment(id uint) (*Department, error) {
	if id == 0 {
		return nil, ErrInvalidInput
	}

	var dept Department
	if err := r.db.First(&dept, id).Error; err != nil {
		return nil, ErrNotFound
	}

	return &dept, nil
}

// DeleteDepartment soft-deletes a department by ID.
func (r *RBAC) DeleteDepartment(id uint) error {
	if id == 0 {
		return ErrInvalidInput
	}

	var dept Department
	if err := r.db.First(&dept, id).Error; err != nil {
		return ErrNotFound
	}

	if err := r.db.Delete(&dept).Error; err != nil {
		return err
	}

	r.logAudit(0, "delete_department", "department", id, "Deleted department")
	return nil
}

// ListDepartments retrieves all departments.
func (r *RBAC) ListDepartments() ([]Department, error) {
	var depts []Department
	if err := r.db.Find(&depts).Error; err != nil {
		return nil, err
	}
	return depts, nil
}
