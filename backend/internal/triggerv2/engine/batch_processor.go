package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mailman/internal/triggerv2/models"
)

// BatchProcessor 批处理处理器
type BatchProcessor struct {
	config    *BatchConfig
	batches   map[string]*EventBatch
	mu        sync.RWMutex
	processor EventProcessor
	metrics   *BatchMetrics
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// BatchConfig 批处理配置
type BatchConfig struct {
	MaxBatchSize      int           `json:"max_batch_size"`     // 最大批处理大小
	MaxWaitTime       time.Duration `json:"max_wait_time"`      // 最大等待时间
	FlushInterval     time.Duration `json:"flush_interval"`     // 刷新间隔
	MaxConcurrency    int           `json:"max_concurrency"`    // 最大并发数
	GroupByFields     []string      `json:"group_by_fields"`    // 分组字段
	EnableCompression bool          `json:"enable_compression"` // 启用压缩
	RetryConfig       *RetryConfig  `json:"retry_config"`       // 重试配置
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// EventBatch 事件批次
type EventBatch struct {
	ID             string          `json:"id"`
	GroupKey       string          `json:"group_key"`
	Events         []*models.Event `json:"events"`
	CreatedAt      time.Time       `json:"created_at"`
	LastUpdated    time.Time       `json:"last_updated"`
	Status         BatchStatus     `json:"status"`
	Priority       int             `json:"priority"`
	RetryCount     int             `json:"retry_count"`
	ProcessedAt    *time.Time      `json:"processed_at,omitempty"`
	CompressedData []byte          `json:"compressed_data,omitempty"`
	mu             sync.RWMutex
}

// BatchStatus 批次状态
type BatchStatus string

const (
	BatchStatusPending    BatchStatus = "pending"
	BatchStatusProcessing BatchStatus = "processing"
	BatchStatusCompleted  BatchStatus = "completed"
	BatchStatusFailed     BatchStatus = "failed"
	BatchStatusRetrying   BatchStatus = "retrying"
)

// EventProcessor 事件处理器接口
type EventProcessor interface {
	ProcessBatch(ctx context.Context, batch *EventBatch) error
}

// BatchMetrics 批处理指标
type BatchMetrics struct {
	TotalBatches       int64         `json:"total_batches"`
	ProcessedBatches   int64         `json:"processed_batches"`
	FailedBatches      int64         `json:"failed_batches"`
	TotalEvents        int64         `json:"total_events"`
	ProcessedEvents    int64         `json:"processed_events"`
	AverageProcessTime time.Duration `json:"average_process_time"`
	mu                 sync.RWMutex
}

// NewBatchProcessor 创建批处理处理器
func NewBatchProcessor(config *BatchConfig, processor EventProcessor) *BatchProcessor {
	if config == nil {
		config = &BatchConfig{
			MaxBatchSize:      100,
			MaxWaitTime:       time.Second * 5,
			FlushInterval:     time.Second,
			MaxConcurrency:    10,
			GroupByFields:     []string{"type"},
			EnableCompression: false,
			RetryConfig: &RetryConfig{
				MaxRetries:    3,
				InitialDelay:  time.Second,
				MaxDelay:      time.Minute,
				BackoffFactor: 2.0,
			},
		}
	}

	// 确保RetryConfig不为空
	if config.RetryConfig == nil {
		config.RetryConfig = &RetryConfig{
			MaxRetries:    3,
			InitialDelay:  time.Second,
			MaxDelay:      time.Minute,
			BackoffFactor: 2.0,
		}
	}

	return &BatchProcessor{
		config:    config,
		batches:   make(map[string]*EventBatch),
		processor: processor,
		metrics:   &BatchMetrics{},
		stopCh:    make(chan struct{}),
	}
}

// Start 启动批处理处理器
func (bp *BatchProcessor) Start(ctx context.Context) error {
	bp.wg.Add(1)
	go bp.flushWorker(ctx)
	return nil
}

// Stop 停止批处理处理器
func (bp *BatchProcessor) Stop() error {
	close(bp.stopCh)
	bp.wg.Wait()
	return nil
}

// AddEvent 添加事件到批处理
func (bp *BatchProcessor) AddEvent(event *models.Event) error {
	groupKey := bp.generateGroupKey(event)

	bp.mu.Lock()
	defer bp.mu.Unlock()

	batch, exists := bp.batches[groupKey]
	if !exists {
		batch = &EventBatch{
			ID:          bp.generateBatchID(),
			GroupKey:    groupKey,
			Events:      make([]*models.Event, 0, bp.config.MaxBatchSize),
			CreatedAt:   time.Now(),
			LastUpdated: time.Now(),
			Status:      BatchStatusPending,
			Priority:    event.Priority,
		}
		bp.batches[groupKey] = batch
	}

	batch.mu.Lock()
	defer batch.mu.Unlock()

	batch.Events = append(batch.Events, event)
	batch.LastUpdated = time.Now()

	// 更新指标
	bp.metrics.mu.Lock()
	bp.metrics.TotalEvents++
	bp.metrics.mu.Unlock()

	// 检查是否需要立即处理
	if len(batch.Events) >= bp.config.MaxBatchSize {
		go bp.processBatch(context.Background(), batch)
		delete(bp.batches, groupKey)
	}

	return nil
}

// ProcessBatch 处理批次
func (bp *BatchProcessor) ProcessBatch(ctx context.Context, batch *EventBatch) error {
	return bp.processBatch(ctx, batch)
}

// processBatch 内部处理批次
func (bp *BatchProcessor) processBatch(ctx context.Context, batch *EventBatch) error {
	batch.mu.Lock()
	if batch.Status == BatchStatusProcessing {
		batch.mu.Unlock()
		return fmt.Errorf("批次 %s 正在处理中", batch.ID)
	}
	batch.Status = BatchStatusProcessing
	batch.mu.Unlock()

	startTime := time.Now()

	// 更新指标
	bp.metrics.mu.Lock()
	bp.metrics.TotalBatches++
	bp.metrics.mu.Unlock()

	// 压缩数据（如果启用）
	if bp.config.EnableCompression {
		if err := bp.compressBatch(batch); err != nil {
			return fmt.Errorf("压缩批次数据失败: %v", err)
		}
	}

	// 执行处理
	var err error
	for attempt := 0; attempt <= bp.config.RetryConfig.MaxRetries; attempt++ {
		err = bp.processor.ProcessBatch(ctx, batch)
		if err == nil {
			break
		}

		if attempt < bp.config.RetryConfig.MaxRetries {
			delay := bp.calculateRetryDelay(attempt)
			select {
			case <-time.After(delay):
				batch.mu.Lock()
				batch.Status = BatchStatusRetrying
				batch.RetryCount++
				batch.mu.Unlock()
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// 更新批次状态
	batch.mu.Lock()
	if err != nil {
		batch.Status = BatchStatusFailed
		bp.metrics.mu.Lock()
		bp.metrics.FailedBatches++
		bp.metrics.mu.Unlock()
	} else {
		batch.Status = BatchStatusCompleted
		now := time.Now()
		batch.ProcessedAt = &now

		// 更新指标
		bp.metrics.mu.Lock()
		bp.metrics.ProcessedBatches++
		bp.metrics.ProcessedEvents += int64(len(batch.Events))

		// 更新平均处理时间
		processingTime := time.Since(startTime)
		if bp.metrics.ProcessedBatches == 1 {
			bp.metrics.AverageProcessTime = processingTime
		} else {
			bp.metrics.AverageProcessTime = time.Duration(
				(int64(bp.metrics.AverageProcessTime)*bp.metrics.ProcessedBatches + int64(processingTime)) /
					(bp.metrics.ProcessedBatches + 1),
			)
		}
		bp.metrics.mu.Unlock()
	}
	batch.mu.Unlock()

	return err
}

// flushWorker 定期刷新工作器
func (bp *BatchProcessor) flushWorker(ctx context.Context) {
	defer bp.wg.Done()

	ticker := time.NewTicker(bp.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bp.flushExpiredBatches(ctx)
		case <-bp.stopCh:
			bp.flushAllBatches(ctx)
			return
		case <-ctx.Done():
			return
		}
	}
}

// flushExpiredBatches 刷新过期批次
func (bp *BatchProcessor) flushExpiredBatches(ctx context.Context) {
	bp.mu.Lock()
	expiredBatches := make([]*EventBatch, 0)
	keysToDelete := make([]string, 0)

	for key, batch := range bp.batches {
		batch.mu.RLock()
		if time.Since(batch.LastUpdated) > bp.config.MaxWaitTime {
			expiredBatches = append(expiredBatches, batch)
			keysToDelete = append(keysToDelete, key)
		}
		batch.mu.RUnlock()
	}

	for _, key := range keysToDelete {
		delete(bp.batches, key)
	}
	bp.mu.Unlock()

	// 处理过期批次
	semaphore := make(chan struct{}, bp.config.MaxConcurrency)
	var wg sync.WaitGroup

	for _, batch := range expiredBatches {
		wg.Add(1)
		go func(b *EventBatch) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := bp.processBatch(ctx, b); err != nil {
				// 记录错误，但不影响其他批次的处理
				fmt.Printf("处理批次 %s 失败: %v\n", b.ID, err)
			}
		}(batch)
	}

	wg.Wait()
}

// flushAllBatches 刷新所有批次
func (bp *BatchProcessor) flushAllBatches(ctx context.Context) {
	bp.mu.Lock()
	allBatches := make([]*EventBatch, 0, len(bp.batches))
	for _, batch := range bp.batches {
		allBatches = append(allBatches, batch)
	}
	bp.batches = make(map[string]*EventBatch)
	bp.mu.Unlock()

	// 处理所有批次
	semaphore := make(chan struct{}, bp.config.MaxConcurrency)
	var wg sync.WaitGroup

	for _, batch := range allBatches {
		wg.Add(1)
		go func(b *EventBatch) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := bp.processBatch(ctx, b); err != nil {
				fmt.Printf("处理批次 %s 失败: %v\n", b.ID, err)
			}
		}(batch)
	}

	wg.Wait()
}

// generateGroupKey 生成分组键
func (bp *BatchProcessor) generateGroupKey(event *models.Event) string {
	if len(bp.config.GroupByFields) == 0 {
		return "default"
	}

	key := ""
	for _, field := range bp.config.GroupByFields {
		switch field {
		case "type":
			key += fmt.Sprintf("type:%s,", event.Type)
		case "source":
			key += fmt.Sprintf("source:%s,", event.Source)
		case "priority":
			key += fmt.Sprintf("priority:%d,", event.Priority)
		case "subject":
			key += fmt.Sprintf("subject:%s,", event.Subject)
		}
	}

	if key == "" {
		return "default"
	}

	return key[:len(key)-1] // 移除最后的逗号
}

// generateBatchID 生成批次ID
func (bp *BatchProcessor) generateBatchID() string {
	return fmt.Sprintf("batch_%d", time.Now().UnixNano())
}

// calculateRetryDelay 计算重试延迟
func (bp *BatchProcessor) calculateRetryDelay(attempt int) time.Duration {
	delay := bp.config.RetryConfig.InitialDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * bp.config.RetryConfig.BackoffFactor)
		if delay > bp.config.RetryConfig.MaxDelay {
			delay = bp.config.RetryConfig.MaxDelay
			break
		}
	}
	return delay
}

// compressBatch 压缩批次数据
func (bp *BatchProcessor) compressBatch(batch *EventBatch) error {
	// 这里可以实现具体的压缩逻辑
	// 例如使用 gzip 或其他压缩算法
	// 为了简化，这里只是示例
	return nil
}

// GetMetrics 获取批处理指标
func (bp *BatchProcessor) GetMetrics() *BatchMetrics {
	bp.metrics.mu.RLock()
	defer bp.metrics.mu.RUnlock()

	return &BatchMetrics{
		TotalBatches:       bp.metrics.TotalBatches,
		ProcessedBatches:   bp.metrics.ProcessedBatches,
		FailedBatches:      bp.metrics.FailedBatches,
		TotalEvents:        bp.metrics.TotalEvents,
		ProcessedEvents:    bp.metrics.ProcessedEvents,
		AverageProcessTime: bp.metrics.AverageProcessTime,
	}
}

// GetBatchStatus 获取批次状态
func (bp *BatchProcessor) GetBatchStatus(batchID string) (*EventBatch, error) {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	for _, batch := range bp.batches {
		batch.mu.RLock()
		if batch.ID == batchID {
			result := &EventBatch{
				ID:          batch.ID,
				GroupKey:    batch.GroupKey,
				Events:      batch.Events,
				CreatedAt:   batch.CreatedAt,
				LastUpdated: batch.LastUpdated,
				Status:      batch.Status,
				Priority:    batch.Priority,
				RetryCount:  batch.RetryCount,
				ProcessedAt: batch.ProcessedAt,
			}
			batch.mu.RUnlock()
			return result, nil
		}
		batch.mu.RUnlock()
	}

	return nil, fmt.Errorf("批次 %s 不存在", batchID)
}

// GetActiveBatches 获取活跃批次
func (bp *BatchProcessor) GetActiveBatches() []*EventBatch {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	batches := make([]*EventBatch, 0)
	for _, batch := range bp.batches {
		batch.mu.RLock()
		// 只返回活跃状态的批次
		if batch.Status == BatchStatusPending ||
			batch.Status == BatchStatusProcessing ||
			batch.Status == BatchStatusRetrying {
			batches = append(batches, &EventBatch{
				ID:          batch.ID,
				GroupKey:    batch.GroupKey,
				Events:      batch.Events,
				CreatedAt:   batch.CreatedAt,
				LastUpdated: batch.LastUpdated,
				Status:      batch.Status,
				Priority:    batch.Priority,
				RetryCount:  batch.RetryCount,
				ProcessedAt: batch.ProcessedAt,
			})
		}
		batch.mu.RUnlock()
	}

	return batches
}

// UpdateConfig 更新配置
func (bp *BatchProcessor) UpdateConfig(config *BatchConfig) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.config = config
}

// GetConfig 获取配置
func (bp *BatchProcessor) GetConfig() *BatchConfig {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	// 确保RetryConfig不为空，提供默认值
	var retryConfig *RetryConfig
	if bp.config.RetryConfig != nil {
		retryConfig = &RetryConfig{
			MaxRetries:    bp.config.RetryConfig.MaxRetries,
			InitialDelay:  bp.config.RetryConfig.InitialDelay,
			MaxDelay:      bp.config.RetryConfig.MaxDelay,
			BackoffFactor: bp.config.RetryConfig.BackoffFactor,
		}
	} else {
		retryConfig = &RetryConfig{
			MaxRetries:    3,
			InitialDelay:  time.Second,
			MaxDelay:      time.Minute,
			BackoffFactor: 2.0,
		}
	}

	return &BatchConfig{
		MaxBatchSize:      bp.config.MaxBatchSize,
		MaxWaitTime:       bp.config.MaxWaitTime,
		FlushInterval:     bp.config.FlushInterval,
		MaxConcurrency:    bp.config.MaxConcurrency,
		GroupByFields:     bp.config.GroupByFields,
		EnableCompression: bp.config.EnableCompression,
		RetryConfig:       retryConfig,
	}
}
