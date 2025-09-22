package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"mailman/internal/triggerv2/models"
)

// SchedulerStatus 调度器状态
type SchedulerStatus int

const (
	SchedulerStopped SchedulerStatus = iota
	SchedulerStarting
	SchedulerRunning
	SchedulerStopping
)

// String 返回状态字符串
func (s SchedulerStatus) String() string {
	switch s {
	case SchedulerStopped:
		return "stopped"
	case SchedulerStarting:
		return "starting"
	case SchedulerRunning:
		return "running"
	case SchedulerStopping:
		return "stopping"
	default:
		return "unknown"
	}
}

// Scheduler 调度器接口
type Scheduler interface {
	// 生命周期管理
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error

	// 状态管理
	GetStatus() SchedulerStatus
	IsRunning() bool
	IsHealthy() bool

	// 事件处理
	ProcessEvent(event *models.Event) error
	ProcessEvents(events []*models.Event) error

	// 触发器管理
	RegisterTrigger(trigger *models.TriggerV2) error
	UnregisterTrigger(triggerID uint) error
	UpdateTrigger(trigger *models.TriggerV2) error
	GetTrigger(triggerID uint) (*models.TriggerV2, error)
	ListTriggers() ([]*models.TriggerV2, error)

	// 任务管理
	SubmitTask(task Task) error
	GetTaskStatus(taskID string) (*TaskResult, error)
	CancelTask(taskID string) error

	// 监控和统计
	GetStats() *SchedulerStats
	GetHealth() *SchedulerHealth
	GetMetrics() *SchedulerMetrics
}

// SchedulerStats 调度器统计信息
type SchedulerStats struct {
	// 基本统计
	Status          SchedulerStatus `json:"status"`
	Uptime          time.Duration   `json:"uptime"`
	TotalEvents     int64           `json:"total_events"`
	ProcessedEvents int64           `json:"processed_events"`
	FailedEvents    int64           `json:"failed_events"`
	TotalTasks      int64           `json:"total_tasks"`
	CompletedTasks  int64           `json:"completed_tasks"`
	FailedTasks     int64           `json:"failed_tasks"`

	// 触发器统计
	RegisteredTriggers int   `json:"registered_triggers"`
	ActiveTriggers     int   `json:"active_triggers"`
	TriggerExecutions  int64 `json:"trigger_executions"`
	TriggerFailures    int64 `json:"trigger_failures"`

	// 组件统计
	EventBusRunning     bool `json:"event_bus_running"`
	ExecutorPoolRunning bool `json:"executor_pool_running"`
	TaskQueueRunning    bool `json:"task_queue_running"`

	// 性能统计
	EventsPerSecond       float64       `json:"events_per_second"`
	TasksPerSecond        float64       `json:"tasks_per_second"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`

	// 时间信息
	StartTime     time.Time  `json:"start_time"`
	LastEventTime *time.Time `json:"last_event_time"`
	LastTaskTime  *time.Time `json:"last_task_time"`
	LastUpdated   time.Time  `json:"last_updated"`
}

// SchedulerHealth 调度器健康状态
type SchedulerHealth struct {
	Status          string          `json:"status"`
	IsHealthy       bool            `json:"is_healthy"`
	ComponentHealth map[string]bool `json:"component_health"`
	Issues          []string        `json:"issues"`
	LastCheck       time.Time       `json:"last_check"`

	// 性能指标
	MemoryUsage    int64   `json:"memory_usage"`
	CPUUsage       float64 `json:"cpu_usage"`
	GoroutineCount int     `json:"goroutine_count"`

	// 阈值检查
	EventQueueUtilization float64 `json:"event_queue_utilization"`
	TaskQueueUtilization  float64 `json:"task_queue_utilization"`
	ExecutorUtilization   float64 `json:"executor_utilization"`
	ErrorRate             float64 `json:"error_rate"`
}

// SchedulerMetrics 调度器指标
type SchedulerMetrics struct {
	// 计数器
	EventsReceived    int64 `json:"events_received"`
	EventsProcessed   int64 `json:"events_processed"`
	EventsDropped     int64 `json:"events_dropped"`
	TasksSubmitted    int64 `json:"tasks_submitted"`
	TasksCompleted    int64 `json:"tasks_completed"`
	TasksRetried      int64 `json:"tasks_retried"`
	TriggerMatches    int64 `json:"trigger_matches"`
	TriggerMismatches int64 `json:"trigger_mismatches"`

	// 时间指标
	EventProcessingTime []time.Duration `json:"event_processing_time"`
	TaskExecutionTime   []time.Duration `json:"task_execution_time"`

	// 错误统计
	ErrorsByType      map[string]int64 `json:"errors_by_type"`
	ErrorsByComponent map[string]int64 `json:"errors_by_component"`

	// 时间戳
	CollectedAt time.Time `json:"collected_at"`
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	// 基本配置
	MaxConcurrency    int           `json:"max_concurrency"`
	EventBufferSize   int           `json:"event_buffer_size"`
	ProcessingTimeout time.Duration `json:"processing_timeout"`
	ShutdownTimeout   time.Duration `json:"shutdown_timeout"`

	// 监控配置
	StatsInterval       time.Duration `json:"stats_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	MetricsInterval     time.Duration `json:"metrics_interval"`

	// 性能配置
	EnableProfiling bool          `json:"enable_profiling"`
	EnableMetrics   bool          `json:"enable_metrics"`
	EnableTracing   bool          `json:"enable_tracing"`
	GCInterval      time.Duration `json:"gc_interval"`

	// 重试配置
	MaxRetries   int           `json:"max_retries"`
	RetryDelay   time.Duration `json:"retry_delay"`
	RetryBackoff float64       `json:"retry_backoff"`
}

// DefaultSchedulerConfig 默认调度器配置
func DefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		MaxConcurrency:      100,
		EventBufferSize:     10000,
		ProcessingTimeout:   30 * time.Second,
		ShutdownTimeout:     60 * time.Second,
		StatsInterval:       30 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		MetricsInterval:     10 * time.Second,
		EnableProfiling:     false,
		EnableMetrics:       true,
		EnableTracing:       false,
		GCInterval:          10 * time.Minute,
		MaxRetries:          3,
		RetryDelay:          1 * time.Second,
		RetryBackoff:        2.0,
	}
}

// TriggerV2Scheduler TriggerV2调度器实现
type TriggerV2Scheduler struct {
	config *SchedulerConfig

	// 状态管理
	status int32
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 触发器管理
	triggers   map[uint]*models.TriggerV2
	triggersMu sync.RWMutex

	// 统计信息
	stats   *SchedulerStats
	health  *SchedulerHealth
	metrics *SchedulerMetrics

	// 同步
	mu sync.RWMutex

	// 计数器
	totalEvents     int64
	processedEvents int64
	failedEvents    int64
	totalTasks      int64
	completedTasks  int64
	failedTasks     int64

	// 时间追踪
	startTime       time.Time
	lastEventTime   *time.Time
	lastTaskTime    *time.Time
	processingTimes []time.Duration
	timesMu         sync.RWMutex

	// 任务结果追踪
	taskResults   map[string]*TaskResult
	taskResultsMu sync.RWMutex

	// 事件通道
	eventChan    chan *models.Event
	shutdownChan chan struct{}

	// 通道状态标志
	eventChanClosed    bool
	shutdownChanClosed bool
	chanMu             sync.Mutex
}

// NewTriggerV2Scheduler 创建TriggerV2调度器
func NewTriggerV2Scheduler(config *SchedulerConfig) Scheduler {
	if config == nil {
		config = DefaultSchedulerConfig()
	}

	scheduler := &TriggerV2Scheduler{
		config:             config,
		triggers:           make(map[uint]*models.TriggerV2),
		taskResults:        make(map[string]*TaskResult),
		processingTimes:    make([]time.Duration, 0, 1000),
		eventChan:          make(chan *models.Event, config.EventBufferSize),
		shutdownChan:       make(chan struct{}),
		eventChanClosed:    false,
		shutdownChanClosed: false,
	}

	// 初始化统计信息
	scheduler.stats = &SchedulerStats{
		Status:    SchedulerStopped,
		StartTime: time.Now(),
	}

	// 初始化健康状态
	scheduler.health = &SchedulerHealth{
		Status:          "stopped",
		IsHealthy:       false,
		ComponentHealth: make(map[string]bool),
		Issues:          []string{},
		LastCheck:       time.Now(),
	}

	// 初始化指标
	scheduler.metrics = &SchedulerMetrics{
		ErrorsByType:      make(map[string]int64),
		ErrorsByComponent: make(map[string]int64),
		CollectedAt:       time.Now(),
	}

	return scheduler
}

// Start 启动调度器
func (s *TriggerV2Scheduler) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&s.status, int32(SchedulerStopped), int32(SchedulerStarting)) {
		return fmt.Errorf("调度器已经在运行或正在启动")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.startTime = time.Now()

	// 重置通道状态标志（重启时需要）
	s.chanMu.Lock()
	if s.eventChanClosed {
		s.eventChan = make(chan *models.Event, s.config.EventBufferSize)
		s.eventChanClosed = false
	}
	if s.shutdownChanClosed {
		s.shutdownChan = make(chan struct{})
		s.shutdownChanClosed = false
	}
	s.chanMu.Unlock()

	log.Printf("[Scheduler] 开始启动调度器...")

	// 启动事件处理器
	s.wg.Add(1)
	go s.eventProcessor()

	// 启动统计更新
	if s.config.StatsInterval > 0 {
		s.wg.Add(1)
		go s.updateStats()
	}

	// 启动健康检查
	if s.config.HealthCheckInterval > 0 {
		s.wg.Add(1)
		go s.healthCheck()
	}

	// 启动指标收集
	if s.config.MetricsInterval > 0 && s.config.EnableMetrics {
		s.wg.Add(1)
		go s.collectMetrics()
	}

	// 启动垃圾回收
	if s.config.GCInterval > 0 {
		s.wg.Add(1)
		go s.gcRoutine()
	}

	// 设置状态为运行中
	atomic.StoreInt32(&s.status, int32(SchedulerRunning))
	s.stats.Status = SchedulerRunning
	s.health.Status = "running"
	s.health.IsHealthy = true

	log.Printf("[Scheduler] 调度器启动成功")
	return nil
}

// Stop 停止调度器
func (s *TriggerV2Scheduler) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&s.status, int32(SchedulerRunning), int32(SchedulerStopping)) {
		return fmt.Errorf("调度器没有在运行")
	}

	log.Printf("[Scheduler] 开始停止调度器...")

	// 安全关闭 channels
	s.chanMu.Lock()

	// 停止接收新事件
	if !s.eventChanClosed {
		close(s.eventChan)
		s.eventChanClosed = true
	}

	// 发送关闭信号
	if !s.shutdownChanClosed {
		close(s.shutdownChan)
		s.shutdownChanClosed = true
	}

	s.chanMu.Unlock()

	// 取消上下文
	if s.cancel != nil {
		s.cancel()
	}

	// 等待所有协程完成
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[Scheduler] 所有协程已停止")
	case <-ctx.Done():
		log.Printf("[Scheduler] 停止超时，强制关闭")
		return ctx.Err()
	}

	// 设置状态为停止
	atomic.StoreInt32(&s.status, int32(SchedulerStopped))
	s.stats.Status = SchedulerStopped
	s.health.Status = "stopped"
	s.health.IsHealthy = false

	log.Printf("[Scheduler] 调度器已停止")
	return nil
}

// Restart 重启调度器
func (s *TriggerV2Scheduler) Restart(ctx context.Context) error {
	if err := s.Stop(ctx); err != nil {
		return fmt.Errorf("停止调度器失败: %w", err)
	}

	// 等待一段时间确保完全停止
	time.Sleep(100 * time.Millisecond)

	if err := s.Start(ctx); err != nil {
		return fmt.Errorf("启动调度器失败: %w", err)
	}

	return nil
}

// GetStatus 获取调度器状态
func (s *TriggerV2Scheduler) GetStatus() SchedulerStatus {
	return SchedulerStatus(atomic.LoadInt32(&s.status))
}

// IsRunning 检查调度器是否在运行
func (s *TriggerV2Scheduler) IsRunning() bool {
	return s.GetStatus() == SchedulerRunning
}

// IsHealthy 检查调度器是否健康
func (s *TriggerV2Scheduler) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.health.IsHealthy
}

// ProcessEvent 处理单个事件
func (s *TriggerV2Scheduler) ProcessEvent(event *models.Event) error {
	if !s.IsRunning() {
		return fmt.Errorf("调度器没有运行")
	}

	if event == nil {
		return fmt.Errorf("事件不能为空")
	}

	// 发送事件到处理通道
	select {
	case s.eventChan <- event:
		atomic.AddInt64(&s.totalEvents, 1)
		now := time.Now()
		s.lastEventTime = &now
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
		return fmt.Errorf("事件缓冲区已满")
	}
}

// ProcessEvents 处理多个事件
func (s *TriggerV2Scheduler) ProcessEvents(events []*models.Event) error {
	if !s.IsRunning() {
		return fmt.Errorf("调度器没有运行")
	}

	var errors []error

	for _, event := range events {
		if err := s.ProcessEvent(event); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("处理事件时发生错误: %v", errors)
	}

	return nil
}

// RegisterTrigger 注册触发器
func (s *TriggerV2Scheduler) RegisterTrigger(trigger *models.TriggerV2) error {
	if trigger == nil {
		return fmt.Errorf("触发器不能为空")
	}

	s.triggersMu.Lock()
	defer s.triggersMu.Unlock()

	s.triggers[trigger.ID] = trigger

	log.Printf("[Scheduler] 注册触发器: %s (ID: %d)", trigger.Name, trigger.ID)
	return nil
}

// UnregisterTrigger 注销触发器
func (s *TriggerV2Scheduler) UnregisterTrigger(triggerID uint) error {
	s.triggersMu.Lock()
	defer s.triggersMu.Unlock()

	if _, exists := s.triggers[triggerID]; !exists {
		return fmt.Errorf("触发器 %d 不存在", triggerID)
	}

	delete(s.triggers, triggerID)

	log.Printf("[Scheduler] 注销触发器: %d", triggerID)
	return nil
}

// UpdateTrigger 更新触发器
func (s *TriggerV2Scheduler) UpdateTrigger(trigger *models.TriggerV2) error {
	if trigger == nil {
		return fmt.Errorf("触发器不能为空")
	}

	s.triggersMu.Lock()
	defer s.triggersMu.Unlock()

	if _, exists := s.triggers[trigger.ID]; !exists {
		return fmt.Errorf("触发器 %d 不存在", trigger.ID)
	}

	s.triggers[trigger.ID] = trigger

	log.Printf("[Scheduler] 更新触发器: %s (ID: %d)", trigger.Name, trigger.ID)
	return nil
}

// GetTrigger 获取触发器
func (s *TriggerV2Scheduler) GetTrigger(triggerID uint) (*models.TriggerV2, error) {
	s.triggersMu.RLock()
	defer s.triggersMu.RUnlock()

	trigger, exists := s.triggers[triggerID]
	if !exists {
		return nil, fmt.Errorf("触发器 %d 不存在", triggerID)
	}

	return trigger, nil
}

// ListTriggers 列出所有触发器
func (s *TriggerV2Scheduler) ListTriggers() ([]*models.TriggerV2, error) {
	s.triggersMu.RLock()
	defer s.triggersMu.RUnlock()

	triggers := make([]*models.TriggerV2, 0, len(s.triggers))
	for _, trigger := range s.triggers {
		triggers = append(triggers, trigger)
	}

	return triggers, nil
}

// SubmitTask 提交任务
func (s *TriggerV2Scheduler) SubmitTask(task Task) error {
	if !s.IsRunning() {
		return fmt.Errorf("调度器没有运行")
	}

	if task == nil {
		return fmt.Errorf("任务不能为空")
	}

	// 这里简化实现，实际应该集成执行器池
	go func() {
		if err := task.Execute(s.ctx); err != nil {
			log.Printf("[Scheduler] 任务执行失败: %v", err)
			atomic.AddInt64(&s.failedTasks, 1)
		} else {
			atomic.AddInt64(&s.completedTasks, 1)
		}
	}()

	atomic.AddInt64(&s.totalTasks, 1)
	now := time.Now()
	s.lastTaskTime = &now

	return nil
}

// GetTaskStatus 获取任务状态
func (s *TriggerV2Scheduler) GetTaskStatus(taskID string) (*TaskResult, error) {
	s.taskResultsMu.RLock()
	defer s.taskResultsMu.RUnlock()

	result, exists := s.taskResults[taskID]
	if !exists {
		return nil, fmt.Errorf("任务 %s 不存在", taskID)
	}

	return result, nil
}

// CancelTask 取消任务
func (s *TriggerV2Scheduler) CancelTask(taskID string) error {
	if !s.IsRunning() {
		return fmt.Errorf("调度器没有运行")
	}

	// 清理任务结果
	s.taskResultsMu.Lock()
	delete(s.taskResults, taskID)
	s.taskResultsMu.Unlock()

	return nil
}

// GetStats 获取统计信息
func (s *TriggerV2Scheduler) GetStats() *SchedulerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := *s.stats
	stats.TotalEvents = atomic.LoadInt64(&s.totalEvents)
	stats.ProcessedEvents = atomic.LoadInt64(&s.processedEvents)
	stats.FailedEvents = atomic.LoadInt64(&s.failedEvents)
	stats.TotalTasks = atomic.LoadInt64(&s.totalTasks)
	stats.CompletedTasks = atomic.LoadInt64(&s.completedTasks)
	stats.FailedTasks = atomic.LoadInt64(&s.failedTasks)

	if !s.startTime.IsZero() {
		stats.Uptime = time.Since(s.startTime)
	}

	// 获取触发器统计
	s.triggersMu.RLock()
	stats.RegisteredTriggers = len(s.triggers)
	activeCount := 0
	for _, trigger := range s.triggers {
		if trigger.Status == models.TriggerV2StatusActive {
			activeCount++
		}
	}
	stats.ActiveTriggers = activeCount
	s.triggersMu.RUnlock()

	// 计算性能指标
	if stats.Uptime > 0 {
		stats.EventsPerSecond = float64(stats.ProcessedEvents) / stats.Uptime.Seconds()
		stats.TasksPerSecond = float64(stats.CompletedTasks) / stats.Uptime.Seconds()
	}

	// 计算平均处理时间
	s.timesMu.RLock()
	if len(s.processingTimes) > 0 {
		var total time.Duration
		for _, t := range s.processingTimes {
			total += t
		}
		stats.AverageProcessingTime = total / time.Duration(len(s.processingTimes))
	}
	s.timesMu.RUnlock()

	stats.LastEventTime = s.lastEventTime
	stats.LastTaskTime = s.lastTaskTime
	stats.LastUpdated = time.Now()

	return &stats
}

// GetHealth 获取健康状态
func (s *TriggerV2Scheduler) GetHealth() *SchedulerHealth {
	s.mu.RLock()
	defer s.mu.RUnlock()

	health := *s.health

	// 获取当前状态
	currentStatus := s.GetStatus()

	// 更新组件健康状态
	health.ComponentHealth["scheduler"] = s.IsRunning()

	// 根据调度器状态设置健康状态
	switch currentStatus {
	case SchedulerStopped:
		health.Status = "stopped"
		health.IsHealthy = false
	case SchedulerStarting:
		health.Status = "starting"
		health.IsHealthy = false
	case SchedulerStopping:
		health.Status = "stopping"
		health.IsHealthy = false
	case SchedulerRunning:
		// 计算整体健康状态
		allHealthy := true
		for _, healthy := range health.ComponentHealth {
			if !healthy {
				allHealthy = false
				break
			}
		}
		health.IsHealthy = allHealthy

		if health.IsHealthy {
			health.Status = "healthy"
		} else {
			health.Status = "unhealthy"
		}
	default:
		health.Status = "unknown"
		health.IsHealthy = false
	}

	health.LastCheck = time.Now()

	return &health
}

// GetMetrics 获取指标
func (s *TriggerV2Scheduler) GetMetrics() *SchedulerMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := *s.metrics
	metrics.EventsReceived = atomic.LoadInt64(&s.totalEvents)
	metrics.EventsProcessed = atomic.LoadInt64(&s.processedEvents)
	metrics.TasksSubmitted = atomic.LoadInt64(&s.totalTasks)
	metrics.TasksCompleted = atomic.LoadInt64(&s.completedTasks)
	metrics.CollectedAt = time.Now()

	return &metrics
}

// 内部方法

// eventProcessor 事件处理器
func (s *TriggerV2Scheduler) eventProcessor() {
	defer s.wg.Done()

	log.Printf("[Scheduler] 事件处理器启动")

	for {
		select {
		case event, ok := <-s.eventChan:
			if !ok {
				log.Printf("[Scheduler] 事件通道已关闭，停止事件处理器")
				return
			}

			s.handleEvent(event)

		case <-s.ctx.Done():
			log.Printf("[Scheduler] 事件处理器收到停止信号")
			return
		}
	}
}

// handleEvent 处理单个事件
func (s *TriggerV2Scheduler) handleEvent(event *models.Event) {
	startTime := time.Now()

	defer func() {
		processingTime := time.Since(startTime)
		s.recordProcessingTime(processingTime)
	}()

	s.triggersMu.RLock()
	triggers := make([]*models.TriggerV2, 0, len(s.triggers))
	for _, trigger := range s.triggers {
		if trigger.Status == models.TriggerV2StatusActive {
			triggers = append(triggers, trigger)
		}
	}
	s.triggersMu.RUnlock()

	// 检查每个触发器
	for _, trigger := range triggers {
		if s.shouldTrigger(trigger, event) {
			if err := s.executeTrigger(trigger, event); err != nil {
				log.Printf("[Scheduler] 执行触发器失败: %v", err)
				atomic.AddInt64(&s.failedEvents, 1)
				s.recordError("trigger_execution", err)
			}
		}
	}

	atomic.AddInt64(&s.processedEvents, 1)
}

// shouldTrigger 检查是否应该触发
func (s *TriggerV2Scheduler) shouldTrigger(trigger *models.TriggerV2, event *models.Event) bool {
	// 简化的触发器匹配逻辑 - 实际应该根据trigger的条件来判断
	// 这里假设所有启用的触发器都会被触发，实际实现需要更复杂的条件匹配
	return true
}

// executeTrigger 执行触发器
func (s *TriggerV2Scheduler) executeTrigger(trigger *models.TriggerV2, event *models.Event) error {
	// 创建触发器任务
	task := &TriggerTask{
		TriggerID:  trigger.ID,
		Event:      event,
		Actions:    trigger.Actions,
		retryCount: 0,
		createdAt:  time.Now(),
		lastError:  nil,
	}

	// 提交任务
	return s.SubmitTask(task)
}

// updateStats 更新统计信息
func (s *TriggerV2Scheduler) updateStats() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.StatsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performStatsUpdate()

		case <-s.ctx.Done():
			return
		}
	}
}

// performStatsUpdate 执行统计更新
func (s *TriggerV2Scheduler) performStatsUpdate() {
	// 更新统计信息的逻辑
	s.mu.Lock()
	s.stats.LastUpdated = time.Now()
	s.mu.Unlock()
}

// healthCheck 健康检查
func (s *TriggerV2Scheduler) healthCheck() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performHealthCheck()

		case <-s.ctx.Done():
			return
		}
	}
}

// performHealthCheck 执行健康检查
func (s *TriggerV2Scheduler) performHealthCheck() {
	issues := []string{}

	// 检查事件处理延迟
	if s.lastEventTime != nil && time.Since(*s.lastEventTime) > 5*time.Minute {
		issues = append(issues, "事件处理延迟过长")
	}

	// 检查任务处理延迟
	if s.lastTaskTime != nil && time.Since(*s.lastTaskTime) > 10*time.Minute {
		issues = append(issues, "任务处理延迟过长")
	}

	// 检查错误率
	totalEvents := atomic.LoadInt64(&s.totalEvents)
	failedEvents := atomic.LoadInt64(&s.failedEvents)
	if totalEvents > 0 {
		errorRate := float64(failedEvents) / float64(totalEvents) * 100
		if errorRate > 10 {
			issues = append(issues, fmt.Sprintf("事件错误率过高: %.1f%%", errorRate))
		}
	}

	s.mu.Lock()
	s.health.Issues = issues
	s.health.LastCheck = time.Now()
	s.mu.Unlock()
}

// collectMetrics 收集指标
func (s *TriggerV2Scheduler) collectMetrics() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performMetricsCollection()

		case <-s.ctx.Done():
			return
		}
	}
}

// performMetricsCollection 执行指标收集
func (s *TriggerV2Scheduler) performMetricsCollection() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics.CollectedAt = time.Now()
	// 这里可以添加更多指标收集逻辑
}

// gcRoutine 垃圾回收
func (s *TriggerV2Scheduler) gcRoutine() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.GCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performGC()

		case <-s.ctx.Done():
			return
		}
	}
}

// performGC 执行垃圾回收
func (s *TriggerV2Scheduler) performGC() {
	// 清理过期的处理时间记录
	s.timesMu.Lock()
	if len(s.processingTimes) > 1000 {
		s.processingTimes = s.processingTimes[len(s.processingTimes)-1000:]
	}
	s.timesMu.Unlock()

	// 清理过期的任务结果
	s.cleanupTaskResults()
}

// cleanupTaskResults 清理过期的任务结果
func (s *TriggerV2Scheduler) cleanupTaskResults() {
	s.taskResultsMu.Lock()
	defer s.taskResultsMu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)

	for taskID, result := range s.taskResults {
		if result.EndTime.Before(cutoff) {
			delete(s.taskResults, taskID)
		}
	}
}

// recordProcessingTime 记录处理时间
func (s *TriggerV2Scheduler) recordProcessingTime(duration time.Duration) {
	s.timesMu.Lock()
	defer s.timesMu.Unlock()

	s.processingTimes = append(s.processingTimes, duration)
	if len(s.processingTimes) > 2000 {
		s.processingTimes = s.processingTimes[1000:]
	}
}

// recordError 记录错误
func (s *TriggerV2Scheduler) recordError(errorType string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics.ErrorsByType[errorType]++

	if err != nil {
		s.metrics.ErrorsByType[err.Error()]++
	}
}

// TriggerTask 触发器任务
type TriggerTask struct {
	TriggerID  uint                  `json:"trigger_id"`
	Event      *models.Event         `json:"event"`
	Actions    []models.ActionConfig `json:"actions"`
	retryCount int                   `json:"retry_count"`
	createdAt  time.Time             `json:"created_at"`
	lastError  error                 `json:"-"`
}

// Execute 执行触发器任务
func (t *TriggerTask) Execute(ctx context.Context) error {
	// 执行触发器动作
	for _, action := range t.Actions {
		if err := t.executeAction(ctx, action); err != nil {
			return fmt.Errorf("执行动作失败: %w", err)
		}
	}

	return nil
}

// executeAction 执行动作
func (t *TriggerTask) executeAction(ctx context.Context, action models.ActionConfig) error {
	// 这里实现具体的动作执行逻辑
	log.Printf("[TriggerTask] 执行动作: %s", action.Type)

	// 模拟动作执行
	time.Sleep(100 * time.Millisecond)

	return nil
}

// GetID 获取任务ID
func (t *TriggerTask) GetID() string {
	return fmt.Sprintf("trigger-%d-%s", t.TriggerID, t.Event.ID)
}

// GetType 获取任务类型
func (t *TriggerTask) GetType() string {
	return "trigger"
}

// GetPriority 获取任务优先级
func (t *TriggerTask) GetPriority() int {
	return 2 // 普通优先级
}

// GetCreatedAt 获取创建时间
func (t *TriggerTask) GetCreatedAt() time.Time {
	return t.createdAt
}

// GetRetryCount 获取重试次数
func (t *TriggerTask) GetRetryCount() int {
	return t.retryCount
}

// IncrementRetry 增加重试次数
func (t *TriggerTask) IncrementRetry() {
	t.retryCount++
}

// CanRetry 检查是否可以重试
func (t *TriggerTask) CanRetry() bool {
	return t.GetRetryCount() < t.GetMaxRetries()
}

// GetMaxRetries 获取最大重试次数
func (t *TriggerTask) GetMaxRetries() int {
	return 3
}

// SetError 设置错误
func (t *TriggerTask) SetError(err error) {
	t.lastError = err
}

// GetError 获取错误
func (t *TriggerTask) GetError() error {
	return t.lastError
}
