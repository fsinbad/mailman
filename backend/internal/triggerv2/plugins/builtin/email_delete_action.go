package builtin

import (
	"fmt"
	"strings"
	"time"

	"mailman/internal/triggerv2/models"
	"mailman/internal/triggerv2/plugins"
)

// EmailDeleteActionPlugin 邮件删除动作插件
type EmailDeleteActionPlugin struct {
	info   *plugins.PluginInfo
	config map[string]interface{}
}

// NewEmailDeleteActionPlugin 创建邮件删除动作插件
func NewEmailDeleteActionPlugin() plugins.ActionPlugin {
	return &EmailDeleteActionPlugin{
		info: &plugins.PluginInfo{
			ID:          "email_delete_action",
			Name:        "邮件删除动作",
			Version:     "1.0.0",
			Description: "删除邮件到垃圾箱或永久删除",
			Author:      "TriggerV2 Team",
			Website:     "https://github.com/triggerv2/plugins",
			License:     "MIT",
			Type:        plugins.PluginTypeAction,
			Status:      plugins.PluginStatusLoaded,
			LoadedAt:    time.Now(),
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"delete_type": map[string]interface{}{
						"type":        "string",
						"description": "删除类型",
						"enum":        []string{"trash", "permanent"},
						"default":     "trash",
					},
					"confirm_delete": map[string]interface{}{
						"type":        "boolean",
						"description": "是否需要确认删除",
						"default":     true,
					},
					"backup_before_delete": map[string]interface{}{
						"type":        "boolean",
						"description": "删除前是否备份",
						"default":     false,
					},
					"backup_location": map[string]interface{}{
						"type":        "string",
						"description": "备份位置",
						"default":     "/tmp/email_backup",
					},
					"delete_attachments": map[string]interface{}{
						"type":        "boolean",
						"description": "是否同时删除附件",
						"default":     true,
					},
					"reason": map[string]interface{}{
						"type":        "string",
						"description": "删除原因",
						"default":     "自动删除",
					},
				},
				"required": []string{"delete_type"},
			},
			DefaultConfig: map[string]interface{}{
				"delete_type":          "trash",
				"confirm_delete":       true,
				"backup_before_delete": false,
				"backup_location":      "/tmp/email_backup",
				"delete_attachments":   true,
				"reason":               "自动删除",
			},
			Dependencies: []string{},
			Permissions:  []string{plugins.PermissionWrite, plugins.PermissionRead},
			Sandbox:      true,
			MinVersion:   "1.0.0",
			MaxVersion:   "",
		},
		config: make(map[string]interface{}),
	}
}

// GetInfo 获取插件信息
func (p *EmailDeleteActionPlugin) GetInfo() *plugins.PluginInfo {
	return p.info
}

// Initialize 初始化插件
func (p *EmailDeleteActionPlugin) Initialize(ctx *plugins.PluginContext) error {
	p.config = p.info.DefaultConfig
	return nil
}

// Cleanup 清理插件
func (p *EmailDeleteActionPlugin) Cleanup() error {
	return nil
}

// OnLoad 加载时回调
func (p *EmailDeleteActionPlugin) OnLoad() error {
	return nil
}

// OnUnload 卸载时回调
func (p *EmailDeleteActionPlugin) OnUnload() error {
	return nil
}

// OnActivate 激活时回调
func (p *EmailDeleteActionPlugin) OnActivate() error {
	p.info.Status = plugins.PluginStatusActive
	return nil
}

// OnDeactivate 停用时回调
func (p *EmailDeleteActionPlugin) OnDeactivate() error {
	p.info.Status = plugins.PluginStatusInactive
	return nil
}

// GetUISchema 获取UI架构
func (p *EmailDeleteActionPlugin) GetUISchema() *plugins.UISchema {
	return &plugins.UISchema{
		Fields: []plugins.UIField{
			{
				Name:        "delete_type",
				Label:       "删除类型",
				Type:        plugins.UIFieldTypeSelect,
				Description: "选择删除方式",
				Required:    true,
				Width:       "1/2",
				Options: []plugins.UIOption{
					{Value: "trash", Label: "移到垃圾箱", Description: "将邮件移到垃圾箱，可以恢复"},
					{Value: "permanent", Label: "永久删除", Description: "彻底删除邮件，无法恢复"},
				},
				DefaultValue: "trash",
			},
			{
				Name:         "backup_before_delete",
				Label:        "删除前备份",
				Type:         plugins.UIFieldTypeBoolean,
				Description:  "删除前是否备份邮件",
				Required:     false,
				Width:        "1/2",
				DefaultValue: true,
			},
			{
				Name:         "confirm_delete",
				Label:        "确认删除",
				Type:         plugins.UIFieldTypeBoolean,
				Description:  "执行删除前需要确认",
				Required:     false,
				Width:        "1/2",
				DefaultValue: true,
				ShowIf: map[string]interface{}{
					"field":  "delete_type",
					"values": []string{"permanent"},
				},
			},
			{
				Name:        "backup_location",
				Label:       "备份位置",
				Type:        plugins.UIFieldTypeText,
				Description: "备份文件的存储位置",
				Placeholder: "例如: /backup/emails/",
				Required:    false,
				Width:       "full",
				ShowIf: map[string]interface{}{
					"field":  "backup_before_delete",
					"values": []string{"true"},
				},
			},
		},
		Layout:            "vertical",
		AllowCustomFields: false,
		AllowNesting:      false,
		MaxNestingLevel:   0,
		HelpText:          "配置邮件删除操作的参数",
		Examples: []plugins.UIExample{
			{
				Title:       "移到垃圾箱",
				Description: "将邮件移到垃圾箱，保留备份",
				Expression: map[string]interface{}{
					"delete_type":          "trash",
					"backup_before_delete": true,
					"confirm_delete":       false,
				},
			},
			{
				Title:       "永久删除",
				Description: "彻底删除邮件，需要确认",
				Expression: map[string]interface{}{
					"delete_type":          "permanent",
					"backup_before_delete": true,
					"confirm_delete":       true,
					"backup_location":      "/backup/emails/",
				},
			},
		},
	}
}

// GetDynamicOptions 获取动态选项
func (p *EmailDeleteActionPlugin) GetDynamicOptions(field string, query string) ([]plugins.UIOption, error) {
	switch field {
	case "backup_location":
		// 返回一些常用的备份位置
		return []plugins.UIOption{
			{Value: "/backup/emails/", Label: "默认备份目录"},
			{Value: "/var/backups/mailman/", Label: "系统备份目录"},
			{Value: "/tmp/email_backup/", Label: "临时备份目录"},
		}, nil
	default:
		return []plugins.UIOption{}, nil
	}
}

// ValidateFieldValue 验证字段值
func (p *EmailDeleteActionPlugin) ValidateFieldValue(field string, value interface{}) error {
	switch field {
	case "delete_type":
		if str, ok := value.(string); ok {
			if str != "trash" && str != "permanent" {
				return fmt.Errorf("删除类型必须是 'trash' 或 'permanent'")
			}
		} else {
			return fmt.Errorf("删除类型必须是字符串")
		}
	case "backup_location":
		if str, ok := value.(string); ok {
			if str != "" && !strings.HasPrefix(str, "/") {
				return fmt.Errorf("备份位置必须是绝对路径")
			}
		}
	}
	return nil
}

// GetFieldSuggestions 获取字段建议
func (p *EmailDeleteActionPlugin) GetFieldSuggestions(field string, prefix string) ([]string, error) {
	switch field {
	case "backup_location":
		return []string{
			"/backup/emails/",
			"/var/backups/mailman/",
			"/tmp/email_backup/",
		}, nil
	default:
		return []string{}, nil
	}
}

// GetDefaultConfig 获取默认配置
func (p *EmailDeleteActionPlugin) GetDefaultConfig() map[string]interface{} {
	return p.info.DefaultConfig
}

// ValidateConfig 验证配置
func (p *EmailDeleteActionPlugin) ValidateConfig(config map[string]interface{}) error {
	// 验证删除类型
	if deleteType, ok := config["delete_type"]; ok {
		if typeStr, ok := deleteType.(string); ok {
			if typeStr != "trash" && typeStr != "permanent" {
				return fmt.Errorf("删除类型必须是 'trash' 或 'permanent'")
			}
		} else {
			return fmt.Errorf("删除类型必须是字符串")
		}
	}

	// 验证确认删除
	if confirm, ok := config["confirm_delete"]; ok {
		if _, ok := confirm.(bool); !ok {
			return fmt.Errorf("确认删除必须是布尔值")
		}
	}

	// 验证备份设置
	if backup, ok := config["backup_before_delete"]; ok {
		if _, ok := backup.(bool); !ok {
			return fmt.Errorf("备份设置必须是布尔值")
		}
	}

	// 验证备份位置
	if location, ok := config["backup_location"]; ok {
		if _, ok := location.(string); !ok {
			return fmt.Errorf("备份位置必须是字符串")
		}
	}

	// 验证删除附件
	if deleteAttach, ok := config["delete_attachments"]; ok {
		if _, ok := deleteAttach.(bool); !ok {
			return fmt.Errorf("删除附件设置必须是布尔值")
		}
	}

	// 验证原因
	if reason, ok := config["reason"]; ok {
		if _, ok := reason.(string); !ok {
			return fmt.Errorf("删除原因必须是字符串")
		}
	}

	return nil
}

// ApplyConfig 应用配置
func (p *EmailDeleteActionPlugin) ApplyConfig(config map[string]interface{}) error {
	if err := p.ValidateConfig(config); err != nil {
		return err
	}

	for key, value := range config {
		p.config[key] = value
	}

	return nil
}

// HealthCheck 健康检查
func (p *EmailDeleteActionPlugin) HealthCheck() error {
	return nil
}

// GetMetrics 获取指标
func (p *EmailDeleteActionPlugin) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"executions": p.info.UsageCount,
		"last_used":  p.info.LastUsed,
		"status":     p.info.Status,
	}
}

// Execute 执行动作
func (p *EmailDeleteActionPlugin) Execute(ctx *plugins.PluginContext, event *models.Event) (*plugins.PluginResult, error) {
	startTime := time.Now()

	// 更新使用统计
	p.info.UsageCount++
	p.info.LastUsed = time.Now()

	// 解析邮件事件数据
	var emailData models.EmailEventData
	if err := event.GetData(&emailData); err != nil {
		return &plugins.PluginResult{
			Success:       false,
			Error:         fmt.Sprintf("解析邮件数据失败: %v", err),
			ExecutionTime: time.Since(startTime),
			Timestamp:     time.Now(),
		}, nil
	}

	// 获取配置
	deleteType := p.getDeleteType()
	confirmDelete := p.getConfirmDelete()
	backupBeforeDelete := p.getBackupBeforeDelete()
	backupLocation := p.getBackupLocation()
	deleteAttachments := p.getDeleteAttachments()
	reason := p.getReason()

	// 如果需要确认删除，这里可以添加确认逻辑
	if confirmDelete {
		// 模拟确认过程
		fmt.Printf("确认删除邮件: %s (ID: %d)\n", emailData.Subject, emailData.EmailID)
	}

	// 如果需要备份，先备份邮件
	if backupBeforeDelete {
		if err := p.backupEmail(emailData, backupLocation); err != nil {
			return &plugins.PluginResult{
				Success:       false,
				Error:         fmt.Sprintf("备份邮件失败: %v", err),
				ExecutionTime: time.Since(startTime),
				Timestamp:     time.Now(),
			}, nil
		}
	}

	// 执行删除操作
	deleteResult := p.deleteEmail(emailData, deleteType, deleteAttachments, reason)

	result := &plugins.PluginResult{
		Success: deleteResult.Success,
		Data: map[string]interface{}{
			"email_id":             emailData.EmailID,
			"subject":              emailData.Subject,
			"from":                 emailData.From,
			"delete_type":          deleteType,
			"confirm_delete":       confirmDelete,
			"backup_before_delete": backupBeforeDelete,
			"backup_location":      backupLocation,
			"delete_attachments":   deleteAttachments,
			"reason":               reason,
			"delete_result":        deleteResult,
			"backup_performed":     backupBeforeDelete,
		},
		Error:         deleteResult.Error,
		ExecutionTime: time.Since(startTime),
		Timestamp:     time.Now(),
	}

	return result, nil
}

// GetDescription 获取描述
func (p *EmailDeleteActionPlugin) GetDescription() string {
	return p.info.Description
}

// GetSupportedEventTypes 获取支持的事件类型
func (p *EmailDeleteActionPlugin) GetSupportedEventTypes() []string {
	return []string{
		string(models.EventTypeEmailReceived),
		string(models.EventTypeEmailUpdated),
	}
}

// GetRequiredConfig 获取必需配置
func (p *EmailDeleteActionPlugin) GetRequiredConfig() []string {
	return []string{"delete_type"}
}

// CanExecute 检查是否可以执行
func (p *EmailDeleteActionPlugin) CanExecute(ctx *plugins.PluginContext, event *models.Event) bool {
	// 检查事件类型是否支持
	supportedTypes := p.GetSupportedEventTypes()
	for _, supportedType := range supportedTypes {
		if string(event.Type) == supportedType {
			return true
		}
	}
	return false
}

// GetExecutionOrder 获取执行顺序
func (p *EmailDeleteActionPlugin) GetExecutionOrder() int {
	return 200 // 较高优先级，在其他动作之后执行
}

// 私有方法

// getDeleteType 获取删除类型
func (p *EmailDeleteActionPlugin) getDeleteType() string {
	if deleteType, ok := p.config["delete_type"]; ok {
		if str, ok := deleteType.(string); ok {
			return str
		}
	}
	return "trash"
}

// getConfirmDelete 获取确认删除设置
func (p *EmailDeleteActionPlugin) getConfirmDelete() bool {
	if confirm, ok := p.config["confirm_delete"]; ok {
		if b, ok := confirm.(bool); ok {
			return b
		}
	}
	return true
}

// getBackupBeforeDelete 获取备份设置
func (p *EmailDeleteActionPlugin) getBackupBeforeDelete() bool {
	if backup, ok := p.config["backup_before_delete"]; ok {
		if b, ok := backup.(bool); ok {
			return b
		}
	}
	return false
}

// getBackupLocation 获取备份位置
func (p *EmailDeleteActionPlugin) getBackupLocation() string {
	if location, ok := p.config["backup_location"]; ok {
		if str, ok := location.(string); ok {
			return str
		}
	}
	return "/tmp/email_backup"
}

// getDeleteAttachments 获取删除附件设置
func (p *EmailDeleteActionPlugin) getDeleteAttachments() bool {
	if deleteAttach, ok := p.config["delete_attachments"]; ok {
		if b, ok := deleteAttach.(bool); ok {
			return b
		}
	}
	return true
}

// getReason 获取删除原因
func (p *EmailDeleteActionPlugin) getReason() string {
	if reason, ok := p.config["reason"]; ok {
		if str, ok := reason.(string); ok {
			return str
		}
	}
	return "自动删除"
}

// DeleteResult 删除结果
type DeleteResult struct {
	Success       bool      `json:"success"`
	Error         string    `json:"error,omitempty"`
	DeletedAt     time.Time `json:"deleted_at"`
	BackupCreated bool      `json:"backup_created"`
	BackupPath    string    `json:"backup_path,omitempty"`
}

// backupEmail 备份邮件
func (p *EmailDeleteActionPlugin) backupEmail(emailData models.EmailEventData, backupLocation string) error {
	// 模拟备份过程
	fmt.Printf("备份邮件到 %s: %s (ID: %d)\n", backupLocation, emailData.Subject, emailData.EmailID)
	return nil
}

// deleteEmail 删除邮件
func (p *EmailDeleteActionPlugin) deleteEmail(emailData models.EmailEventData, deleteType string, deleteAttachments bool, reason string) *DeleteResult {
	// 模拟删除过程
	switch deleteType {
	case "trash":
		fmt.Printf("移动邮件到垃圾箱: %s (ID: %d), 原因: %s\n", emailData.Subject, emailData.EmailID, reason)
	case "permanent":
		fmt.Printf("永久删除邮件: %s (ID: %d), 原因: %s\n", emailData.Subject, emailData.EmailID, reason)
	}

	if deleteAttachments && emailData.HasAttachment {
		fmt.Printf("删除附件: 邮件ID %d\n", emailData.EmailID)
	}

	return &DeleteResult{
		Success:       true,
		DeletedAt:     time.Now(),
		BackupCreated: false,
	}
}
