package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// SystemConfigValueType 系统配置值类型
type SystemConfigValueType string

const (
	ConfigTypeString  SystemConfigValueType = "string"  // 字符串
	ConfigTypeNumber  SystemConfigValueType = "number"  // 数值
	ConfigTypeFloat   SystemConfigValueType = "float"   // 小数
	ConfigTypeBoolean SystemConfigValueType = "boolean" // 真假
	ConfigTypeJSON    SystemConfigValueType = "json"    // JSON对象
)

// SystemConfig 系统配置字典模型
type SystemConfig struct {
	ID           uint                  `json:"id" gorm:"primaryKey;autoIncrement"`
	Key          string                `json:"key" gorm:"uniqueIndex;not null;size:255;comment:配置键名"`
	Name         string                `json:"name" gorm:"not null;size:255;comment:配置名称"`
	Description  string                `json:"description" gorm:"size:500;comment:配置描述"`
	ValueType    SystemConfigValueType `json:"value_type" gorm:"not null;size:50;comment:值类型"`
	CurrentValue JSONMap               `json:"current_value" gorm:"type:text;comment:当前值"`
	DefaultValue JSONMap               `json:"default_value" gorm:"type:text;comment:默认值"`
	Category     string                `json:"category" gorm:"size:100;comment:配置分类"`
	IsEditable   bool                  `json:"is_editable" gorm:"default:true;comment:是否可编辑"`
	IsVisible    bool                  `json:"is_visible" gorm:"default:true;comment:是否在UI中可见"`
	SortOrder    int                   `json:"sort_order" gorm:"default:0;comment:排序顺序"`
	CreatedAt    time.Time             `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time             `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt        `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "system_configs"
}

// GetValue 获取配置值
func (sc *SystemConfig) GetValue() interface{} {
	if sc.CurrentValue == nil {
		return sc.GetDefaultValue()
	}

	valueStr := sc.CurrentValue["value"]
	return sc.parseValue(valueStr, false)
}

// GetDefaultValue 获取默认值
func (sc *SystemConfig) GetDefaultValue() interface{} {
	if sc.DefaultValue == nil {
		return sc.getTypeDefaultValue()
	}

	valueStr := sc.DefaultValue["value"]
	return sc.parseValue(valueStr, true)
}

// parseValue 解析字符串值为对应类型
func (sc *SystemConfig) parseValue(valueStr string, isDefault bool) interface{} {
	switch sc.ValueType {
	case ConfigTypeString:
		return valueStr
	case ConfigTypeNumber:
		if valueStr == "" {
			return sc.getTypeDefaultValue()
		}
		if val, err := strconv.Atoi(valueStr); err == nil {
			return val
		}
		return sc.getTypeDefaultValue()
	case ConfigTypeFloat:
		if valueStr == "" {
			return sc.getTypeDefaultValue()
		}
		if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
			return val
		}
		return sc.getTypeDefaultValue()
	case ConfigTypeBoolean:
		if valueStr == "" {
			return sc.getTypeDefaultValue()
		}
		if val, err := strconv.ParseBool(valueStr); err == nil {
			return val
		}
		return sc.getTypeDefaultValue()
	case ConfigTypeJSON:
		if valueStr == "" {
			return sc.getTypeDefaultValue()
		}
		var val interface{}
		if err := json.Unmarshal([]byte(valueStr), &val); err == nil {
			return val
		}
		return sc.getTypeDefaultValue()
	}
	return sc.getTypeDefaultValue()
}

// getTypeDefaultValue 获取类型的默认值
func (sc *SystemConfig) getTypeDefaultValue() interface{} {
	switch sc.ValueType {
	case ConfigTypeString:
		return ""
	case ConfigTypeNumber:
		return 0
	case ConfigTypeFloat:
		return 0.0
	case ConfigTypeBoolean:
		return false
	case ConfigTypeJSON:
		return nil
	}
	return nil
}

// SetValue 设置配置值
func (sc *SystemConfig) SetValue(value interface{}) error {
	if sc.CurrentValue == nil {
		sc.CurrentValue = make(JSONMap)
	}

	switch sc.ValueType {
	case ConfigTypeString:
		if val, ok := value.(string); ok {
			sc.CurrentValue["value"] = val
		} else {
			sc.CurrentValue["value"] = fmt.Sprintf("%v", value)
		}
	case ConfigTypeNumber:
		switch val := value.(type) {
		case int:
			sc.CurrentValue["value"] = strconv.Itoa(val)
		case float64:
			sc.CurrentValue["value"] = strconv.Itoa(int(val))
		default:
			return fmt.Errorf("invalid number value: %v", value)
		}
	case ConfigTypeFloat:
		if val, ok := value.(float64); ok {
			sc.CurrentValue["value"] = strconv.FormatFloat(val, 'f', -1, 64)
		} else {
			return fmt.Errorf("invalid float value: %v", value)
		}
	case ConfigTypeBoolean:
		if val, ok := value.(bool); ok {
			sc.CurrentValue["value"] = strconv.FormatBool(val)
		} else {
			return fmt.Errorf("invalid boolean value: %v", value)
		}
	case ConfigTypeJSON:
		bytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("invalid JSON value: %v", err)
		}
		sc.CurrentValue["value"] = string(bytes)
	default:
		return fmt.Errorf("unsupported value type: %s", sc.ValueType)
	}

	return nil
}

// GetBoolValue 获取布尔值配置
func (sc *SystemConfig) GetBoolValue() bool {
	if val := sc.GetValue(); val != nil {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return false
}

// GetStringValue 获取字符串配置
func (sc *SystemConfig) GetStringValue() string {
	if val := sc.GetValue(); val != nil {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}

// GetIntValue 获取整数配置
func (sc *SystemConfig) GetIntValue() int {
	if val := sc.GetValue(); val != nil {
		if intVal, ok := val.(int); ok {
			return intVal
		}
	}
	return 0
}

// GetFloatValue 获取浮点数配置
func (sc *SystemConfig) GetFloatValue() float64 {
	if val := sc.GetValue(); val != nil {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
	}
	return 0.0
}

// ResetToDefault 重置为默认值
func (sc *SystemConfig) ResetToDefault() {
	sc.CurrentValue = nil
}

// SystemConfigRequest 系统配置请求结构
type SystemConfigRequest struct {
	Value interface{} `json:"value" binding:"required"`
}

// SystemConfigResponse 系统配置响应结构
type SystemConfigResponse struct {
	Key          string                `json:"key"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	ValueType    SystemConfigValueType `json:"value_type"`
	CurrentValue interface{}           `json:"current_value"`
	DefaultValue interface{}           `json:"default_value"`
	Category     string                `json:"category"`
	IsEditable   bool                  `json:"is_editable"`
	IsVisible    bool                  `json:"is_visible"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// ToResponse 转换为响应格式
func (sc *SystemConfig) ToResponse() SystemConfigResponse {
	return SystemConfigResponse{
		Key:          sc.Key,
		Name:         sc.Name,
		Description:  sc.Description,
		ValueType:    sc.ValueType,
		CurrentValue: sc.GetValue(),
		DefaultValue: sc.GetDefaultValue(),
		Category:     sc.Category,
		IsEditable:   sc.IsEditable,
		IsVisible:    sc.IsVisible,
		UpdatedAt:    sc.UpdatedAt,
	}
}
