package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Task 任务接口
type Task interface {
	Execute(ctx context.Context) error
	GetID() string
	GetType() string
	GetPriority() int
	GetCreatedAt() time.Time
	GetRetryCount() int
	IncrementRetry()
	CanRetry() bool
	GetMaxRetries() int
	SetError(err error)
	GetError() error
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID        string        `json:"task_id"`
	TaskType      string        `json:"task_type"`
	Success       bool          `json:"success"`
	Error         error         `json:"error,omitempty"`
	ExecutionTime time.Duration `json:"execution_time"`
	WorkerID      int           `json:"worker_id"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	RetryCount    int           `json:"retry_count"`
}

// TaskExecutor 任务执行器接口
type TaskExecutor interface {
	Execute(ctx context.Context, task Task) *TaskResult
	GetID() int
	GetStats() *ExecutorStats
	IsActive() bool
}

// ExecutorPool 执行器池接口
type ExecutorPool interface {
	// 生命周期管理
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// 任务提交
	Submit(task Task) error
	SubmitWithCallback(task Task, callback func(*TaskResult)) error

	// 池管理
	Resize(newSize int) error
	GetSize() int
	GetActiveWorkers() int

	// 监控和统计
	GetStats() *PoolStats
	GetHealth() *PoolHealth

	// 任务管理
	GetPendingTasks() int
	GetRunningTasks() int
	CancelTask(taskID string) error
}

// ExecutorStats 执行器统计信息
type ExecutorStats struct {
	ID             int           `json:"id"`
	TasksProcessed int64         `json:"tasks_processed"`
	TasksSucceeded int64         `json:"tasks_succeeded"`
	TasksFailed    int64         `json:"tasks_failed"`
	AverageTime    time.Duration `json:"average_time"`
	LastTaskTime   *time.Time    `json:"last_task_time"`
	IsActive       bool          `json:"is_active"`
	CreatedAt      time.Time     `json:"created_at"`
}

// PoolStats 池统计信息
type PoolStats struct {
	PoolSize            int                    `json:"pool_size"`
	ActiveWorkers       int                    `json:"active_workers"`
	PendingTasks        int                    `json:"pending_tasks"`
	RunningTasks        int                    `json:"running_tasks"`
	TotalTasks          int64                  `json:"total_tasks"`
	CompletedTasks      int64                  `json:"completed_tasks"`
	FailedTasks         int64                  `json:"failed_tasks"`
	AverageTime         time.Duration          `json:"average_time"`
	ThroughputPerSecond float64                `json:"throughput_per_second"`
	ExecutorStats       map[int]*ExecutorStats `json:"executor_stats"`
	CreatedAt           time.Time              `json:"created_at"`
	LastUpdated         time.Time              `json:"last_updated"`
}

// PoolHealth 池健康状态
type PoolHealth struct {
	Status           string        `json:"status"`
	IsRunning        bool          `json:"is_running"`
	QueueCapacity    int           `json:"queue_capacity"`
	QueueSize        int           `json:"queue_size"`
	QueueUtilization float64       `json:"queue_utilization"`
	ActiveWorkers    int           `json:"active_workers"`
	IdleWorkers      int           `json:"idle_workers"`
	ErrorRate        float64       `json:"error_rate"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	Issues           []string      `json:"issues"`
	LastCheck        time.Time     `json:"last_check"`
}

// PoolConfig 执行器池配置
type PoolConfig struct {
	// 基本配置
	PoolSize    int           `json:"pool_size"`
	QueueSize   int           `json:"queue_size"`
	TaskTimeout time.Duration `json:"task_timeout"`
	IdleTimeout time.Duration `json:"idle_timeout"`

	// 重试配置
	MaxRetries   int           `json:"max_retries"`
	RetryDelay   time.Duration `json:"retry_delay"`
	RetryBackoff float64       `json:"retry_backoff"`

	// 监控配置
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	StatsInterval       time.Duration `json:"stats_interval"`
	EnableMetrics       bool          `json:"enable_metrics"`

	// 性能配置
	EnableProfiling bool          `json:"enable_profiling"`
	GCInterval      time.Duration `json:"gc_interval"`
}

// DefaultPoolConfig 默认配置
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		PoolSize:            10,
		QueueSize:           1000,
		TaskTimeout:         30 * time.Second,
		IdleTimeout:         5 * time.Minute,
		MaxRetries:          3,
		RetryDelay:          1 * time.Second,
		RetryBackoff:        2.0,
		HealthCheckInterval: 30 * time.Second,
		StatsInterval:       10 * time.Second,
		EnableMetrics:       true,
		EnableProfiling:     false,
		GCInterval:          10 * time.Minute,
	}
}

// WorkerExecutorPool 工作器执行器池实现
type WorkerExecutorPool struct {
	config    *PoolConfig
	taskQueue chan *TaskWrapper
	executors []*WorkerExecutor
	stats     *PoolStats
	health    *PoolHealth
	callbacks map[string]func(*TaskResult)

	// 状态管理
	running int32
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// 同步
	mu         sync.RWMutex
	callbackMu sync.RWMutex

	// 计数器
	totalTasks     int64
	completedTasks int64
	failedTasks    int64

	// 时间追踪
	createdAt     time.Time
	lastStatsTime time.Time
	taskTimes     []time.Duration
	timesMu       sync.RWMutex
}

// TaskWrapper 任务包装器
type TaskWrapper struct {
	Task        Task
	Callback    func(*TaskResult)
	SubmittedAt time.Time
}

// WorkerExecutor 工作器执行器
type WorkerExecutor struct {
	id     int
	pool   *WorkerExecutorPool
	stats  *ExecutorStats
	active int32
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// NewWorkerExecutorPool 创建工作器执行器池
func NewWorkerExecutorPool(config *PoolConfig) ExecutorPool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	pool := &WorkerExecutorPool{
		config:        config,
		taskQueue:     make(chan *TaskWrapper, config.QueueSize),
		executors:     make([]*WorkerExecutor, 0, config.PoolSize),
		callbacks:     make(map[string]func(*TaskResult)),
		createdAt:     time.Now(),
		lastStatsTime: time.Now(),
		taskTimes:     make([]time.Duration, 0, 1000),
	}

	// 初始化统计信息
	pool.stats = &PoolStats{
		PoolSize:      config.PoolSize,
		ExecutorStats: make(map[int]*ExecutorStats),
		CreatedAt:     pool.createdAt,
	}

	// 初始化健康状态
	pool.health = &PoolHealth{
		Status:        "stopped",
		IsRunning:     false,
		QueueCapacity: config.QueueSize,
		Issues:        []string{},
		LastCheck:     time.Now(),
	}

	return pool
}

// Start 启动执行器池
func (p *WorkerExecutorPool) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		return fmt.Errorf("执行器池已经在运行")
	}

	p.ctx, p.cancel = context.WithCancel(ctx)

	// 创建工作器执行器
	for i := 0; i < p.config.PoolSize; i++ {
		executor := &WorkerExecutor{
			id:   i,
			pool: p,
			stats: &ExecutorStats{
				ID:        i,
				CreatedAt: time.Now(),
			},
		}
		executor.ctx, executor.cancel = context.WithCancel(p.ctx)

		p.executors = append(p.executors, executor)
		p.stats.ExecutorStats[i] = executor.stats

		p.wg.Add(1)
		go executor.run()
	}

	// 启动健康检查
	if p.config.HealthCheckInterval > 0 {
		p.wg.Add(1)
		go p.healthCheck()
	}

	// 启动统计更新
	if p.config.StatsInterval > 0 {
		p.wg.Add(1)
		go p.updateStats()
	}

	// 启动垃圾回收
	if p.config.GCInterval > 0 {
		p.wg.Add(1)
		go p.gcRoutine()
	}

	p.health.Status = "running"
	p.health.IsRunning = true
	p.health.ActiveWorkers = p.config.PoolSize

	log.Printf("[ExecutorPool] 执行器池已启动，工作器数量: %d", p.config.PoolSize)
	return nil
}

// Stop 停止执行器池
func (p *WorkerExecutorPool) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		return fmt.Errorf("执行器池没有在运行")
	}

	// 停止接收新任务
	close(p.taskQueue)

	// 取消所有执行器
	p.cancel()

	// 等待所有协程完成
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[ExecutorPool] 执行器池已停止")
	case <-ctx.Done():
		log.Printf("[ExecutorPool] 执行器池停止超时")
		return ctx.Err()
	}

	p.health.Status = "stopped"
	p.health.IsRunning = false
	p.health.ActiveWorkers = 0

	return nil
}

// Submit 提交任务
func (p *WorkerExecutorPool) Submit(task Task) error {
	return p.SubmitWithCallback(task, nil)
}

// SubmitWithCallback 提交任务并指定回调
func (p *WorkerExecutorPool) SubmitWithCallback(task Task, callback func(*TaskResult)) error {
	if atomic.LoadInt32(&p.running) == 0 {
		return fmt.Errorf("执行器池没有在运行")
	}

	if task == nil {
		return fmt.Errorf("任务不能为空")
	}

	wrapper := &TaskWrapper{
		Task:        task,
		Callback:    callback,
		SubmittedAt: time.Now(),
	}

	// 如果有回调函数，保存它
	if callback != nil {
		p.callbackMu.Lock()
		p.callbacks[task.GetID()] = callback
		p.callbackMu.Unlock()
	}

	// 提交任务到队列
	select {
	case p.taskQueue <- wrapper:
		atomic.AddInt64(&p.totalTasks, 1)
		return nil
	default:
		return fmt.Errorf("任务队列已满")
	}
}

// Resize 调整池大小
func (p *WorkerExecutorPool) Resize(newSize int) error {
	if newSize <= 0 {
		return fmt.Errorf("池大小必须大于0")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	currentSize := len(p.executors)
	if newSize == currentSize {
		return nil
	}

	if newSize > currentSize {
		// 增加执行器
		for i := currentSize; i < newSize; i++ {
			executor := &WorkerExecutor{
				id:   i,
				pool: p,
				stats: &ExecutorStats{
					ID:        i,
					CreatedAt: time.Now(),
				},
			}
			executor.ctx, executor.cancel = context.WithCancel(p.ctx)

			p.executors = append(p.executors, executor)
			p.stats.ExecutorStats[i] = executor.stats

			p.wg.Add(1)
			go executor.run()
		}
	} else {
		// 减少执行器
		for i := newSize; i < currentSize; i++ {
			p.executors[i].cancel()
			delete(p.stats.ExecutorStats, i)
		}
		p.executors = p.executors[:newSize]
	}

	p.config.PoolSize = newSize
	p.stats.PoolSize = newSize

	log.Printf("[ExecutorPool] 池大小已调整为: %d", newSize)
	return nil
}

// GetSize 获取池大小
func (p *WorkerExecutorPool) GetSize() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.executors)
}

// GetActiveWorkers 获取活跃工作器数量
func (p *WorkerExecutorPool) GetActiveWorkers() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	active := 0
	for _, executor := range p.executors {
		if executor.IsActive() {
			active++
		}
	}
	return active
}

// GetPendingTasks 获取待处理任务数量
func (p *WorkerExecutorPool) GetPendingTasks() int {
	return len(p.taskQueue)
}

// GetRunningTasks 获取运行中任务数量
func (p *WorkerExecutorPool) GetRunningTasks() int {
	return p.GetActiveWorkers()
}

// CancelTask 取消任务
func (p *WorkerExecutorPool) CancelTask(taskID string) error {
	// 移除回调
	p.callbackMu.Lock()
	delete(p.callbacks, taskID)
	p.callbackMu.Unlock()

	// 注意：这里无法取消已经在执行的任务，只能移除回调
	// 如果需要取消正在执行的任务，需要在Task接口中添加Cancel方法
	return nil
}

// GetStats 获取统计信息
func (p *WorkerExecutorPool) GetStats() *PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := *p.stats
	stats.PendingTasks = len(p.taskQueue)
	stats.RunningTasks = p.GetActiveWorkers()
	stats.TotalTasks = atomic.LoadInt64(&p.totalTasks)
	stats.CompletedTasks = atomic.LoadInt64(&p.completedTasks)
	stats.FailedTasks = atomic.LoadInt64(&p.failedTasks)
	stats.ActiveWorkers = p.GetActiveWorkers()
	stats.LastUpdated = time.Now()

	// 计算平均时间
	p.timesMu.RLock()
	if len(p.taskTimes) > 0 {
		var total time.Duration
		for _, t := range p.taskTimes {
			total += t
		}
		stats.AverageTime = total / time.Duration(len(p.taskTimes))
	}
	p.timesMu.RUnlock()

	// 计算吞吐量
	elapsed := time.Since(p.lastStatsTime)
	if elapsed > 0 {
		stats.ThroughputPerSecond = float64(stats.CompletedTasks) / elapsed.Seconds()
	}

	return &stats
}

// GetHealth 获取健康状态
func (p *WorkerExecutorPool) GetHealth() *PoolHealth {
	p.mu.RLock()
	defer p.mu.RUnlock()

	health := *p.health
	health.QueueSize = len(p.taskQueue)
	health.QueueUtilization = float64(health.QueueSize) / float64(health.QueueCapacity) * 100
	health.ActiveWorkers = p.GetActiveWorkers()
	health.IdleWorkers = p.GetSize() - health.ActiveWorkers
	health.LastCheck = time.Now()

	// 计算错误率
	totalTasks := atomic.LoadInt64(&p.totalTasks)
	failedTasks := atomic.LoadInt64(&p.failedTasks)
	if totalTasks > 0 {
		health.ErrorRate = float64(failedTasks) / float64(totalTasks) * 100
	}

	// 计算平均响应时间
	p.timesMu.RLock()
	if len(p.taskTimes) > 0 {
		var total time.Duration
		for _, t := range p.taskTimes {
			total += t
		}
		health.AvgResponseTime = total / time.Duration(len(p.taskTimes))
	}
	p.timesMu.RUnlock()

	return &health
}

// healthCheck 健康检查
func (p *WorkerExecutorPool) healthCheck() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (p *WorkerExecutorPool) performHealthCheck() {
	p.mu.Lock()
	defer p.mu.Unlock()

	issues := []string{}

	// 检查队列使用率
	queueSize := len(p.taskQueue)
	utilization := float64(queueSize) / float64(p.config.QueueSize) * 100
	if utilization > 80 {
		issues = append(issues, fmt.Sprintf("队列使用率过高: %.1f%%", utilization))
	}

	// 检查活跃工作器数量
	activeWorkers := p.GetActiveWorkers()
	if activeWorkers < p.config.PoolSize/2 {
		issues = append(issues, fmt.Sprintf("活跃工作器数量过低: %d/%d", activeWorkers, p.config.PoolSize))
	}

	// 检查错误率
	totalTasks := atomic.LoadInt64(&p.totalTasks)
	failedTasks := atomic.LoadInt64(&p.failedTasks)
	if totalTasks > 0 {
		errorRate := float64(failedTasks) / float64(totalTasks) * 100
		if errorRate > 20 {
			issues = append(issues, fmt.Sprintf("错误率过高: %.1f%%", errorRate))
		}
	}

	// 检查平均响应时间
	p.timesMu.RLock()
	if len(p.taskTimes) > 0 {
		var total time.Duration
		for _, t := range p.taskTimes {
			total += t
		}
		avgTime := total / time.Duration(len(p.taskTimes))
		if avgTime > p.config.TaskTimeout/2 {
			issues = append(issues, fmt.Sprintf("平均响应时间过长: %v", avgTime))
		}
	}
	p.timesMu.RUnlock()

	p.health.Issues = issues
	p.health.LastCheck = time.Now()

	if len(issues) > 0 {
		p.health.Status = "unhealthy"
		log.Printf("[ExecutorPool] 健康检查发现问题: %v", issues)
	} else {
		p.health.Status = "healthy"
	}
}

// updateStats 更新统计信息
func (p *WorkerExecutorPool) updateStats() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.StatsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.performStatsUpdate()
		}
	}
}

// performStatsUpdate 执行统计更新
func (p *WorkerExecutorPool) performStatsUpdate() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(p.lastStatsTime)

	// 更新吞吐量
	if elapsed > 0 {
		completedTasks := atomic.LoadInt64(&p.completedTasks)
		p.stats.ThroughputPerSecond = float64(completedTasks) / elapsed.Seconds()
	}

	p.lastStatsTime = now
	p.stats.LastUpdated = now
}

// gcRoutine 垃圾回收
func (p *WorkerExecutorPool) gcRoutine() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.GCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.performGC()
		}
	}
}

// performGC 执行垃圾回收
func (p *WorkerExecutorPool) performGC() {
	// 清理过期的任务时间记录
	p.timesMu.Lock()
	if len(p.taskTimes) > 1000 {
		// 保留最近1000个记录
		p.taskTimes = p.taskTimes[len(p.taskTimes)-1000:]
	}
	p.timesMu.Unlock()

	// 清理过期的回调
	p.callbackMu.Lock()
	// 这里可以添加过期回调清理逻辑
	p.callbackMu.Unlock()
}

// recordTaskTime 记录任务执行时间
func (p *WorkerExecutorPool) recordTaskTime(duration time.Duration) {
	p.timesMu.Lock()
	defer p.timesMu.Unlock()

	p.taskTimes = append(p.taskTimes, duration)
	if len(p.taskTimes) > 2000 {
		p.taskTimes = p.taskTimes[1000:]
	}
}

// run 运行工作器执行器
func (e *WorkerExecutor) run() {
	defer e.pool.wg.Done()

	log.Printf("[ExecutorPool] 工作器 %d 启动", e.id)

	for {
		select {
		case <-e.ctx.Done():
			log.Printf("[ExecutorPool] 工作器 %d 停止", e.id)
			return

		case wrapper, ok := <-e.pool.taskQueue:
			if !ok {
				log.Printf("[ExecutorPool] 工作器 %d 停止，队列已关闭", e.id)
				return
			}

			result := e.Execute(e.ctx, wrapper.Task)

			// 执行回调
			if wrapper.Callback != nil {
				wrapper.Callback(result)
			}

			// 查找并执行全局回调
			e.pool.callbackMu.RLock()
			if callback, exists := e.pool.callbacks[wrapper.Task.GetID()]; exists {
				callback(result)
			}
			e.pool.callbackMu.RUnlock()

			// 清理回调
			e.pool.callbackMu.Lock()
			delete(e.pool.callbacks, wrapper.Task.GetID())
			e.pool.callbackMu.Unlock()
		}
	}
}

// Execute 执行任务
func (e *WorkerExecutor) Execute(ctx context.Context, task Task) *TaskResult {
	startTime := time.Now()

	// 标记为活跃
	atomic.StoreInt32(&e.active, 1)
	defer atomic.StoreInt32(&e.active, 0)

	// 创建带超时的上下文
	taskCtx, cancel := context.WithTimeout(ctx, e.pool.config.TaskTimeout)
	defer cancel()

	// 执行任务
	err := task.Execute(taskCtx)
	endTime := time.Now()
	executionTime := endTime.Sub(startTime)

	// 创建结果
	result := &TaskResult{
		TaskID:        task.GetID(),
		TaskType:      task.GetType(),
		Success:       err == nil,
		Error:         err,
		ExecutionTime: executionTime,
		WorkerID:      e.id,
		StartTime:     startTime,
		EndTime:       endTime,
		RetryCount:    task.GetRetryCount(),
	}

	// 更新统计信息
	e.updateStats(result)

	// 记录任务时间
	e.pool.recordTaskTime(executionTime)

	// 更新全局计数器
	if result.Success {
		atomic.AddInt64(&e.pool.completedTasks, 1)
	} else {
		atomic.AddInt64(&e.pool.failedTasks, 1)

		// 如果任务失败且可以重试，重新提交
		if task.CanRetry() {
			task.IncrementRetry()
			task.SetError(err)

			// 计算重试延迟
			retryDelay := time.Duration(float64(e.pool.config.RetryDelay) *
				float64(task.GetRetryCount()) * e.pool.config.RetryBackoff)

			log.Printf("[ExecutorPool] 任务 %s 将在 %v 后重试，重试次数: %d",
				task.GetID(), retryDelay, task.GetRetryCount())

			// 延迟后重新提交
			go func() {
				time.Sleep(retryDelay)
				e.pool.Submit(task)
			}()
		}
	}

	return result
}

// GetID 获取执行器ID
func (e *WorkerExecutor) GetID() int {
	return e.id
}

// GetStats 获取执行器统计信息
func (e *WorkerExecutor) GetStats() *ExecutorStats {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := *e.stats
	stats.IsActive = e.IsActive()
	return &stats
}

// IsActive 检查执行器是否活跃
func (e *WorkerExecutor) IsActive() bool {
	return atomic.LoadInt32(&e.active) == 1
}

// updateStats 更新执行器统计信息
func (e *WorkerExecutor) updateStats(result *TaskResult) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.stats.TasksProcessed++
	if result.Success {
		e.stats.TasksSucceeded++
	} else {
		e.stats.TasksFailed++
	}

	// 更新平均时间
	if e.stats.AverageTime == 0 {
		e.stats.AverageTime = result.ExecutionTime
	} else {
		e.stats.AverageTime = (e.stats.AverageTime + result.ExecutionTime) / 2
	}

	e.stats.LastTaskTime = &result.EndTime
	e.stats.IsActive = e.IsActive()
}
