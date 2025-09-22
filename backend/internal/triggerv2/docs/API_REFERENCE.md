# TriggerV2 API 参考文档

## 概述

TriggerV2 提供了完整的 Go API 接口，用于事件处理、触发器管理、条件评估、批处理、监控等功能。

## 核心模块

### 1. 事件总线 (EventBus)

事件总线是系统的核心组件，负责事件的发布、订阅和路由。

#### 类型定义

```go
type EventBus struct {
    // 私有字段
}

type EventBusConfig struct {
    MaxEventSize    int           // 最大事件大小（字节）
    BufferSize      int           // 缓冲区大小
    WorkerCount     int           // 工作线程数
    ProcessTimeout  time.Duration // 处理超时时间
    EnableMetrics   bool          // 启用指标收集
}
```

#### 构造函数

```go
func NewEventBus() *EventBus
func NewEventBusWithConfig(config EventBusConfig) *EventBus
```

#### 主要方法

##### PublishEvent
发布事件到事件总线。

```go
func (eb *EventBus) PublishEvent(ctx context.Context, event *models.Event) error
```

**参数：**
- `ctx` - 上下文
- `event` - 要发布的事件

**返回值：**
- `error` - 发布失败时返回错误

**示例：**
```go
event := &models.Event{
    ID:     "event-001",
    Type:   "user.created",
    Source: "user-service",
    Data: map[string]interface{}{
        "user_id": "user-123",
        "email":   "user@example.com",
    },
    Timestamp: time.Now(),
}

err := eventBus.PublishEvent(ctx, event)
if err != nil {
    log.Printf("Failed to publish event: %v", err)
}
```

##### RegisterTrigger
注册触发器到事件总线。

```go
func (eb *EventBus) RegisterTrigger(trigger *models.Trigger) error
```

**参数：**
- `trigger` - 要注册的触发器

**返回值：**
- `error` - 注册失败时返回错误

##### UnregisterTrigger
注销触发器。

```go
func (eb *EventBus) UnregisterTrigger(triggerID string) error
```

##### GetTriggers
获取所有注册的触发器。

```go
func (eb *EventBus) GetTriggers() []*models.Trigger
```

##### GetStats
获取事件总线统计信息。

```go
func (eb *EventBus) GetStats() *EventBusStats

type EventBusStats struct {
    EventsPublished   int64     // 发布的事件总数
    EventsProcessed   int64     // 处理的事件总数
    EventsFailed      int64     // 失败的事件总数
    TriggerCount      int       // 触发器总数
    AverageProcessTime time.Duration // 平均处理时间
    LastEventTime     time.Time // 最后事件时间
}
```

### 2. 条件引擎 (ConditionEngine)

条件引擎负责评估复杂的条件逻辑。

#### 类型定义

```go
type ConditionEngine struct {
    // 私有字段
}

type ConditionEngineConfig struct {
    MaxDepth          int           // 最大嵌套深度
    EvaluationTimeout time.Duration // 评估超时时间
    CacheSize         int           // 缓存大小
}
```

#### 构造函数

```go
func NewConditionEngine() *ConditionEngine
func NewConditionEngineWithConfig(config ConditionEngineConfig) *ConditionEngine
```

#### 主要方法

##### EvaluateCondition
评估条件表达式。

```go
func (ce *ConditionEngine) EvaluateCondition(
    ctx context.Context,
    condition map[string]interface{},
    event *models.Event,
) (bool, error)
```

**参数：**
- `ctx` - 上下文
- `condition` - 条件表达式
- `event` - 事件数据

**返回值：**
- `bool` - 条件是否满足
- `error` - 评估失败时返回错误

**示例：**
```go
condition := map[string]interface{}{
    "type": "logical",
    "operator": "and",
    "conditions": []map[string]interface{}{
        {
            "type":     "comparison",
            "field":    "user.age",
            "operator": "gte",
            "value":    18,
        },
        {
            "type":     "comparison",
            "field":    "order.total",
            "operator": "gt",
            "value":    100,
        },
    },
}

result, err := conditionEngine.EvaluateCondition(ctx, condition, event)
if err != nil {
    log.Printf("Failed to evaluate condition: %v", err)
}
```

##### RegisterOperator
注册自定义操作符。

```go
func (ce *ConditionEngine) RegisterOperator(name string, op Operator)

type Operator interface {
    Evaluate(left, right interface{}) (bool, error)
}
```

##### RegisterFunction
注册自定义函数。

```go
func (ce *ConditionEngine) RegisterFunction(name string, fn Function)

type Function interface {
    Call(args []interface{}) (interface{}, error)
}
```

### 3. 批处理器 (BatchProcessor)

批处理器用于批量处理事件，提高处理效率。

#### 类型定义

```go
type BatchProcessor struct {
    // 私有字段
}

type BatchProcessorConfig struct {
    MaxBatchSize   int           // 最大批处理大小
    FlushInterval  time.Duration // 刷新间隔
    MaxRetries     int           // 最大重试次数
    RetryInterval  time.Duration // 重试间隔
    MaxConcurrency int           // 最大并发数
}
```

#### 构造函数

```go
func NewBatchProcessor(config BatchProcessorConfig) *BatchProcessor
```

#### 主要方法

##### AddEvent
添加事件到批处理器。

```go
func (bp *BatchProcessor) AddEvent(
    ctx context.Context,
    event *models.Event,
    processor BatchEventProcessor,
) error

type BatchEventProcessor func(ctx context.Context, events []*models.Event) error
```

**参数：**
- `ctx` - 上下文
- `event` - 要处理的事件
- `processor` - 批处理函数

**示例：**
```go
processor := func(ctx context.Context, events []*models.Event) error {
    log.Printf("Processing batch of %d events", len(events))
    for _, event := range events {
        // 处理单个事件
        if err := processEvent(event); err != nil {
            return err
        }
    }
    return nil
}

err := batchProcessor.AddEvent(ctx, event, processor)
if err != nil {
    log.Printf("Failed to add event to batch: %v", err)
}
```

##### GetMetrics
获取批处理指标。

```go
func (bp *BatchProcessor) GetMetrics() *BatchMetrics

type BatchMetrics struct {
    EventsReceived    int64         // 接收的事件总数
    EventsProcessed   int64         // 处理的事件总数
    BatchesProcessed  int64         // 处理的批次总数
    AverageBatchSize  float64       // 平均批处理大小
    ProcessingTime    time.Duration // 总处理时间
    RetryCount        int64         // 重试次数
}
```

### 4. 监控系统 (Monitor)

监控系统提供指标收集、告警管理和健康检查功能。

#### 类型定义

```go
type Monitor struct {
    // 私有字段
}

type MonitorConfig struct {
    MetricsRetention    time.Duration // 指标保留时间
    AlertCooldown       time.Duration // 告警冷却时间
    HealthCheckInterval time.Duration // 健康检查间隔
}
```

#### 构造函数

```go
func NewMonitor(config MonitorConfig) *Monitor
```

#### 主要方法

##### RegisterCollector
注册指标收集器。

```go
func (m *Monitor) RegisterCollector(collector MetricsCollector)

type MetricsCollector interface {
    GetName() string
    GetInterval() time.Duration
    Collect(ctx context.Context) (*MetricSet, error)
}
```

##### AddAlertRule
添加告警规则。

```go
func (m *Monitor) AddAlertRule(rule *AlertRule) error

type AlertRule struct {
    Name        string         // 规则名称
    Description string         // 规则描述
    Condition   *AlertCondition // 告警条件
    Severity    AlertSeverity  // 告警级别
    Actions     []AlertAction  // 告警动作
}
```

##### GetHealth
获取系统健康状态。

```go
func (m *Monitor) GetHealth() *HealthStatus

type HealthStatus struct {
    Status     string                    // 健康状态
    Components map[string]ComponentHealth // 组件健康状态
    Timestamp  time.Time                 // 检查时间
}
```

### 5. 插件管理器 (PluginManager)

插件管理器负责插件的注册、管理和执行。

#### 类型定义

```go
type PluginManager struct {
    // 私有字段
}

type Plugin interface {
    GetName() string
    GetVersion() string
    Execute(ctx context.Context, action *models.Action, event *models.Event) error
}
```

#### 构造函数

```go
func NewPluginManager() *PluginManager
```

#### 主要方法

##### RegisterPlugin
注册插件。

```go
func (pm *PluginManager) RegisterPlugin(plugin Plugin) error
```

##### GetPlugin
获取插件。

```go
func (pm *PluginManager) GetPlugin(name string) Plugin
```

##### GetAvailablePlugins
获取可用插件列表。

```go
func (pm *PluginManager) GetAvailablePlugins() []string
```

## 数据模型

### Event 事件模型

```go
type Event struct {
    ID        string                 `json:"id"`        // 事件唯一标识
    Type      string                 `json:"type"`      // 事件类型
    Source    string                 `json:"source"`    // 事件来源
    Data      map[string]interface{} `json:"data"`      // 事件数据
    Timestamp time.Time              `json:"timestamp"` // 事件时间戳
    Version   string                 `json:"version"`   // 事件版本
    Metadata  map[string]interface{} `json:"metadata"`  // 元数据
}
```

**字段说明：**
- `ID` - 事件的唯一标识符，用于事件去重和追踪
- `Type` - 事件类型，建议使用 `域.动作` 格式
- `Source` - 事件来源，标识产生事件的系统或组件
- `Data` - 事件的业务数据，包含事件的详细信息
- `Timestamp` - 事件发生的时间戳
- `Version` - 事件模式版本，用于兼容性管理
- `Metadata` - 元数据，用于存储额外的系统信息

### Trigger 触发器模型

```go
type Trigger struct {
    ID          string                 `json:"id"`          // 触发器唯一标识
    Name        string                 `json:"name"`        // 触发器名称
    Description string                 `json:"description"` // 触发器描述
    EventTypes  []string               `json:"event_types"` // 监听的事件类型
    Conditions  map[string]interface{} `json:"conditions"`  // 触发条件
    Actions     []Action               `json:"actions"`     // 执行动作
    Status      TriggerStatus          `json:"status"`      // 触发器状态
    CreatedAt   time.Time              `json:"created_at"`  // 创建时间
    UpdatedAt   time.Time              `json:"updated_at"`  // 更新时间
    Version     int                    `json:"version"`     // 版本号
}

type TriggerStatus string

const (
    TriggerStatusActive   TriggerStatus = "ACTIVE"   // 活跃状态
    TriggerStatusInactive TriggerStatus = "INACTIVE" // 非活跃状态
    TriggerStatusDraft    TriggerStatus = "DRAFT"    // 草稿状态
)
```

### Action 动作模型

```go
type Action struct {
    Type   string                 `json:"type"`   // 动作类型
    Config map[string]interface{} `json:"config"` // 动作配置
}
```

**支持的动作类型：**
- `email` - 邮件通知
- `webhook` - HTTP 回调
- `slack` - Slack 通知
- `sms` - 短信通知
- `database` - 数据库操作

## 使用模式

### 1. 基本事件处理

```go
package main

import (
    "context"
    "log"
    "time"
    
    "mailman/backend/internal/triggerv2/core"
    "mailman/backend/internal/triggerv2/models"
)

func main() {
    // 创建事件总线
    eventBus := core.NewEventBus()
    
    // 创建触发器
    trigger := &models.Trigger{
        ID:         "user-welcome",
        Name:       "用户欢迎",
        EventTypes: []string{"user.created"},
        Conditions: map[string]interface{}{
            "type":     "comparison",
            "field":    "user.verified",
            "operator": "eq",
            "value":    true,
        },
        Actions: []models.Action{
            {
                Type: "email",
                Config: map[string]interface{}{
                    "template": "welcome",
                    "to":       "{{user.email}}",
                },
            },
        },
        Status: models.TriggerStatusActive,
    }
    
    // 注册触发器
    if err := eventBus.RegisterTrigger(trigger); err != nil {
        log.Fatal(err)
    }
    
    // 发布事件
    event := &models.Event{
        ID:     "user-001",
        Type:   "user.created",
        Source: "user-service",
        Data: map[string]interface{}{
            "user": map[string]interface{}{
                "id":       "user-001",
                "email":    "user@example.com",
                "verified": true,
            },
        },
        Timestamp: time.Now(),
    }
    
    ctx := context.Background()
    if err := eventBus.PublishEvent(ctx, event); err != nil {
        log.Printf("Failed to publish event: %v", err)
    }
}
```

### 2. 批处理模式

```go
package main

import (
    "context"
    "log"
    "time"
    
    "mailman/backend/internal/triggerv2/engine"
    "mailman/backend/internal/triggerv2/models"
)

func main() {
    // 创建批处理器
    batchProcessor := engine.NewBatchProcessor(engine.BatchProcessorConfig{
        MaxBatchSize:   100,
        FlushInterval:  5 * time.Second,
        MaxRetries:     3,
        RetryInterval:  time.Second,
        MaxConcurrency: 10,
    })
    
    // 批处理函数
    processor := func(ctx context.Context, events []*models.Event) error {
        log.Printf("Processing batch of %d events", len(events))
        
        // 批量处理事件
        for _, event := range events {
            // 处理单个事件
            if err := processEvent(event); err != nil {
                return err
            }
        }
        
        return nil
    }
    
    ctx := context.Background()
    
    // 添加事件到批处理器
    for i := 0; i < 1000; i++ {
        event := &models.Event{
            ID:     fmt.Sprintf("batch-event-%d", i),
            Type:   "batch.test",
            Source: "batch-service",
            Data: map[string]interface{}{
                "index": i,
            },
            Timestamp: time.Now(),
        }
        
        if err := batchProcessor.AddEvent(ctx, event, processor); err != nil {
            log.Printf("Failed to add event: %v", err)
        }
    }
}

func processEvent(event *models.Event) error {
    // 处理单个事件的逻辑
    return nil
}
```

### 3. 监控和告警

```go
package main

import (
    "context"
    "log"
    "time"
    
    "mailman/backend/internal/triggerv2/monitoring"
)

func main() {
    // 创建监控器
    monitor := monitoring.NewMonitor(monitoring.MonitorConfig{
        MetricsRetention: 24 * time.Hour,
        AlertCooldown:    5 * time.Minute,
    })
    
    // 创建自定义指标收集器
    collector := &CustomMetricsCollector{
        name:     "custom-metrics",
        interval: time.Minute,
    }
    
    // 注册收集器
    monitor.RegisterCollector(collector)
    
    // 添加告警规则
    alertRule := &monitoring.AlertRule{
        Name:        "high-error-rate",
        Description: "高错误率告警",
        Condition: &monitoring.AlertCondition{
            MetricName: "error_rate",
            Operator:   "gt",
            Threshold:  0.05,
        },
        Severity: monitoring.AlertSeverityWarning,
        Actions: []monitoring.AlertAction{
            {
                Type: "email",
                Config: map[string]interface{}{
                    "to": "admin@example.com",
                },
            },
        },
    }
    
    if err := monitor.AddAlertRule(alertRule); err != nil {
        log.Printf("Failed to add alert rule: %v", err)
    }
    
    // 启动监控
    ctx := context.Background()
    monitor.Start(ctx)
    
    // 检查健康状态
    health := monitor.GetHealth()
    log.Printf("System health: %s", health.Status)
}

type CustomMetricsCollector struct {
    name     string
    interval time.Duration
}

func (c *CustomMetricsCollector) GetName() string {
    return c.name
}

func (c *CustomMetricsCollector) GetInterval() time.Duration {
    return c.interval
}

func (c *CustomMetricsCollector) Collect(ctx context.Context) (*monitoring.MetricSet, error) {
    return &monitoring.MetricSet{
        Name:      c.name,
        Timestamp: time.Now(),
        Metrics: map[string]interface{}{
            "custom_metric": 42,
            "error_rate":    0.02,
        },
    }, nil
}
```

## 错误处理

### 错误类型

```go
// 验证错误
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// 条件评估错误
type ConditionEvaluationError struct {
    Condition string
    Cause     error
}

func (e *ConditionEvaluationError) Error() string {
    return fmt.Sprintf("condition evaluation error: %s, cause: %v", e.Condition, e.Cause)
}

// 插件执行错误
type PluginExecutionError struct {
    PluginName string
    ActionType string
    Cause      error
}

func (e *PluginExecutionError) Error() string {
    return fmt.Sprintf("plugin execution error: %s/%s, cause: %v", 
        e.PluginName, e.ActionType, e.Cause)
}
```

### 错误处理最佳实践

1. **使用上下文取消**：
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := eventBus.PublishEvent(ctx, event)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Printf("Event publish timed out")
    }
}
```

2. **错误重试**：
```go
func publishEventWithRetry(eventBus *core.EventBus, event *models.Event) error {
    maxRetries := 3
    var lastErr error
    
    for i := 0; i < maxRetries; i++ {
        ctx := context.Background()
        err := eventBus.PublishEvent(ctx, event)
        if err == nil {
            return nil
        }
        
        lastErr = err
        time.Sleep(time.Second * time.Duration(i+1))
    }
    
    return fmt.Errorf("failed to publish event after %d retries: %w", maxRetries, lastErr)
}
```

3. **错误日志记录**：
```go
func handleError(err error, event *models.Event) {
    switch e := err.(type) {
    case *ValidationError:
        log.Printf("Validation error for event %s: %v", event.ID, e)
    case *ConditionEvaluationError:
        log.Printf("Condition evaluation error for event %s: %v", event.ID, e)
    case *PluginExecutionError:
        log.Printf("Plugin execution error for event %s: %v", event.ID, e)
    default:
        log.Printf("Unknown error for event %s: %v", event.ID, e)
    }
}
```

## 配置参考

### 环境变量配置

```bash
# 事件总线配置
TRIGGERV2_EVENTBUS_MAX_EVENT_SIZE=1048576    # 1MB
TRIGGERV2_EVENTBUS_BUFFER_SIZE=1000
TRIGGERV2_EVENTBUS_WORKER_COUNT=10
TRIGGERV2_EVENTBUS_PROCESS_TIMEOUT=30s

# 批处理配置
TRIGGERV2_BATCH_MAX_SIZE=100
TRIGGERV2_BATCH_FLUSH_INTERVAL=5s
TRIGGERV2_BATCH_MAX_RETRIES=3
TRIGGERV2_BATCH_RETRY_INTERVAL=1s
TRIGGERV2_BATCH_MAX_CONCURRENCY=10

# 监控配置
TRIGGERV2_MONITOR_METRICS_RETENTION=24h
TRIGGERV2_MONITOR_ALERT_COOLDOWN=5m
TRIGGERV2_MONITOR_HEALTH_CHECK_INTERVAL=30s
```

### 配置文件示例

```yaml
# config.yaml
triggerv2:
  eventbus:
    max_event_size: 1048576
    buffer_size: 1000
    worker_count: 10
    process_timeout: 30s
    enable_metrics: true
    
  batch_processor:
    max_batch_size: 100
    flush_interval: 5s
    max_retries: 3
    retry_interval: 1s
    max_concurrency: 10
    
  monitor:
    metrics_retention: 24h
    alert_cooldown: 5m
    health_check_interval: 30s
    
  plugins:
    email:
      smtp_host: smtp.example.com
      smtp_port: 587
      username: user@example.com
      password: password
    
    webhook:
      timeout: 30s
      max_retries: 3
      
    slack:
      webhook_url: https://hooks.slack.com/services/...
```

## 版本兼容性

### API 版本控制

TriggerV2 使用语义化版本控制：

- **主版本号**：不兼容的 API 更改
- **次版本号**：向后兼容的功能新增
- **修订号**：向后兼容的问题修复

### 版本迁移指南

从 v1.x 迁移到 v2.x：

1. **更新导入路径**：
```go
// v1.x
import "mailman/backend/internal/trigger/core"

// v2.x
import "mailman/backend/internal/triggerv2/core"
```

2. **更新配置结构**：
```go
// v1.x
config := trigger.Config{
    Workers: 10,
}

// v2.x
config := core.EventBusConfig{
    WorkerCount: 10,
}
```

3. **更新接口调用**：
```go
// v1.x
err := eventBus.Publish(event)

// v2.x
err := eventBus.PublishEvent(ctx, event)
```

---

*本文档基于 TriggerV2 v2.0.0 版本编写，最后更新于：2024年*