package builtin

import (
	"fmt"
	"strings"

	"mailman/internal/triggerv2/models"
	"mailman/internal/triggerv2/plugins"
)

// EmailConditionPlugin 邮件条件插件
type EmailConditionPlugin struct {
	info   *plugins.PluginInfo
	config map[string]interface{}
}

// NewEmailConditionPlugin 创建邮件条件插件
func NewEmailConditionPlugin() *EmailConditionPlugin {
	return &EmailConditionPlugin{
		info: &plugins.PluginInfo{
			ID:          "builtin.email_condition",
			Name:        "邮件条件",
			Version:     "1.0.0",
			Description: "基于邮件属性的条件判断",
			Author:      "Mailman Team",
			Type:        plugins.PluginTypeCondition,
			Status:      plugins.PluginStatusActive,
		},
		config: make(map[string]interface{}),
	}
}

// GetInfo 获取插件信息
func (p *EmailConditionPlugin) GetInfo() *plugins.PluginInfo {
	return p.info
}

// GetUISchema 获取UI架构
func (p *EmailConditionPlugin) GetUISchema() *plugins.UISchema {
	return &plugins.UISchema{
		Fields: []plugins.UIField{
			{
				Name:        "email.from",
				Label:       "发件人",
				Type:        plugins.UIFieldTypeDynamic,
				Description: "邮件发件人地址",
				Placeholder: "输入或选择邮箱地址",
				Required:    false,
				Width:       "full",
				OptionsAPI:  "/api/plugins/builtin.email_condition/callbacks/get-email-addresses",
			},
			{
				Name:        "email.to",
				Label:       "收件人",
				Type:        plugins.UIFieldTypeDynamic,
				Description: "邮件收件人地址",
				Placeholder: "输入或选择邮箱地址",
				Required:    false,
				Width:       "full",
				OptionsAPI:  "/api/plugins/builtin.email_condition/callbacks/get-email-addresses",
			},
			{
				Name:        "email.subject",
				Label:       "主题",
				Type:        plugins.UIFieldTypeText,
				Description: "邮件主题",
				Placeholder: "输入邮件主题关键词",
				Required:    false,
				Width:       "full",
			},
			{
				Name:        "email.body",
				Label:       "正文",
				Type:        plugins.UIFieldTypeText,
				Description: "邮件正文内容",
				Placeholder: "输入正文关键词",
				Required:    false,
				Width:       "full",
			},
			{
				Name:        "email.has_attachments",
				Label:       "包含附件",
				Type:        plugins.UIFieldTypeBoolean,
				Description: "是否包含附件",
				Required:    false,
				Width:       "half",
			},
			{
				Name:        "email.attachment_type",
				Label:       "附件类型",
				Type:        plugins.UIFieldTypeSelect,
				Description: "附件文件类型",
				Required:    false,
				Width:       "half",
				Options: []plugins.UIOption{
					{Value: "pdf", Label: "PDF文档", Icon: "file-pdf"},
					{Value: "doc", Label: "Word文档", Icon: "file-word"},
					{Value: "xls", Label: "Excel表格", Icon: "file-excel"},
					{Value: "img", Label: "图片", Icon: "file-image"},
					{Value: "zip", Label: "压缩文件", Icon: "file-zip"},
				},
				ShowIf: map[string]interface{}{
					"email.has_attachments": true,
				},
			},
			{
				Name:        "email.size",
				Label:       "邮件大小",
				Type:        plugins.UIFieldTypeNumber,
				Description: "邮件大小（KB）",
				Placeholder: "输入大小",
				Required:    false,
				Width:       "half",
				Min:         0,
				Max:         1048576, // 1GB
			},
			{
				Name:        "email.priority",
				Label:       "优先级",
				Type:        plugins.UIFieldTypeSelect,
				Description: "邮件优先级",
				Required:    false,
				Width:       "half",
				Options: []plugins.UIOption{
					{Value: "high", Label: "高", Color: "red"},
					{Value: "normal", Label: "普通", Color: "gray"},
					{Value: "low", Label: "低", Color: "blue"},
				},
			},
		},
		Operators: []plugins.UIOperator{
			{Value: "equals", Label: "等于", ApplicableTo: []string{"text", "number", "select", "dynamic"}},
			{Value: "not_equals", Label: "不等于", ApplicableTo: []string{"text", "number", "select", "dynamic"}},
			{Value: "contains", Label: "包含", ApplicableTo: []string{"text", "dynamic"}},
			{Value: "not_contains", Label: "不包含", ApplicableTo: []string{"text", "dynamic"}},
			{Value: "starts_with", Label: "开头是", ApplicableTo: []string{"text", "dynamic"}},
			{Value: "ends_with", Label: "结尾是", ApplicableTo: []string{"text", "dynamic"}},
			{Value: "greater_than", Label: "大于", ApplicableTo: []string{"number"}},
			{Value: "less_than", Label: "小于", ApplicableTo: []string{"number"}},
			{Value: "in", Label: "在列表中", ApplicableTo: []string{"text", "select", "dynamic"}},
			{Value: "not_in", Label: "不在列表中", ApplicableTo: []string{"text", "select", "dynamic"}},
			{Value: "is_true", Label: "为真", ApplicableTo: []string{"boolean"}},
			{Value: "is_false", Label: "为假", ApplicableTo: []string{"boolean"}},
		},
		Layout:            "vertical",
		AllowCustomFields: true,
		AllowNesting:      true,
		MaxNestingLevel:   3,
		HelpText:          "配置邮件相关的条件判断规则",
		Examples: []plugins.UIExample{
			{
				Title:       "垃圾邮件过滤",
				Description: "过滤来自特定域名的邮件",
				Expression: map[string]interface{}{
					"field":    "email.from",
					"operator": "ends_with",
					"value":    "@spam.com",
				},
			},
			{
				Title:       "重要邮件标记",
				Description: "标记包含特定关键词的邮件",
				Expression: map[string]interface{}{
					"type":     "group",
					"operator": "and",
					"conditions": []map[string]interface{}{
						{
							"field":    "email.subject",
							"operator": "contains",
							"value":    "紧急",
						},
						{
							"field":    "email.priority",
							"operator": "equals",
							"value":    "high",
						},
					},
				},
			},
		},
	}
}

// GetDynamicOptions 获取动态选项
func (p *EmailConditionPlugin) GetDynamicOptions(field string, query string) ([]plugins.UIOption, error) {
	switch field {
	case "email.from", "email.to":
		// 这里应该从数据库获取邮箱地址
		// 模拟一些数据
		emails := []string{
			"admin@example.com",
			"support@example.com",
			"noreply@example.com",
			"user1@example.com",
			"user2@example.com",
		}

		var options []plugins.UIOption
		for _, email := range emails {
			if query == "" || strings.Contains(strings.ToLower(email), strings.ToLower(query)) {
				options = append(options, plugins.UIOption{
					Value: email,
					Label: email,
					Icon:  "mail",
				})
			}
		}
		return options, nil
	}
	return nil, fmt.Errorf("unsupported dynamic field: %s", field)
}

// ValidateFieldValue 验证字段值
func (p *EmailConditionPlugin) ValidateFieldValue(field string, value interface{}) error {
	switch field {
	case "email.from", "email.to":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("邮箱地址必须是字符串")
		}
		if !strings.Contains(str, "@") {
			return fmt.Errorf("无效的邮箱地址格式")
		}
	case "email.size":
		num, ok := value.(float64)
		if !ok {
			return fmt.Errorf("邮件大小必须是数字")
		}
		if num < 0 {
			return fmt.Errorf("邮件大小不能为负数")
		}
	}
	return nil
}

// GetFieldSuggestions 获取字段建议
func (p *EmailConditionPlugin) GetFieldSuggestions(field string, prefix string) ([]string, error) {
	suggestions := map[string][]string{
		"email.subject": {
			"订单",
			"发票",
			"通知",
			"提醒",
			"确认",
			"重要",
			"紧急",
		},
		"email.body": {
			"感谢您的订单",
			"您的订单已发货",
			"请确认",
			"附件",
			"详情请见",
		},
	}

	if fieldSuggestions, ok := suggestions[field]; ok {
		var filtered []string
		for _, s := range fieldSuggestions {
			if prefix == "" || strings.HasPrefix(s, prefix) {
				filtered = append(filtered, s)
			}
		}
		return filtered, nil
	}

	return nil, nil
}

// 实现其他必需的插件接口方法...
func (p *EmailConditionPlugin) Initialize(ctx *plugins.PluginContext) error {
	return nil
}

func (p *EmailConditionPlugin) Cleanup() error {
	return nil
}

func (p *EmailConditionPlugin) OnLoad() error {
	return nil
}

func (p *EmailConditionPlugin) OnUnload() error {
	return nil
}

func (p *EmailConditionPlugin) OnActivate() error {
	return nil
}

func (p *EmailConditionPlugin) OnDeactivate() error {
	return nil
}

func (p *EmailConditionPlugin) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{}
}

func (p *EmailConditionPlugin) ValidateConfig(config map[string]interface{}) error {
	return nil
}

func (p *EmailConditionPlugin) ApplyConfig(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *EmailConditionPlugin) HealthCheck() error {
	return nil
}

func (p *EmailConditionPlugin) GetMetrics() map[string]interface{} {
	return map[string]interface{}{}
}

func (p *EmailConditionPlugin) Evaluate(ctx *plugins.PluginContext, event *models.Event) (*plugins.PluginResult, error) {
	// 实际的条件评估逻辑
	return &plugins.PluginResult{
		Success: true,
		Data:    map[string]interface{}{"result": true},
	}, nil
}

func (p *EmailConditionPlugin) GetDescription() string {
	return p.info.Description
}

func (p *EmailConditionPlugin) GetSupportedEventTypes() []string {
	return []string{"email.received", "email.sent"}
}

func (p *EmailConditionPlugin) GetRequiredFields() []string {
	return []string{}
}
