package repository

import (
	"errors"
	"mailman/internal/models"

	"gorm.io/gorm"
)

// OAuth2GlobalConfigRepository handles database operations for OAuth2GlobalConfig
type OAuth2GlobalConfigRepository struct {
	db *gorm.DB
}

// NewOAuth2GlobalConfigRepository creates a new OAuth2GlobalConfigRepository
func NewOAuth2GlobalConfigRepository(db *gorm.DB) *OAuth2GlobalConfigRepository {
	return &OAuth2GlobalConfigRepository{db: db}
}

// Create creates a new OAuth2 global config
func (r *OAuth2GlobalConfigRepository) Create(config *models.OAuth2GlobalConfig) error {
	return r.db.Create(config).Error
}

// GetByID retrieves an OAuth2 global config by ID
func (r *OAuth2GlobalConfigRepository) GetByID(id uint) (*models.OAuth2GlobalConfig, error) {
	var config models.OAuth2GlobalConfig
	err := r.db.First(&config, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("OAuth2 global config not found")
		}
		return nil, err
	}
	return &config, nil
}

// GetByProviderType retrieves an OAuth2 global config by provider type
func (r *OAuth2GlobalConfigRepository) GetByProviderType(providerType models.MailProviderType) (*models.OAuth2GlobalConfig, error) {
	var config models.OAuth2GlobalConfig
	err := r.db.Where("provider_type = ? AND is_enabled = ?", providerType, true).First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("OAuth2 global config not found")
		}
		return nil, err
	}
	return &config, nil
}

// GetCompleteConfigByProviderType retrieves a complete OAuth2 global config by provider type
// 优先返回配置完整的记录（有client_id、client_secret和redirect_uri）
func (r *OAuth2GlobalConfigRepository) GetCompleteConfigByProviderType(providerType models.MailProviderType) (*models.OAuth2GlobalConfig, error) {
	var config models.OAuth2GlobalConfig
	
	// 首先尝试获取配置完整的记录
	err := r.db.Where("provider_type = ? AND is_enabled = ? AND client_id != '' AND client_secret != '' AND redirect_uri != ''",
		providerType, true).First(&config).Error
	
	if err == nil {
		return &config, nil
	}
	
	// 如果没有找到完整配置，则返回任何启用的配置（包括空配置）
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = r.db.Where("provider_type = ? AND is_enabled = ?", providerType, true).First(&config).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("OAuth2 global config not found")
			}
			return nil, err
		}
		return &config, nil
	}
	
	return nil, err
}

// GetByName retrieves an OAuth2 global config by name
func (r *OAuth2GlobalConfigRepository) GetByName(name string) (*models.OAuth2GlobalConfig, error) {
	var config models.OAuth2GlobalConfig
	err := r.db.Where("name = ? AND is_enabled = ?", name, true).First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("OAuth2 global config not found")
		}
		return nil, err
	}
	return &config, nil
}

// GetByProviderTypeAll retrieves all OAuth2 global configs by provider type
func (r *OAuth2GlobalConfigRepository) GetByProviderTypeAll(providerType models.MailProviderType) ([]models.OAuth2GlobalConfig, error) {
	var configs []models.OAuth2GlobalConfig
	err := r.db.Where("provider_type = ? AND is_enabled = ?", providerType, true).Find(&configs).Error
	return configs, err
}

// GetAll retrieves all OAuth2 global configs
func (r *OAuth2GlobalConfigRepository) GetAll() ([]models.OAuth2GlobalConfig, error) {
	var configs []models.OAuth2GlobalConfig
	err := r.db.Find(&configs).Error
	return configs, err
}

// GetEnabled retrieves all enabled OAuth2 global configs
func (r *OAuth2GlobalConfigRepository) GetEnabled() ([]models.OAuth2GlobalConfig, error) {
	var configs []models.OAuth2GlobalConfig
	err := r.db.Where("is_enabled = ?", true).Find(&configs).Error
	return configs, err
}

// Update updates an OAuth2 global config
func (r *OAuth2GlobalConfigRepository) Update(config *models.OAuth2GlobalConfig) error {
	// 使用Where().Updates()来确保是更新操作而不是插入
	result := r.db.Where("id = ?", config.ID).Updates(config)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("no rows affected, record not found")
	}
	return nil
}

// Delete soft deletes an OAuth2 global config
func (r *OAuth2GlobalConfigRepository) Delete(id uint) error {
	return r.db.Delete(&models.OAuth2GlobalConfig{}, id).Error
}

// CreateOrUpdate creates or updates an OAuth2 global config for a provider
func (r *OAuth2GlobalConfigRepository) CreateOrUpdate(config *models.OAuth2GlobalConfig) error {
	var existing models.OAuth2GlobalConfig
	err := r.db.Where("provider_type = ?", config.ProviderType).First(&existing).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new
			return r.Create(config)
		}
		return err
	}

	// Update existing
	config.ID = existing.ID
	config.CreatedAt = existing.CreatedAt
	return r.Update(config)
}

// SeedDefaultConfigs seeds the database with default OAuth2 configs
func (r *OAuth2GlobalConfigRepository) SeedDefaultConfigs() error {
	// Check and create Gmail config
	_, err := r.GetByProviderType(models.ProviderTypeGmail)
	if err != nil {
		// Create default Gmail config (disabled by default)
		gmailConfig := &models.OAuth2GlobalConfig{
			Name:         "Gmail 默认配置",
			ProviderType: models.ProviderTypeGmail,
			ClientID:     "",
			ClientSecret: "",
			RedirectURI:  "",
			Scopes:       models.StringSlice{"https://mail.google.com/", "https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			IsEnabled:    false,
		}
		if err := r.Create(gmailConfig); err != nil {
			return err
		}
	}

	// Check and create Outlook config
	_, err = r.GetByProviderType(models.ProviderTypeOutlook)
	if err != nil {
		// Create default Outlook config (disabled by default)
		outlookConfig := &models.OAuth2GlobalConfig{
			Name:         "Outlook 默认配置",
			ProviderType: models.ProviderTypeOutlook,
			ClientID:     "",
			ClientSecret: "",
			RedirectURI:  "",
			Scopes:       models.StringSlice{"https://outlook.office.com/IMAP.AccessAsUser.All", "https://outlook.office.com/SMTP.Send", "offline_access"},
			IsEnabled:    false,
		}
		if err := r.Create(outlookConfig); err != nil {
			return err
		}
	}

	return nil
}
