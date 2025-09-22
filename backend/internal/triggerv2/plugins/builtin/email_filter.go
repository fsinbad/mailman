package builtin

import (
	"fmt"
	"strings"
	"time"

	"mailman/internal/triggerv2/models"
	"mailman/internal/triggerv2/plugins"
)

// EmailFilterPlugin 邮件过滤插件
type EmailFilterPlugin struct {
	info   *plugins.PluginInfo
	config map[string]interface{}
}

// NewEmailFilterPlugin 创建邮件过滤插件
func NewEmailFilterPlugin() plugins.ConditionPlugin {
	return &EmailFilterPlugin{
		info: &plugins.PluginInfo{
			ID:          "email_filter",
			Name:        "邮件过滤器",
			Version:     "1.0.0",
			Description: "根据邮件内容过滤邮件",
			Author:      "TriggerV2 Team",
			Website:     "https://github.com/triggerv2/plugins",
			License:     "MIT",
			Type:        plugins.PluginTypeCondition,
			Status:      plugins.PluginStatusLoaded,
			LoadedAt:    time.Now(),
			ConfigSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"keywords": map[string]interface{}{
						"type":        "array",
						"description": "关键词列表",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"sender_domains": map[string]interface{}{
						"type":        "array",
						"description": "发件人域名列表",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"case_sensitive": map[string]interface{}{
						"type":        "boolean",
						"description": "是否区分大小写",
						"default":     false,
					},
					"match_mode": map[string]interface{}{
						"type":        "string",
						"description": "匹配模式: any, all",
						"default":     "any",
						"enum":        []string{"any", "all"},
					},
				},
				"required": []string{"keywords"},
			},
			DefaultConfig: map[string]interface{}{
				"keywords":       []string{},
				"sender_domains": []string{},
				"case_sensitive": false,
				"match_mode":     "any",
			},
			Dependencies: []string{},
			Permissions:  []string{plugins.PermissionRead},
			Sandbox:      true,
			MinVersion:   "1.0.0",
			MaxVersion:   "",
		},
		config: make(map[string]interface{}),
	}
}

// GetInfo 获取插件信息
func (p *EmailFilterPlugin) GetInfo() *plugins.PluginInfo {
	return p.info
}

// Initialize 初始化插件
func (p *EmailFilterPlugin) Initialize(ctx *plugins.PluginContext) error {
	p.config = p.info.DefaultConfig
	return nil
}

// Cleanup 清理插件
func (p *EmailFilterPlugin) Cleanup() error {
	return nil
}

// OnLoad 加载时回调
func (p *EmailFilterPlugin) OnLoad() error {
	return nil
}

// OnUnload 卸载时回调
func (p *EmailFilterPlugin) OnUnload() error {
	return nil
}

// OnActivate 激活时回调
func (p *EmailFilterPlugin) OnActivate() error {
	p.info.Status = plugins.PluginStatusActive
	return nil
}

// OnDeactivate 停用时回调
func (p *EmailFilterPlugin) OnDeactivate() error {
	p.info.Status = plugins.PluginStatusInactive
	return nil
}

// GetDefaultConfig 获取默认配置
func (p *EmailFilterPlugin) GetDefaultConfig() map[string]interface{} {
	return p.info.DefaultConfig
}

// ValidateConfig 验证配置
func (p *EmailFilterPlugin) ValidateConfig(config map[string]interface{}) error {
	// 验证关键词
	if keywords, ok := config["keywords"]; ok {
		if keywordList, ok := keywords.([]interface{}); ok {
			for _, keyword := range keywordList {
				if _, ok := keyword.(string); !ok {
					return fmt.Errorf("关键词必须是字符串")
				}
			}
		} else {
			return fmt.Errorf("关键词必须是数组")
		}
	}

	// 验证发件人域名
	if domains, ok := config["sender_domains"]; ok {
		if domainList, ok := domains.([]interface{}); ok {
			for _, domain := range domainList {
				if _, ok := domain.(string); !ok {
					return fmt.Errorf("发件人域名必须是字符串")
				}
			}
		} else {
			return fmt.Errorf("发件人域名必须是数组")
		}
	}

	// 验证匹配模式
	if mode, ok := config["match_mode"]; ok {
		if modeStr, ok := mode.(string); ok {
			if modeStr != "any" && modeStr != "all" {
				return fmt.Errorf("匹配模式必须是 'any' 或 'all'")
			}
		} else {
			return fmt.Errorf("匹配模式必须是字符串")
		}
	}

	return nil
}

// ApplyConfig 应用配置
func (p *EmailFilterPlugin) ApplyConfig(config map[string]interface{}) error {
	if err := p.ValidateConfig(config); err != nil {
		return err
	}

	for key, value := range config {
		p.config[key] = value
	}

	return nil
}

// HealthCheck 健康检查
func (p *EmailFilterPlugin) HealthCheck() error {
	return nil
}

// GetMetrics 获取指标
func (p *EmailFilterPlugin) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"evaluations": p.info.UsageCount,
		"last_used":   p.info.LastUsed,
		"status":      p.info.Status,
	}
}

// Evaluate 评估条件
func (p *EmailFilterPlugin) Evaluate(ctx *plugins.PluginContext, event *models.Event) (*plugins.PluginResult, error) {
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
	keywords := p.getKeywords()
	senderDomains := p.getSenderDomains()
	caseSensitive := p.getCaseSensitive()
	matchMode := p.getMatchMode()

	// 如果没有配置条件，返回 true
	if len(keywords) == 0 && len(senderDomains) == 0 {
		return &plugins.PluginResult{
			Success:       true,
			Data:          map[string]interface{}{"matched": true, "reason": "无过滤条件"},
			ExecutionTime: time.Since(startTime),
			Timestamp:     time.Now(),
		}, nil
	}

	matches := []bool{}
	reasons := []string{}

	// 检查关键词匹配
	if len(keywords) > 0 {
		keywordMatched := p.checkKeywords(emailData, keywords, caseSensitive)
		matches = append(matches, keywordMatched)
		if keywordMatched {
			reasons = append(reasons, "关键词匹配")
		}
	}

	// 检查发件人域名匹配
	if len(senderDomains) > 0 {
		domainMatched := p.checkSenderDomains(emailData, senderDomains)
		matches = append(matches, domainMatched)
		if domainMatched {
			reasons = append(reasons, "发件人域名匹配")
		}
	}

	// 根据匹配模式决定结果
	var finalResult bool
	if matchMode == "all" {
		finalResult = p.allTrue(matches)
	} else {
		finalResult = p.anyTrue(matches)
	}

	result := &plugins.PluginResult{
		Success: true,
		Data: map[string]interface{}{
			"matched":        finalResult,
			"reasons":        reasons,
			"match_mode":     matchMode,
			"keywords":       keywords,
			"sender_domains": senderDomains,
		},
		ExecutionTime: time.Since(startTime),
		Timestamp:     time.Now(),
	}

	return result, nil
}

// GetDescription 获取描述
func (p *EmailFilterPlugin) GetDescription() string {
	return p.info.Description
}

// GetSupportedEventTypes 获取支持的事件类型
func (p *EmailFilterPlugin) GetSupportedEventTypes() []string {
	return []string{
		string(models.EventTypeEmailReceived),
		string(models.EventTypeEmailUpdated),
	}
}

// GetRequiredFields 获取必需字段
func (p *EmailFilterPlugin) GetRequiredFields() []string {
	return []string{"subject", "from", "to"}
}

// 私有方法

// getKeywords 获取关键词配置
func (p *EmailFilterPlugin) getKeywords() []string {
	if keywords, ok := p.config["keywords"]; ok {
		if keywordList, ok := keywords.([]interface{}); ok {
			result := make([]string, 0, len(keywordList))
			for _, keyword := range keywordList {
				if str, ok := keyword.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{}
}

// getSenderDomains 获取发件人域名配置
func (p *EmailFilterPlugin) getSenderDomains() []string {
	if domains, ok := p.config["sender_domains"]; ok {
		if domainList, ok := domains.([]interface{}); ok {
			result := make([]string, 0, len(domainList))
			for _, domain := range domainList {
				if str, ok := domain.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{}
}

// getCaseSensitive 获取大小写敏感配置
func (p *EmailFilterPlugin) getCaseSensitive() bool {
	if caseSensitive, ok := p.config["case_sensitive"]; ok {
		if b, ok := caseSensitive.(bool); ok {
			return b
		}
	}
	return false
}

// getMatchMode 获取匹配模式配置
func (p *EmailFilterPlugin) getMatchMode() string {
	if mode, ok := p.config["match_mode"]; ok {
		if str, ok := mode.(string); ok {
			return str
		}
	}
	return "any"
}

// checkKeywords 检查关键词匹配
func (p *EmailFilterPlugin) checkKeywords(emailData models.EmailEventData, keywords []string, caseSensitive bool) bool {
	searchText := emailData.Subject + " " + emailData.From + " " + emailData.To

	if !caseSensitive {
		searchText = strings.ToLower(searchText)
	}

	for _, keyword := range keywords {
		checkKeyword := keyword
		if !caseSensitive {
			checkKeyword = strings.ToLower(keyword)
		}

		if strings.Contains(searchText, checkKeyword) {
			return true
		}
	}

	return false
}

// checkSenderDomains 检查发件人域名匹配
func (p *EmailFilterPlugin) checkSenderDomains(emailData models.EmailEventData, domains []string) bool {
	// 从邮件地址中提取域名
	fromParts := strings.Split(emailData.From, "@")
	if len(fromParts) < 2 {
		return false
	}

	senderDomain := strings.ToLower(fromParts[1])

	for _, domain := range domains {
		if strings.ToLower(domain) == senderDomain {
			return true
		}
	}

	return false
}

// allTrue 检查所有值是否为真
func (p *EmailFilterPlugin) allTrue(values []bool) bool {
	for _, value := range values {
		if !value {
			return false
		}
	}
	return len(values) > 0
}

// anyTrue 检查是否有任何值为真
func (p *EmailFilterPlugin) anyTrue(values []bool) bool {
	for _, value := range values {
		if value {
			return true
		}
	}
	return false
}
