package models

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"
	
	mainModels "mailman/internal/models"
)

// EventType 事件类型
type EventType string

const (
	// 邮件相关事件
	EventTypeEmailReceived EventType = "email.received"
	EventTypeEmailUpdated  EventType = "email.updated"
	EventTypeEmailDeleted  EventType = "email.deleted"

	// 触发器相关事件
	EventTypeTriggerCreated  EventType = "trigger.created"
	EventTypeTriggerUpdated  EventType = "trigger.updated"
	EventTypeTriggerDeleted  EventType = "trigger.deleted"
	EventTypeTriggerExecuted EventType = "trigger.executed"

	// 系统相关事件
	EventTypeSystemStart EventType = "system.start"
	EventTypeSystemStop  EventType = "system.stop"
)

// EventStatus 事件状态
type EventStatus string

const (
	EventStatusPending    EventStatus = "pending"
	EventStatusProcessing EventStatus = "processing"
	EventStatusCompleted  EventStatus = "completed"
	EventStatusFailed     EventStatus = "failed"
)

// Event 事件模型
type Event struct {
	ID          string            `json:"id"`
	Type        EventType         `json:"type"`
	Status      EventStatus       `json:"status"`
	Source      string            `json:"source"`
	Subject     string            `json:"subject"`
	Data        json.RawMessage   `json:"data"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Priority    int               `json:"priority"`
	RetryCount  int               `json:"retry_count"`
	MaxRetries  int               `json:"max_retries"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	ProcessedAt *time.Time        `json:"processed_at,omitempty"`
}

// EmailEventData 邮件事件数据
type EmailEventData struct {
	// 原有字段（保持向后兼容）
	EmailID       uint      `json:"email_id"`
	AccountID     uint      `json:"account_id"`
	MailboxID     uint      `json:"mailbox_id"`
	Subject       string    `json:"subject"`
	From          string    `json:"from"`
	To            string    `json:"to"`
	MessageID     string    `json:"message_id"`
	ThreadID      string    `json:"thread_id"`
	IsRead        bool      `json:"is_read"`
	HasAttachment bool      `json:"has_attachment"`
	Labels        []string  `json:"labels"`
	ReceivedAt    time.Time `json:"received_at"`
	
	// 新增字段 - 完整的邮件对象（可选）
	Email         *mainModels.Email    `json:"email,omitempty"`
	
	// 事件元数据（新增）
	EventType     string    `json:"event_type,omitempty"`   // "received", "updated", "deleted"
	MailboxName   string    `json:"mailbox_name,omitempty"`
	Changes       map[string]interface{} `json:"changes,omitempty"`
	TriggerID     uint      `json:"trigger_id,omitempty"`
	ExecutionID   string    `json:"execution_id,omitempty"`
}

// TriggerEventData 触发器事件数据
type TriggerEventData struct {
	TriggerID     uint      `json:"trigger_id"`
	TriggerName   string    `json:"trigger_name"`
	EmailID       uint      `json:"email_id"`
	ConditionMet  bool      `json:"condition_met"`
	ExecutionID   string    `json:"execution_id"`
	ExecutionTime time.Time `json:"execution_time"`
	Success       bool      `json:"success"`
	Error         string    `json:"error,omitempty"`
}

// NewEvent 创建新事件
func NewEvent(eventType EventType, source string, subject string, data interface{}) (*Event, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &Event{
		ID:         generateEventID(),
		Type:       eventType,
		Status:     EventStatusPending,
		Source:     source,
		Subject:    subject,
		Data:       dataBytes,
		Metadata:   make(map[string]string),
		Priority:   0,
		RetryCount: 0,
		MaxRetries: 3,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// GetData 获取事件数据
func (e *Event) GetData(target interface{}) error {
	return json.Unmarshal(e.Data, target)
}

// SetData 设置事件数据
func (e *Event) SetData(data interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	e.Data = dataBytes
	e.UpdatedAt = time.Now()
	return nil
}

// CanRetry 是否可以重试
func (e *Event) CanRetry() bool {
	return e.RetryCount < e.MaxRetries
}

// IncrementRetry 增加重试次数
func (e *Event) IncrementRetry() {
	e.RetryCount++
	e.UpdatedAt = time.Now()
}

// MarkProcessing 标记为处理中
func (e *Event) MarkProcessing() {
	e.Status = EventStatusProcessing
	e.UpdatedAt = time.Now()
}

// MarkCompleted 标记为完成
func (e *Event) MarkCompleted() {
	now := time.Now()
	e.Status = EventStatusCompleted
	e.UpdatedAt = now
	e.ProcessedAt = &now
}

// MarkFailed 标记为失败
func (e *Event) MarkFailed() {
	now := time.Now()
	e.Status = EventStatusFailed
	e.UpdatedAt = now
	e.ProcessedAt = &now
}

// generateEventID 生成事件ID
func generateEventID() string {
	// 使用时间戳（包含纳秒）和随机数生成唯一ID
	now := time.Now()
	timestamp := now.Format("20060102150405")
	nanos := now.UnixNano() % 1000000 // 取微秒部分
	return timestamp + fmt.Sprintf("%06d", nanos) + generateRandomString(6)
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	// 使用crypto/rand生成真正的随机数
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// 如果crypto/rand失败，使用时间+索引作为后备
			result[i] = charset[(time.Now().UnixNano()+int64(i))%int64(len(charset))]
		} else {
			result[i] = charset[num.Int64()]
		}
	}
	return string(result)
}
