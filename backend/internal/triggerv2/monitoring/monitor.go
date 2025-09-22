package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Monitor 监控系统
type Monitor struct {
	config        *MonitorConfig
	collectors    map[string]MetricsCollector
	alertManager  *AlertManager
	healthChecker *HealthChecker
	metrics       *SystemMetrics
	mu            sync.RWMutex
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	CollectInterval     time.Duration `json:"collect_interval"`
	AlertCheckInterval  time.Duration `json:"alert_check_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	MetricsRetention    time.Duration `json:"metrics_retention"`
	EnableAlerts        bool          `json:"enable_alerts"`
	EnableHealthCheck   bool          `json:"enable_health_check"`
	AlertRules          []*AlertRule  `json:"alert_rules"`
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	GetName() string
	Collect(ctx context.Context) (*MetricSet, error)
	GetInterval() time.Duration
}

// MetricSet 指标集合
type MetricSet struct {
	Name      string                 `json:"name"`
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// SystemMetrics 系统指标
type SystemMetrics struct {
	CollectedAt time.Time                    `json:"collected_at"`
	Collectors  map[string]*CollectorMetrics `json:"collectors"`
	Alerts      []*Alert                     `json:"alerts"`
	Health      *HealthStatus                `json:"health"`
	mu          sync.RWMutex
}

// CollectorMetrics 收集器指标
type CollectorMetrics struct {
	Name            string      `json:"name"`
	LastCollected   time.Time   `json:"last_collected"`
	CollectionCount int64       `json:"collection_count"`
	ErrorCount      int64       `json:"error_count"`
	LastError       string      `json:"last_error,omitempty"`
	Metrics         []MetricSet `json:"metrics"`
	mu              sync.RWMutex
}

// Alert 告警
type Alert struct {
	ID         string      `json:"id"`
	Rule       *AlertRule  `json:"rule"`
	Status     AlertStatus `json:"status"`
	Message    string      `json:"message"`
	Value      interface{} `json:"value"`
	Threshold  interface{} `json:"threshold"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
	ResolvedAt *time.Time  `json:"resolved_at,omitempty"`
	NotifiedAt *time.Time  `json:"notified_at,omitempty"`
	Count      int         `json:"count"`
}

// AlertRule 告警规则
type AlertRule struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Metric         string        `json:"metric"`
	Condition      string        `json:"condition"`
	Threshold      interface{}   `json:"threshold"`
	Duration       time.Duration `json:"duration"`
	Severity       AlertSeverity `json:"severity"`
	Enabled        bool          `json:"enabled"`
	NotifyChannels []string      `json:"notify_channels"`
}

// AlertStatus 告警状态
type AlertStatus string

const (
	AlertStatusPending  AlertStatus = "pending"
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
)

// AlertSeverity 告警严重程度
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// HealthStatus 健康状态
type HealthStatus struct {
	Overall    HealthLevel                 `json:"overall"`
	Components map[string]*ComponentHealth `json:"components"`
	CheckedAt  time.Time                   `json:"checked_at"`
	Message    string                      `json:"message"`
}

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Name      string                 `json:"name"`
	Status    HealthLevel            `json:"status"`
	Message   string                 `json:"message"`
	CheckedAt time.Time              `json:"checked_at"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthLevel 健康级别
type HealthLevel string

const (
	HealthLevelHealthy   HealthLevel = "healthy"
	HealthLevelWarning   HealthLevel = "warning"
	HealthLevelUnhealthy HealthLevel = "unhealthy"
)

// NewMonitor 创建监控系统
func NewMonitor(config *MonitorConfig) *Monitor {
	if config == nil {
		config = &MonitorConfig{
			CollectInterval:     time.Second * 30,
			AlertCheckInterval:  time.Second * 10,
			HealthCheckInterval: time.Second * 60,
			MetricsRetention:    time.Hour * 24,
			EnableAlerts:        true,
			EnableHealthCheck:   true,
			AlertRules:          []*AlertRule{},
		}
	}

	monitor := &Monitor{
		config:     config,
		collectors: make(map[string]MetricsCollector),
		metrics: &SystemMetrics{
			Collectors: make(map[string]*CollectorMetrics),
			Alerts:     make([]*Alert, 0),
			Health: &HealthStatus{
				Overall:    HealthLevelHealthy,
				Components: make(map[string]*ComponentHealth),
				CheckedAt:  time.Now(),
			},
		},
		stopCh: make(chan struct{}),
	}

	monitor.alertManager = NewAlertManager(config.AlertRules)
	monitor.healthChecker = NewHealthChecker()

	return monitor
}

// Start 启动监控系统
func (m *Monitor) Start(ctx context.Context) error {
	// 启动指标收集
	m.wg.Add(1)
	go m.metricsCollectionWorker(ctx)

	// 启动告警检查
	if m.config.EnableAlerts {
		m.wg.Add(1)
		go m.alertCheckWorker(ctx)
	}

	// 启动健康检查
	if m.config.EnableHealthCheck {
		m.wg.Add(1)
		go m.healthCheckWorker(ctx)
	}

	// 启动数据清理
	m.wg.Add(1)
	go m.cleanupWorker(ctx)

	return nil
}

// Stop 停止监控系统
func (m *Monitor) Stop() error {
	close(m.stopCh)
	m.wg.Wait()
	return nil
}

// RegisterCollector 注册指标收集器
func (m *Monitor) RegisterCollector(collector MetricsCollector) {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := collector.GetName()
	m.collectors[name] = collector

	// 初始化收集器指标
	m.metrics.mu.Lock()
	m.metrics.Collectors[name] = &CollectorMetrics{
		Name:    name,
		Metrics: make([]MetricSet, 0),
	}
	m.metrics.mu.Unlock()
}

// UnregisterCollector 注销指标收集器
func (m *Monitor) UnregisterCollector(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.collectors, name)

	m.metrics.mu.Lock()
	delete(m.metrics.Collectors, name)
	m.metrics.mu.Unlock()
}

// GetMetrics 获取系统指标
func (m *Monitor) GetMetrics() *SystemMetrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	// 深拷贝指标数据
	result := &SystemMetrics{
		CollectedAt: m.metrics.CollectedAt,
		Collectors:  make(map[string]*CollectorMetrics),
		Alerts:      make([]*Alert, len(m.metrics.Alerts)),
		Health:      m.copyHealthStatus(m.metrics.Health),
	}

	for name, collector := range m.metrics.Collectors {
		result.Collectors[name] = m.copyCollectorMetrics(collector)
	}

	copy(result.Alerts, m.metrics.Alerts)

	return result
}

// GetCollectorMetrics 获取特定收集器的指标
func (m *Monitor) GetCollectorMetrics(name string) (*CollectorMetrics, error) {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	collector, exists := m.metrics.Collectors[name]
	if !exists {
		return nil, fmt.Errorf("收集器 %s 不存在", name)
	}

	return m.copyCollectorMetrics(collector), nil
}

// GetAlerts 获取告警
func (m *Monitor) GetAlerts() []*Alert {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	alerts := make([]*Alert, len(m.metrics.Alerts))
	copy(alerts, m.metrics.Alerts)
	return alerts
}

// GetHealthStatus 获取健康状态
func (m *Monitor) GetHealthStatus() *HealthStatus {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	return m.copyHealthStatus(m.metrics.Health)
}

// AddAlertRule 添加告警规则
func (m *Monitor) AddAlertRule(rule *AlertRule) {
	m.alertManager.AddRule(rule)
}

// RemoveAlertRule 删除告警规则
func (m *Monitor) RemoveAlertRule(ruleID string) {
	m.alertManager.RemoveRule(ruleID)
}

// metricsCollectionWorker 指标收集工作器
func (m *Monitor) metricsCollectionWorker(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CollectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.collectMetrics(ctx)
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// collectMetrics 收集指标
func (m *Monitor) collectMetrics(ctx context.Context) {
	m.mu.RLock()
	collectors := make(map[string]MetricsCollector)
	for name, collector := range m.collectors {
		collectors[name] = collector
	}
	m.mu.RUnlock()

	// 并发收集指标
	var wg sync.WaitGroup
	for name, collector := range collectors {
		wg.Add(1)
		go func(name string, collector MetricsCollector) {
			defer wg.Done()
			m.collectSingleMetric(ctx, name, collector)
		}(name, collector)
	}

	wg.Wait()

	// 更新收集时间
	m.metrics.mu.Lock()
	m.metrics.CollectedAt = time.Now()
	m.metrics.mu.Unlock()
}

// collectSingleMetric 收集单个指标
func (m *Monitor) collectSingleMetric(ctx context.Context, name string, collector MetricsCollector) {
	metricSet, err := collector.Collect(ctx)

	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	collectorMetrics, exists := m.metrics.Collectors[name]
	if !exists {
		collectorMetrics = &CollectorMetrics{
			Name:    name,
			Metrics: make([]MetricSet, 0),
		}
		m.metrics.Collectors[name] = collectorMetrics
	}

	collectorMetrics.mu.Lock()
	defer collectorMetrics.mu.Unlock()

	collectorMetrics.CollectionCount++
	collectorMetrics.LastCollected = time.Now()

	if err != nil {
		collectorMetrics.ErrorCount++
		collectorMetrics.LastError = err.Error()
		return
	}

	// 添加新的指标数据
	if metricSet != nil {
		collectorMetrics.Metrics = append(collectorMetrics.Metrics, *metricSet)

		// 限制保留的指标数量
		maxMetrics := int(m.config.MetricsRetention / m.config.CollectInterval)
		if len(collectorMetrics.Metrics) > maxMetrics {
			collectorMetrics.Metrics = collectorMetrics.Metrics[len(collectorMetrics.Metrics)-maxMetrics:]
		}
	}

	collectorMetrics.LastError = ""
}

// alertCheckWorker 告警检查工作器
func (m *Monitor) alertCheckWorker(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.AlertCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAlerts(ctx)
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkAlerts 检查告警
func (m *Monitor) checkAlerts(ctx context.Context) {
	alerts := m.alertManager.CheckAlerts(m.metrics)

	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	// 更新告警状态
	for _, alert := range alerts {
		existingAlert := m.findExistingAlert(alert.Rule.ID)
		if existingAlert != nil {
			existingAlert.Status = alert.Status
			existingAlert.Message = alert.Message
			existingAlert.Value = alert.Value
			existingAlert.UpdatedAt = time.Now()
			existingAlert.Count++

			if alert.Status == AlertStatusResolved {
				now := time.Now()
				existingAlert.ResolvedAt = &now
			}
		} else {
			m.metrics.Alerts = append(m.metrics.Alerts, alert)
		}
	}

	// 清理已解决的旧告警
	m.cleanupResolvedAlerts()
}

// findExistingAlert 查找现有告警
func (m *Monitor) findExistingAlert(ruleID string) *Alert {
	for _, alert := range m.metrics.Alerts {
		if alert.Rule.ID == ruleID && alert.Status != AlertStatusResolved {
			return alert
		}
	}
	return nil
}

// cleanupResolvedAlerts 清理已解决的告警
func (m *Monitor) cleanupResolvedAlerts() {
	cutoff := time.Now().Add(-time.Hour) // 保留1小时内的已解决告警

	activeAlerts := make([]*Alert, 0)
	for _, alert := range m.metrics.Alerts {
		if alert.Status != AlertStatusResolved ||
			(alert.ResolvedAt != nil && alert.ResolvedAt.After(cutoff)) {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	m.metrics.Alerts = activeAlerts
}

// healthCheckWorker 健康检查工作器
func (m *Monitor) healthCheckWorker(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkHealth(ctx)
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkHealth 检查健康状态
func (m *Monitor) checkHealth(ctx context.Context) {
	healthStatus := m.healthChecker.CheckHealth(ctx, m.metrics)

	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.Health = healthStatus
}

// cleanupWorker 数据清理工作器
func (m *Monitor) cleanupWorker(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(time.Hour) // 每小时清理一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// cleanup 清理过期数据
func (m *Monitor) cleanup() {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	cutoff := time.Now().Add(-m.config.MetricsRetention)

	// 清理过期的指标数据
	for _, collector := range m.metrics.Collectors {
		collector.mu.Lock()
		activeMetrics := make([]MetricSet, 0)
		for _, metric := range collector.Metrics {
			if metric.Timestamp.After(cutoff) {
				activeMetrics = append(activeMetrics, metric)
			}
		}
		collector.Metrics = activeMetrics
		collector.mu.Unlock()
	}
}

// copyCollectorMetrics 复制收集器指标
func (m *Monitor) copyCollectorMetrics(source *CollectorMetrics) *CollectorMetrics {
	source.mu.RLock()
	defer source.mu.RUnlock()

	result := &CollectorMetrics{
		Name:            source.Name,
		LastCollected:   source.LastCollected,
		CollectionCount: source.CollectionCount,
		ErrorCount:      source.ErrorCount,
		LastError:       source.LastError,
		Metrics:         make([]MetricSet, len(source.Metrics)),
	}

	copy(result.Metrics, source.Metrics)
	return result
}

// copyHealthStatus 复制健康状态
func (m *Monitor) copyHealthStatus(source *HealthStatus) *HealthStatus {
	result := &HealthStatus{
		Overall:    source.Overall,
		Components: make(map[string]*ComponentHealth),
		CheckedAt:  source.CheckedAt,
		Message:    source.Message,
	}

	for name, component := range source.Components {
		result.Components[name] = &ComponentHealth{
			Name:      component.Name,
			Status:    component.Status,
			Message:   component.Message,
			CheckedAt: component.CheckedAt,
			Details:   make(map[string]interface{}),
		}

		// 复制详情
		for k, v := range component.Details {
			result.Components[name].Details[k] = v
		}
	}

	return result
}
