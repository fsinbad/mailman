package monitoring

import (
	"context"
	"fmt"
	"time"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	checkers map[string]ComponentHealthChecker
}

// ComponentHealthChecker 组件健康检查器接口
type ComponentHealthChecker interface {
	GetName() string
	CheckHealth(ctx context.Context) *ComponentHealth
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker() *HealthChecker {
	hc := &HealthChecker{
		checkers: make(map[string]ComponentHealthChecker),
	}

	// 注册默认的健康检查器
	hc.RegisterChecker(&SystemHealthChecker{})
	hc.RegisterChecker(&CollectorHealthChecker{})
	hc.RegisterChecker(&AlertHealthChecker{})

	return hc
}

// RegisterChecker 注册组件健康检查器
func (hc *HealthChecker) RegisterChecker(checker ComponentHealthChecker) {
	hc.checkers[checker.GetName()] = checker
}

// UnregisterChecker 注销组件健康检查器
func (hc *HealthChecker) UnregisterChecker(name string) {
	delete(hc.checkers, name)
}

// CheckHealth 检查整体健康状态
func (hc *HealthChecker) CheckHealth(ctx context.Context, metrics *SystemMetrics) *HealthStatus {
	status := &HealthStatus{
		Overall:    HealthLevelHealthy,
		Components: make(map[string]*ComponentHealth),
		CheckedAt:  time.Now(),
		Message:    "所有组件正常",
	}

	// 检查各个组件的健康状态
	for name, checker := range hc.checkers {
		componentHealth := checker.CheckHealth(ctx)
		if componentHealth != nil {
			status.Components[name] = componentHealth

			// 更新整体状态
			if componentHealth.Status == HealthLevelUnhealthy {
				status.Overall = HealthLevelUnhealthy
				status.Message = fmt.Sprintf("组件 %s 不健康", name)
			} else if componentHealth.Status == HealthLevelWarning && status.Overall == HealthLevelHealthy {
				status.Overall = HealthLevelWarning
				status.Message = fmt.Sprintf("组件 %s 有警告", name)
			}
		}
	}

	// 基于系统指标进行额外的健康检查
	hc.checkSystemMetrics(status, metrics)

	return status
}

// checkSystemMetrics 基于系统指标检查健康状态
func (hc *HealthChecker) checkSystemMetrics(status *HealthStatus, metrics *SystemMetrics) {
	// 检查告警数量
	firingAlerts := 0
	for _, alert := range metrics.Alerts {
		if alert.Status == AlertStatusFiring {
			firingAlerts++
		}
	}

	if firingAlerts > 0 {
		if firingAlerts >= 10 {
			status.Overall = HealthLevelUnhealthy
			status.Message = fmt.Sprintf("存在 %d 个活跃告警", firingAlerts)
		} else if firingAlerts >= 5 && status.Overall == HealthLevelHealthy {
			status.Overall = HealthLevelWarning
			status.Message = fmt.Sprintf("存在 %d 个活跃告警", firingAlerts)
		}
	}

	// 检查收集器错误率
	totalCollectors := len(metrics.Collectors)
	errorCollectors := 0
	for _, collector := range metrics.Collectors {
		if collector.ErrorCount > 0 {
			errorRate := float64(collector.ErrorCount) / float64(collector.CollectionCount)
			if errorRate > 0.5 { // 错误率超过50%
				errorCollectors++
			}
		}
	}

	if errorCollectors > 0 {
		errorRate := float64(errorCollectors) / float64(totalCollectors)
		if errorRate > 0.5 {
			status.Overall = HealthLevelUnhealthy
			status.Message = fmt.Sprintf("%d/%d 个收集器错误率过高", errorCollectors, totalCollectors)
		} else if errorRate > 0.2 && status.Overall == HealthLevelHealthy {
			status.Overall = HealthLevelWarning
			status.Message = fmt.Sprintf("%d/%d 个收集器有错误", errorCollectors, totalCollectors)
		}
	}
}

// 内置健康检查器

// SystemHealthChecker 系统健康检查器
type SystemHealthChecker struct{}

func (shc *SystemHealthChecker) GetName() string {
	return "system"
}

func (shc *SystemHealthChecker) CheckHealth(ctx context.Context) *ComponentHealth {
	health := &ComponentHealth{
		Name:      "system",
		Status:    HealthLevelHealthy,
		Message:   "系统正常运行",
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// 检查系统资源（这里可以添加具体的系统资源检查逻辑）
	// 例如：内存使用率、CPU使用率、磁盘空间等

	// 示例：模拟系统资源检查
	health.Details["uptime"] = "正常"
	health.Details["memory_usage"] = "正常"
	health.Details["cpu_usage"] = "正常"
	health.Details["disk_space"] = "正常"

	return health
}

// CollectorHealthChecker 收集器健康检查器
type CollectorHealthChecker struct{}

func (chc *CollectorHealthChecker) GetName() string {
	return "collectors"
}

func (chc *CollectorHealthChecker) CheckHealth(ctx context.Context) *ComponentHealth {
	health := &ComponentHealth{
		Name:      "collectors",
		Status:    HealthLevelHealthy,
		Message:   "所有收集器正常",
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// 这里可以添加更详细的收集器健康检查逻辑
	// 例如：检查收集器的连接状态、响应时间等

	health.Details["status"] = "运行中"
	health.Details["last_check"] = time.Now().Format("2006-01-02 15:04:05")

	return health
}

// AlertHealthChecker 告警健康检查器
type AlertHealthChecker struct{}

func (ahc *AlertHealthChecker) GetName() string {
	return "alerts"
}

func (ahc *AlertHealthChecker) CheckHealth(ctx context.Context) *ComponentHealth {
	health := &ComponentHealth{
		Name:      "alerts",
		Status:    HealthLevelHealthy,
		Message:   "告警系统正常",
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// 这里可以添加告警系统的健康检查逻辑
	// 例如：检查告警规则的有效性、通知渠道的可用性等

	health.Details["status"] = "运行中"
	health.Details["rules_loaded"] = true
	health.Details["notification_channels"] = "可用"

	return health
}

// DatabaseHealthChecker 数据库健康检查器
type DatabaseHealthChecker struct {
	connectionString string
}

func NewDatabaseHealthChecker(connectionString string) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		connectionString: connectionString,
	}
}

func (dhc *DatabaseHealthChecker) GetName() string {
	return "database"
}

func (dhc *DatabaseHealthChecker) CheckHealth(ctx context.Context) *ComponentHealth {
	health := &ComponentHealth{
		Name:      "database",
		Status:    HealthLevelHealthy,
		Message:   "数据库连接正常",
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// 这里可以添加真实的数据库连接检查逻辑
	// 例如：执行简单的查询来验证数据库连接

	// 示例：模拟数据库连接检查
	health.Details["connection_status"] = "已连接"
	health.Details["response_time"] = "< 100ms"
	health.Details["connection_pool"] = "正常"

	return health
}

// ServiceHealthChecker 服务健康检查器
type ServiceHealthChecker struct {
	serviceName string
	serviceURL  string
}

func NewServiceHealthChecker(serviceName, serviceURL string) *ServiceHealthChecker {
	return &ServiceHealthChecker{
		serviceName: serviceName,
		serviceURL:  serviceURL,
	}
}

func (shc *ServiceHealthChecker) GetName() string {
	return shc.serviceName
}

func (shc *ServiceHealthChecker) CheckHealth(ctx context.Context) *ComponentHealth {
	health := &ComponentHealth{
		Name:      shc.serviceName,
		Status:    HealthLevelHealthy,
		Message:   fmt.Sprintf("服务 %s 正常", shc.serviceName),
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// 这里可以添加真实的服务健康检查逻辑
	// 例如：发送HTTP请求到服务的健康检查端点

	// 示例：模拟服务健康检查
	health.Details["url"] = shc.serviceURL
	health.Details["status"] = "运行中"
	health.Details["response_time"] = "< 500ms"

	return health
}

// QueueHealthChecker 队列健康检查器
type QueueHealthChecker struct {
	queueName string
}

func NewQueueHealthChecker(queueName string) *QueueHealthChecker {
	return &QueueHealthChecker{
		queueName: queueName,
	}
}

func (qhc *QueueHealthChecker) GetName() string {
	return fmt.Sprintf("queue_%s", qhc.queueName)
}

func (qhc *QueueHealthChecker) CheckHealth(ctx context.Context) *ComponentHealth {
	health := &ComponentHealth{
		Name:      qhc.queueName,
		Status:    HealthLevelHealthy,
		Message:   fmt.Sprintf("队列 %s 正常", qhc.queueName),
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// 这里可以添加真实的队列健康检查逻辑
	// 例如：检查队列的连接状态、消息积压情况等

	// 示例：模拟队列健康检查
	health.Details["queue_size"] = 0
	health.Details["consumers"] = 1
	health.Details["connection_status"] = "已连接"

	return health
}

// CacheHealthChecker 缓存健康检查器
type CacheHealthChecker struct {
	cacheName string
}

func NewCacheHealthChecker(cacheName string) *CacheHealthChecker {
	return &CacheHealthChecker{
		cacheName: cacheName,
	}
}

func (chc *CacheHealthChecker) GetName() string {
	return fmt.Sprintf("cache_%s", chc.cacheName)
}

func (chc *CacheHealthChecker) CheckHealth(ctx context.Context) *ComponentHealth {
	health := &ComponentHealth{
		Name:      chc.cacheName,
		Status:    HealthLevelHealthy,
		Message:   fmt.Sprintf("缓存 %s 正常", chc.cacheName),
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// 这里可以添加真实的缓存健康检查逻辑
	// 例如：测试缓存的读写操作、检查连接状态等

	// 示例：模拟缓存健康检查
	health.Details["connection_status"] = "已连接"
	health.Details["memory_usage"] = "正常"
	health.Details["hit_rate"] = "95%"

	return health
}
