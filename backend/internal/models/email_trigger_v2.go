package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// EmailTriggerV2Status 触发器状态
type EmailTriggerV2Status string

const (
	EmailTriggerV2StatusEnabled  EmailTriggerV2Status = "enabled"
	EmailTriggerV2StatusDisabled EmailTriggerV2Status = "disabled"
)

// TriggerExpressionType 表达式类型
type TriggerExpressionType string

const (
	TriggerExpressionTypeGroup     TriggerExpressionType = "group"
	TriggerExpressionTypeCondition TriggerExpressionType = "condition"
)

// TriggerOperator 操作符类型
type TriggerOperator string

const (
	TriggerOperatorAnd TriggerOperator = "and"
	TriggerOperatorOr  TriggerOperator = "or"
	TriggerOperatorNot TriggerOperator = "not"
)

// TriggerExpression 触发器表达式
type TriggerExpression struct {
	ID         string               `json:"id"`
	Type       TriggerExpressionType `json:"type"`
	Operator   *TriggerOperator     `json:"operator,omitempty"`
	Field      *string              `json:"field,omitempty"`
	Value      interface{}          `json:"value,omitempty"`
	Conditions []TriggerExpression  `json:"conditions,omitempty"`
	Not        *bool                `json:"not,omitempty"`
}

// TriggerExpressions 触发器表达式数组
type TriggerExpressions []TriggerExpression

// Scan implements the sql.Scanner interface
func (te *TriggerExpressions) Scan(value interface{}) error {
	if value == nil {
		*te = []TriggerExpression{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, te)
}

// Value implements the driver.Valuer interface
func (te TriggerExpressions) Value() (driver.Value, error) {
	if len(te) == 0 {
		return "[]", nil
	}
	return json.Marshal(te)
}

// TriggerAction 触发器动作
type TriggerAction struct {
	ID          string                 `json:"id"`
	PluginID    string                 `json:"pluginId"`
	PluginName  string                 `json:"pluginName"`
	Config      map[string]interface{} `json:"config"`
	Enabled     bool                   `json:"enabled"`
	ExecutionOrder int                 `json:"executionOrder"`
}

// TriggerActions 触发器动作数组
type TriggerActions []TriggerAction

// Scan implements the sql.Scanner interface
func (ta *TriggerActions) Scan(value interface{}) error {
	if value == nil {
		*ta = []TriggerAction{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, ta)
}

// Value implements the driver.Valuer interface
func (ta TriggerActions) Value() (driver.Value, error) {
	if len(ta) == 0 {
		return "[]", nil
	}
	return json.Marshal(ta)
}

// EmailTriggerV2 邮件触发器V2
type EmailTriggerV2 struct {
	ID                uint                `gorm:"primaryKey" json:"id"`
	Name              string              `gorm:"not null;type:varchar(255)" json:"name"`
	Description       string              `json:"description,omitempty"`
	Enabled           bool                `gorm:"not null;default:false" json:"enabled"`
	Expressions       TriggerExpressions  `gorm:"type:json;not null" json:"expressions"`
	Actions           TriggerActions      `gorm:"type:json;not null" json:"actions"`
	TotalExecutions   int64               `gorm:"default:0" json:"totalExecutions"`
	SuccessExecutions int64               `gorm:"default:0" json:"successExecutions"`
	LastExecutedAt    *time.Time          `json:"lastExecutedAt,omitempty"`
	LastError         string              `json:"lastError,omitempty"`
	CreatedAt         time.Time           `json:"createdAt"`
	UpdatedAt         time.Time           `json:"updatedAt"`
	DeletedAt         DeletedAt           `gorm:"index" json:"deletedAt,omitempty"`
}

// TriggerExecutionV2Status 触发器执行状态
type TriggerExecutionV2Status string

const (
	TriggerExecutionV2StatusSuccess TriggerExecutionV2Status = "success"
	TriggerExecutionV2StatusFailed  TriggerExecutionV2Status = "failed"
	TriggerExecutionV2StatusPartial TriggerExecutionV2Status = "partial" // 部分成功
)

// ActionExecutionResult 动作执行结果
type ActionExecutionResult struct {
	ActionID    string                 `json:"actionId"`
	PluginID    string                 `json:"pluginId"`
	PluginName  string                 `json:"pluginName"`
	Success     bool                   `json:"success"`
	StartTime   time.Time              `json:"startTime"`
	EndTime     time.Time              `json:"endTime"`
	Duration    int64                  `json:"duration"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// ActionExecutionResults 动作执行结果数组
type ActionExecutionResults []ActionExecutionResult

// Scan implements the sql.Scanner interface
func (aer *ActionExecutionResults) Scan(value interface{}) error {
	if value == nil {
		*aer = []ActionExecutionResult{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, aer)
}

// Value implements the driver.Valuer interface
func (aer ActionExecutionResults) Value() (driver.Value, error) {
	if len(aer) == 0 {
		return "[]", nil
	}
	return json.Marshal(aer)
}

// TriggerExecutionLogV2 触发器执行日志V2
type TriggerExecutionLogV2 struct {
	ID               uint                     `gorm:"primaryKey" json:"id"`
	TriggerID        uint                     `gorm:"not null;index" json:"triggerId"`
	TriggerName      string                   `gorm:"not null;type:varchar(255)" json:"triggerName"`
	EmailID          uint                     `gorm:"not null;index" json:"emailId"`
	Status           TriggerExecutionV2Status `gorm:"not null" json:"status"`
	StartTime        time.Time                `gorm:"not null" json:"startTime"`
	EndTime          time.Time                `gorm:"not null" json:"endTime"`
	Duration         int64                    `gorm:"not null" json:"duration"` // 毫秒
	ConditionResult  bool                     `gorm:"not null" json:"conditionResult"`
	ConditionEval    JSONMap                  `gorm:"type:json" json:"conditionEvaluation"`
	ActionsExecuted  int                      `gorm:"not null" json:"actionsExecuted"`
	ActionsSucceeded int                      `gorm:"not null" json:"actionsSucceeded"`
	Error            string                   `json:"error,omitempty"`
	ActionResults    ActionExecutionResults   `gorm:"type:json" json:"actionResults"`
	CreatedAt        time.Time                `json:"createdAt"`
}