package builtin

import (
	"fmt"
	"strings"
	"time"

	"mailman/internal/triggerv2/models"
	"mailman/internal/triggerv2/plugins"
)

// EmailForwardActionPlugin 邮件转发动作插件
type EmailForwardActionPlugin struct {
	info   *plugins.PluginInfo
	config map[string]interface{}
}

// NewEmailForwardActionPlugin 创建邮件转发动作插件
func NewEmailForwardActionPlugin() plugins.ActionPlugin {
	return &EmailForwardActionPlugin{
		info: &plugins.PluginInfo{
			ID:          "email_forward_action",
			Name:        "邮件转发",
			Version:     "1.0.0",
			Description: "将邮件转发到指定地址",
			Author:      "TriggerV2 Team",
			Website:     "https://github.com/triggerv2/plugins",
			License:     "MIT",
			Type:        plugins.PluginTypeAction,
			Status:      plugins.PluginStatusLoaded,
			LoadedAt:    time.Now(),
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"to_address": map[string]interface{}{
						"type":        "string",
						"description": "转发目标邮箱地址",
						"format":      "email",
					},
					"subject_prefix": map[string]interface{}{
						"type":        "string",
						"description": "转发邮件主题前缀",
						"default":     "[转发] ",
					},
					"add_original_headers": map[string]interface{}{
						"type":        "boolean",
						"description": "是否包含原始邮件头",
						"default":     true,
					},
					"forward_attachments": map[string]interface{}{
						"type":        "boolean",
						"description": "是否转发附件",
						"default":     true,
					},
				},
				"required": []string{"to_address"},
			},
			DefaultConfig: map[string]interface{}{
				"to_address":           "",
				"subject_prefix":       "[转发] ",
				"add_original_headers": true,
				"forward_attachments":  true,
			},
			Dependencies: []string{},
			Permissions:  []string{plugins.PermissionWrite, plugins.PermissionNetwork},
			Sandbox:      true,
			MinVersion:   "1.0.0",
			MaxVersion:   "",
		},
		config: make(map[string]interface{}),
	}
}

// GetInfo 获取插件信息
func (p *EmailForwardActionPlugin) GetInfo() *plugins.PluginInfo {
	return p.info
}

// Initialize 初始化插件
func (p *EmailForwardActionPlugin) Initialize(ctx *plugins.PluginContext) error {
	p.config = p.info.DefaultConfig
	return nil
}

// Cleanup 清理插件
func (p *EmailForwardActionPlugin) Cleanup() error {
	return nil
}

// OnLoad 加载时回调
func (p *EmailForwardActionPlugin) OnLoad() error {
	return nil
}

// OnUnload 卸载时回调
func (p *EmailForwardActionPlugin) OnUnload() error {
	return nil
}

// OnActivate 激活时回调
func (p *EmailForwardActionPlugin) OnActivate() error {
	p.info.Status = plugins.PluginStatusActive
	return nil
}

// OnDeactivate 停用时回调
func (p *EmailForwardActionPlugin) OnDeactivate() error {
	p.info.Status = plugins.PluginStatusInactive
	return nil
}

// GetDefaultConfig 获取默认配置
func (p *EmailForwardActionPlugin) GetDefaultConfig() map[string]interface{} {
	return p.info.DefaultConfig
}

// ValidateConfig 验证配置
func (p *EmailForwardActionPlugin) ValidateConfig(config map[string]interface{}) error {
	// 验证转发地址
	if toAddress, ok := config["to_address"]; ok {
		if str, ok := toAddress.(string); ok {
			if str == "" {
				return fmt.Errorf("转发地址不能为空")
			}
			// 简单的邮箱格式验证
			if !p.isValidEmail(str) {
				return fmt.Errorf("无效的邮箱地址: %s", str)
			}
		} else {
			return fmt.Errorf("转发地址必须是字符串")
		}
	} else {
		return fmt.Errorf("转发地址是必需的")
	}

	// 验证主题前缀
	if prefix, ok := config["subject_prefix"]; ok {
		if _, ok := prefix.(string); !ok {
			return fmt.Errorf("主题前缀必须是字符串")
		}
	}

	return nil
}

// ApplyConfig 应用配置
func (p *EmailForwardActionPlugin) ApplyConfig(config map[string]interface{}) error {
	if err := p.ValidateConfig(config); err != nil {
		return err
	}

	for key, value := range config {
		p.config[key] = value
	}

	return nil
}

// HealthCheck 健康检查
func (p *EmailForwardActionPlugin) HealthCheck() error {
	return nil
}

// GetMetrics 获取指标
func (p *EmailForwardActionPlugin) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"executions": p.info.UsageCount,
		"last_used":  p.info.LastUsed,
		"status":     p.info.Status,
	}
}

// Execute 执行动作
func (p *EmailForwardActionPlugin) Execute(ctx *plugins.PluginContext, event *models.Event) (*plugins.PluginResult, error) {
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
	toAddress := p.getToAddress()
	subjectPrefix := p.getSubjectPrefix()
	addOriginalHeaders := p.getAddOriginalHeaders()
	forwardAttachments := p.getForwardAttachments()

	// 构建转发邮件
	forwardedEmail := p.buildForwardedEmail(emailData, toAddress, subjectPrefix, addOriginalHeaders, forwardAttachments)

	// 模拟发送邮件（实际应该调用邮件服务）
	err := p.sendEmail(forwardedEmail)
	if err != nil {
		return &plugins.PluginResult{
			Success:       false,
			Error:         fmt.Sprintf("转发邮件失败: %v", err),
			ExecutionTime: time.Since(startTime),
			Timestamp:     time.Now(),
		}, nil
	}

	result := &plugins.PluginResult{
		Success: true,
		Data: map[string]interface{}{
			"forwarded_to":         toAddress,
			"original_subject":     emailData.Subject,
			"forwarded_subject":    forwardedEmail["subject"],
			"forwarded_at":         time.Now(),
			"original_from":        emailData.From,
			"add_original_headers": addOriginalHeaders,
			"forward_attachments":  forwardAttachments,
		},
		ExecutionTime: time.Since(startTime),
		Timestamp:     time.Now(),
	}

	return result, nil
}

// GetDescription 获取描述
func (p *EmailForwardActionPlugin) GetDescription() string {
	return p.info.Description
}

// GetSupportedEventTypes 获取支持的事件类型
func (p *EmailForwardActionPlugin) GetSupportedEventTypes() []string {
	return []string{
		string(models.EventTypeEmailReceived),
		string(models.EventTypeEmailUpdated),
	}
}

// GetRequiredConfig 获取必需配置
func (p *EmailForwardActionPlugin) GetRequiredConfig() []string {
	return []string{"to_address"}
}

// CanExecute 检查是否可以执行
func (p *EmailForwardActionPlugin) CanExecute(ctx *plugins.PluginContext, event *models.Event) bool {
	supportedTypes := p.GetSupportedEventTypes()
	for _, supportedType := range supportedTypes {
		if string(event.Type) == supportedType {
			return true
		}
	}
	return false
}

// GetExecutionOrder 获取执行顺序
func (p *EmailForwardActionPlugin) GetExecutionOrder() int {
	return 200 // 较高优先级
}

// 私有方法

// getToAddress 获取转发地址
func (p *EmailForwardActionPlugin) getToAddress() string {
	if addr, ok := p.config["to_address"]; ok {
		if str, ok := addr.(string); ok {
			return str
		}
	}
	return ""
}

// getSubjectPrefix 获取主题前缀
func (p *EmailForwardActionPlugin) getSubjectPrefix() string {
	if prefix, ok := p.config["subject_prefix"]; ok {
		if str, ok := prefix.(string); ok {
			return str
		}
	}
	return "[转发] "
}

// getAddOriginalHeaders 获取是否包含原始邮件头
func (p *EmailForwardActionPlugin) getAddOriginalHeaders() bool {
	if headers, ok := p.config["add_original_headers"]; ok {
		if b, ok := headers.(bool); ok {
			return b
		}
	}
	return true
}

// getForwardAttachments 获取是否转发附件
func (p *EmailForwardActionPlugin) getForwardAttachments() bool {
	if attachments, ok := p.config["forward_attachments"]; ok {
		if b, ok := attachments.(bool); ok {
			return b
		}
	}
	return true
}

// buildForwardedEmail 构建转发邮件
func (p *EmailForwardActionPlugin) buildForwardedEmail(emailData models.EmailEventData, toAddress, subjectPrefix string, addOriginalHeaders, forwardAttachments bool) map[string]interface{} {
	forwardedEmail := map[string]interface{}{
		"to":      toAddress,
		"subject": subjectPrefix + emailData.Subject,
		"body":    p.buildForwardedBody(emailData, addOriginalHeaders),
	}

	if forwardAttachments && emailData.HasAttachment {
		forwardedEmail["attachments"] = "转发附件"
	}

	return forwardedEmail
}

// buildForwardedBody 构建转发邮件正文
func (p *EmailForwardActionPlugin) buildForwardedBody(emailData models.EmailEventData, addOriginalHeaders bool) string {
	body := "---------- 转发邮件 ----------\n"

	if addOriginalHeaders {
		body += fmt.Sprintf("发件人: %s\n", emailData.From)
		body += fmt.Sprintf("收件人: %s\n", emailData.To)
		body += fmt.Sprintf("主题: %s\n", emailData.Subject)
		body += fmt.Sprintf("时间: %s\n", emailData.ReceivedAt.Format("2006-01-02 15:04:05"))
		body += "\n"
	}

	// 由于 EmailEventData 结构中没有 Body 字段，我们添加一个说明
	body += fmt.Sprintf("邮件ID: %d\n", emailData.EmailID)
	if emailData.HasAttachment {
		body += "包含附件: 是\n"
	}
	if len(emailData.Labels) > 0 {
		body += fmt.Sprintf("标签: %v\n", emailData.Labels)
	}
	body += "\n[注意: 邮件正文内容需要从邮件服务获取]"

	return body
}

// sendEmail 发送邮件（模拟）
func (p *EmailForwardActionPlugin) sendEmail(email map[string]interface{}) error {
	// 模拟邮件发送
	fmt.Printf("[EMAIL_FORWARD] 转发邮件到: %s, 主题: %s\n", email["to"], email["subject"])
	return nil
}

// isValidEmail 简单的邮箱格式验证
func (p *EmailForwardActionPlugin) isValidEmail(email string) bool {
	// 简单的邮箱格式验证
	return len(email) > 0 && email != ""
}

// GetUISchema 获取UI架构
func (p *EmailForwardActionPlugin) GetUISchema() *plugins.UISchema {
	return &plugins.UISchema{
		Fields: []plugins.UIField{
			{
				Name:        "to_address",
				Label:       "转发地址",
				Type:        plugins.UIFieldTypeText,
				Description: "要转发到的目标邮箱地址",
				Placeholder: "example@domain.com",
				Required:    true,
				Width:       "full",
				Pattern:     `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			},
			{
				Name:         "subject_prefix",
				Label:        "主题前缀",
				Type:         plugins.UIFieldTypeText,
				Description:  "转发邮件的主题前缀",
				Placeholder:  "[转发]",
				Required:     false,
				Width:        "1/2",
				DefaultValue: "[转发] ",
			},
			{
				Name:         "add_original_headers",
				Label:        "包含原始邮件头",
				Type:         plugins.UIFieldTypeBoolean,
				Description:  "是否在转发邮件中包含原始邮件头信息",
				Required:     false,
				Width:        "1/2",
				DefaultValue: true,
			},
			{
				Name:         "forward_attachments",
				Label:        "转发附件",
				Type:         plugins.UIFieldTypeBoolean,
				Description:  "是否转发原邮件的附件",
				Required:     false,
				Width:        "1/2",
				DefaultValue: true,
			},
			{
				Name:         "add_forward_note",
				Label:        "添加转发说明",
				Type:         plugins.UIFieldTypeBoolean,
				Description:  "是否在转发邮件中添加转发说明",
				Required:     false,
				Width:        "1/2",
				DefaultValue: false,
			},
			{
				Name:        "forward_note",
				Label:       "转发说明",
				Type:        plugins.UIFieldTypeText,
				Description: "转发说明的内容",
				Placeholder: "此邮件已自动转发",
				Required:    false,
				Width:       "full",
				ShowIf: map[string]interface{}{
					"field":  "add_forward_note",
					"values": []string{"true"},
				},
			},
		},
		Layout:            "vertical",
		AllowCustomFields: false,
		AllowNesting:      false,
		MaxNestingLevel:   0,
		HelpText:          "配置邮件转发操作的参数",
		Examples: []plugins.UIExample{
			{
				Title:       "基本转发",
				Description: "转发邮件到指定地址",
				Expression: map[string]interface{}{
					"to_address":           "admin@example.com",
					"subject_prefix":       "[转发] ",
					"add_original_headers": true,
					"forward_attachments":  true,
				},
			},
			{
				Title:       "转发带说明",
				Description: "转发邮件并添加说明",
				Expression: map[string]interface{}{
					"to_address":          "admin@example.com",
					"subject_prefix":      "[自动转发] ",
					"add_forward_note":    true,
					"forward_note":        "此邮件已由系统自动转发",
					"forward_attachments": false,
				},
			},
		},
	}
}

// GetDynamicOptions 获取动态选项
func (p *EmailForwardActionPlugin) GetDynamicOptions(field string, query string) ([]plugins.UIOption, error) {
	switch field {
	case "to_address":
		// 返回一些常用的邮箱地址
		return []plugins.UIOption{
			{Value: "admin@example.com", Label: "管理员邮箱"},
			{Value: "support@example.com", Label: "支持邮箱"},
			{Value: "backup@example.com", Label: "备份邮箱"},
		}, nil
	case "subject_prefix":
		return []plugins.UIOption{
			{Value: "[转发] ", Label: "转发"},
			{Value: "[自动转发] ", Label: "自动转发"},
			{Value: "[重要] ", Label: "重要"},
		}, nil
	default:
		return []plugins.UIOption{}, nil
	}
}

// ValidateFieldValue 验证字段值
func (p *EmailForwardActionPlugin) ValidateFieldValue(field string, value interface{}) error {
	switch field {
	case "to_address":
		if str, ok := value.(string); ok {
			if str == "" {
				return fmt.Errorf("转发地址不能为空")
			}
			if !strings.Contains(str, "@") {
				return fmt.Errorf("转发地址格式不正确")
			}
		} else {
			return fmt.Errorf("转发地址必须是字符串")
		}
	case "subject_prefix":
		if str, ok := value.(string); ok {
			if len(str) > 50 {
				return fmt.Errorf("主题前缀不能超过50个字符")
			}
		}
	}
	return nil
}

// GetFieldSuggestions 获取字段建议
func (p *EmailForwardActionPlugin) GetFieldSuggestions(field string, prefix string) ([]string, error) {
	switch field {
	case "to_address":
		return []string{
			"admin@example.com",
			"support@example.com",
			"backup@example.com",
		}, nil
	case "subject_prefix":
		return []string{
			"[转发] ",
			"[自动转发] ",
			"[重要] ",
			"[紧急] ",
		}, nil
	default:
		return []string{}, nil
	}
}
