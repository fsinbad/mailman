package builtin

import (
	"fmt"
	"strings"
	"time"

	"mailman/internal/triggerv2/models"
	"mailman/internal/triggerv2/plugins"
)

// EmailLabelActionPlugin 邮件标记动作插件
type EmailLabelActionPlugin struct {
	info   *plugins.PluginInfo
	config map[string]interface{}
}

// NewEmailLabelActionPlugin 创建邮件标记动作插件
func NewEmailLabelActionPlugin() plugins.ActionPlugin {
	return &EmailLabelActionPlugin{
		info: &plugins.PluginInfo{
			ID:          "email_label_action",
			Name:        "邮件标记动作",
			Version:     "1.0.0",
			Description: "为邮件添加、移除或修改标签",
			Author:      "TriggerV2 Team",
			Website:     "https://github.com/triggerv2/plugins",
			License:     "MIT",
			Type:        plugins.PluginTypeAction,
			Status:      plugins.PluginStatusLoaded,
			LoadedAt:    time.Now(),
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"description": "标签操作类型",
						"enum":        []string{"add", "remove", "replace", "clear"},
						"default":     "add",
					},
					"labels": map[string]interface{}{
						"type":        "array",
						"description": "要操作的标签列表",
						"items": map[string]interface{}{
							"type": "string",
						},
						"default": []string{"重要"},
					},
					"color": map[string]interface{}{
						"type":        "string",
						"description": "标签颜色",
						"enum":        []string{"red", "blue", "green", "yellow", "purple", "orange", "gray"},
						"default":     "blue",
					},
					"create_if_not_exists": map[string]interface{}{
						"type":        "boolean",
						"description": "如果标签不存在是否创建",
						"default":     true,
					},
					"case_sensitive": map[string]interface{}{
						"type":        "boolean",
						"description": "标签名称是否区分大小写",
						"default":     false,
					},
					"auto_cleanup": map[string]interface{}{
						"type":        "boolean",
						"description": "自动清理空标签",
						"default":     true,
					},
					"max_labels": map[string]interface{}{
						"type":        "integer",
						"description": "最大标签数量",
						"minimum":     1,
						"maximum":     50,
						"default":     10,
					},
					"notify_on_change": map[string]interface{}{
						"type":        "boolean",
						"description": "标签变更时是否通知",
						"default":     false,
					},
				},
				"required": []string{"action", "labels"},
			},
			DefaultConfig: map[string]interface{}{
				"action":               "add",
				"labels":               []string{"重要"},
				"color":                "blue",
				"create_if_not_exists": true,
				"case_sensitive":       false,
				"auto_cleanup":         true,
				"max_labels":           10,
				"notify_on_change":     false,
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
func (p *EmailLabelActionPlugin) GetInfo() *plugins.PluginInfo {
	return p.info
}

// Initialize 初始化插件
func (p *EmailLabelActionPlugin) Initialize(ctx *plugins.PluginContext) error {
	p.config = p.info.DefaultConfig
	return nil
}

// Cleanup 清理插件
func (p *EmailLabelActionPlugin) Cleanup() error {
	return nil
}

// OnLoad 加载时回调
func (p *EmailLabelActionPlugin) OnLoad() error {
	return nil
}

// OnUnload 卸载时回调
func (p *EmailLabelActionPlugin) OnUnload() error {
	return nil
}

// OnActivate 激活时回调
func (p *EmailLabelActionPlugin) OnActivate() error {
	p.info.Status = plugins.PluginStatusActive
	return nil
}

// OnDeactivate 停用时回调
func (p *EmailLabelActionPlugin) OnDeactivate() error {
	p.info.Status = plugins.PluginStatusInactive
	return nil
}

// GetDefaultConfig 获取默认配置
func (p *EmailLabelActionPlugin) GetDefaultConfig() map[string]interface{} {
	return p.info.DefaultConfig
}

// ValidateConfig 验证配置
func (p *EmailLabelActionPlugin) ValidateConfig(config map[string]interface{}) error {
	// 验证动作类型
	if action, ok := config["action"]; ok {
		if actionStr, ok := action.(string); ok {
			validActions := map[string]bool{
				"add":     true,
				"remove":  true,
				"replace": true,
				"clear":   true,
			}
			if !validActions[actionStr] {
				return fmt.Errorf("无效的动作类型: %s", actionStr)
			}
		} else {
			return fmt.Errorf("动作类型必须是字符串")
		}
	}

	// 验证标签列表
	if labels, ok := config["labels"]; ok {
		if labelList, ok := labels.([]interface{}); ok {
			for _, label := range labelList {
				if _, ok := label.(string); !ok {
					return fmt.Errorf("标签必须是字符串")
				}
			}
		} else {
			return fmt.Errorf("标签必须是数组")
		}
	}

	// 验证颜色
	if color, ok := config["color"]; ok {
		if colorStr, ok := color.(string); ok {
			validColors := map[string]bool{
				"red":    true,
				"blue":   true,
				"green":  true,
				"yellow": true,
				"purple": true,
				"orange": true,
				"gray":   true,
			}
			if !validColors[colorStr] {
				return fmt.Errorf("无效的颜色: %s", colorStr)
			}
		} else {
			return fmt.Errorf("颜色必须是字符串")
		}
	}

	// 验证布尔值配置
	boolConfigs := []string{"create_if_not_exists", "case_sensitive", "auto_cleanup", "notify_on_change"}
	for _, key := range boolConfigs {
		if value, ok := config[key]; ok {
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("%s 必须是布尔值", key)
			}
		}
	}

	// 验证最大标签数量
	if maxLabels, ok := config["max_labels"]; ok {
		if num, ok := maxLabels.(int); ok {
			if num < 1 || num > 50 {
				return fmt.Errorf("最大标签数量必须在1-50之间")
			}
		} else {
			return fmt.Errorf("最大标签数量必须是整数")
		}
	}

	return nil
}

// ApplyConfig 应用配置
func (p *EmailLabelActionPlugin) ApplyConfig(config map[string]interface{}) error {
	if err := p.ValidateConfig(config); err != nil {
		return err
	}

	for key, value := range config {
		p.config[key] = value
	}

	return nil
}

// HealthCheck 健康检查
func (p *EmailLabelActionPlugin) HealthCheck() error {
	return nil
}

// GetMetrics 获取指标
func (p *EmailLabelActionPlugin) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"executions": p.info.UsageCount,
		"last_used":  p.info.LastUsed,
		"status":     p.info.Status,
	}
}

// Execute 执行动作
func (p *EmailLabelActionPlugin) Execute(ctx *plugins.PluginContext, event *models.Event) (*plugins.PluginResult, error) {
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
	action := p.getAction()
	labels := p.getLabels()
	color := p.getColor()
	createIfNotExists := p.getCreateIfNotExists()
	caseSensitive := p.getCaseSensitive()
	autoCleanup := p.getAutoCleanup()
	maxLabels := p.getMaxLabels()
	notifyOnChange := p.getNotifyOnChange()

	// 处理标签操作
	labelResult := p.processLabelAction(emailData, action, labels, color, createIfNotExists, caseSensitive, maxLabels)

	// 如果启用了自动清理，清理空标签
	if autoCleanup {
		p.cleanupEmptyLabels(emailData)
	}

	// 如果启用了变更通知
	if notifyOnChange && labelResult.Changed {
		p.notifyLabelChange(emailData, labelResult)
	}

	result := &plugins.PluginResult{
		Success: labelResult.Success,
		Data: map[string]interface{}{
			"email_id":             emailData.EmailID,
			"subject":              emailData.Subject,
			"from":                 emailData.From,
			"original_labels":      emailData.Labels,
			"action":               action,
			"target_labels":        labels,
			"color":                color,
			"create_if_not_exists": createIfNotExists,
			"case_sensitive":       caseSensitive,
			"auto_cleanup":         autoCleanup,
			"max_labels":           maxLabels,
			"notify_on_change":     notifyOnChange,
			"label_result":         labelResult,
		},
		Error:         labelResult.Error,
		ExecutionTime: time.Since(startTime),
		Timestamp:     time.Now(),
	}

	return result, nil
}

// GetDescription 获取描述
func (p *EmailLabelActionPlugin) GetDescription() string {
	return p.info.Description
}

// GetSupportedEventTypes 获取支持的事件类型
func (p *EmailLabelActionPlugin) GetSupportedEventTypes() []string {
	return []string{
		string(models.EventTypeEmailReceived),
		string(models.EventTypeEmailUpdated),
	}
}

// GetRequiredConfig 获取必需配置
func (p *EmailLabelActionPlugin) GetRequiredConfig() []string {
	return []string{"action", "labels"}
}

// CanExecute 检查是否可以执行
func (p *EmailLabelActionPlugin) CanExecute(ctx *plugins.PluginContext, event *models.Event) bool {
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
func (p *EmailLabelActionPlugin) GetExecutionOrder() int {
	return 50 // 中等优先级
}

// 私有方法

// getAction 获取动作类型
func (p *EmailLabelActionPlugin) getAction() string {
	if action, ok := p.config["action"]; ok {
		if str, ok := action.(string); ok {
			return str
		}
	}
	return "add"
}

// getLabels 获取标签列表
func (p *EmailLabelActionPlugin) getLabels() []string {
	if labels, ok := p.config["labels"]; ok {
		if labelList, ok := labels.([]interface{}); ok {
			result := make([]string, 0, len(labelList))
			for _, label := range labelList {
				if str, ok := label.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{"重要"}
}

// getColor 获取颜色
func (p *EmailLabelActionPlugin) getColor() string {
	if color, ok := p.config["color"]; ok {
		if str, ok := color.(string); ok {
			return str
		}
	}
	return "blue"
}

// getCreateIfNotExists 获取创建不存在标签的设置
func (p *EmailLabelActionPlugin) getCreateIfNotExists() bool {
	if create, ok := p.config["create_if_not_exists"]; ok {
		if b, ok := create.(bool); ok {
			return b
		}
	}
	return true
}

// getCaseSensitive 获取大小写敏感设置
func (p *EmailLabelActionPlugin) getCaseSensitive() bool {
	if caseSensitive, ok := p.config["case_sensitive"]; ok {
		if b, ok := caseSensitive.(bool); ok {
			return b
		}
	}
	return false
}

// getAutoCleanup 获取自动清理设置
func (p *EmailLabelActionPlugin) getAutoCleanup() bool {
	if cleanup, ok := p.config["auto_cleanup"]; ok {
		if b, ok := cleanup.(bool); ok {
			return b
		}
	}
	return true
}

// getMaxLabels 获取最大标签数量
func (p *EmailLabelActionPlugin) getMaxLabels() int {
	if maxLabels, ok := p.config["max_labels"]; ok {
		if num, ok := maxLabels.(int); ok {
			return num
		}
	}
	return 10
}

// getNotifyOnChange 获取变更通知设置
func (p *EmailLabelActionPlugin) getNotifyOnChange() bool {
	if notify, ok := p.config["notify_on_change"]; ok {
		if b, ok := notify.(bool); ok {
			return b
		}
	}
	return false
}

// LabelResult 标签操作结果
type LabelResult struct {
	Success        bool      `json:"success"`
	Changed        bool      `json:"changed"`
	Error          string    `json:"error,omitempty"`
	OriginalLabels []string  `json:"original_labels"`
	FinalLabels    []string  `json:"final_labels"`
	AddedLabels    []string  `json:"added_labels"`
	RemovedLabels  []string  `json:"removed_labels"`
	CreatedLabels  []string  `json:"created_labels"`
	ProcessedAt    time.Time `json:"processed_at"`
}

// processLabelAction 处理标签操作
func (p *EmailLabelActionPlugin) processLabelAction(emailData models.EmailEventData, action string, labels []string, color string, createIfNotExists bool, caseSensitive bool, maxLabels int) *LabelResult {
	result := &LabelResult{
		Success:        true,
		Changed:        false,
		OriginalLabels: make([]string, len(emailData.Labels)),
		FinalLabels:    make([]string, len(emailData.Labels)),
		AddedLabels:    []string{},
		RemovedLabels:  []string{},
		CreatedLabels:  []string{},
		ProcessedAt:    time.Now(),
	}

	// 复制原始标签
	copy(result.OriginalLabels, emailData.Labels)
	copy(result.FinalLabels, emailData.Labels)

	// 根据动作类型处理标签
	switch action {
	case "add":
		p.addLabels(result, labels, caseSensitive, createIfNotExists, maxLabels, color)
	case "remove":
		p.removeLabels(result, labels, caseSensitive)
	case "replace":
		p.replaceLabels(result, labels, caseSensitive, createIfNotExists, maxLabels, color)
	case "clear":
		p.clearLabels(result)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("不支持的动作类型: %s", action)
	}

	// 检查是否有变更
	result.Changed = !p.labelsEqual(result.OriginalLabels, result.FinalLabels)

	return result
}

// addLabels 添加标签
func (p *EmailLabelActionPlugin) addLabels(result *LabelResult, labels []string, caseSensitive bool, createIfNotExists bool, maxLabels int, color string) {
	for _, label := range labels {
		if len(result.FinalLabels) >= maxLabels {
			result.Error = fmt.Sprintf("已达到最大标签数量限制: %d", maxLabels)
			break
		}

		if !p.hasLabel(result.FinalLabels, label, caseSensitive) {
			result.FinalLabels = append(result.FinalLabels, label)
			result.AddedLabels = append(result.AddedLabels, label)

			if createIfNotExists {
				result.CreatedLabels = append(result.CreatedLabels, label)
				fmt.Printf("创建新标签: %s (颜色: %s)\n", label, color)
			}
		}
	}
}

// removeLabels 移除标签
func (p *EmailLabelActionPlugin) removeLabels(result *LabelResult, labels []string, caseSensitive bool) {
	for _, label := range labels {
		if index := p.findLabelIndex(result.FinalLabels, label, caseSensitive); index != -1 {
			result.FinalLabels = append(result.FinalLabels[:index], result.FinalLabels[index+1:]...)
			result.RemovedLabels = append(result.RemovedLabels, label)
		}
	}
}

// replaceLabels 替换标签
func (p *EmailLabelActionPlugin) replaceLabels(result *LabelResult, labels []string, caseSensitive bool, createIfNotExists bool, maxLabels int, color string) {
	// 记录被移除的标签
	result.RemovedLabels = make([]string, len(result.FinalLabels))
	copy(result.RemovedLabels, result.FinalLabels)

	// 清空现有标签
	result.FinalLabels = []string{}

	// 添加新标签
	p.addLabels(result, labels, caseSensitive, createIfNotExists, maxLabels, color)
}

// clearLabels 清空标签
func (p *EmailLabelActionPlugin) clearLabels(result *LabelResult) {
	result.RemovedLabels = make([]string, len(result.FinalLabels))
	copy(result.RemovedLabels, result.FinalLabels)
	result.FinalLabels = []string{}
}

// hasLabel 检查是否有指定标签
func (p *EmailLabelActionPlugin) hasLabel(labels []string, label string, caseSensitive bool) bool {
	return p.findLabelIndex(labels, label, caseSensitive) != -1
}

// findLabelIndex 查找标签索引
func (p *EmailLabelActionPlugin) findLabelIndex(labels []string, label string, caseSensitive bool) int {
	for i, l := range labels {
		if caseSensitive {
			if l == label {
				return i
			}
		} else {
			if strings.EqualFold(l, label) {
				return i
			}
		}
	}
	return -1
}

// labelsEqual 检查两个标签列表是否相等
func (p *EmailLabelActionPlugin) labelsEqual(labels1, labels2 []string) bool {
	if len(labels1) != len(labels2) {
		return false
	}

	for i, label := range labels1 {
		if label != labels2[i] {
			return false
		}
	}

	return true
}

// cleanupEmptyLabels 清理空标签
func (p *EmailLabelActionPlugin) cleanupEmptyLabels(emailData models.EmailEventData) {
	// 模拟清理空标签
	fmt.Printf("清理空标签: 邮件ID %d\n", emailData.EmailID)
}

// notifyLabelChange 通知标签变更
func (p *EmailLabelActionPlugin) notifyLabelChange(emailData models.EmailEventData, result *LabelResult) {
	fmt.Printf("标签变更通知: 邮件 %s (ID: %d)\n", emailData.Subject, emailData.EmailID)
	fmt.Printf("  添加的标签: %v\n", result.AddedLabels)
	fmt.Printf("  移除的标签: %v\n", result.RemovedLabels)
}

// GetUISchema 获取UI架构
func (p *EmailLabelActionPlugin) GetUISchema() *plugins.UISchema {
	return &plugins.UISchema{
		Fields: []plugins.UIField{
			{
				Name:        "action_type",
				Label:       "操作类型",
				Type:        plugins.UIFieldTypeSelect,
				Description: "选择标签操作类型",
				Required:    true,
				Width:       "1/2",
				Options: []plugins.UIOption{
					{Value: "add", Label: "添加标签", Description: "为邮件添加新标签"},
					{Value: "remove", Label: "移除标签", Description: "从邮件中移除指定标签"},
					{Value: "replace", Label: "替换标签", Description: "替换邮件的所有标签"},
					{Value: "clear", Label: "清空标签", Description: "清空邮件的所有标签"},
				},
				DefaultValue: "add",
			},
			{
				Name:        "labels",
				Label:       "标签列表",
				Type:        plugins.UIFieldTypeText,
				Description: "标签名称，多个标签用逗号分隔",
				Placeholder: "重要,紧急,工作",
				Required:    false,
				Width:       "1/2",
				ShowIf: map[string]interface{}{
					"field":  "action_type",
					"values": []string{"add", "remove", "replace"},
				},
			},
			{
				Name:        "label_color",
				Label:       "标签颜色",
				Type:        plugins.UIFieldTypeSelect,
				Description: "选择标签的颜色",
				Required:    false,
				Width:       "1/3",
				Options: []plugins.UIOption{
					{Value: "red", Label: "红色", Color: "#ff0000"},
					{Value: "green", Label: "绿色", Color: "#00ff00"},
					{Value: "blue", Label: "蓝色", Color: "#0000ff"},
					{Value: "yellow", Label: "黄色", Color: "#ffff00"},
					{Value: "purple", Label: "紫色", Color: "#800080"},
					{Value: "orange", Label: "橙色", Color: "#ffa500"},
				},
				DefaultValue: "blue",
				ShowIf: map[string]interface{}{
					"field":  "action_type",
					"values": []string{"add", "replace"},
				},
			},
			{
				Name:         "create_if_not_exists",
				Label:        "自动创建标签",
				Type:         plugins.UIFieldTypeBoolean,
				Description:  "如果标签不存在，是否自动创建",
				Required:     false,
				Width:        "1/3",
				DefaultValue: true,
				ShowIf: map[string]interface{}{
					"field":  "action_type",
					"values": []string{"add", "replace"},
				},
			},
			{
				Name:         "cleanup_empty_labels",
				Label:        "清理空标签",
				Type:         plugins.UIFieldTypeBoolean,
				Description:  "是否清理空的标签",
				Required:     false,
				Width:        "1/3",
				DefaultValue: false,
			},
		},
		Layout:            "vertical",
		AllowCustomFields: false,
		AllowNesting:      false,
		MaxNestingLevel:   0,
		HelpText:          "配置邮件标签操作的参数",
		Examples: []plugins.UIExample{
			{
				Title:       "添加标签",
				Description: "为邮件添加重要和紧急标签",
				Expression: map[string]interface{}{
					"action_type":          "add",
					"labels":               "重要,紧急",
					"label_color":          "red",
					"create_if_not_exists": true,
				},
			},
			{
				Title:       "移除标签",
				Description: "从邮件中移除指定标签",
				Expression: map[string]interface{}{
					"action_type": "remove",
					"labels":      "临时,草稿",
				},
			},
			{
				Title:       "清空标签",
				Description: "清空邮件的所有标签",
				Expression: map[string]interface{}{
					"action_type": "clear",
				},
			},
		},
	}
}

// GetDynamicOptions 获取动态选项
func (p *EmailLabelActionPlugin) GetDynamicOptions(field string, query string) ([]plugins.UIOption, error) {
	switch field {
	case "labels":
		// 返回一些常用的标签
		return []plugins.UIOption{
			{Value: "重要", Label: "重要", Color: "red"},
			{Value: "紧急", Label: "紧急", Color: "orange"},
			{Value: "工作", Label: "工作", Color: "blue"},
			{Value: "个人", Label: "个人", Color: "green"},
			{Value: "待办", Label: "待办", Color: "yellow"},
			{Value: "已完成", Label: "已完成", Color: "gray"},
		}, nil
	default:
		return []plugins.UIOption{}, nil
	}
}

// ValidateFieldValue 验证字段值
func (p *EmailLabelActionPlugin) ValidateFieldValue(field string, value interface{}) error {
	switch field {
	case "action_type":
		if str, ok := value.(string); ok {
			validTypes := []string{"add", "remove", "replace", "clear"}
			valid := false
			for _, t := range validTypes {
				if str == t {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("操作类型必须是: %v", validTypes)
			}
		} else {
			return fmt.Errorf("操作类型必须是字符串")
		}
	case "labels":
		if str, ok := value.(string); ok {
			if str == "" {
				return fmt.Errorf("标签列表不能为空")
			}
		}
	}
	return nil
}

// GetFieldSuggestions 获取字段建议
func (p *EmailLabelActionPlugin) GetFieldSuggestions(field string, prefix string) ([]string, error) {
	switch field {
	case "labels":
		return []string{
			"重要",
			"紧急",
			"工作",
			"个人",
			"待办",
			"已完成",
			"草稿",
			"临时",
		}, nil
	default:
		return []string{}, nil
	}
}
