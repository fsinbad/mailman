package monitoring

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// AlertManager 告警管理器
type AlertManager struct {
	rules     map[string]*AlertRule
	evaluator *ConditionEvaluator
	notifier  *AlertNotifier
}

// ConditionEvaluator 条件评估器
type ConditionEvaluator struct{}

// AlertNotifier 告警通知器
type AlertNotifier struct {
	channels map[string]NotificationChannel
}

// NotificationChannel 通知渠道接口
type NotificationChannel interface {
	GetName() string
	SendNotification(alert *Alert) error
}

// NewAlertManager 创建告警管理器
func NewAlertManager(rules []*AlertRule) *AlertManager {
	am := &AlertManager{
		rules:     make(map[string]*AlertRule),
		evaluator: &ConditionEvaluator{},
		notifier:  NewAlertNotifier(),
	}

	// 加载规则
	for _, rule := range rules {
		am.rules[rule.ID] = rule
	}

	return am
}

// NewAlertNotifier 创建告警通知器
func NewAlertNotifier() *AlertNotifier {
	return &AlertNotifier{
		channels: make(map[string]NotificationChannel),
	}
}

// AddRule 添加告警规则
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.rules[rule.ID] = rule
}

// RemoveRule 删除告警规则
func (am *AlertManager) RemoveRule(ruleID string) {
	delete(am.rules, ruleID)
}

// GetRule 获取告警规则
func (am *AlertManager) GetRule(ruleID string) (*AlertRule, bool) {
	rule, exists := am.rules[ruleID]
	return rule, exists
}

// GetRules 获取所有告警规则
func (am *AlertManager) GetRules() []*AlertRule {
	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		rules = append(rules, rule)
	}
	return rules
}

// CheckAlerts 检查告警
func (am *AlertManager) CheckAlerts(metrics *SystemMetrics) []*Alert {
	alerts := make([]*Alert, 0)

	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		alert := am.evaluateRule(rule, metrics)
		if alert != nil {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// evaluateRule 评估告警规则
func (am *AlertManager) evaluateRule(rule *AlertRule, metrics *SystemMetrics) *Alert {
	// 获取指标值
	value, err := am.getMetricValue(rule.Metric, metrics)
	if err != nil {
		return nil
	}

	// 评估条件
	triggered, err := am.evaluator.Evaluate(rule.Condition, value, rule.Threshold)
	if err != nil {
		return nil
	}

	if !triggered {
		return nil
	}

	// 创建告警
	alert := &Alert{
		ID:        fmt.Sprintf("%s_%d", rule.ID, time.Now().Unix()),
		Rule:      rule,
		Status:    AlertStatusFiring,
		Message:   am.formatAlertMessage(rule, value),
		Value:     value,
		Threshold: rule.Threshold,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Count:     1,
	}

	// 发送通知
	if len(rule.NotifyChannels) > 0 {
		go am.sendNotification(alert)
	}

	return alert
}

// getMetricValue 获取指标值
func (am *AlertManager) getMetricValue(metric string, metrics *SystemMetrics) (interface{}, error) {
	// 解析指标路径，例如: "collector.name.metric.key"
	parts := strings.Split(metric, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("无效的指标路径: %s", metric)
	}

	// 处理系统级指标
	if parts[0] == "system" {
		return am.getSystemMetricValue(parts[1:], metrics)
	}

	// 处理收集器指标
	if parts[0] == "collector" && len(parts) >= 3 {
		collectorName := parts[1]
		metricPath := parts[2:]
		return am.getCollectorMetricValue(collectorName, metricPath, metrics)
	}

	return nil, fmt.Errorf("不支持的指标类型: %s", parts[0])
}

// getSystemMetricValue 获取系统级指标值
func (am *AlertManager) getSystemMetricValue(path []string, metrics *SystemMetrics) (interface{}, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("系统指标路径为空")
	}

	switch path[0] {
	case "alerts":
		if len(path) == 1 {
			return len(metrics.Alerts), nil
		}
		if path[1] == "count" {
			return len(metrics.Alerts), nil
		}
		if path[1] == "firing" {
			count := 0
			for _, alert := range metrics.Alerts {
				if alert.Status == AlertStatusFiring {
					count++
				}
			}
			return count, nil
		}
	case "health":
		if len(path) == 1 {
			return metrics.Health.Overall, nil
		}
		if path[1] == "overall" {
			return metrics.Health.Overall, nil
		}
		if path[1] == "components" && len(path) >= 3 {
			componentName := path[2]
			if component, exists := metrics.Health.Components[componentName]; exists {
				if len(path) == 3 {
					return component.Status, nil
				}
				if path[3] == "status" {
					return component.Status, nil
				}
			}
		}
	case "collectors":
		if len(path) == 1 {
			return len(metrics.Collectors), nil
		}
		if path[1] == "count" {
			return len(metrics.Collectors), nil
		}
		if path[1] == "errors" {
			totalErrors := int64(0)
			for _, collector := range metrics.Collectors {
				totalErrors += collector.ErrorCount
			}
			return totalErrors, nil
		}
	}

	return nil, fmt.Errorf("未知的系统指标: %s", strings.Join(path, "."))
}

// getCollectorMetricValue 获取收集器指标值
func (am *AlertManager) getCollectorMetricValue(collectorName string, path []string, metrics *SystemMetrics) (interface{}, error) {
	collector, exists := metrics.Collectors[collectorName]
	if !exists {
		return nil, fmt.Errorf("收集器 %s 不存在", collectorName)
	}

	if len(path) == 0 {
		return nil, fmt.Errorf("收集器指标路径为空")
	}

	switch path[0] {
	case "error_count":
		return collector.ErrorCount, nil
	case "collection_count":
		return collector.CollectionCount, nil
	case "last_error":
		return collector.LastError, nil
	case "metrics":
		if len(path) == 1 {
			return len(collector.Metrics), nil
		}

		// 获取最新的指标数据
		if len(collector.Metrics) == 0 {
			return nil, fmt.Errorf("收集器 %s 没有指标数据", collectorName)
		}

		latestMetric := collector.Metrics[len(collector.Metrics)-1]
		return am.getMetricFromSet(latestMetric.Metrics, path[1:])
	}

	return nil, fmt.Errorf("未知的收集器指标: %s", path[0])
}

// getMetricFromSet 从指标集合中获取值
func (am *AlertManager) getMetricFromSet(metrics map[string]interface{}, path []string) (interface{}, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("指标路径为空")
	}

	current := metrics
	for i, key := range path {
		if i == len(path)-1 {
			// 最后一个键，返回值
			if value, exists := current[key]; exists {
				return value, nil
			}
			return nil, fmt.Errorf("指标键 %s 不存在", key)
		}

		// 中间键，继续遍历
		if next, exists := current[key]; exists {
			if nextMap, ok := next.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return nil, fmt.Errorf("指标键 %s 不是对象", key)
			}
		} else {
			return nil, fmt.Errorf("指标键 %s 不存在", key)
		}
	}

	return nil, fmt.Errorf("无法获取指标值")
}

// formatAlertMessage 格式化告警消息
func (am *AlertManager) formatAlertMessage(rule *AlertRule, value interface{}) string {
	return fmt.Sprintf("%s: %s %s %v (当前值: %v)",
		rule.Name, rule.Metric, rule.Condition, rule.Threshold, value)
}

// sendNotification 发送通知
func (am *AlertManager) sendNotification(alert *Alert) {
	for _, channelName := range alert.Rule.NotifyChannels {
		if channel, exists := am.notifier.channels[channelName]; exists {
			if err := channel.SendNotification(alert); err != nil {
				fmt.Printf("发送通知失败 (渠道: %s): %v\n", channelName, err)
			} else {
				now := time.Now()
				alert.NotifiedAt = &now
			}
		}
	}
}

// RegisterNotificationChannel 注册通知渠道
func (am *AlertManager) RegisterNotificationChannel(channel NotificationChannel) {
	am.notifier.channels[channel.GetName()] = channel
}

// UnregisterNotificationChannel 注销通知渠道
func (am *AlertManager) UnregisterNotificationChannel(channelName string) {
	delete(am.notifier.channels, channelName)
}

// Evaluate 评估条件
func (ce *ConditionEvaluator) Evaluate(condition string, value, threshold interface{}) (bool, error) {
	switch condition {
	case "==", "eq":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", threshold), nil
	case "!=", "ne":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", threshold), nil
	case ">", "gt":
		return ce.compareNumbers(value, threshold, func(a, b float64) bool { return a > b })
	case ">=", "gte":
		return ce.compareNumbers(value, threshold, func(a, b float64) bool { return a >= b })
	case "<", "lt":
		return ce.compareNumbers(value, threshold, func(a, b float64) bool { return a < b })
	case "<=", "lte":
		return ce.compareNumbers(value, threshold, func(a, b float64) bool { return a <= b })
	case "contains":
		valueStr := fmt.Sprintf("%v", value)
		thresholdStr := fmt.Sprintf("%v", threshold)
		return strings.Contains(valueStr, thresholdStr), nil
	case "not_contains":
		valueStr := fmt.Sprintf("%v", value)
		thresholdStr := fmt.Sprintf("%v", threshold)
		return !strings.Contains(valueStr, thresholdStr), nil
	default:
		return false, fmt.Errorf("不支持的条件: %s", condition)
	}
}

// compareNumbers 比较数字
func (ce *ConditionEvaluator) compareNumbers(value, threshold interface{}, compareFn func(float64, float64) bool) (bool, error) {
	valueNum, err := ce.toFloat64(value)
	if err != nil {
		return false, fmt.Errorf("无法将值转换为数字: %v", value)
	}

	thresholdNum, err := ce.toFloat64(threshold)
	if err != nil {
		return false, fmt.Errorf("无法将阈值转换为数字: %v", threshold)
	}

	return compareFn(valueNum, thresholdNum), nil
}

// toFloat64 转换为 float64
func (ce *ConditionEvaluator) toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("无法转换类型 %T 为 float64", value)
	}
}

// 内置通知渠道

// ConsoleNotificationChannel 控制台通知渠道
type ConsoleNotificationChannel struct{}

func (c *ConsoleNotificationChannel) GetName() string {
	return "console"
}

func (c *ConsoleNotificationChannel) SendNotification(alert *Alert) error {
	fmt.Printf("[ALERT] %s - %s (严重程度: %s)\n",
		alert.CreatedAt.Format("2006-01-02 15:04:05"),
		alert.Message,
		alert.Rule.Severity)
	return nil
}

// EmailNotificationChannel 邮件通知渠道
type EmailNotificationChannel struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
	To       []string
}

func (e *EmailNotificationChannel) GetName() string {
	return "email"
}

func (e *EmailNotificationChannel) SendNotification(alert *Alert) error {
	// 这里可以实现真正的邮件发送逻辑
	fmt.Printf("[EMAIL] 发送告警邮件: %s\n", alert.Message)
	return nil
}

// WebhookNotificationChannel Webhook通知渠道
type WebhookNotificationChannel struct {
	URL     string
	Method  string
	Headers map[string]string
}

func (w *WebhookNotificationChannel) GetName() string {
	return "webhook"
}

func (w *WebhookNotificationChannel) SendNotification(alert *Alert) error {
	// 这里可以实现真正的HTTP请求发送逻辑
	fmt.Printf("[WEBHOOK] 发送告警到 %s: %s\n", w.URL, alert.Message)
	return nil
}
