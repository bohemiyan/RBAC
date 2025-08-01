package rbac

import (
	"time"

)

// logAudit creates an audit log entry.
func (r *RBAC) logAudit(actorEmpID uint, action, targetType string, targetID uint, details string) {
	audit := &AuditLog{
		ActorEmpID: actorEmpID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Details:    details,
		CreatedAt:  time.Now(),
	}
	r.db.Create(audit)
}

// GetAuditLog retrieves an audit log by ID.
func (r *RBAC) GetAuditLog(id uint) (*AuditLog, error) {
	if id == 0 {
		return nil, ErrInvalidInput
	}

	var audit AuditLog
	if err := r.db.First(&audit, id).Error; err != nil {
		return nil, ErrNotFound
	}

	return &audit, nil
}

// ListAuditLogs retrieves audit logs, optionally filtered by actor or target.
func (r *RBAC) ListAuditLogs(actorEmpID, targetID *uint) ([]AuditLog, error) {
	var audits []AuditLog
	query := r.db.Order("created_at DESC")
	if actorEmpID != nil {
		query = query.Where("actor_emp_id = ?", *actorEmpID)
	}
	if targetID != nil {
		query = query.Where("target_id = ?", *targetID)
	}
	if err := query.Find(&audits).Error; err != nil {
		return nil, err
	}
	return audits, nil
}
