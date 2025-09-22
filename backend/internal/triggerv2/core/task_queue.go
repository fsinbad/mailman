package core

import (
	"container/heap"
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// TaskPriority 任务优先级
type TaskPriority int

const (
	LowPriority    TaskPriority = 1
	NormalPriority TaskPriority = 2
	HighPriority   TaskPriority = 3
	UrgentPriority TaskPriority = 4
)

// QueuedTask 队列任务
type QueuedTask struct {
	Task        Task              `json:"task"`
	Priority    TaskPriority      `json:"priority"`
	ScheduledAt time.Time         `json:"scheduled_at"`
	QueuedAt    time.Time         `json:"queued_at"`
	Attempts    int               `json:"attempts"`
	MaxAttempts int               `json:"max_attempts"`
	RetryDelay  time.Duration     `json:"retry_delay"`
	Tags        []string          `json:"tags"`
	Callback    func(*TaskResult) `json:"-"`
	Index       int               `json:"-"` // heap index
}

// TaskQueue 任务队列接口
type TaskQueue interface {
	// 任务管理
	Push(task *QueuedTask) error
	Pop() (*QueuedTask, error)
	Peek() (*QueuedTask, error)

	// 调度任务
	Schedule(task *QueuedTask, delay time.Duration) error
	ScheduleAt(task *QueuedTask, scheduledAt time.Time) error

	// 批处理
	PopBatch(maxSize int) ([]*QueuedTask, error)
	PushBatch(tasks []*QueuedTask) error

	// 队列管理
	Size() int
	IsEmpty() bool
	Clear() error

	// 任务查询
	GetByID(taskID string) (*QueuedTask, error)
	GetByTags(tags []string) ([]*QueuedTask, error)
	GetByPriority(priority TaskPriority) ([]*QueuedTask, error)

	// 任务操作
	RemoveByID(taskID string) error
	UpdatePriority(taskID string, priority TaskPriority) error

	// 监控
	GetStats() *QueueStats
	GetHealth() *QueueHealth

	// 生命周期
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// QueueStats 队列统计信息
type QueueStats struct {
	TotalTasks          int64                `json:"total_tasks"`
	PendingTasks        int                  `json:"pending_tasks"`
	ProcessedTasks      int64                `json:"processed_tasks"`
	FailedTasks         int64                `json:"failed_tasks"`
	ScheduledTasks      int                  `json:"scheduled_tasks"`
	PriorityBreakdown   map[TaskPriority]int `json:"priority_breakdown"`
	TagBreakdown        map[string]int       `json:"tag_breakdown"`
	AverageWaitTime     time.Duration        `json:"average_wait_time"`
	ThroughputPerSecond float64              `json:"throughput_per_second"`
	CreatedAt           time.Time            `json:"created_at"`
	LastUpdated         time.Time            `json:"last_updated"`
}

// QueueHealth 队列健康状态
type QueueHealth struct {
	Status          string        `json:"status"`
	IsRunning       bool          `json:"is_running"`
	QueueSize       int           `json:"queue_size"`
	ScheduledSize   int           `json:"scheduled_size"`
	Capacity        int           `json:"capacity"`
	Utilization     float64       `json:"utilization"`
	OldestTaskAge   time.Duration `json:"oldest_task_age"`
	AverageWaitTime time.Duration `json:"average_wait_time"`
	ProcessingRate  float64       `json:"processing_rate"`
	ErrorRate       float64       `json:"error_rate"`
	Issues          []string      `json:"issues"`
	LastCheck       time.Time     `json:"last_check"`
}

// QueueConfig 队列配置
type QueueConfig struct {
	// 基本配置
	MaxSize         int          `json:"max_size"`
	MaxScheduled    int          `json:"max_scheduled"`
	DefaultPriority TaskPriority `json:"default_priority"`

	// 调度配置
	ScheduleInterval time.Duration `json:"schedule_interval"`
	MaxScheduleDelay time.Duration `json:"max_schedule_delay"`

	// 批处理配置
	BatchSize    int           `json:"batch_size"`
	BatchTimeout time.Duration `json:"batch_timeout"`

	// 监控配置
	StatsInterval  time.Duration `json:"stats_interval"`
	HealthInterval time.Duration `json:"health_interval"`

	// 性能配置
	EnableMetrics   bool          `json:"enable_metrics"`
	EnableProfiling bool          `json:"enable_profiling"`
	GCInterval      time.Duration `json:"gc_interval"`
}

// DefaultQueueConfig 默认队列配置
func DefaultQueueConfig() *QueueConfig {
	return &QueueConfig{
		MaxSize:          10000,
		MaxScheduled:     1000,
		DefaultPriority:  NormalPriority,
		ScheduleInterval: 100 * time.Millisecond,
		MaxScheduleDelay: 24 * time.Hour,
		BatchSize:        100,
		BatchTimeout:     1 * time.Second,
		StatsInterval:    10 * time.Second,
		HealthInterval:   30 * time.Second,
		EnableMetrics:    true,
		EnableProfiling:  false,
		GCInterval:       10 * time.Minute,
	}
}

// PriorityTaskQueue 优先级任务队列实现
type PriorityTaskQueue struct {
	config    *QueueConfig
	heap      *TaskHeap
	scheduled map[string]*QueuedTask
	taskIndex map[string]*QueuedTask
	stats     *QueueStats
	health    *QueueHealth

	// 状态管理
	running int32
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// 同步
	mu          sync.RWMutex
	scheduledMu sync.RWMutex
	indexMu     sync.RWMutex

	// 计数器
	totalTasks     int64
	processedTasks int64
	failedTasks    int64

	// 时间追踪
	createdAt     time.Time
	lastStatsTime time.Time
	waitTimes     []time.Duration
	timesMu       sync.RWMutex
}

// TaskHeap 任务堆
type TaskHeap struct {
	tasks []*QueuedTask
	mu    sync.RWMutex
}

// NewPriorityTaskQueue 创建优先级任务队列
func NewPriorityTaskQueue(config *QueueConfig) TaskQueue {
	if config == nil {
		config = DefaultQueueConfig()
	}

	queue := &PriorityTaskQueue{
		config:        config,
		heap:          &TaskHeap{tasks: make([]*QueuedTask, 0)},
		scheduled:     make(map[string]*QueuedTask),
		taskIndex:     make(map[string]*QueuedTask),
		createdAt:     time.Now(),
		lastStatsTime: time.Now(),
		waitTimes:     make([]time.Duration, 0, 1000),
	}

	// 初始化统计信息
	queue.stats = &QueueStats{
		PriorityBreakdown: make(map[TaskPriority]int),
		TagBreakdown:      make(map[string]int),
		CreatedAt:         queue.createdAt,
	}

	// 初始化健康状态
	queue.health = &QueueHealth{
		Status:    "stopped",
		IsRunning: false,
		Capacity:  config.MaxSize,
		Issues:    []string{},
		LastCheck: time.Now(),
	}

	return queue
}

// Start 启动任务队列
func (q *PriorityTaskQueue) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&q.running, 0, 1) {
		return fmt.Errorf("任务队列已经在运行")
	}

	q.ctx, q.cancel = context.WithCancel(ctx)

	// 启动调度器
	if q.config.ScheduleInterval > 0 {
		q.wg.Add(1)
		go q.scheduleRoutine()
	}

	// 启动统计更新
	if q.config.StatsInterval > 0 {
		q.wg.Add(1)
		go q.updateStats()
	}

	// 启动健康检查
	if q.config.HealthInterval > 0 {
		q.wg.Add(1)
		go q.healthCheck()
	}

	// 启动垃圾回收
	if q.config.GCInterval > 0 {
		q.wg.Add(1)
		go q.gcRoutine()
	}

	q.health.Status = "running"
	q.health.IsRunning = true

	log.Printf("[TaskQueue] 任务队列已启动，最大容量: %d", q.config.MaxSize)
	return nil
}

// Stop 停止任务队列
func (q *PriorityTaskQueue) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&q.running, 1, 0) {
		return fmt.Errorf("任务队列没有在运行")
	}

	q.cancel()

	// 等待所有协程完成
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[TaskQueue] 任务队列已停止")
	case <-ctx.Done():
		log.Printf("[TaskQueue] 任务队列停止超时")
		return ctx.Err()
	}

	q.health.Status = "stopped"
	q.health.IsRunning = false

	return nil
}

// Push 推送任务到队列
func (q *PriorityTaskQueue) Push(task *QueuedTask) error {
	if atomic.LoadInt32(&q.running) == 0 {
		return fmt.Errorf("任务队列没有在运行")
	}

	if task == nil {
		return fmt.Errorf("任务不能为空")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// 检查队列大小
	if q.heap.Len() >= q.config.MaxSize {
		return fmt.Errorf("任务队列已满")
	}

	// 设置默认值
	if task.Priority == 0 {
		task.Priority = q.config.DefaultPriority
	}
	if task.QueuedAt.IsZero() {
		task.QueuedAt = time.Now()
	}
	if task.MaxAttempts == 0 {
		task.MaxAttempts = 3
	}

	// 添加到堆
	heap.Push(q.heap, task)

	// 添加到索引
	q.indexMu.Lock()
	q.taskIndex[task.Task.GetID()] = task
	q.indexMu.Unlock()

	// 更新统计
	atomic.AddInt64(&q.totalTasks, 1)
	q.updatePriorityStats(task.Priority, 1)
	q.updateTagStats(task.Tags, 1)

	return nil
}

// Pop 从队列弹出任务
func (q *PriorityTaskQueue) Pop() (*QueuedTask, error) {
	if atomic.LoadInt32(&q.running) == 0 {
		return nil, fmt.Errorf("任务队列没有在运行")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.heap.Len() == 0 {
		return nil, fmt.Errorf("队列为空")
	}

	result := heap.Pop(q.heap)
	if result == nil {
		return nil, fmt.Errorf("队列为空")
	}
	task := result.(*QueuedTask)
	if task == nil {
		return nil, fmt.Errorf("队列为空")
	}

	// 从索引中移除
	q.indexMu.Lock()
	delete(q.taskIndex, task.Task.GetID())
	q.indexMu.Unlock()

	// 记录等待时间
	waitTime := time.Since(task.QueuedAt)
	q.recordWaitTime(waitTime)

	// 更新统计
	atomic.AddInt64(&q.processedTasks, 1)
	q.updatePriorityStats(task.Priority, -1)
	q.updateTagStats(task.Tags, -1)

	return task, nil
}

// Peek 查看队列顶部任务
func (q *PriorityTaskQueue) Peek() (*QueuedTask, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.heap.Len() == 0 {
		return nil, fmt.Errorf("队列为空")
	}

	return q.heap.Peek(), nil
}

// Schedule 调度任务
func (q *PriorityTaskQueue) Schedule(task *QueuedTask, delay time.Duration) error {
	return q.ScheduleAt(task, time.Now().Add(delay))
}

// ScheduleAt 在指定时间调度任务
func (q *PriorityTaskQueue) ScheduleAt(task *QueuedTask, scheduledAt time.Time) error {
	if atomic.LoadInt32(&q.running) == 0 {
		return fmt.Errorf("任务队列没有在运行")
	}

	if task == nil {
		return fmt.Errorf("任务不能为空")
	}

	// 检查调度时间
	if scheduledAt.Before(time.Now()) {
		return fmt.Errorf("调度时间不能在过去")
	}

	if scheduledAt.Sub(time.Now()) > q.config.MaxScheduleDelay {
		return fmt.Errorf("调度延迟超过最大限制")
	}

	q.scheduledMu.Lock()
	defer q.scheduledMu.Unlock()

	// 检查调度队列大小
	if len(q.scheduled) >= q.config.MaxScheduled {
		return fmt.Errorf("调度队列已满")
	}

	// 设置调度时间
	task.ScheduledAt = scheduledAt
	task.QueuedAt = time.Now()

	// 添加到调度队列
	q.scheduled[task.Task.GetID()] = task

	log.Printf("[TaskQueue] 任务 %s 已调度在 %v", task.Task.GetID(), scheduledAt)
	return nil
}

// PopBatch 批量弹出任务
func (q *PriorityTaskQueue) PopBatch(maxSize int) ([]*QueuedTask, error) {
	if atomic.LoadInt32(&q.running) == 0 {
		return nil, fmt.Errorf("任务队列没有在运行")
	}

	if maxSize <= 0 {
		maxSize = q.config.BatchSize
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	available := q.heap.Len()
	if available == 0 {
		return []*QueuedTask{}, nil
	}

	batchSize := available
	if batchSize > maxSize {
		batchSize = maxSize
	}

	tasks := make([]*QueuedTask, 0, batchSize)

	for i := 0; i < batchSize; i++ {
		if q.heap.Len() == 0 {
			break
		}

		result := heap.Pop(q.heap)
		if result == nil {
			break
		}
		task := result.(*QueuedTask)
		if task == nil {
			break
		}

		// 从索引中移除
		q.indexMu.Lock()
		delete(q.taskIndex, task.Task.GetID())
		q.indexMu.Unlock()

		// 记录等待时间
		waitTime := time.Since(task.QueuedAt)
		q.recordWaitTime(waitTime)

		// 更新统计
		atomic.AddInt64(&q.processedTasks, 1)
		q.updatePriorityStats(task.Priority, -1)
		q.updateTagStats(task.Tags, -1)

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// PushBatch 批量推送任务
func (q *PriorityTaskQueue) PushBatch(tasks []*QueuedTask) error {
	if atomic.LoadInt32(&q.running) == 0 {
		return fmt.Errorf("任务队列没有在运行")
	}

	if len(tasks) == 0 {
		return nil
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// 检查队列容量
	if q.heap.Len()+len(tasks) > q.config.MaxSize {
		return fmt.Errorf("批量任务超过队列容量")
	}

	// 批量添加任务
	for _, task := range tasks {
		if task == nil {
			continue
		}

		// 设置默认值
		if task.Priority == 0 {
			task.Priority = q.config.DefaultPriority
		}
		if task.QueuedAt.IsZero() {
			task.QueuedAt = time.Now()
		}
		if task.MaxAttempts == 0 {
			task.MaxAttempts = 3
		}

		// 添加到堆
		heap.Push(q.heap, task)

		// 添加到索引
		q.indexMu.Lock()
		q.taskIndex[task.Task.GetID()] = task
		q.indexMu.Unlock()

		// 更新统计
		atomic.AddInt64(&q.totalTasks, 1)
		q.updatePriorityStats(task.Priority, 1)
		q.updateTagStats(task.Tags, 1)
	}

	return nil
}

// Size 获取队列大小
func (q *PriorityTaskQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.heap.Len()
}

// IsEmpty 检查队列是否为空
func (q *PriorityTaskQueue) IsEmpty() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.heap.Len() == 0
}

// Clear 清空队列
func (q *PriorityTaskQueue) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.heap.Clear()

	q.indexMu.Lock()
	q.taskIndex = make(map[string]*QueuedTask)
	q.indexMu.Unlock()

	q.scheduledMu.Lock()
	q.scheduled = make(map[string]*QueuedTask)
	q.scheduledMu.Unlock()

	// 重置统计
	atomic.StoreInt64(&q.totalTasks, 0)
	atomic.StoreInt64(&q.processedTasks, 0)
	atomic.StoreInt64(&q.failedTasks, 0)

	q.stats.PriorityBreakdown = make(map[TaskPriority]int)
	q.stats.TagBreakdown = make(map[string]int)

	log.Printf("[TaskQueue] 队列已清空")
	return nil
}

// GetByID 根据ID获取任务
func (q *PriorityTaskQueue) GetByID(taskID string) (*QueuedTask, error) {
	q.indexMu.RLock()
	defer q.indexMu.RUnlock()

	if task, exists := q.taskIndex[taskID]; exists {
		return task, nil
	}

	q.scheduledMu.RLock()
	defer q.scheduledMu.RUnlock()

	if task, exists := q.scheduled[taskID]; exists {
		return task, nil
	}

	return nil, fmt.Errorf("任务 %s 不存在", taskID)
}

// GetByTags 根据标签获取任务
func (q *PriorityTaskQueue) GetByTags(tags []string) ([]*QueuedTask, error) {
	if len(tags) == 0 {
		return []*QueuedTask{}, nil
	}

	q.indexMu.RLock()
	defer q.indexMu.RUnlock()

	var result []*QueuedTask

	for _, task := range q.taskIndex {
		if q.hasAllTags(task.Tags, tags) {
			result = append(result, task)
		}
	}

	return result, nil
}

// GetByPriority 根据优先级获取任务
func (q *PriorityTaskQueue) GetByPriority(priority TaskPriority) ([]*QueuedTask, error) {
	q.indexMu.RLock()
	defer q.indexMu.RUnlock()

	var result []*QueuedTask

	for _, task := range q.taskIndex {
		if task.Priority == priority {
			result = append(result, task)
		}
	}

	return result, nil
}

// RemoveByID 根据ID移除任务
func (q *PriorityTaskQueue) RemoveByID(taskID string) error {
	q.indexMu.Lock()
	defer q.indexMu.Unlock()

	// 从队列中移除
	if task, exists := q.taskIndex[taskID]; exists {
		q.mu.Lock()
		q.heap.Remove(task)
		q.mu.Unlock()

		delete(q.taskIndex, taskID)

		// 更新统计
		q.updatePriorityStats(task.Priority, -1)
		q.updateTagStats(task.Tags, -1)

		return nil
	}

	// 从调度队列中移除
	q.scheduledMu.Lock()
	defer q.scheduledMu.Unlock()

	if _, exists := q.scheduled[taskID]; exists {
		delete(q.scheduled, taskID)
		return nil
	}

	return fmt.Errorf("任务 %s 不存在", taskID)
}

// UpdatePriority 更新任务优先级
func (q *PriorityTaskQueue) UpdatePriority(taskID string, priority TaskPriority) error {
	q.indexMu.Lock()
	defer q.indexMu.Unlock()

	task, exists := q.taskIndex[taskID]
	if !exists {
		return fmt.Errorf("任务 %s 不存在", taskID)
	}

	oldPriority := task.Priority
	task.Priority = priority

	// 更新堆 - 需要调用heap.Fix来维护堆性质
	q.mu.Lock()
	if task.Index >= 0 && task.Index < q.heap.Len() {
		heap.Fix(q.heap, task.Index)
	}
	q.mu.Unlock()

	// 更新统计
	q.updatePriorityStats(oldPriority, -1)
	q.updatePriorityStats(priority, 1)

	return nil
}

// GetStats 获取统计信息
func (q *PriorityTaskQueue) GetStats() *QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := *q.stats
	stats.PendingTasks = q.heap.Len()
	stats.TotalTasks = atomic.LoadInt64(&q.totalTasks)
	stats.ProcessedTasks = atomic.LoadInt64(&q.processedTasks)
	stats.FailedTasks = atomic.LoadInt64(&q.failedTasks)

	q.scheduledMu.RLock()
	stats.ScheduledTasks = len(q.scheduled)
	q.scheduledMu.RUnlock()

	// 计算平均等待时间
	q.timesMu.RLock()
	if len(q.waitTimes) > 0 {
		var total time.Duration
		for _, t := range q.waitTimes {
			total += t
		}
		stats.AverageWaitTime = total / time.Duration(len(q.waitTimes))
	}
	q.timesMu.RUnlock()

	// 计算吞吐量
	elapsed := time.Since(q.lastStatsTime)
	if elapsed > 0 {
		stats.ThroughputPerSecond = float64(stats.ProcessedTasks) / elapsed.Seconds()
	}

	stats.LastUpdated = time.Now()

	return &stats
}

// GetHealth 获取健康状态
func (q *PriorityTaskQueue) GetHealth() *QueueHealth {
	q.mu.RLock()
	defer q.mu.RUnlock()

	health := *q.health
	health.QueueSize = q.heap.Len()

	q.scheduledMu.RLock()
	health.ScheduledSize = len(q.scheduled)
	q.scheduledMu.RUnlock()

	health.Utilization = float64(health.QueueSize) / float64(health.Capacity) * 100

	// 计算最老任务年龄
	if q.heap.Len() > 0 {
		oldestTask := q.heap.Peek()
		health.OldestTaskAge = time.Since(oldestTask.QueuedAt)
	}

	// 计算平均等待时间
	q.timesMu.RLock()
	if len(q.waitTimes) > 0 {
		var total time.Duration
		for _, t := range q.waitTimes {
			total += t
		}
		health.AverageWaitTime = total / time.Duration(len(q.waitTimes))
	}
	q.timesMu.RUnlock()

	// 计算处理率
	totalTasks := atomic.LoadInt64(&q.totalTasks)
	processedTasks := atomic.LoadInt64(&q.processedTasks)
	if totalTasks > 0 {
		health.ProcessingRate = float64(processedTasks) / float64(totalTasks) * 100
	}

	// 计算错误率
	failedTasks := atomic.LoadInt64(&q.failedTasks)
	if totalTasks > 0 {
		health.ErrorRate = float64(failedTasks) / float64(totalTasks) * 100
	}

	health.LastCheck = time.Now()

	return &health
}

// scheduleRoutine 调度协程
func (q *PriorityTaskQueue) scheduleRoutine() {
	defer q.wg.Done()

	ticker := time.NewTicker(q.config.ScheduleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			q.processScheduledTasks()
		}
	}
}

// processScheduledTasks 处理调度任务
func (q *PriorityTaskQueue) processScheduledTasks() {
	now := time.Now()
	var tasksToProcess []*QueuedTask

	q.scheduledMu.Lock()
	for taskID, task := range q.scheduled {
		if task.ScheduledAt.Before(now) || task.ScheduledAt.Equal(now) {
			tasksToProcess = append(tasksToProcess, task)
			delete(q.scheduled, taskID)
		}
	}
	q.scheduledMu.Unlock()

	// 将到期的任务添加到队列
	for _, task := range tasksToProcess {
		if err := q.Push(task); err != nil {
			log.Printf("[TaskQueue] 调度任务失败: %v", err)
			// 重新加入调度队列
			q.scheduledMu.Lock()
			q.scheduled[task.Task.GetID()] = task
			q.scheduledMu.Unlock()
		}
	}
}

// updateStats 更新统计信息
func (q *PriorityTaskQueue) updateStats() {
	defer q.wg.Done()

	ticker := time.NewTicker(q.config.StatsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			q.performStatsUpdate()
		}
	}
}

// performStatsUpdate 执行统计更新
func (q *PriorityTaskQueue) performStatsUpdate() {
	now := time.Now()
	elapsed := now.Sub(q.lastStatsTime)

	// 更新吞吐量
	if elapsed > 0 {
		processedTasks := atomic.LoadInt64(&q.processedTasks)
		q.stats.ThroughputPerSecond = float64(processedTasks) / elapsed.Seconds()
	}

	q.lastStatsTime = now
	q.stats.LastUpdated = now
}

// healthCheck 健康检查
func (q *PriorityTaskQueue) healthCheck() {
	defer q.wg.Done()

	ticker := time.NewTicker(q.config.HealthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			q.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (q *PriorityTaskQueue) performHealthCheck() {
	issues := []string{}

	// 检查队列使用率
	queueSize := q.Size()
	utilization := float64(queueSize) / float64(q.config.MaxSize) * 100
	if utilization > 90 {
		issues = append(issues, fmt.Sprintf("队列使用率过高: %.1f%%", utilization))
	}

	// 检查最老任务
	if queueSize > 0 {
		if oldestTask, err := q.Peek(); err == nil {
			age := time.Since(oldestTask.QueuedAt)
			if age > 10*time.Minute {
				issues = append(issues, fmt.Sprintf("存在过期任务: %v", age))
			}
		}
	}

	// 检查调度队列
	q.scheduledMu.RLock()
	scheduledSize := len(q.scheduled)
	q.scheduledMu.RUnlock()

	if scheduledSize > q.config.MaxScheduled/2 {
		issues = append(issues, fmt.Sprintf("调度队列使用率过高: %d/%d", scheduledSize, q.config.MaxScheduled))
	}

	// 检查错误率
	totalTasks := atomic.LoadInt64(&q.totalTasks)
	failedTasks := atomic.LoadInt64(&q.failedTasks)
	if totalTasks > 0 {
		errorRate := float64(failedTasks) / float64(totalTasks) * 100
		if errorRate > 20 {
			issues = append(issues, fmt.Sprintf("错误率过高: %.1f%%", errorRate))
		}
	}

	q.health.Issues = issues
	q.health.LastCheck = time.Now()

	if len(issues) > 0 {
		q.health.Status = "unhealthy"
		log.Printf("[TaskQueue] 健康检查发现问题: %v", issues)
	} else {
		q.health.Status = "healthy"
	}
}

// gcRoutine 垃圾回收协程
func (q *PriorityTaskQueue) gcRoutine() {
	defer q.wg.Done()

	ticker := time.NewTicker(q.config.GCInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			q.performGC()
		}
	}
}

// performGC 执行垃圾回收
func (q *PriorityTaskQueue) performGC() {
	// 清理过期的等待时间记录
	q.timesMu.Lock()
	if len(q.waitTimes) > 1000 {
		q.waitTimes = q.waitTimes[len(q.waitTimes)-1000:]
	}
	q.timesMu.Unlock()

	// 清理过期的调度任务
	q.scheduledMu.Lock()
	now := time.Now()
	expiredTasks := []string{}

	for taskID, task := range q.scheduled {
		if now.Sub(task.ScheduledAt) > q.config.MaxScheduleDelay {
			expiredTasks = append(expiredTasks, taskID)
		}
	}

	for _, taskID := range expiredTasks {
		delete(q.scheduled, taskID)
		log.Printf("[TaskQueue] 清理过期调度任务: %s", taskID)
	}
	q.scheduledMu.Unlock()
}

// 辅助方法

// updatePriorityStats 更新优先级统计
func (q *PriorityTaskQueue) updatePriorityStats(priority TaskPriority, delta int) {
	q.stats.PriorityBreakdown[priority] += delta
	if q.stats.PriorityBreakdown[priority] <= 0 {
		delete(q.stats.PriorityBreakdown, priority)
	}
}

// updateTagStats 更新标签统计
func (q *PriorityTaskQueue) updateTagStats(tags []string, delta int) {
	for _, tag := range tags {
		q.stats.TagBreakdown[tag] += delta
		if q.stats.TagBreakdown[tag] <= 0 {
			delete(q.stats.TagBreakdown, tag)
		}
	}
}

// hasAllTags 检查是否包含所有标签
func (q *PriorityTaskQueue) hasAllTags(taskTags, requiredTags []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range taskTags {
		tagSet[tag] = true
	}

	for _, tag := range requiredTags {
		if !tagSet[tag] {
			return false
		}
	}

	return true
}

// recordWaitTime 记录等待时间
func (q *PriorityTaskQueue) recordWaitTime(duration time.Duration) {
	q.timesMu.Lock()
	defer q.timesMu.Unlock()

	q.waitTimes = append(q.waitTimes, duration)
	if len(q.waitTimes) > 2000 {
		q.waitTimes = q.waitTimes[1000:]
	}
}

// 堆操作实现

// Len 返回堆长度
func (h *TaskHeap) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.tasks)
}

// Less 比较两个任务的优先级
func (h *TaskHeap) Less(i, j int) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 优先级高的任务在前（对于最小堆，我们需要反转比较逻辑）
	if h.tasks[i].Priority != h.tasks[j].Priority {
		return h.tasks[i].Priority > h.tasks[j].Priority
	}

	// 相同优先级按排队时间排序（早排队的在前）
	return h.tasks[i].QueuedAt.Before(h.tasks[j].QueuedAt)
}

// Swap 交换两个任务
func (h *TaskHeap) Swap(i, j int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.tasks[i], h.tasks[j] = h.tasks[j], h.tasks[i]
	h.tasks[i].Index = i
	h.tasks[j].Index = j
}

// Push 推送任务到堆
func (h *TaskHeap) Push(x interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	task := x.(*QueuedTask)
	task.Index = len(h.tasks)
	h.tasks = append(h.tasks, task)
	// 注意：不在这里调用heap.Fix，因为会导致死锁
	// heap的维护由外部调用heap.Push()时完成
}

// Pop 从堆弹出任务 - 实现heap.Interface
func (h *TaskHeap) Pop() any {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.tasks) == 0 {
		return nil
	}

	task := h.tasks[len(h.tasks)-1]
	h.tasks = h.tasks[:len(h.tasks)-1]
	return task
}

// PopTask 从堆弹出任务 - 返回具体类型
func (h *TaskHeap) PopTask() *QueuedTask {
	result := heap.Pop(h)
	if result == nil {
		return nil
	}
	return result.(*QueuedTask)
}

// Peek 查看堆顶任务
func (h *TaskHeap) Peek() *QueuedTask {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.tasks) == 0 {
		return nil
	}

	return h.tasks[0]
}

// Remove 从堆中移除任务
func (h *TaskHeap) Remove(task *QueuedTask) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if task.Index < 0 || task.Index >= len(h.tasks) {
		return
	}

	lastIndex := len(h.tasks) - 1
	if task.Index != lastIndex {
		h.tasks[task.Index] = h.tasks[lastIndex]
		h.tasks[task.Index].Index = task.Index
	}

	h.tasks = h.tasks[:lastIndex]

	if task.Index < len(h.tasks) {
		heap.Fix(h, task.Index)
	}
}

// Update 更新堆中的任务
func (h *TaskHeap) Update(task *QueuedTask) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if task.Index >= 0 && task.Index < len(h.tasks) {
		// 注意：不能在这里调用heap.Fix，因为会导致死锁
		// 我们只更新任务信息，堆的维护由外部处理
	}
}

// Clear 清空堆
func (h *TaskHeap) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.tasks = h.tasks[:0]
}
