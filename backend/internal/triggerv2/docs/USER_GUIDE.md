# TriggerV2 用户指南

## 概述

TriggerV2 是一个高性能、可扩展的事件驱动触发器系统，支持复杂的条件逻辑、批处理优化和全面的监控告警功能。

## 核心概念

### 事件（Event）

事件是系统中的基本数据单元，代表系统中发生的特定活动或状态变更。

```go
type Event struct {
    ID        string                 // 事件唯一标识
    Type      string                 // 事件类型
    Source    string                 // 事件来源
    Data      map[string]interface{} // 事件数据
    Timestamp time.Time              // 事件时间戳
    Version   string                 // 事件版本
    Metadata  map[string]interface{} // 元数据
}
```

**事件类型示例：**
- `user.created` - 用户创建
- `order.placed` - 订单创建
- `payment.completed` - 支付完成
- `system.alert` - 系统告警

### 触发器（Trigger）

触发器定义了对特定事件的响应逻辑，包括条件判断和执行动作。

```go
type Trigger struct {
    ID          string                 // 触发器唯一标识
    Name        string                 // 触发器名称
    Description string                 // 触发器描述
    EventTypes  []string              // 监听的事件类型
    Conditions  map[string]interface{} // 触发条件
    Actions     []Action              // 执行动作
    Status      TriggerStatus         // 触发器状态
    CreatedAt   time.Time             // 创建时间
    UpdatedAt   time.Time             // 更新时间
}
```

**触发器状态：**
- `ACTIVE` - 活跃状态，正常处理事件
- `INACTIVE` - 非活跃状态，暂停处理事件
- `DRAFT` - 草稿状态，尚未发布

### 动作（Action）

动作定义了触发器匹配后要执行的操作。

```go
type Action struct {
    Type   string                 // 动作类型
    Config map[string]interface{} // 动作配置
}
```

**支持的动作类型：**
- `email` - 发送邮件
- `webhook` - HTTP回调
- `slack` - Slack通知
- `sms` - 短信通知
- `database` - 数据库操作

## 快速开始

### 1. 基本使用

```go
package main

import (
    "context"
    "time"
    
    "mailman/backend/internal/triggerv2/core"
    "mailman/backend/internal/triggerv2/models"
)

func main() {
    // 创建事件总线
    eventBus := core.NewEventBus()
    
    // 创建触发器
    trigger := &models.Trigger{
        ID:         "welcome-trigger",
        Name:       "用户欢迎触发器",
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
                    "template": "welcome_email",
                    "to":       "{{user.email}}",
                },
            },
        },
        Status: models.TriggerStatusActive,
    }
    
    // 注册触发器
    eventBus.RegisterTrigger(trigger)
    
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
    eventBus.PublishEvent(ctx, event)
}
```

### 2. 条件配置

#### 简单条件

```json
{
    "type": "comparison",
    "field": "order.total",
    "operator": "gt",
    "value": 100
}
```

#### 逻辑条件

```json
{
    "type": "logical",
    "operator": "and",
    "conditions": [
        {
            "type": "comparison",
            "field": "user.age",
            "operator": "gte",
            "value": 18
        },
        {
            "type": "comparison",
            "field": "order.total",
            "operator": "gt",
            "value": 1000
        }
    ]
}
```

#### 函数条件

```json
{
    "type": "function",
    "function": "len",
    "args": ["order.items"],
    "operator": "gt",
    "value": 5
}
```

### 3. 支持的操作符

#### 比较操作符
- `eq` - 等于
- `ne` - 不等于
- `gt` - 大于
- `gte` - 大于等于
- `lt` - 小于
- `lte` - 小于等于
- `in` - 包含于
- `nin` - 不包含于
- `contains` - 包含
- `regex` - 正则匹配

#### 逻辑操作符
- `and` - 逻辑与
- `or` - 逻辑或
- `not` - 逻辑非

#### 支持的函数
- `len(field)` - 获取长度
- `upper(field)` - 转大写
- `lower(field)` - 转小写
- `trim(field)` - 去除空格
- `substr(field, start, length)` - 字符串截取
- `contains(field, substring)` - 包含检查
- `startswith(field, prefix)` - 前缀检查
- `endswith(field, suffix)` - 后缀检查

### 4. 动作配置

#### 邮件动作

```json
{
    "type": "email",
    "config": {
        "template": "order_confirmation",
        "to": "{{user.email}}",
        "cc": ["admin@example.com"],
        "subject": "订单确认 - {{order.id}}",
        "variables": {
            "order_id": "{{order.id}}",
            "user_name": "{{user.name}}"
        }
    }
}
```

#### Webhook动作

```json
{
    "type": "webhook",
    "config": {
        "url": "https://api.example.com/webhook",
        "method": "POST",
        "headers": {
            "Content-Type": "application/json",
            "Authorization": "Bearer {{api_token}}"
        },
        "body": {
            "event_type": "{{event.type}}",
            "data": "{{event.data}}"
        },
        "timeout": 30
    }
}
```

#### Slack动作

```json
{
    "type": "slack",
    "config": {
        "webhook_url": "https://hooks.slack.com/services/...",
        "channel": "#alerts",
        "username": "TriggerV2",
        "text": "新订单：{{order.id}}，金额：{{order.total}}",
        "attachments": [
            {
                "color": "good",
                "fields": [
                    {
                        "title": "订单ID",
                        "value": "{{order.id}}",
                        "short": true
                    },
                    {
                        "title": "客户",
                        "value": "{{user.name}}",
                        "short": true
                    }
                ]
            }
        ]
    }
}
```

## 高级功能

### 1. 批处理

批处理功能可以将多个事件合并处理，提高处理效率。

```go
// 创建批处理器
batchProcessor := engine.NewBatchProcessor(engine.BatchProcessorConfig{
    MaxBatchSize:   100,           // 最大批处理大小
    FlushInterval:  5 * time.Second, // 刷新间隔
    MaxRetries:     3,             // 最大重试次数
    MaxConcurrency: 10,            // 最大并发数
})

// 批处理处理函数
processor := func(ctx context.Context, events []*models.Event) error {
    // 处理一批事件
    for _, event := range events {
        // 处理单个事件
        processEvent(event)
    }
    return nil
}

// 添加事件到批处理器
batchProcessor.AddEvent(ctx, event, processor)
```

### 2. 监控和告警

系统提供全面的监控和告警功能。

```go
// 创建监控器
monitor := monitoring.NewMonitor(monitoring.MonitorConfig{
    MetricsRetention: 24 * time.Hour,
    AlertCooldown:    5 * time.Minute,
})

// 添加告警规则
alertRule := &monitoring.AlertRule{
    Name:        "高错误率告警",
    Description: "当错误率超过5%时触发告警",
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

monitor.AddAlertRule(alertRule)
```

### 3. 插件系统

系统支持自定义插件扩展功能。

```go
// 实现自定义插件
type CustomPlugin struct {
    name string
}

func (p *CustomPlugin) GetName() string {
    return p.name
}

func (p *CustomPlugin) GetVersion() string {
    return "1.0.0"
}

func (p *CustomPlugin) Execute(ctx context.Context, action *models.Action, event *models.Event) error {
    // 实现自定义逻辑
    return nil
}

// 注册插件
pluginManager := plugins.NewPluginManager()
pluginManager.RegisterPlugin(&CustomPlugin{name: "custom"})
```

## 最佳实践

### 1. 事件设计

- **使用清晰的事件类型命名**：采用`域.动作`的格式，如`user.created`、`order.updated`
- **保持事件数据结构一致**：相同类型的事件应有相同的数据结构
- **包含完整的上下文信息**：事件数据应包含处理所需的所有信息
- **避免敏感信息**：不要在事件数据中包含密码、令牌等敏感信息

### 2. 触发器设计

- **明确的触发器命名**：使用描述性的名称，便于理解和维护
- **简化条件逻辑**：避免过于复杂的条件，可以分解为多个简单触发器
- **合理的重试策略**：设置适当的重试次数和间隔
- **监控触发器性能**：定期检查触发器的执行效率

### 3. 性能优化

- **使用批处理**：对于大量事件，使用批处理可以显著提高性能
- **优化条件逻辑**：将最可能失败的条件放在前面，利用短路求值
- **合理的并发设置**：根据系统资源调整并发数
- **定期清理数据**：清理过期的事件和指标数据

### 4. 错误处理

- **幂等性设计**：确保重复执行不会产生副作用
- **记录详细日志**：记录足够的信息用于问题诊断
- **设置告警**：对关键错误设置告警通知
- **优雅降级**：在系统负载过高时，优雅地降级处理

## 配置示例

### 完整的电商场景配置

```json
{
    "triggers": [
        {
            "id": "high-value-order",
            "name": "高价值订单触发器",
            "description": "当订单金额超过1000元时，发送通知给销售团队",
            "event_types": ["order.created", "order.updated"],
            "conditions": {
                "type": "logical",
                "operator": "and",
                "conditions": [
                    {
                        "type": "comparison",
                        "field": "order.total",
                        "operator": "gt",
                        "value": 1000
                    },
                    {
                        "type": "comparison",
                        "field": "order.status",
                        "operator": "eq",
                        "value": "confirmed"
                    }
                ]
            },
            "actions": [
                {
                    "type": "email",
                    "config": {
                        "template": "high_value_order",
                        "to": "sales@example.com",
                        "subject": "高价值订单通知 - {{order.id}}"
                    }
                },
                {
                    "type": "slack",
                    "config": {
                        "channel": "#sales",
                        "text": "新的高价值订单：{{order.id}}，金额：{{order.total}}"
                    }
                }
            ],
            "status": "ACTIVE"
        },
        {
            "id": "new-user-welcome",
            "name": "新用户欢迎触发器",
            "description": "向新注册的用户发送欢迎邮件",
            "event_types": ["user.created"],
            "conditions": {
                "type": "comparison",
                "field": "user.verified",
                "operator": "eq",
                "value": true
            },
            "actions": [
                {
                    "type": "email",
                    "config": {
                        "template": "welcome_email",
                        "to": "{{user.email}}",
                        "subject": "欢迎加入我们！"
                    }
                }
            ],
            "status": "ACTIVE"
        }
    ]
}
```

## 故障排除

### 常见问题

1. **事件没有被处理**
   - 检查触发器状态是否为ACTIVE
   - 确认事件类型是否匹配
   - 检查条件逻辑是否正确

2. **动作执行失败**
   - 检查动作配置是否正确
   - 确认外部服务（如邮件服务）是否可用
   - 查看错误日志获取详细信息

3. **性能问题**
   - 检查系统资源使用情况
   - 优化条件逻辑复杂度
   - 考虑使用批处理

4. **内存使用过高**
   - 检查事件数据大小
   - 调整批处理配置
   - 清理过期数据

### 调试技巧

1. **启用调试日志**
   ```go
   // 设置日志级别
   log.SetLevel(log.DebugLevel)
   ```

2. **使用监控指标**
   ```go
   // 获取系统统计信息
   stats := eventBus.GetStats()
   fmt.Printf("Events processed: %d\n", stats.EventsProcessed)
   ```

3. **检查健康状态**
   ```go
   // 检查系统健康状态
   health := monitor.GetHealth()
   fmt.Printf("System health: %s\n", health.Status)
   ```

## 更新日志

### v2.0.0
- 新增复杂条件逻辑引擎
- 支持批处理优化
- 完整的监控告警系统
- 插件系统重构

### v1.0.0
- 基础事件处理功能
- 简单条件匹配
- 基本动作支持

## 支持和反馈

如果您在使用过程中遇到问题或有建议，请：

1. 查看本文档的故障排除部分
2. 检查系统日志获取详细错误信息
3. 联系技术支持团队

---

*本文档最后更新于：2024年*