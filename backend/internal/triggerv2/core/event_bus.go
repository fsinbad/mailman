package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"mailman/internal/triggerv2/models"
)

// EventHandler 事件处理器接口
type EventHandler interface {
	Handle(ctx context.Context, event *models.Event) error
	CanHandle(event *models.Event) bool
	GetName() string
	GetPriority() int
}

// EventFilter 事件过滤器接口
type EventFilter interface {
	Filter(event *models.Event) bool
	GetName() string
}

// EventBus 事件总线接口
type EventBus interface {
	// 事件发布
	Publish(ctx context.Context, event *models.Event) error
	PublishAsync(ctx context.Context, event *models.Event) error

	// 事件订阅
	Subscribe(eventType models.EventType, handler EventHandler) error
	SubscribeWithFilter(eventType models.EventType, handler EventHandler, filter EventFilter) error
	Unsubscribe(eventType models.EventType, handlerName string) error

	// 生命周期管理
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// 监控和统计
	GetStats() *EventBusStats
	GetHealth() *EventBusHealth
}

// EventBusStats 事件总线统计信息
type EventBusStats struct {
	TotalEvents     int64                      `json:"total_events"`
	ProcessedEvents int64                      `json:"processed_events"`
	FailedEvents    int64                      `json:"failed_events"`
	EventsByType    map[models.EventType]int64 `json:"events_by_type"`
	HandlerStats    map[string]*HandlerStats   `json:"handler_stats"`
	AverageLatency  time.Duration              `json:"average_latency"`
	LastEventTime   *time.Time                 `json:"last_event_time"`
	QueueSize       int                        `json:"queue_size"`
	ActiveHandlers  int                        `json:"active_handlers"`
}

// HandlerStats 处理器统计信息
type HandlerStats struct {
	Name            string        `json:"name"`
	ProcessedCount  int64         `json:"processed_count"`
	FailedCount     int64         `json:"failed_count"`
	AverageLatency  time.Duration `json:"average_latency"`
	LastProcessTime *time.Time    `json:"last_process_time"`
}

// EventBusHealth 事件总线健康状态
type EventBusHealth struct {
	Status          string    `json:"status"`
	IsRunning       bool      `json:"is_running"`
	QueueCapacity   int       `json:"queue_capacity"`
	QueueSize       int       `json:"queue_size"`
	ActiveWorkers   int       `json:"active_workers"`
	LastHealthCheck time.Time `json:"last_health_check"`
	ErrorRate       float64   `json:"error_rate"`
	Issues          []string  `json:"issues"`
}

// EventBusConfig 事件总线配置
type EventBusConfig struct {
	WorkerCount         int           `json:"worker_count"`
	QueueSize           int           `json:"queue_size"`
	MaxRetries          int           `json:"max_retries"`
	RetryDelay          time.Duration `json:"retry_delay"`
	RetryBackoff        float64       `json:"retry_backoff"`
	ProcessTimeout      time.Duration `json:"process_timeout"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	EnableMetrics       bool          `json:"enable_metrics"`
	EnableTracing       bool          `json:"enable_tracing"`
	PersistEvents       bool          `json:"persist_events"`
}

// DefaultEventBusConfig 默认事件总线配置
func DefaultEventBusConfig() *EventBusConfig {
	return &EventBusConfig{
		WorkerCount:         10,
		QueueSize:           1000,
		MaxRetries:          3,
		RetryDelay:          1 * time.Second,
		RetryBackoff:        2.0,
		ProcessTimeout:      30 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		EnableMetrics:       true,
		EnableTracing:       false,
		PersistEvents:       true,
	}
}

// InMemoryEventBus 内存事件总线实现
type InMemoryEventBus struct {
	config   *EventBusConfig
	handlers map[models.EventType][]HandlerWithFilter
	filters  map[string]EventFilter
	queue    chan *EventTask
	stats    *EventBusStats
	health   *EventBusHealth
	workers  []*EventWorker
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
}

// HandlerWithFilter 带过滤器的处理器
type HandlerWithFilter struct {
	Handler EventHandler
	Filter  EventFilter
}

// EventTask 事件任务
type EventTask struct {
	Event     *models.Event
	Context   context.Context
	Retry     int
	CreatedAt time.Time
}

// EventWorker 事件工作器
type EventWorker struct {
	ID        int
	bus       *InMemoryEventBus
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	mu        sync.RWMutex
}

// NewInMemoryEventBus 创建内存事件总线
func NewInMemoryEventBus(config *EventBusConfig) EventBus {
	if config == nil {
		config = DefaultEventBusConfig()
	}

	return &InMemoryEventBus{
		config:   config,
		handlers: make(map[models.EventType][]HandlerWithFilter),
		filters:  make(map[string]EventFilter),
		queue:    make(chan *EventTask, config.QueueSize),
		stats: &EventBusStats{
			EventsByType: make(map[models.EventType]int64),
			HandlerStats: make(map[string]*HandlerStats),
		},
		health: &EventBusHealth{
			Status:          "stopped",
			IsRunning:       false,
			QueueCapacity:   config.QueueSize,
			LastHealthCheck: time.Now(),
			Issues:          []string{},
		},
		workers: make([]*EventWorker, 0, config.WorkerCount),
	}
}

// Start 启动事件总线
func (bus *InMemoryEventBus) Start(ctx context.Context) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if bus.running {
		return fmt.Errorf("事件总线已经在运行")
	}

	bus.ctx, bus.cancel = context.WithCancel(ctx)

	// 启动工作器
	for i := 0; i < bus.config.WorkerCount; i++ {
		worker := &EventWorker{
			ID:  i,
			bus: bus,
		}
		worker.ctx, worker.cancel = context.WithCancel(bus.ctx)
		bus.workers = append(bus.workers, worker)

		bus.wg.Add(1)
		go worker.run()
	}

	// 启动健康检查
	if bus.config.HealthCheckInterval > 0 {
		bus.wg.Add(1)
		go bus.healthCheck()
	}

	bus.running = true
	bus.health.Status = "running"
	bus.health.IsRunning = true
	bus.health.ActiveWorkers = bus.config.WorkerCount

	log.Printf("[EventBus] 事件总线已启动，工作器数量: %d", bus.config.WorkerCount)
	return nil
}

// Stop 停止事件总线
func (bus *InMemoryEventBus) Stop(ctx context.Context) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if !bus.running {
		return fmt.Errorf("事件总线没有在运行")
	}

	// 停止接收新事件
	close(bus.queue)

	// 取消所有工作器
	bus.cancel()

	// 等待所有工作器完成
	done := make(chan struct{})
	go func() {
		bus.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[EventBus] 事件总线已停止")
	case <-ctx.Done():
		log.Printf("[EventBus] 事件总线停止超时")
		return ctx.Err()
	}

	bus.running = false
	bus.health.Status = "stopped"
	bus.health.IsRunning = false
	bus.health.ActiveWorkers = 0

	return nil
}

// Publish 发布事件（同步）
func (bus *InMemoryEventBus) Publish(ctx context.Context, event *models.Event) error {
	return bus.publishEvent(ctx, event, false)
}

// PublishAsync 异步发布事件
func (bus *InMemoryEventBus) PublishAsync(ctx context.Context, event *models.Event) error {
	return bus.publishEvent(ctx, event, true)
}

// publishEvent 发布事件的内部实现
func (bus *InMemoryEventBus) publishEvent(ctx context.Context, event *models.Event, async bool) error {
	if event == nil {
		return fmt.Errorf("事件不能为空")
	}

	bus.mu.RLock()
	if !bus.running {
		bus.mu.RUnlock()
		return fmt.Errorf("事件总线没有在运行")
	}
	bus.mu.RUnlock()

	// 更新统计信息
	bus.updateStats(event)

	// 创建事件任务
	task := &EventTask{
		Event:     event,
		Context:   ctx,
		Retry:     0,
		CreatedAt: time.Now(),
	}

	// 将事件放入队列
	if async {
		select {
		case bus.queue <- task:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fmt.Errorf("事件队列已满")
		}
	} else {
		select {
		case bus.queue <- task:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Subscribe 订阅事件类型
func (bus *InMemoryEventBus) Subscribe(eventType models.EventType, handler EventHandler) error {
	return bus.SubscribeWithFilter(eventType, handler, nil)
}

// SubscribeWithFilter 带过滤器的事件订阅
func (bus *InMemoryEventBus) SubscribeWithFilter(eventType models.EventType, handler EventHandler, filter EventFilter) error {
	if handler == nil {
		return fmt.Errorf("事件处理器不能为空")
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	handlerWithFilter := HandlerWithFilter{
		Handler: handler,
		Filter:  filter,
	}

	bus.handlers[eventType] = append(bus.handlers[eventType], handlerWithFilter)

	// 初始化处理器统计信息
	if bus.stats.HandlerStats[handler.GetName()] == nil {
		bus.stats.HandlerStats[handler.GetName()] = &HandlerStats{
			Name: handler.GetName(),
		}
	}

	log.Printf("[EventBus] 订阅事件类型: %s, 处理器: %s", eventType, handler.GetName())
	return nil
}

// Unsubscribe 取消订阅
func (bus *InMemoryEventBus) Unsubscribe(eventType models.EventType, handlerName string) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	handlers, exists := bus.handlers[eventType]
	if !exists {
		return fmt.Errorf("事件类型 %s 没有订阅者", eventType)
	}

	for i, handler := range handlers {
		if handler.Handler.GetName() == handlerName {
			bus.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			log.Printf("[EventBus] 取消订阅事件类型: %s, 处理器: %s", eventType, handlerName)
			return nil
		}
	}

	return fmt.Errorf("找不到处理器: %s", handlerName)
}

// GetStats 获取统计信息
func (bus *InMemoryEventBus) GetStats() *EventBusStats {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	stats := *bus.stats
	stats.QueueSize = len(bus.queue)
	stats.ActiveHandlers = len(bus.handlers)

	return &stats
}

// GetHealth 获取健康状态
func (bus *InMemoryEventBus) GetHealth() *EventBusHealth {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	health := *bus.health
	health.QueueSize = len(bus.queue)
	health.LastHealthCheck = time.Now()

	return &health
}

// updateStats 更新统计信息
func (bus *InMemoryEventBus) updateStats(event *models.Event) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.stats.TotalEvents++
	bus.stats.EventsByType[event.Type]++
	now := time.Now()
	bus.stats.LastEventTime = &now
}

// healthCheck 健康检查
func (bus *InMemoryEventBus) healthCheck() {
	defer bus.wg.Done()

	ticker := time.NewTicker(bus.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bus.ctx.Done():
			return
		case <-ticker.C:
			bus.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (bus *InMemoryEventBus) performHealthCheck() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	issues := []string{}

	// 检查队列大小
	queueSize := len(bus.queue)
	if queueSize > bus.config.QueueSize*8/10 {
		issues = append(issues, fmt.Sprintf("队列使用率过高: %d/%d", queueSize, bus.config.QueueSize))
	}

	// 检查工作器状态
	activeWorkers := 0
	for _, worker := range bus.workers {
		worker.mu.RLock()
		if worker.isRunning {
			activeWorkers++
		}
		worker.mu.RUnlock()
	}

	if activeWorkers < bus.config.WorkerCount {
		issues = append(issues, fmt.Sprintf("活跃工作器数量不足: %d/%d", activeWorkers, bus.config.WorkerCount))
	}

	// 计算错误率
	errorRate := 0.0
	if bus.stats.TotalEvents > 0 {
		errorRate = float64(bus.stats.FailedEvents) / float64(bus.stats.TotalEvents) * 100
	}

	if errorRate > 10.0 {
		issues = append(issues, fmt.Sprintf("错误率过高: %.2f%%", errorRate))
	}

	bus.health.QueueSize = queueSize
	bus.health.ActiveWorkers = activeWorkers
	bus.health.ErrorRate = errorRate
	bus.health.Issues = issues
	bus.health.LastHealthCheck = time.Now()

	if len(issues) > 0 {
		bus.health.Status = "unhealthy"
		log.Printf("[EventBus] 健康检查发现问题: %v", issues)
	} else {
		bus.health.Status = "healthy"
	}
}

// run 运行事件工作器
func (worker *EventWorker) run() {
	defer worker.bus.wg.Done()

	worker.mu.Lock()
	worker.isRunning = true
	worker.mu.Unlock()

	log.Printf("[EventBus] 工作器 %d 启动", worker.ID)

	for {
		select {
		case <-worker.ctx.Done():
			log.Printf("[EventBus] 工作器 %d 停止", worker.ID)
			worker.mu.Lock()
			worker.isRunning = false
			worker.mu.Unlock()
			return

		case task, ok := <-worker.bus.queue:
			if !ok {
				log.Printf("[EventBus] 工作器 %d 停止，队列已关闭", worker.ID)
				worker.mu.Lock()
				worker.isRunning = false
				worker.mu.Unlock()
				return
			}

			worker.processEvent(task)
		}
	}
}

// processEvent 处理事件
func (worker *EventWorker) processEvent(task *EventTask) {
	startTime := time.Now()

	// 获取事件处理器
	worker.bus.mu.RLock()
	handlers, exists := worker.bus.handlers[task.Event.Type]
	worker.bus.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		log.Printf("[EventBus] 没有找到事件类型 %s 的处理器", task.Event.Type)
		return
	}

	// 标记事件为处理中
	task.Event.MarkProcessing()

	// 处理事件
	processed := false
	for _, handlerWithFilter := range handlers {
		handler := handlerWithFilter.Handler
		filter := handlerWithFilter.Filter

		// 检查处理器是否能处理该事件
		if !handler.CanHandle(task.Event) {
			continue
		}

		// 应用过滤器
		if filter != nil && !filter.Filter(task.Event) {
			continue
		}

		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(task.Context, worker.bus.config.ProcessTimeout)

		// 处理事件
		err := handler.Handle(ctx, task.Event)
		cancel()

		// 更新处理器统计信息
		worker.updateHandlerStats(handler.GetName(), err, time.Since(startTime))

		if err != nil {
			log.Printf("[EventBus] 处理器 %s 处理事件失败: %v", handler.GetName(), err)

			// 检查是否需要重试
			if task.Retry < worker.bus.config.MaxRetries {
				task.Retry++

				// 计算重试延迟
				delay := time.Duration(float64(worker.bus.config.RetryDelay) *
					float64(task.Retry) * worker.bus.config.RetryBackoff)

				log.Printf("[EventBus] 事件将在 %v 后重试，重试次数: %d", delay, task.Retry)

				// 延迟后重新加入队列
				go func() {
					time.Sleep(delay)
					select {
					case worker.bus.queue <- task:
					case <-worker.ctx.Done():
					}
				}()

				return
			}

			task.Event.MarkFailed()
			worker.bus.mu.Lock()
			worker.bus.stats.FailedEvents++
			worker.bus.mu.Unlock()
		} else {
			processed = true
		}
	}

	if processed {
		task.Event.MarkCompleted()
		worker.bus.mu.Lock()
		worker.bus.stats.ProcessedEvents++
		worker.bus.mu.Unlock()
	}

	// 更新平均延迟
	latency := time.Since(startTime)
	worker.bus.mu.Lock()
	if worker.bus.stats.AverageLatency == 0 {
		worker.bus.stats.AverageLatency = latency
	} else {
		worker.bus.stats.AverageLatency = (worker.bus.stats.AverageLatency + latency) / 2
	}
	worker.bus.mu.Unlock()
}

// updateHandlerStats 更新处理器统计信息
func (worker *EventWorker) updateHandlerStats(handlerName string, err error, latency time.Duration) {
	worker.bus.mu.Lock()
	defer worker.bus.mu.Unlock()

	stats, exists := worker.bus.stats.HandlerStats[handlerName]
	if !exists {
		stats = &HandlerStats{
			Name: handlerName,
		}
		worker.bus.stats.HandlerStats[handlerName] = stats
	}

	if err != nil {
		stats.FailedCount++
	} else {
		stats.ProcessedCount++
	}

	// 更新平均延迟
	if stats.AverageLatency == 0 {
		stats.AverageLatency = latency
	} else {
		stats.AverageLatency = (stats.AverageLatency + latency) / 2
	}

	now := time.Now()
	stats.LastProcessTime = &now
}
