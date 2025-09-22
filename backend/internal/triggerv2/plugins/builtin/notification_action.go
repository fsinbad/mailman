package builtin

import (
	"fmt"
	"time"

	"mailman/internal/triggerv2/models"
	"mailman/internal/triggerv2/plugins"
)

// NotificationActionPlugin 通知动作插件
type NotificationActionPlugin struct {
	info   *plugins.PluginInfo
	config map[string]interface{}
}

// NewNotificationActionPlugin 创建通知动作插件
func NewNotificationActionPlugin() plugins.ActionPlugin {
	return &NotificationActionPlugin{
		info: &plugins.PluginInfo{
			ID:          "notification_action",
			Name:        "通知动作",
			Version:     "1.0.0",
			Description: "发送通知消息",
			Author:      "TriggerV2 Team",
			Website:     "https://github.com/triggerv2/plugins",
			License:     "MIT",
			Type:        plugins.PluginTypeAction,
			Status:      plugins.PluginStatusLoaded,
			LoadedAt:    time.Now(),
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type":        "string",
						"description": "通知消息内容",
						"default":     "新邮件通知",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "通知标题",
						"default":     "邮件通知",
					},
					"channels": map[string]interface{}{
						"type":        "array",
						"description": "通知渠道",
						"items": map[string]interface{}{
							"type": "string",
							"enum": []string{"email", "sms", "webhook", "console"},
						},
						"default": []string{"console"},
					},
					"priority": map[string]interface{}{
						"type":        "string",
						"description": "优先级",
						"enum":        []string{"low", "normal", "high", "urgent"},
						"default":     "normal",
					},
					"template": map[string]interface{}{
						"type":        "string",
						"description": "消息模板",
						"default":     "{{.Title}}: {{.Message}}",
					},
					"webhook_url": map[string]interface{}{
						"type":        "string",
						"description": "Webhook URL",
						"default":     "",
					},
				},
				"required": []string{"message"},
			},
			DefaultConfig: map[string]interface{}{
				"message":     "新邮件通知",
				"title":       "邮件通知",
				"channels":    []string{"console"},
				"priority":    "normal",
				"template":    "{{.Title}}: {{.Message}}",
				"webhook_url": "",
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
func (p *NotificationActionPlugin) GetInfo() *plugins.PluginInfo {
	return p.info
}

// Initialize 初始化插件
func (p *NotificationActionPlugin) Initialize(ctx *plugins.PluginContext) error {
	p.config = p.info.DefaultConfig
	return nil
}

// Cleanup 清理插件
func (p *NotificationActionPlugin) Cleanup() error {
	return nil
}

// OnLoad 加载时回调
func (p *NotificationActionPlugin) OnLoad() error {
	return nil
}

// OnUnload 卸载时回调
func (p *NotificationActionPlugin) OnUnload() error {
	return nil
}

// OnActivate 激活时回调
func (p *NotificationActionPlugin) OnActivate() error {
	p.info.Status = plugins.PluginStatusActive
	return nil
}

// OnDeactivate 停用时回调
func (p *NotificationActionPlugin) OnDeactivate() error {
	p.info.Status = plugins.PluginStatusInactive
	return nil
}

// GetDefaultConfig 获取默认配置
func (p *NotificationActionPlugin) GetDefaultConfig() map[string]interface{} {
	return p.info.DefaultConfig
}

// ValidateConfig 验证配置
func (p *NotificationActionPlugin) ValidateConfig(config map[string]interface{}) error {
	// 验证消息
	if message, ok := config["message"]; ok {
		if _, ok := message.(string); !ok {
			return fmt.Errorf("消息必须是字符串")
		}
	}

	// 验证标题
	if title, ok := config["title"]; ok {
		if _, ok := title.(string); !ok {
			return fmt.Errorf("标题必须是字符串")
		}
	}

	// 验证通知渠道
	if channels, ok := config["channels"]; ok {
		if channelList, ok := channels.([]interface{}); ok {
			validChannels := map[string]bool{
				"email":   true,
				"sms":     true,
				"webhook": true,
				"console": true,
			}
			for _, channel := range channelList {
				if channelStr, ok := channel.(string); ok {
					if !validChannels[channelStr] {
						return fmt.Errorf("无效的通知渠道: %s", channelStr)
					}
				} else {
					return fmt.Errorf("通知渠道必须是字符串")
				}
			}
		} else {
			return fmt.Errorf("通知渠道必须是数组")
		}
	}

	// 验证优先级
	if priority, ok := config["priority"]; ok {
		if priorityStr, ok := priority.(string); ok {
			validPriorities := map[string]bool{
				"low":    true,
				"normal": true,
				"high":   true,
				"urgent": true,
			}
			if !validPriorities[priorityStr] {
				return fmt.Errorf("无效的优先级: %s", priorityStr)
			}
		} else {
			return fmt.Errorf("优先级必须是字符串")
		}
	}

	return nil
}

// ApplyConfig 应用配置
func (p *NotificationActionPlugin) ApplyConfig(config map[string]interface{}) error {
	if err := p.ValidateConfig(config); err != nil {
		return err
	}

	for key, value := range config {
		p.config[key] = value
	}

	return nil
}

// HealthCheck 健康检查
func (p *NotificationActionPlugin) HealthCheck() error {
	return nil
}

// GetMetrics 获取指标
func (p *NotificationActionPlugin) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"executions": p.info.UsageCount,
		"last_used":  p.info.LastUsed,
		"status":     p.info.Status,
	}
}

// Execute 执行动作
func (p *NotificationActionPlugin) Execute(ctx *plugins.PluginContext, event *models.Event) (*plugins.PluginResult, error) {
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
	message := p.getMessage()
	title := p.getTitle()
	channels := p.getChannels()
	priority := p.getPriority()

	// 构建通知内容
	notificationContent := p.buildNotificationContent(title, message, emailData)

	// 发送通知
	results := make(map[string]interface{})
	var errors []string

	for _, channel := range channels {
		err := p.sendNotification(channel, notificationContent, priority)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", channel, err))
			results[channel] = map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			}
		} else {
			results[channel] = map[string]interface{}{
				"success": true,
				"sent_at": time.Now(),
			}
		}
	}

	success := len(errors) == 0
	var errorMsg string
	if !success {
		errorMsg = fmt.Sprintf("部分通知发送失败: %v", errors)
	}

	result := &plugins.PluginResult{
		Success: success,
		Data: map[string]interface{}{
			"title":      title,
			"message":    message,
			"channels":   channels,
			"priority":   priority,
			"results":    results,
			"email_data": emailData,
		},
		Error:         errorMsg,
		ExecutionTime: time.Since(startTime),
		Timestamp:     time.Now(),
	}

	return result, nil
}

// GetDescription 获取描述
func (p *NotificationActionPlugin) GetDescription() string {
	return p.info.Description
}

// GetSupportedEventTypes 获取支持的事件类型
func (p *NotificationActionPlugin) GetSupportedEventTypes() []string {
	return []string{
		string(models.EventTypeEmailReceived),
		string(models.EventTypeEmailUpdated),
		string(models.EventTypeTriggerExecuted),
	}
}

// GetRequiredConfig 获取必需配置
func (p *NotificationActionPlugin) GetRequiredConfig() []string {
	return []string{"message"}
}

// CanExecute 检查是否可以执行
func (p *NotificationActionPlugin) CanExecute(ctx *plugins.PluginContext, event *models.Event) bool {
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
func (p *NotificationActionPlugin) GetExecutionOrder() int {
	return 100 // 中等优先级
}

// 私有方法

// getMessage 获取消息配置
func (p *NotificationActionPlugin) getMessage() string {
	if message, ok := p.config["message"]; ok {
		if str, ok := message.(string); ok {
			return str
		}
	}
	return "新邮件通知"
}

// getTitle 获取标题配置
func (p *NotificationActionPlugin) getTitle() string {
	if title, ok := p.config["title"]; ok {
		if str, ok := title.(string); ok {
			return str
		}
	}
	return "邮件通知"
}

// getChannels 获取通知渠道配置
func (p *NotificationActionPlugin) getChannels() []string {
	if channels, ok := p.config["channels"]; ok {
		if channelList, ok := channels.([]interface{}); ok {
			result := make([]string, 0, len(channelList))
			for _, channel := range channelList {
				if str, ok := channel.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{"console"}
}

// getPriority 获取优先级配置
func (p *NotificationActionPlugin) getPriority() string {
	if priority, ok := p.config["priority"]; ok {
		if str, ok := priority.(string); ok {
			return str
		}
	}
	return "normal"
}

// buildNotificationContent 构建通知内容
func (p *NotificationActionPlugin) buildNotificationContent(title, message string, emailData models.EmailEventData) string {
	// 简单的模板替换
	content := fmt.Sprintf("%s: %s", title, message)
	content += fmt.Sprintf("\n发件人: %s", emailData.From)
	content += fmt.Sprintf("\n主题: %s", emailData.Subject)
	content += fmt.Sprintf("\n收件人: %s", emailData.To)
	content += fmt.Sprintf("\n接收时间: %s", emailData.ReceivedAt.Format("2006-01-02 15:04:05"))

	if emailData.HasAttachment {
		content += "\n包含附件"
	}

	if len(emailData.Labels) > 0 {
		content += fmt.Sprintf("\n标签: %v", emailData.Labels)
	}

	return content
}

// sendNotification 发送通知
func (p *NotificationActionPlugin) sendNotification(channel, content, priority string) error {
	switch channel {
	case "console":
		return p.sendConsoleNotification(content, priority)
	case "email":
		return p.sendEmailNotification(content, priority)
	case "sms":
		return p.sendSMSNotification(content, priority)
	case "webhook":
		return p.sendWebhookNotification(content, priority)
	default:
		return fmt.Errorf("不支持的通知渠道: %s", channel)
	}
}

// sendConsoleNotification 发送控制台通知
func (p *NotificationActionPlugin) sendConsoleNotification(content, priority string) error {
	// 简单的控制台输出
	fmt.Printf("[%s] %s: %s\n", time.Now().Format("2006-01-02 15:04:05"), priority, content)
	return nil
}

// sendEmailNotification 发送邮件通知
func (p *NotificationActionPlugin) sendEmailNotification(content, priority string) error {
	// 这里应该实现真正的邮件发送逻辑
	fmt.Printf("[EMAIL] %s: %s\n", priority, content)
	return nil
}

// sendSMSNotification 发送短信通知
func (p *NotificationActionPlugin) sendSMSNotification(content, priority string) error {
	// 这里应该实现真正的短信发送逻辑
	fmt.Printf("[SMS] %s: %s\n", priority, content)
	return nil
}

// sendWebhookNotification 发送Webhook通知
func (p *NotificationActionPlugin) sendWebhookNotification(content, priority string) error {
	// 这里应该实现真正的Webhook发送逻辑
	webhookURL := p.getWebhookURL()
	if webhookURL == "" {
		return fmt.Errorf("未配置Webhook URL")
	}

	fmt.Printf("[WEBHOOK] %s to %s: %s\n", priority, webhookURL, content)
	return nil
}

// getWebhookURL 获取Webhook URL配置
func (p *NotificationActionPlugin) getWebhookURL() string {
	if url, ok := p.config["webhook_url"]; ok {
		if str, ok := url.(string); ok {
			return str
		}
	}
	return ""
}
