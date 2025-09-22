package repository

import (
	"errors"
	"mailman/internal/models"
	"time"

	"gorm.io/gorm"
)

// EmailTriggerV2Repository handles database operations for EmailTriggerV2
type EmailTriggerV2Repository struct {
	db *gorm.DB
}

// NewEmailTriggerV2Repository creates a new EmailTriggerV2Repository
func NewEmailTriggerV2Repository(db *gorm.DB) *EmailTriggerV2Repository {
	return &EmailTriggerV2Repository{db: db}
}

// Create creates a new email trigger
func (r *EmailTriggerV2Repository) Create(trigger *models.EmailTriggerV2) error {
	return r.db.Create(trigger).Error
}

// GetByID retrieves an email trigger by ID
func (r *EmailTriggerV2Repository) GetByID(id uint) (*models.EmailTriggerV2, error) {
	var trigger models.EmailTriggerV2
	err := r.db.First(&trigger, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("email trigger not found")
		}
		return nil, err
	}
	return &trigger, nil
}

// GetAll retrieves all email triggers
func (r *EmailTriggerV2Repository) GetAll() ([]models.EmailTriggerV2, error) {
	var triggers []models.EmailTriggerV2
	err := r.db.Find(&triggers).Error
	return triggers, err
}

// GetAllPaginated retrieves email triggers with pagination and search
func (r *EmailTriggerV2Repository) GetAllPaginated(page, limit int, sortBy, sortOrder, search string) ([]models.EmailTriggerV2, int64, error) {
	var triggers []models.EmailTriggerV2
	var total int64

	// Build query
	query := r.db.Model(&models.EmailTriggerV2{})

	// Apply search filter
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", searchPattern, searchPattern)
	}

	// Get total count
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Apply sorting
	orderClause := sortBy + " " + sortOrder
	query = query.Order(orderClause)

	// Apply pagination
	offset := (page - 1) * limit
	err = query.Offset(offset).Limit(limit).Find(&triggers).Error
	if err != nil {
		return nil, 0, err
	}

	return triggers, total, nil
}

// GetByStatus retrieves email triggers by enabled status
func (r *EmailTriggerV2Repository) GetByStatus(enabled bool) ([]models.EmailTriggerV2, error) {
	var triggers []models.EmailTriggerV2
	err := r.db.Where("enabled = ?", enabled).Find(&triggers).Error
	return triggers, err
}

// Update updates an email trigger
func (r *EmailTriggerV2Repository) Update(trigger *models.EmailTriggerV2) error {
	return r.db.Save(trigger).Error
}

// UpdateStatus updates email trigger enabled status
func (r *EmailTriggerV2Repository) UpdateStatus(id uint, enabled bool) error {
	return r.db.Model(&models.EmailTriggerV2{}).Where("id = ?", id).Update("enabled", enabled).Error
}

// UpdateStatistics updates email trigger execution statistics
func (r *EmailTriggerV2Repository) UpdateStatistics(id uint, totalExecutions, successExecutions int64, lastExecutedAt *time.Time, lastError string) error {
	updates := map[string]interface{}{
		"total_executions":   totalExecutions,
		"success_executions": successExecutions,
		"last_executed_at":   lastExecutedAt,
		"last_error":         lastError,
	}
	return r.db.Model(&models.EmailTriggerV2{}).Where("id = ?", id).Updates(updates).Error
}

// Delete soft deletes an email trigger
func (r *EmailTriggerV2Repository) Delete(id uint) error {
	return r.db.Delete(&models.EmailTriggerV2{}, id).Error
}

// GetCount returns the total count of email triggers
func (r *EmailTriggerV2Repository) GetCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.EmailTriggerV2{}).Count(&count).Error
	return count, err
}

// GetCountByStatus returns the count of email triggers by enabled status
func (r *EmailTriggerV2Repository) GetCountByStatus(enabled bool) (int64, error) {
	var count int64
	err := r.db.Model(&models.EmailTriggerV2{}).Where("enabled = ?", enabled).Count(&count).Error
	return count, err
}

// TriggerExecutionLogV2Repository handles database operations for TriggerExecutionLogV2
type TriggerExecutionLogV2Repository struct {
	db *gorm.DB
}

// NewTriggerExecutionLogV2Repository creates a new TriggerExecutionLogV2Repository
func NewTriggerExecutionLogV2Repository(db *gorm.DB) *TriggerExecutionLogV2Repository {
	return &TriggerExecutionLogV2Repository{db: db}
}

// Create creates a new execution log
func (r *TriggerExecutionLogV2Repository) Create(log *models.TriggerExecutionLogV2) error {
	return r.db.Create(log).Error
}

// GetByID retrieves an execution log by ID
func (r *TriggerExecutionLogV2Repository) GetByID(id uint) (*models.TriggerExecutionLogV2, error) {
	var log models.TriggerExecutionLogV2
	err := r.db.First(&log, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("execution log not found")
		}
		return nil, err
	}
	return &log, nil
}

// GetByTriggerID retrieves execution logs by trigger ID with pagination
func (r *TriggerExecutionLogV2Repository) GetByTriggerID(triggerID uint, page, limit int) ([]models.TriggerExecutionLogV2, int64, error) {
	var logs []models.TriggerExecutionLogV2
	var total int64

	// Build query
	query := r.db.Model(&models.TriggerExecutionLogV2{}).Where("trigger_id = ?", triggerID)

	// Get total count
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination and ordering
	offset := (page - 1) * limit
	err = query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetAllPaginated retrieves execution logs with pagination and filtering
func (r *TriggerExecutionLogV2Repository) GetAllPaginated(page, limit int, triggerID *uint, status *models.TriggerExecutionV2Status, startDate, endDate *time.Time) ([]models.TriggerExecutionLogV2, int64, error) {
	var logs []models.TriggerExecutionLogV2
	var total int64

	// Build query
	query := r.db.Model(&models.TriggerExecutionLogV2{})

	// Apply filters
	if triggerID != nil {
		query = query.Where("trigger_id = ?", *triggerID)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	// Get total count
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination and ordering
	offset := (page - 1) * limit
	err = query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetLatestByTriggerID retrieves the latest execution log for a trigger
func (r *TriggerExecutionLogV2Repository) GetLatestByTriggerID(triggerID uint) (*models.TriggerExecutionLogV2, error) {
	var log models.TriggerExecutionLogV2
	err := r.db.Where("trigger_id = ?", triggerID).Order("created_at DESC").First(&log).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No logs found, not an error
		}
		return nil, err
	}
	return &log, nil
}

// DeleteOldLogs deletes execution logs older than the specified date
func (r *TriggerExecutionLogV2Repository) DeleteOldLogs(beforeDate time.Time) (int64, error) {
	result := r.db.Where("created_at < ?", beforeDate).Delete(&models.TriggerExecutionLogV2{})
	return result.RowsAffected, result.Error
}

// GetStatistics retrieves execution statistics for a trigger
func (r *TriggerExecutionLogV2Repository) GetStatistics(triggerID uint, startDate, endDate *time.Time) (map[string]interface{}, error) {
	query := r.db.Model(&models.TriggerExecutionLogV2{}).Where("trigger_id = ?", triggerID)

	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	// Get total count
	var totalCount int64
	err := query.Count(&totalCount).Error
	if err != nil {
		return nil, err
	}

	// Get success count
	var successCount int64
	err = query.Where("status = ?", models.TriggerExecutionV2StatusSuccess).Count(&successCount).Error
	if err != nil {
		return nil, err
	}

	// Get failed count
	var failedCount int64
	err = query.Where("status = ?", models.TriggerExecutionV2StatusFailed).Count(&failedCount).Error
	if err != nil {
		return nil, err
	}

	// Get partial count
	var partialCount int64
	err = query.Where("status = ?", models.TriggerExecutionV2StatusPartial).Count(&partialCount).Error
	if err != nil {
		return nil, err
	}

	// Calculate average execution time
	var avgExecutionTime float64
	err = query.Select("AVG(duration)").Scan(&avgExecutionTime).Error
	if err != nil {
		return nil, err
	}

	// Calculate success rate
	var successRate float64
	if totalCount > 0 {
		successRate = float64(successCount) / float64(totalCount) * 100
	}

	return map[string]interface{}{
		"total_executions":   totalCount,
		"success_executions": successCount,
		"failed_executions":  failedCount,
		"partial_executions": partialCount,
		"avg_execution_time": avgExecutionTime,
		"success_rate":       successRate,
	}, nil
}