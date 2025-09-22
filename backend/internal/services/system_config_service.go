package services

import (
	"fmt"
	"mailman/internal/models"
	"mailman/internal/repository"
)

// SystemConfigService 系统配置服务
type SystemConfigService struct {
	repo *repository.SystemConfigRepository
}

// NewSystemConfigService 创建系统配置服务
func NewSystemConfigService(repo *repository.SystemConfigRepository) *SystemConfigService {
	return &SystemConfigService{
		repo: repo,
	}
}

// GetAllConfigs 获取所有可见配置
func (s *SystemConfigService) GetAllConfigs() ([]models.SystemConfigResponse, error) {
	configs, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}

	responses := make([]models.SystemConfigResponse, len(configs))
	for i, config := range configs {
		responses[i] = config.ToResponse()
	}

	return responses, nil
}

// GetConfigsByCategory 根据分类获取配置
func (s *SystemConfigService) GetConfigsByCategory(category string) ([]models.SystemConfigResponse, error) {
	configs, err := s.repo.GetByCategory(category)
	if err != nil {
		return nil, err
	}

	responses := make([]models.SystemConfigResponse, len(configs))
	for i, config := range configs {
		responses[i] = config.ToResponse()
	}

	return responses, nil
}

// GetConfigByKey 根据键获取配置
func (s *SystemConfigService) GetConfigByKey(key string) (*models.SystemConfigResponse, error) {
	config, err := s.repo.EnsureConfigExists(key)
	if err != nil {
		return nil, err
	}

	response := config.ToResponse()
	return &response, nil
}

// UpdateConfigValue 更新配置值
func (s *SystemConfigService) UpdateConfigValue(key string, value interface{}) error {
	return s.repo.UpdateValue(key, value)
}

// GetConfigValue 获取配置值
func (s *SystemConfigService) GetConfigValue(key string) (interface{}, error) {
	return s.repo.GetValueByKey(key)
}

// GetBoolConfig 获取布尔值配置
func (s *SystemConfigService) GetBoolConfig(key string) (bool, error) {
	return s.repo.GetBoolValueByKey(key)
}

// GetStringConfig 获取字符串配置
func (s *SystemConfigService) GetStringConfig(key string) (string, error) {
	return s.repo.GetStringValueByKey(key)
}

// ResetConfigToDefault 重置配置为默认值
func (s *SystemConfigService) ResetConfigToDefault(key string) error {
	config, err := s.repo.GetByKey(key)
	if err != nil {
		return err
	}

	if !config.IsEditable {
		return fmt.Errorf("configuration '%s' is not editable", key)
	}

	config.ResetToDefault()
	return s.repo.Update(config)
}

// InitializeDefaults 初始化默认配置
func (s *SystemConfigService) InitializeDefaults() error {
	return s.repo.InitializeDefaultConfigs()
}

// ValidateConfigValue 验证配置值
func (s *SystemConfigService) ValidateConfigValue(key string, value interface{}) error {
	config, err := s.repo.EnsureConfigExists(key)
	if err != nil {
		return err
	}

	if !config.IsEditable {
		return fmt.Errorf("configuration '%s' is not editable", key)
	}

	// 创建临时配置对象进行验证
	tempConfig := *config
	return tempConfig.SetValue(value)
}

// GetOAuth2AutoOpenConfig 获取OAuth2自动打开配置的便捷方法
func (s *SystemConfigService) GetOAuth2AutoOpenConfig() (bool, error) {
	return s.GetBoolConfig("oauth2-auto-open")
}

// SetOAuth2AutoOpenConfig 设置OAuth2自动打开配置的便捷方法
func (s *SystemConfigService) SetOAuth2AutoOpenConfig(autoOpen bool) error {
	return s.UpdateConfigValue("oauth2-auto-open", autoOpen)
}
