package builtin

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"

	"mailman/internal/models"
	triggerModels "mailman/internal/triggerv2/models"
	"mailman/internal/triggerv2/plugins"
)

// EmailTransformActionPlugin 邮件数据转换动作插件
type EmailTransformActionPlugin struct {
	info   *plugins.PluginInfo
	config map[string]interface{}
}

// NewEmailTransformActionPlugin 创建邮件数据转换动作插件
func NewEmailTransformActionPlugin() plugins.ActionPlugin {
	return &EmailTransformActionPlugin{
		info: &plugins.PluginInfo{
			ID:          "email_transform_action",
			Name:        "邮件数据转换",
			Version:     "1.0.0",
			Description: "修改邮件的各种属性，支持JS/Go template等转换方式",
			Author:      "TriggerV2 Team",
			Website:     "https://github.com/triggerv2/plugins",
			License:     "MIT",
			Type:        plugins.PluginTypeAction,
			Status:      plugins.PluginStatusLoaded,
			LoadedAt:    time.Now(),
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target_field": map[string]interface{}{
						"type":        "string",
						"description": "要修改的邮件字段",
						"enum":        []string{"subject", "from", "to", "message_id", "thread_id", "labels"},
						"default":     "subject",
					},
					"transform_type": map[string]interface{}{
						"type":        "string",
						"description": "转换类型",
						"enum":        []string{"template", "javascript", "regex", "prefix", "suffix", "replace"},
						"default":     "template",
					},
				},
				"required": []string{"target_field", "transform_type"},
			},
			DefaultConfig: map[string]interface{}{
				"target_field":   "subject",
				"transform_type": "template",
			},
		},
		config: make(map[string]interface{}),
	}
}

// GetInfo 获取插件信息
func (p *EmailTransformActionPlugin) GetInfo() *plugins.PluginInfo {
	return p.info
}

// Initialize 初始化插件
func (p *EmailTransformActionPlugin) Initialize(ctx *plugins.PluginContext) error {
	return nil
}

// Cleanup 清理插件
func (p *EmailTransformActionPlugin) Cleanup() error {
	return nil
}

// OnLoad 加载时触发
func (p *EmailTransformActionPlugin) OnLoad() error {
	return nil
}

// OnUnload 卸载时触发
func (p *EmailTransformActionPlugin) OnUnload() error {
	return nil
}

// OnActivate 激活时触发
func (p *EmailTransformActionPlugin) OnActivate() error {
	return nil
}

// OnDeactivate 停用时触发
func (p *EmailTransformActionPlugin) OnDeactivate() error {
	return nil
}

// GetDefaultConfig 获取默认配置
func (p *EmailTransformActionPlugin) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"target_field":   "subject",
		"transform_type": "template",
	}
}

// ValidateConfig 验证配置
func (p *EmailTransformActionPlugin) ValidateConfig(config map[string]interface{}) error {
	targetField, ok := config["target_field"].(string)
	if !ok {
		return fmt.Errorf("target_field必须是字符串")
	}

	validFields := []string{"subject", "from", "to", "message_id", "thread_id", "labels"}
	found := false
	for _, field := range validFields {
		if field == targetField {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("不支持的字段: %s", targetField)
	}

	transformType, ok := config["transform_type"].(string)
	if !ok {
		return fmt.Errorf("transform_type必须是字符串")
	}

	validTypes := []string{"template", "javascript", "regex", "prefix", "suffix", "replace"}
	found = false
	for _, t := range validTypes {
		if t == transformType {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("不支持的转换类型: %s", transformType)
	}

	return nil
}

// ApplyConfig 应用配置
func (p *EmailTransformActionPlugin) ApplyConfig(config map[string]interface{}) error {
	// 合并默认配置和传入配置
	mergedConfig := make(map[string]interface{})

	// 先应用默认配置
	for key, value := range p.info.DefaultConfig {
		mergedConfig[key] = value
	}

	// 再应用传入的配置（覆盖默认值）
	for key, value := range config {
		mergedConfig[key] = value
	}

	if err := p.ValidateConfig(mergedConfig); err != nil {
		return err
	}

	p.config = mergedConfig
	return nil
}

// HealthCheck 健康检查
func (p *EmailTransformActionPlugin) HealthCheck() error {
	return nil
}

// GetMetrics 获取指标
func (p *EmailTransformActionPlugin) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"avg_execution_time": p.info.AvgExecutionTime,
		"usage_count":        p.info.UsageCount,
		"error_rate":         p.info.ErrorRate,
	}
}

// GetDescription 获取描述
func (p *EmailTransformActionPlugin) GetDescription() string {
	return "修改邮件的各种属性，支持模板、JavaScript、正则表达式等多种转换方式"
}

// GetSupportedEventTypes 获取支持的事件类型
func (p *EmailTransformActionPlugin) GetSupportedEventTypes() []string {
	return []string{
		string(triggerModels.EventTypeEmailReceived),
		string(triggerModels.EventTypeEmailUpdated),
	}
}

// GetRequiredConfig 获取必需的配置
func (p *EmailTransformActionPlugin) GetRequiredConfig() []string {
	return []string{
		"target_field",
		"transform_type",
	}
}

// CanExecute 检查是否可以执行
func (p *EmailTransformActionPlugin) CanExecute(ctx *plugins.PluginContext, event *triggerModels.Event) bool {
	// 检查事件类型
	if event.Type != triggerModels.EventTypeEmailReceived && event.Type != triggerModels.EventTypeEmailUpdated {
		return false
	}

	// 检查必需配置
	targetField := p.getStringConfig("target_field", "")
	transformType := p.getStringConfig("transform_type", "")

	return targetField != "" && transformType != ""
}

// GetExecutionOrder 获取执行顺序
func (p *EmailTransformActionPlugin) GetExecutionOrder() int {
	return 100 // 中等优先级
}

// Execute 执行动作
func (p *EmailTransformActionPlugin) Execute(ctx *plugins.PluginContext, event *triggerModels.Event) (*plugins.PluginResult, error) {
	startTime := time.Now()

	// 更新使用统计
	p.info.UsageCount++
	p.info.LastUsed = time.Now()

	// 尝试解析完整的邮件事件数据
	var emailEventData triggerModels.EmailEventData
	var email *models.Email

	if err := event.GetData(&emailEventData); err == nil && emailEventData.Email != nil {
		// 新格式：包含完整Email对象
		email = emailEventData.Email
	} else {
		// 兼容旧格式：从测试数据构建Email对象
		var eventData map[string]interface{}
		if err := event.GetData(&eventData); err != nil {
			return &plugins.PluginResult{
				Success:       false,
				Error:         fmt.Sprintf("解析事件数据失败: %v", err),
				ExecutionTime: time.Since(startTime),
				Timestamp:     time.Now(),
			}, nil
		}

		// 从测试数据构建Email对象
		email = p.buildEmailFromTestData(eventData)
		if email == nil {
			return &plugins.PluginResult{
				Success:       false,
				Error:         "无法构建邮件对象",
				ExecutionTime: time.Since(startTime),
				Timestamp:     time.Now(),
			}, nil
		}
	}

	// 获取配置
	targetField := p.getStringConfig("target_field", "")
	transformType := p.getStringConfig("transform_type", "")

	// 保存原始值
	originalValue := p.getEmailFieldValue(email, targetField)

	// 根据转换类型执行相应的转换
	var err error
	switch transformType {
	case "template":
		err = p.executeTemplateOnEmail(ctx, email, targetField)
	case "javascript":
		err = p.executeJavaScriptOnEmail(ctx, email, targetField)
	case "regex":
		err = p.executeRegexOnEmail(ctx, email, targetField)
	case "prefix":
		err = p.executePrefixOnEmail(ctx, email, targetField)
	case "suffix":
		err = p.executeSuffixOnEmail(ctx, email, targetField)
	case "replace":
		err = p.executeReplaceOnEmail(ctx, email, targetField)
	default:
		return &plugins.PluginResult{
			Success:       false,
			Error:         fmt.Sprintf("不支持的转换类型: %s", transformType),
			ExecutionTime: time.Since(startTime),
			Timestamp:     time.Now(),
		}, nil
	}

	if err != nil {
		return &plugins.PluginResult{
			Success:       false,
			Error:         fmt.Sprintf("转换失败: %v", err),
			ExecutionTime: time.Since(startTime),
			Timestamp:     time.Now(),
		}, nil
	}

	// 获取转换后的值
	newValue := p.getEmailFieldValue(email, targetField)

	// 计算执行时间
	duration := time.Since(startTime)
	p.info.AvgExecutionTime = (p.info.AvgExecutionTime + duration) / 2

	return &plugins.PluginResult{
		Success:       true,
		ExecutionTime: duration,
		Timestamp:     time.Now(),
		Data: map[string]interface{}{
			"transformed_field": targetField,
			"transform_type":    transformType,
			"original_value":    originalValue,
			"new_value":         newValue,
			"transformed_email": map[string]interface{}{
				"id":         email.ID,
				"subject":    email.Subject,
				"from":       email.From,
				"to":         email.To,
				"cc":         email.Cc,
				"bcc":        email.Bcc,
				"message_id": email.MessageID,
				"body":       email.Body,
				"html_body":  email.HTMLBody,
				"date":       email.Date,
				"size":       email.Size,
				"flags":      email.Flags,
				"mailbox":    email.MailboxName,
			},
		},
	}, nil
}

// executeTemplateOnEmail 在完整Email对象上执行模板转换
func (p *EmailTransformActionPlugin) executeTemplateOnEmail(ctx *plugins.PluginContext, email *models.Email, targetField string) error {
	templateContent := p.getStringConfig("template_content", "")
	if templateContent == "" {
		return nil
	}

	// 使用Go template引擎
	result, err := p.executeGoTemplate(templateContent, email)
	if err != nil {
		return fmt.Errorf("模板执行失败: %v", err)
	}

	return p.setEmailFieldValue(email, targetField, result)
}

// executeJavaScriptOnEmail 在完整Email对象上执行JavaScript转换
func (p *EmailTransformActionPlugin) executeJavaScriptOnEmail(ctx *plugins.PluginContext, email *models.Email, targetField string) error {
	scriptContent := p.getStringConfig("javascript_script", "")
	if scriptContent == "" {
		return nil
	}

	// 使用JavaScript引擎执行脚本
	result, err := p.executeJavaScript(scriptContent, email, targetField)
	if err != nil {
		return fmt.Errorf("JavaScript执行失败: %v", err)
	}

	return p.setEmailFieldValue(email, targetField, result)
}

// executeRegexOnEmail 在完整Email对象上执行正则表达式转换
func (p *EmailTransformActionPlugin) executeRegexOnEmail(ctx *plugins.PluginContext, email *models.Email, targetField string) error {
	pattern := p.getStringConfig("regex_pattern", "")
	replacement := p.getStringConfig("regex_replacement", "")

	if pattern == "" {
		return nil
	}

	originalValue := p.getEmailFieldValue(email, targetField)

	// 使用真正的正则表达式
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("正则表达式编译失败: %v", err)
	}

	// 执行正则表达式替换
	newValue := regex.ReplaceAllString(originalValue, replacement)

	return p.setEmailFieldValue(email, targetField, newValue)
}

// executePrefixOnEmail 在完整Email对象上执行前缀添加
func (p *EmailTransformActionPlugin) executePrefixOnEmail(ctx *plugins.PluginContext, email *models.Email, targetField string) error {
	textContent := p.getStringConfig("text_content", "")
	if textContent == "" {
		return nil
	}

	originalValue := p.getEmailFieldValue(email, targetField)
	newValue := textContent + originalValue

	return p.setEmailFieldValue(email, targetField, newValue)
}

// executeSuffixOnEmail 在完整Email对象上执行后缀添加
func (p *EmailTransformActionPlugin) executeSuffixOnEmail(ctx *plugins.PluginContext, email *models.Email, targetField string) error {
	textContent := p.getStringConfig("text_content", "")
	if textContent == "" {
		return nil
	}

	originalValue := p.getEmailFieldValue(email, targetField)
	newValue := originalValue + textContent

	return p.setEmailFieldValue(email, targetField, newValue)
}

// executeReplaceOnEmail 在完整Email对象上执行替换
func (p *EmailTransformActionPlugin) executeReplaceOnEmail(ctx *plugins.PluginContext, email *models.Email, targetField string) error {
	oldText := p.getStringConfig("old_text", "")
	newText := p.getStringConfig("new_text", "")

	if oldText == "" {
		return nil
	}

	originalValue := p.getEmailFieldValue(email, targetField)
	newValue := strings.ReplaceAll(originalValue, oldText, newText)

	return p.setEmailFieldValue(email, targetField, newValue)
}

// getStringConfig 获取字符串配置
func (p *EmailTransformActionPlugin) getStringConfig(key string, defaultValue string) string {
	if value, exists := p.config[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// getBoolConfig 获取布尔配置
func (p *EmailTransformActionPlugin) getBoolConfig(key string, defaultValue bool) bool {
	if value, exists := p.config[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetUISchema 获取UI架构
func (p *EmailTransformActionPlugin) GetUISchema() *plugins.UISchema {
	return &plugins.UISchema{
		Fields: []plugins.UIField{
			{
				Name:        "target_field",
				Label:       "目标字段",
				Type:        plugins.UIFieldTypeSelect,
				Description: "选择要修改的邮件字段",
				Required:    true,
				Width:       "1/2",
				Options: []plugins.UIOption{
					{Value: "subject", Label: "主题", Description: "邮件主题"},
					{Value: "from", Label: "发件人", Description: "发件人地址"},
					{Value: "to", Label: "收件人", Description: "收件人地址"},
					{Value: "message_id", Label: "消息ID", Description: "邮件消息ID"},
					{Value: "thread_id", Label: "线程ID", Description: "邮件线程ID"},
					{Value: "labels", Label: "标签", Description: "邮件标签"},
				},
				DefaultValue: "subject",
			},
			{
				Name:        "transform_type",
				Label:       "转换类型",
				Type:        plugins.UIFieldTypeSelect,
				Description: "选择转换方式",
				Required:    true,
				Width:       "1/2",
				Options: []plugins.UIOption{
					{Value: "template", Label: "模板", Description: "使用模板语法转换"},
					{Value: "javascript", Label: "JavaScript", Description: "使用JavaScript代码转换"},
					{Value: "regex", Label: "正则表达式", Description: "使用正则表达式替换"},
					{Value: "prefix", Label: "前缀添加", Description: "在原内容前添加文本"},
					{Value: "suffix", Label: "后缀添加", Description: "在原内容后添加文本"},
					{Value: "replace", Label: "替换", Description: "完全替换原内容"},
				},
				DefaultValue: "template",
			},
			{
				Name:        "template_content",
				Label:       "模板内容",
				Type:        plugins.UIFieldTypeText,
				Description: "模板内容，支持变量：{{subject}}, {{from}}, {{to}}, {{message_id}}, {{thread_id}}, {{labels}}",
				Required:    false,
				Width:       "full",
				Placeholder: "{{subject}} - 已转换",
				ShowIf: map[string]interface{}{
					"transform_type": "template",
				},
			},
			{
				Name:        "javascript_script",
				Label:       "JavaScript代码",
				Type:        plugins.UIFieldTypeCode,
				Description: "JavaScript转换代码",
				Required:    false,
				Width:       "full",
				Placeholder: "return value.toUpperCase();",
				ShowIf: map[string]interface{}{
					"transform_type": "javascript",
				},
			},
			{
				Name:        "regex_pattern",
				Label:       "正则表达式",
				Type:        plugins.UIFieldTypeText,
				Description: "要匹配的正则表达式模式",
				Required:    false,
				Width:       "1/2",
				Placeholder: "\\d+",
				ShowIf: map[string]interface{}{
					"transform_type": "regex",
				},
			},
			{
				Name:        "regex_replacement",
				Label:       "替换内容",
				Type:        plugins.UIFieldTypeText,
				Description: "替换匹配项的内容",
				Required:    false,
				Width:       "1/2",
				Placeholder: "NUMBER",
				ShowIf: map[string]interface{}{
					"transform_type": "regex",
				},
			},
			{
				Name:        "text_content",
				Label:       "文本内容",
				Type:        plugins.UIFieldTypeText,
				Description: "要添加的文本内容",
				Required:    false,
				Width:       "full",
				Placeholder: "输入文本内容",
				ShowIf: map[string]interface{}{
					"transform_type": []string{"prefix", "suffix"},
				},
			},
			{
				Name:        "old_text",
				Label:       "原始文本",
				Type:        plugins.UIFieldTypeText,
				Description: "要被替换的原始文本",
				Required:    false,
				Width:       "1/2",
				Placeholder: "原始文本",
				ShowIf: map[string]interface{}{
					"transform_type": "replace",
				},
			},
			{
				Name:        "new_text",
				Label:       "新文本",
				Type:        plugins.UIFieldTypeText,
				Description: "替换后的新文本",
				Required:    false,
				Width:       "1/2",
				Placeholder: "新文本",
				ShowIf: map[string]interface{}{
					"transform_type": "replace",
				},
			},
		},
		Examples: []plugins.UIExample{
			{
				Title:       "主题添加前缀",
				Description: "为邮件主题添加[重要]前缀",
				Expression: map[string]interface{}{
					"target_field":   "subject",
					"transform_type": "prefix",
					"text_content":   "[重要] ",
				},
			},
			{
				Title:       "使用模板转换",
				Description: "使用模板语法转换邮件主题",
				Expression: map[string]interface{}{
					"target_field":     "subject",
					"transform_type":   "template",
					"template_content": "来自 {{from}} 的邮件: {{subject}}",
				},
			},
		},
	}
}

// GetDynamicOptions 获取动态选项
func (p *EmailTransformActionPlugin) GetDynamicOptions(field string, query string) ([]plugins.UIOption, error) {
	switch field {
	case "template_content":
		return []plugins.UIOption{
			{Value: "{{subject}}", Label: "主题变量"},
			{Value: "{{from}}", Label: "发件人变量"},
			{Value: "{{to}}", Label: "收件人变量"},
			{Value: "{{message_id}}", Label: "消息ID变量"},
			{Value: "{{thread_id}}", Label: "线程ID变量"},
			{Value: "{{labels}}", Label: "标签变量"},
		}, nil
	default:
		return []plugins.UIOption{}, nil
	}
}

// ValidateFieldValue 验证字段值
func (p *EmailTransformActionPlugin) ValidateFieldValue(field string, value interface{}) error {
	switch field {
	case "target_field":
		if str, ok := value.(string); ok {
			validFields := []string{"subject", "from", "to", "message_id", "thread_id", "labels"}
			for _, validField := range validFields {
				if str == validField {
					return nil
				}
			}
			return fmt.Errorf("不支持的字段: %s", str)
		}
		return fmt.Errorf("target_field必须是字符串")
	case "transform_type":
		if str, ok := value.(string); ok {
			validTypes := []string{"template", "javascript", "regex", "prefix", "suffix", "replace"}
			for _, validType := range validTypes {
				if str == validType {
					return nil
				}
			}
			return fmt.Errorf("不支持的转换类型: %s", str)
		}
		return fmt.Errorf("transform_type必须是字符串")
	}
	return nil
}

// GetFieldSuggestions 获取字段建议
func (p *EmailTransformActionPlugin) GetFieldSuggestions(field string, prefix string) ([]string, error) {
	switch field {
	case "template_content":
		if prefix == "" {
			return []string{
				"{{subject}} - 已处理",
				"转发自 {{from}}: {{subject}}",
				"[{{labels}}] {{subject}}",
			}, nil
		}
	case "javascript_script":
		if prefix == "" {
			return []string{
				"return value.toUpperCase();",
				"return value.toLowerCase();",
				"return '[处理] ' + value;",
			}, nil
		}
	}
	return []string{}, nil
}

// getEmailFieldValue 获取完整Email对象的字段值
func (p *EmailTransformActionPlugin) getEmailFieldValue(email *models.Email, field string) string {
	switch field {
	case "subject":
		return email.Subject
	case "from":
		return strings.Join(email.From, ",")
	case "to":
		return strings.Join(email.To, ",")
	case "cc":
		return strings.Join(email.Cc, ",")
	case "bcc":
		return strings.Join(email.Bcc, ",")
	case "message_id":
		return email.MessageID
	case "body":
		return email.Body
	case "html_body":
		return email.HTMLBody
	case "mailbox":
		return email.MailboxName
	case "flags":
		return strings.Join(email.Flags, ",")
	default:
		return ""
	}
}

// setEmailFieldValue 设置完整Email对象的字段值
func (p *EmailTransformActionPlugin) setEmailFieldValue(email *models.Email, field string, value string) error {
	switch field {
	case "subject":
		email.Subject = value
	case "from":
		email.From = strings.Split(value, ",")
	case "to":
		email.To = strings.Split(value, ",")
	case "cc":
		email.Cc = strings.Split(value, ",")
	case "bcc":
		email.Bcc = strings.Split(value, ",")
	case "message_id":
		email.MessageID = value
	case "body":
		email.Body = value
	case "html_body":
		email.HTMLBody = value
	case "mailbox":
		email.MailboxName = value
	case "flags":
		email.Flags = strings.Split(value, ",")
	default:
		return fmt.Errorf("不支持的字段: %s", field)
	}
	return nil
}

// buildEmailFromTestData 从测试数据构建Email对象
func (p *EmailTransformActionPlugin) buildEmailFromTestData(eventData map[string]interface{}) *models.Email {
	// 从事件数据中提取邮件信息
	var emailInfo map[string]interface{}

	if eventInfo, ok := eventData["event"].(map[string]interface{}); ok {
		if data, ok := eventInfo["data"].(map[string]interface{}); ok {
			emailInfo = data
		}
	}

	if emailInfo == nil {
		return nil
	}

	// 构建Email对象
	email := &models.Email{
		Subject:   getStringFromMap(emailInfo, "subject"),
		MessageID: getStringFromMap(emailInfo, "messageId"),
		Body:      getStringFromMap(emailInfo, "body"),
		Size:      int64(getIntFromMap(emailInfo, "size")),
	}

	// 处理From字段（可能是字符串或数组）
	if fromValue, ok := emailInfo["from"]; ok {
		if fromStr, ok := fromValue.(string); ok {
			email.From = []string{fromStr}
		} else if fromArray, ok := fromValue.([]interface{}); ok {
			email.From = make([]string, len(fromArray))
			for i, v := range fromArray {
				if str, ok := v.(string); ok {
					email.From[i] = str
				}
			}
		}
	}

	// 处理To字段（可能是字符串或数组）
	if toValue, ok := emailInfo["to"]; ok {
		if toStr, ok := toValue.(string); ok {
			email.To = []string{toStr}
		} else if toArray, ok := toValue.([]interface{}); ok {
			email.To = make([]string, len(toArray))
			for i, v := range toArray {
				if str, ok := v.(string); ok {
					email.To[i] = str
				}
			}
		}
	}

	// 处理附件信息
	if attachments, ok := emailInfo["attachments"].([]interface{}); ok {
		email.Attachments = make([]models.Attachment, len(attachments))
		for i, att := range attachments {
			if attMap, ok := att.(map[string]interface{}); ok {
				email.Attachments[i] = models.Attachment{
					Filename: getStringFromMap(attMap, "filename"),
					MIMEType: getStringFromMap(attMap, "mimeType"),
					Size:     int64(getIntFromMap(attMap, "size")),
				}
			}
		}
	}

	return email
}

// getStringFromMap 从map中获取字符串值的辅助函数
func getStringFromMap(m map[string]interface{}, key string) string {
	if value, ok := m[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// getIntFromMap 从map中获取整数值的辅助函数
func getIntFromMap(m map[string]interface{}, key string) int {
	if value, ok := m[key]; ok {
		if intVal, ok := value.(int); ok {
			return intVal
		}
		if floatVal, ok := value.(float64); ok {
			return int(floatVal)
		}
	}
	return 0
}

// executeGoTemplate 执行Go模板
func (p *EmailTransformActionPlugin) executeGoTemplate(templateContent string, email *models.Email) (string, error) {
	// 创建模板数据
	templateData := map[string]interface{}{
		"subject":    email.Subject,
		"from":       strings.Join(email.From, ","),
		"to":         strings.Join(email.To, ","),
		"cc":         strings.Join(email.Cc, ","),
		"bcc":        strings.Join(email.Bcc, ","),
		"message_id": email.MessageID,
		"body":       email.Body,
		"html_body":  email.HTMLBody,
		"mailbox":    email.MailboxName,
		"flags":      strings.Join(email.Flags, ","),
		"size":       email.Size,
		"date":       email.Date,
		"now":        time.Now(),
	}

	// 创建模板并添加自定义函数
	tmpl, err := template.New("email_transform").Funcs(template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"trim":  strings.TrimSpace,
		"join": func(sep string, elems []string) string {
			return strings.Join(elems, sep)
		},
		"split": func(sep, s string) []string {
			return strings.Split(s, sep)
		},
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"replace": func(old, new, s string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"formatTime": func(layout string, t time.Time) string {
			return t.Format(layout)
		},
	}).Parse(templateContent)

	if err != nil {
		return "", fmt.Errorf("模板解析失败: %v", err)
	}

	// 执行模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("模板执行失败: %v", err)
	}

	return buf.String(), nil
}

// executeJavaScript 执行JavaScript代码（简化实现）
func (p *EmailTransformActionPlugin) executeJavaScript(scriptContent string, email *models.Email, targetField string) (string, error) {
	// 获取当前字段值
	currentValue := p.getEmailFieldValue(email, targetField)

	// 创建JavaScript执行上下文
	jsContext := map[string]interface{}{
		"value":      currentValue,
		"subject":    email.Subject,
		"from":       strings.Join(email.From, ","),
		"to":         strings.Join(email.To, ","),
		"cc":         strings.Join(email.Cc, ","),
		"bcc":        strings.Join(email.Bcc, ","),
		"message_id": email.MessageID,
		"body":       email.Body,
		"html_body":  email.HTMLBody,
		"mailbox":    email.MailboxName,
		"flags":      strings.Join(email.Flags, ","),
		"size":       email.Size,
		"date":       email.Date,
		"now":        time.Now(),
	}

	// 简化的JavaScript执行器
	// 注意：这是一个简化实现，仅支持基本的字符串操作
	result, err := p.executeSimpleJavaScript(scriptContent, jsContext)
	if err != nil {
		return "", fmt.Errorf("JavaScript执行失败: %v", err)
	}

	return result, nil
}

// executeSimpleJavaScript 简化的JavaScript执行器
func (p *EmailTransformActionPlugin) executeSimpleJavaScript(script string, context map[string]interface{}) (string, error) {
	// 这是一个非常简化的JavaScript执行器
	// 在生产环境中，建议使用真正的JavaScript引擎如goja

	// 获取当前值
	value, ok := context["value"].(string)
	if !ok {
		return "", fmt.Errorf("无法获取当前值")
	}

	// 支持一些常见的JavaScript操作
	script = strings.TrimSpace(script)

	// 简单的模式匹配和执行
	switch {
	case strings.Contains(script, "toUpperCase()"):
		return strings.ToUpper(value), nil
	case strings.Contains(script, "toLowerCase()"):
		return strings.ToLower(value), nil
	case strings.Contains(script, "trim()"):
		return strings.TrimSpace(value), nil
	case strings.HasPrefix(script, "return ") && strings.HasSuffix(script, ";"):
		// 处理简单的return语句
		return p.executeSimpleReturnStatement(script, context)
	default:
		return "", fmt.Errorf("不支持的JavaScript操作: %s", script)
	}
}

// executeSimpleReturnStatement 执行简单的return语句
func (p *EmailTransformActionPlugin) executeSimpleReturnStatement(script string, context map[string]interface{}) (string, error) {
	// 移除 "return " 和 ";"
	expression := strings.TrimSpace(script)
	expression = strings.TrimPrefix(expression, "return ")
	expression = strings.TrimSuffix(expression, ";")
	expression = strings.TrimSpace(expression)

	value, ok := context["value"].(string)
	if !ok {
		return "", fmt.Errorf("无法获取当前值")
	}

	// 处理一些常见的表达式
	switch {
	case expression == "value.toUpperCase()":
		return strings.ToUpper(value), nil
	case expression == "value.toLowerCase()":
		return strings.ToLower(value), nil
	case expression == "value.trim()":
		return strings.TrimSpace(value), nil
	case strings.HasPrefix(expression, "'") && strings.HasSuffix(expression, "' + value"):
		// 处理 'prefix' + value
		prefix := strings.Trim(strings.TrimSuffix(expression, " + value"), "'\"")
		return prefix + value, nil
	case strings.HasPrefix(expression, "value + '") && strings.HasSuffix(expression, "'"):
		// 处理 value + 'suffix'
		suffix := strings.Trim(strings.TrimPrefix(expression, "value + "), "'\"")
		return value + suffix, nil
	case strings.HasPrefix(expression, "'") && strings.HasSuffix(expression, "'"):
		// 处理纯字符串
		return strings.Trim(expression, "'\""), nil
	default:
		return "", fmt.Errorf("不支持的JavaScript表达式: %s", expression)
	}
}
