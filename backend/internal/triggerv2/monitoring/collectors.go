package monitoring

import (
	"context"
	"runtime"
	"time"
)

// SystemMetricsCollector 系统指标收集器
type SystemMetricsCollector struct {
	name     string
	interval time.Duration
}

// NewSystemMetricsCollector 创建系统指标收集器
func NewSystemMetricsCollector() *SystemMetricsCollector {
	return &SystemMetricsCollector{
		name:     "system",
		interval: time.Second * 30,
	}
}

func (smc *SystemMetricsCollector) GetName() string {
	return smc.name
}

func (smc *SystemMetricsCollector) GetInterval() time.Duration {
	return smc.interval
}

func (smc *SystemMetricsCollector) Collect(ctx context.Context) (*MetricSet, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metrics := map[string]interface{}{
		"memory": map[string]interface{}{
			"alloc":           memStats.Alloc,
			"total_alloc":     memStats.TotalAlloc,
			"sys":             memStats.Sys,
			"heap_alloc":      memStats.HeapAlloc,
			"heap_sys":        memStats.HeapSys,
			"heap_idle":       memStats.HeapIdle,
			"heap_inuse":      memStats.HeapInuse,
			"heap_released":   memStats.HeapReleased,
			"heap_objects":    memStats.HeapObjects,
			"stack_inuse":     memStats.StackInuse,
			"stack_sys":       memStats.StackSys,
			"gc_sys":          memStats.GCSys,
			"other_sys":       memStats.OtherSys,
		},
		"gc": map[string]interface{}{
			"num_gc":         memStats.NumGC,
			"num_forced_gc":  memStats.NumForcedGC,
			"gc_cpu_fraction": memStats.GCCPUFraction,
			"pause_total_ns": memStats.PauseTotalNs,
			"pause_ns":       memStats.PauseNs,
		},
		"goroutines": runtime.NumGoroutine(),
		"num_cpu":    runtime.NumCPU(),
		"gomaxprocs": runtime.GOMAXPROCS(0),
	}

	return &MetricSet{
		Name:      smc.name,
		Timestamp: time.Now(),
		Metrics:   metrics,
	}, nil
}

// TriggerV2MetricsCollector TriggerV2系统指标收集器
type TriggerV2MetricsCollector struct {
	name       string
	interval   time.Duration
	eventBus   EventBusMetrics
	scheduler  SchedulerMetrics
	executor   ExecutorMetrics
	taskQueue  TaskQueueMetrics
}

// EventBusMetrics 事件总线指标接口
type EventBusMetrics interface {
	GetTotalEvents() int64
	GetProcessedEvents() int64
	GetFailedEvents() int64
	GetSubscriberCount() int
}

// SchedulerMetrics 调度器指标接口
type SchedulerMetrics interface {
	GetScheduledTasks() int64
	GetCompletedTasks() int64
	GetFailedTasks() int64
	GetAverageExecutionTime() time.Duration
}

// ExecutorMetrics 执行器指标接口
type ExecutorMetrics interface {
	GetActiveWorkers() int
	GetQueuedTasks() int64
	GetCompletedTasks() int64
	GetFailedTasks() int64
	GetAverageExecutionTime() time.Duration
}

// TaskQueueMetrics 任务队列指标接口
type TaskQueueMetrics interface {
	GetQueueSize() int64
	GetProcessedTasks() int64
	GetFailedTasks() int64
	GetAverageWaitTime() time.Duration
}

// NewTriggerV2MetricsCollector 创建TriggerV2指标收集器
func NewTriggerV2MetricsCollector(eventBus EventBusMetrics, scheduler SchedulerMetrics, executor ExecutorMetrics, taskQueue TaskQueueMetrics) *TriggerV2MetricsCollector {
	return &TriggerV2MetricsCollector{
		name:       "triggerv2",
		interval:   time.Second * 15,
		eventBus:   eventBus,
		scheduler:  scheduler,
		executor:   executor,
		taskQueue:  taskQueue,
	}
}

func (tmc *TriggerV2MetricsCollector) GetName() string {
	return tmc.name
}

func (tmc *TriggerV2MetricsCollector) GetInterval() time.Duration {
	return tmc.interval
}

func (tmc *TriggerV2MetricsCollector) Collect(ctx context.Context) (*MetricSet, error) {
	metrics := map[string]interface{}{
		"event_bus": map[string]interface{}{
			"total_events":     tmc.eventBus.GetTotalEvents(),
			"processed_events": tmc.eventBus.GetProcessedEvents(),
			"failed_events":    tmc.eventBus.GetFailedEvents(),
			"subscriber_count": tmc.eventBus.GetSubscriberCount(),
		},
		"scheduler": map[string]interface{}{
			"scheduled_tasks":        tmc.scheduler.GetScheduledTasks(),
			"completed_tasks":        tmc.scheduler.GetCompletedTasks(),
			"failed_tasks":           tmc.scheduler.GetFailedTasks(),
			"average_execution_time": tmc.scheduler.GetAverageExecutionTime().Milliseconds(),
		},
		"executor": map[string]interface{}{
			"active_workers":         tmc.executor.GetActiveWorkers(),
			"queued_tasks":           tmc.executor.GetQueuedTasks(),
			"completed_tasks":        tmc.executor.GetCompletedTasks(),
			"failed_tasks":           tmc.executor.GetFailedTasks(),
			"average_execution_time": tmc.executor.GetAverageExecutionTime().Milliseconds(),
		},
		"task_queue": map[string]interface{}{
			"queue_size":        tmc.taskQueue.GetQueueSize(),
			"processed_tasks":   tmc.taskQueue.GetProcessedTasks(),
			"failed_tasks":      tmc.taskQueue.GetFailedTasks(),
			"average_wait_time": tmc.taskQueue.GetAverageWaitTime().Milliseconds(),
		},
	}

	return &MetricSet{
		Name:      tmc.name,
		Timestamp: time.Now(),
		Metrics:   metrics,
	}, nil
}

// BatchProcessorMetricsCollector 批处理器指标收集器
type BatchProcessorMetricsCollector struct {
	name           string
	interval       time.Duration
	batchProcessor BatchProcessorMetrics
}

// BatchProcessorMetrics 批处理器指标接口
type BatchProcessorMetrics interface {
	GetTotalBatches() int64
	GetProcessedBatches() int64
	GetFailedBatches() int64
	GetTotalEvents() int64
	GetProcessedEvents() int64
	GetAverageProcessTime() time.Duration
	GetActiveBatchCount() int
}

// NewBatchProcessorMetricsCollector 创建批处理器指标收集器
func NewBatchProcessorMetricsCollector(batchProcessor BatchProcessorMetrics) *BatchProcessorMetricsCollector {
	return &BatchProcessorMetricsCollector{
		name:           "batch_processor",
		interval:       time.Second * 30,
		batchProcessor: batchProcessor,
	}
}

func (bmc *BatchProcessorMetricsCollector) GetName() string {
	return bmc.name
}

func (bmc *BatchProcessorMetricsCollector) GetInterval() time.Duration {
	return bmc.interval
}

func (bmc *BatchProcessorMetricsCollector) Collect(ctx context.Context) (*MetricSet, error) {
	metrics := map[string]interface{}{
		"total_batches":        bmc.batchProcessor.GetTotalBatches(),
		"processed_batches":    bmc.batchProcessor.GetProcessedBatches(),
		"failed_batches":       bmc.batchProcessor.GetFailedBatches(),
		"total_events":         bmc.batchProcessor.GetTotalEvents(),
		"processed_events":     bmc.batchProcessor.GetProcessedEvents(),
		"average_process_time": bmc.batchProcessor.GetAverageProcessTime().Milliseconds(),
		"active_batch_count":   bmc.batchProcessor.GetActiveBatchCount(),
		"batch_success_rate":   bmc.calculateSuccessRate(),
		"event_throughput":     bmc.calculateEventThroughput(),
	}

	return &MetricSet{
		Name:      bmc.name,
		Timestamp: time.Now(),
		Metrics:   metrics,
	}, nil
}

func (bmc *BatchProcessorMetricsCollector) calculateSuccessRate() float64 {
	total := bmc.batchProcessor.GetTotalBatches()
	if total == 0 {
		return 0.0
	}
	processed := bmc.batchProcessor.GetProcessedBatches()
	return float64(processed) / float64(total) * 100.0
}

func (bmc *BatchProcessorMetricsCollector) calculateEventThroughput() float64 {
	processed := bmc.batchProcessor.GetProcessedEvents()
	avgTime := bmc.batchProcessor.GetAverageProcessTime()
	if avgTime == 0 {
		return 0.0
	}
	return float64(processed) / avgTime.Seconds()
}

// PluginMetricsCollector 插件系统指标收集器
type PluginMetricsCollector struct {
	name          string
	interval      time.Duration
	pluginManager PluginManagerMetrics
}

// PluginManagerMetrics 插件管理器指标接口
type PluginManagerMetrics interface {
	GetRegisteredPlugins() int
	GetActivePlugins() int
	GetExecutedPlugins() int64
	GetFailedPlugins() int64
	GetAverageExecutionTime() time.Duration
}

// NewPluginMetricsCollector 创建插件指标收集器
func NewPluginMetricsCollector(pluginManager PluginManagerMetrics) *PluginMetricsCollector {
	return &PluginMetricsCollector{
		name:          "plugins",
		interval:      time.Second * 30,
		pluginManager: pluginManager,
	}
}

func (pmc *PluginMetricsCollector) GetName() string {
	return pmc.name
}

func (pmc *PluginMetricsCollector) GetInterval() time.Duration {
	return pmc.interval
}

func (pmc *PluginMetricsCollector) Collect(ctx context.Context) (*MetricSet, error) {
	metrics := map[string]interface{}{
		"registered_plugins":     pmc.pluginManager.GetRegisteredPlugins(),
		"active_plugins":         pmc.pluginManager.GetActivePlugins(),
		"executed_plugins":       pmc.pluginManager.GetExecutedPlugins(),
		"failed_plugins":         pmc.pluginManager.GetFailedPlugins(),
		"average_execution_time": pmc.pluginManager.GetAverageExecutionTime().Milliseconds(),
		"plugin_success_rate":    pmc.calculateSuccessRate(),
	}

	return &MetricSet{
		Name:      pmc.name,
		Timestamp: time.Now(),
		Metrics:   metrics,
	}, nil
}

func (pmc *PluginMetricsCollector) calculateSuccessRate() float64 {
	total := pmc.pluginManager.GetExecutedPlugins()
	if total == 0 {
		return 0.0
	}
	failed := pmc.pluginManager.GetFailedPlugins()
	successful := total - failed
	return float64(successful) / float64(total) * 100.0
}

// ConditionEngineMetricsCollector 条件引擎指标收集器
type ConditionEngineMetricsCollector struct {
	name            string
	interval        time.Duration
	conditionEngine ConditionEngineMetrics
}

// ConditionEngineMetrics 条件引擎指标接口
type ConditionEngineMetrics interface {
	GetTotalEvaluations() int64
	GetSuccessfulEvaluations() int64
	GetFailedEvaluations() int64
	GetAverageEvaluationTime() time.Duration
	GetRegisteredOperators() int
	GetRegisteredFunctions() int
}

// NewConditionEngineMetricsCollector 创建条件引擎指标收集器
func NewConditionEngineMetricsCollector(conditionEngine ConditionEngineMetrics) *ConditionEngineMetricsCollector {
	return &ConditionEngineMetricsCollector{
		name:            "condition_engine",
		interval:        time.Second * 30,
		conditionEngine: conditionEngine,
	}
}

func (cmc *ConditionEngineMetricsCollector) GetName() string {
	return cmc.name
}

func (cmc *ConditionEngineMetricsCollector) GetInterval() time.Duration {
	return cmc.interval
}

func (cmc *ConditionEngineMetricsCollector) Collect(ctx context.Context) (*MetricSet, error) {
	metrics := map[string]interface{}{
		"total_evaluations":       cmc.conditionEngine.GetTotalEvaluations(),
		"successful_evaluations":  cmc.conditionEngine.GetSuccessfulEvaluations(),
		"failed_evaluations":      cmc.conditionEngine.GetFailedEvaluations(),
		"average_evaluation_time": cmc.conditionEngine.GetAverageEvaluationTime().Microseconds(),
		"registered_operators":    cmc.conditionEngine.GetRegisteredOperators(),
		"registered_functions":    cmc.conditionEngine.GetRegisteredFunctions(),
		"evaluation_success_rate": cmc.calculateSuccessRate(),
	}

	return &MetricSet{
		Name:      cmc.name,
		Timestamp: time.Now(),
		Metrics:   metrics,
	}, nil
}

func (cmc *ConditionEngineMetricsCollector) calculateSuccessRate() float64 {
	total := cmc.conditionEngine.GetTotalEvaluations()
	if total == 0 {
		return 0.0
	}
	successful := cmc.conditionEngine.GetSuccessfulEvaluations()
	return float64(successful) / float64(total) * 100.0
}

// CustomMetricsCollector 自定义指标收集器
type CustomMetricsCollector struct {
	name     string
	interval time.Duration
	collectFn func(ctx context.Context) (map[string]interface{}, error)
}

// NewCustomMetricsCollector 创建自定义指标收集器
func NewCustomMetricsCollector(name string, interval time.Duration, collectFn func(ctx context.Context) (map[string]interface{}, error)) *CustomMetricsCollector {
	return &CustomMetricsCollector{
		name:      name,
		interval:  interval,
		collectFn: collectFn,
	}
}

func (cmc *CustomMetricsCollector) GetName() string {
	return cmc.name
}

func (cmc *CustomMetricsCollector) GetInterval() time.Duration {
	return cmc.interval
}

func (cmc *CustomMetricsCollector) Collect(ctx context.Context) (*MetricSet, error) {
	metrics, err := cmc.collectFn(ctx)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		Name:      cmc.name,
		Timestamp: time.Now(),
		Metrics:   metrics,
	}, nil
}

// AggregatedMetricsCollector 聚合指标收集器
type AggregatedMetricsCollector struct {
	name       string
	interval   time.Duration
	collectors []MetricsCollector
}

// NewAggregatedMetricsCollector 创建聚合指标收集器
func NewAggregatedMetricsCollector(name string, interval time.Duration, collectors []MetricsCollector) *AggregatedMetricsCollector {
	return &AggregatedMetricsCollector{
		name:       name,
		interval:   interval,
		collectors: collectors,
	}
}

func (amc *AggregatedMetricsCollector) GetName() string {
	return amc.name
}

func (amc *AggregatedMetricsCollector) GetInterval() time.Duration {
	return amc.interval
}

func (amc *AggregatedMetricsCollector) Collect(ctx context.Context) (*MetricSet, error) {
	aggregatedMetrics := make(map[string]interface{})

	for _, collector := range amc.collectors {
		metricSet, err := collector.Collect(ctx)
		if err != nil {
			// 记录错误但继续收集其他指标
			aggregatedMetrics[collector.GetName()+"_error"] = err.Error()
			continue
		}

		if metricSet != nil {
			aggregatedMetrics[collector.GetName()] = metricSet.Metrics
		}
	}

	return &MetricSet{
		Name:      amc.name,
		Timestamp: time.Now(),
		Metrics:   aggregatedMetrics,
	}, nil
}

// AddCollector 添加收集器
func (amc *AggregatedMetricsCollector) AddCollector(collector MetricsCollector) {
	amc.collectors = append(amc.collectors, collector)
}

// RemoveCollector 移除收集器
func (amc *AggregatedMetricsCollector) RemoveCollector(name string) {
	for i, collector := range amc.collectors {
		if collector.GetName() == name {
			amc.collectors = append(amc.collectors[:i], amc.collectors[i+1:]...)
			break
		}
	}
}