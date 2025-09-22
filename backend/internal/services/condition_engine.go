package services

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"mailman/internal/models"

	"github.com/patrickmn/go-cache"
)

// ConditionEngine evaluates trigger conditions against emails
type ConditionEngine struct {
	// 操作符映射表
	operators map[string]Operator
	// 函数映射表
	functions map[string]Function
	// 结果缓存
	resultCache *cache.Cache
	// 字段值缓存
	fieldValueCache *cache.Cache
	// 缓存锁
	cacheMutex sync.RWMutex
	// 正则表达式缓存
	regexCache map[string]*regexp.Regexp
}

// NewConditionEngine creates a new ConditionEngine
func NewConditionEngine() *ConditionEngine {
	engine := &ConditionEngine{
		operators: make(map[string]Operator),
		functions: make(map[string]Function),
		// 创建缓存，默认过期时间5分钟，每10分钟清理一次过期项
		resultCache:     cache.New(5*time.Minute, 10*time.Minute),
		fieldValueCache: cache.New(5*time.Minute, 10*time.Minute),
		regexCache:      make(map[string]*regexp.Regexp),
	}

	// 注册默认操作符
	engine.registerDefaultOperators()

	// 注册默认函数
	engine.registerDefaultFunctions()

	return engine
}

// Operator 操作符接口
type Operator interface {
	// 评估操作符两侧的值
	Evaluate(left, right interface{}) (bool, error)
	// 获取操作符名称
	GetName() string
}

// Function 函数接口
type Function interface {
	// 执行函数
	Execute(args []interface{}) (interface{}, error)
	// 获取函数名称
	GetName() string
	// 获取参数数量
	GetArgCount() int
}

// EvaluationContext contains the data needed for condition evaluation
type EvaluationContext struct {
	Email models.Email
	// 额外的上下文数据，用于存储评估过程中的临时数据
	Data map[string]interface{}
}

// NewEvaluationContext 创建新的评估上下文
func NewEvaluationContext(email models.Email) *EvaluationContext {
	return &EvaluationContext{
		Email: email,
		Data:  make(map[string]interface{}),
	}
}

// Evaluate evaluates a trigger expression against an evaluation context
func (e *ConditionEngine) Evaluate(expression models.TriggerExpression, context *EvaluationContext) (bool, models.JSONMap, error) {
	log.Printf("[ConditionEngine] Evaluating expression: %s", expression.ID)

	// 生成缓存键
	cacheKey := e.generateCacheKey(expression, context)

	// 尝试从缓存获取结果
	e.cacheMutex.RLock()
	if cachedResult, found := e.resultCache.Get(cacheKey); found {
		e.cacheMutex.RUnlock()
		result := cachedResult.(struct {
			Result  bool
			Details models.JSONMap
		})
		log.Printf("[ConditionEngine] Cache hit for expression: %s", expression.ID)
		return result.Result, result.Details, nil
	}
	e.cacheMutex.RUnlock()

	// 根据表达式类型进行评估
	var result bool
	var details models.JSONMap
	var err error

	switch expression.Type {
	case models.TriggerExpressionTypeCondition:
		// 单一条件表达式
		result, details, err = e.evaluateCondition(expression, context)
	case models.TriggerExpressionTypeGroup:
		// 条件组表达式
		result, details, err = e.evaluateGroup(expression, context)
	default:
		return false, models.JSONMap{
			"expressionId": expression.ID,
			"type":         string(expression.Type),
			"result":       "false",
			"message":      fmt.Sprintf("Unknown expression type: %s", expression.Type),
		}, fmt.Errorf("unknown expression type: %s", expression.Type)
	}

	// 如果评估成功，缓存结果
	if err == nil {
		e.cacheMutex.Lock()
		e.resultCache.Set(cacheKey, struct {
			Result  bool
			Details models.JSONMap
		}{
			Result:  result,
			Details: details,
		}, cache.DefaultExpiration)
		e.cacheMutex.Unlock()
	}

	return result, details, err
}

// generateCacheKey 生成缓存键
func (e *ConditionEngine) generateCacheKey(expression models.TriggerExpression, context *EvaluationContext) string {
	// 使用表达式ID和邮件ID作为缓存键
	return fmt.Sprintf("%s:%d", expression.ID, context.Email.ID)
}

// evaluateCondition 评估单一条件表达式
func (e *ConditionEngine) evaluateCondition(expression models.TriggerExpression, context *EvaluationContext) (bool, models.JSONMap, error) {
	if expression.Field == nil || expression.Value == nil {
		return false, models.JSONMap{
			"expressionId": expression.ID,
			"type":         string(expression.Type),
			"result":       "false",
			"message":      "Missing field or value in condition expression",
		}, fmt.Errorf("missing field or value in condition expression: %s", expression.ID)
	}

	// 获取字段值
	fieldValue, err := e.getFieldValue(*expression.Field, context)
	if err != nil {
		return false, models.JSONMap{
			"expressionId": expression.ID,
			"type":         string(expression.Type),
			"result":       "false",
			"message":      fmt.Sprintf("Failed to get field value: %v", err),
			"field":        *expression.Field,
		}, fmt.Errorf("failed to get field value: %w", err)
	}

	// 获取操作符
	var operator string
	if expression.Operator != nil {
		operator = string(*expression.Operator)
	} else {
		// 默认使用相等操作符
		operator = "equals"
	}

	// 执行操作符评估
	op, exists := e.operators[operator]
	if !exists {
		return false, models.JSONMap{
			"expressionId": expression.ID,
			"type":         string(expression.Type),
			"result":       "false",
			"message":      fmt.Sprintf("Unknown operator: %s", operator),
			"field":        *expression.Field,
			"operator":     string(operator),
		}, fmt.Errorf("unknown operator: %s", operator)
	}

	// 评估条件
	result, err := op.Evaluate(fieldValue, expression.Value)
	if err != nil {
		return false, models.JSONMap{
			"expressionId": expression.ID,
			"type":         string(expression.Type),
			"result":       "false",
			"message":      fmt.Sprintf("Evaluation error: %v", err),
			"field":        *expression.Field,
			"operator":     string(operator),
			"fieldValue":   fmt.Sprintf("%v", fieldValue),
			"value":        fmt.Sprintf("%v", expression.Value),
		}, fmt.Errorf("evaluation error: %w", err)
	}

	// 处理否定条件
	if expression.Not != nil && *expression.Not {
		result = !result
	}

	return result, models.JSONMap{
		"expressionId": expression.ID,
		"type":         string(expression.Type),
		"result":       fmt.Sprintf("%v", result),
		"field":        *expression.Field,
		"operator":     string(operator),
		"fieldValue":   fmt.Sprintf("%v", fieldValue),
		"value":        fmt.Sprintf("%v", expression.Value),
		"not":          fmt.Sprintf("%v", expression.Not != nil && *expression.Not),
	}, nil
}

// evaluateGroup 评估条件组表达式
func (e *ConditionEngine) evaluateGroup(expression models.TriggerExpression, context *EvaluationContext) (bool, models.JSONMap, error) {
	if expression.Conditions == nil || len(expression.Conditions) == 0 {
		return false, models.JSONMap{
			"expressionId": expression.ID,
			"type":         string(expression.Type),
			"result":       "false",
			"message":      "Empty conditions in group expression",
		}, fmt.Errorf("empty conditions in group expression: %s", expression.ID)
	}

	if expression.Operator == nil {
		return false, models.JSONMap{
			"expressionId": expression.ID,
			"type":         string(expression.Type),
			"result":       "false",
			"message":      "Missing operator in group expression",
		}, fmt.Errorf("missing operator in group expression: %s", expression.ID)
	}

	operator := *expression.Operator

	// 评估所有子条件
	results := make([]models.JSONMap, len(expression.Conditions))
	conditionResults := make([]bool, len(expression.Conditions))

	for i, condition := range expression.Conditions {
		result, evalDetails, err := e.Evaluate(condition, context)
		if err != nil {
			return false, models.JSONMap{
				"expressionId": expression.ID,
				"type":         string(expression.Type),
				"result":       "false",
				"message":      fmt.Sprintf("Failed to evaluate condition %s: %v", condition.ID, err),
				"operator":     string(operator),
			}, fmt.Errorf("failed to evaluate condition %s: %w", condition.ID, err)
		}

		results[i] = evalDetails
		conditionResults[i] = result

		// 短路评估
		if operator == models.TriggerOperatorAnd && !result {
			// AND 操作符，一个条件为假则整个表达式为假
			return false, models.JSONMap{
				"expressionId": expression.ID,
				"type":         string(expression.Type),
				"result":       "false",
				"operator":     string(operator),

				"message": "AND condition failed",
			}, nil
		} else if operator == models.TriggerOperatorOr && result {
			// OR 操作符，一个条件为真则整个表达式为真
			return true, models.JSONMap{
				"expressionId": expression.ID,
				"type":         string(expression.Type),
				"result":       "true",
				"operator":     string(operator),

				"message": "OR condition succeeded",
			}, nil
		}
	}

	// 根据操作符计算最终结果
	var finalResult bool
	var message string

	switch operator {
	case models.TriggerOperatorAnd:
		// 所有条件都为真
		finalResult = true
		message = "All AND conditions succeeded"
	case models.TriggerOperatorOr:
		// 所有条件都为假
		finalResult = false
		message = "All OR conditions failed"
	case models.TriggerOperatorNot:
		// 取反第一个条件的结果
		if len(conditionResults) > 0 {
			finalResult = !conditionResults[0]
			if finalResult {
				message = "NOT condition succeeded"
			} else {
				message = "NOT condition failed"
			}
		} else {
			finalResult = false
			message = "Empty NOT condition"
		}
	default:
		return false, models.JSONMap{
			"expressionId": expression.ID,
			"type":         string(expression.Type),
			"result":       "false",
			"operator":     string(operator),
			"message":      fmt.Sprintf("Unknown operator: %s", operator),
		}, fmt.Errorf("unknown operator: %s", operator)
	}

	// 处理否定条件
	if expression.Not != nil && *expression.Not {
		finalResult = !finalResult
		message = fmt.Sprintf("NOT (%s)", message)
	}

	return finalResult, models.JSONMap{
		"expressionId": expression.ID,
		"type":         string(expression.Type),
		"result":       fmt.Sprintf("%v", finalResult),
		"operator":     string(operator),

		"message": message,
		"not":     fmt.Sprintf("%v", expression.Not != nil && *expression.Not),
	}, nil
}

// EvaluateExpressions evaluates a list of trigger expressions against an evaluation context
func (e *ConditionEngine) EvaluateExpressions(expressions []models.TriggerExpression, context *EvaluationContext) (bool, models.JSONMap, error) {
	log.Printf("[ConditionEngine] Evaluating %d expressions", len(expressions))

	if len(expressions) == 0 {
		return true, models.JSONMap{"result": "true", "message": "No expressions to evaluate"}, nil
	}

	// 创建一个根级别的 AND 组表达式
	rootExpression := models.TriggerExpression{
		ID:         "root",
		Type:       models.TriggerExpressionTypeGroup,
		Operator:   (*models.TriggerOperator)(stringPtr(string(models.TriggerOperatorAnd))),
		Conditions: expressions,
	}

	// 评估根表达式
	result, evalDetails, err := e.Evaluate(rootExpression, context)
	if err != nil {
		return false, nil, fmt.Errorf("failed to evaluate expressions: %w", err)
	}

	return result, evalDetails, nil
}

// getFieldValue 从评估上下文中获取字段值
func (e *ConditionEngine) getFieldValue(fieldName string, context *EvaluationContext) (interface{}, error) {
	// 生成缓存键
	cacheKey := fmt.Sprintf("%d:%s", context.Email.ID, fieldName)

	// 尝试从缓存获取字段值
	e.cacheMutex.RLock()
	if cachedValue, found := e.fieldValueCache.Get(cacheKey); found {
		e.cacheMutex.RUnlock()
		return cachedValue, nil
	}
	e.cacheMutex.RUnlock()

	// 支持点表示法访问嵌套字段
	parts := strings.Split(fieldName, ".")

	var value interface{}
	var err error

	// 检查是否是邮件字段
	if len(parts) > 0 && parts[0] == "email" {
		// 从邮件对象获取字段
		value, err = e.getEmailFieldValue(parts[1:], context.Email)
	} else {
		// 检查是否是上下文数据字段
		if contextValue, exists := context.Data[fieldName]; exists {
			value = contextValue
		} else {
			err = fmt.Errorf("unknown field: %s", fieldName)
		}
	}

	// 如果获取成功，缓存字段值
	if err == nil {
		e.cacheMutex.Lock()
		e.fieldValueCache.Set(cacheKey, value, cache.DefaultExpiration)
		e.cacheMutex.Unlock()
	}

	return value, err
}

// getEmailFieldValue 从邮件对象获取字段值
func (e *ConditionEngine) getEmailFieldValue(parts []string, email models.Email) (interface{}, error) {
	if len(parts) == 0 {
		return email, nil
	}

	fieldName := parts[0]

	switch fieldName {
	case "id":
		return email.ID, nil
	case "subject":
		return email.Subject, nil
	case "from":
		if len(parts) > 1 {
			index, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid index for from field: %s", parts[1])
			}
			if index >= 0 && index < len(email.From) {
				return email.From[index], nil
			}
			return nil, fmt.Errorf("index out of range for from field: %d", index)
		}
		return email.From, nil
	case "to":
		if len(parts) > 1 {
			index, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid index for to field: %s", parts[1])
			}
			if index >= 0 && index < len(email.To) {
				return email.To[index], nil
			}
			return nil, fmt.Errorf("index out of range for to field: %d", index)
		}
		return email.To, nil
	case "cc":
		if len(parts) > 1 {
			index, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid index for cc field: %s", parts[1])
			}
			if index >= 0 && index < len(email.Cc) {
				return email.Cc[index], nil
			}
			return nil, fmt.Errorf("index out of range for cc field: %d", index)
		}
		return email.Cc, nil
	case "bcc":
		if len(parts) > 1 {
			index, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid index for bcc field: %s", parts[1])
			}
			if index >= 0 && index < len(email.Bcc) {
				return email.Bcc[index], nil
			}
			return nil, fmt.Errorf("index out of range for bcc field: %d", index)
		}
		return email.Bcc, nil
	case "date":
		return email.Date, nil
	case "receivedAt":
		return email.ReceivedAt, nil
	case "messageId":
		return email.MessageID, nil
	case "inReplyTo":
		return email.InReplyTo, nil
	case "references":
		if len(parts) > 1 {
			index, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid index for references field: %s", parts[1])
			}
			if index >= 0 && index < len(email.References) {
				return email.References[index], nil
			}
			return nil, fmt.Errorf("index out of range for references field: %d", index)
		}
		return email.References, nil
	case "htmlBody":
		return email.HTMLBody, nil
	case "textBody":
		return email.TextBody, nil
	case "hasAttachments":
		return email.HasAttachments, nil
	case "attachments":
		if len(email.Attachments) == 0 {
			return []models.Attachment{}, nil
		}
		if len(parts) > 1 {
			index, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid index for attachments field: %s", parts[1])
			}
			if index >= 0 && index < len(email.Attachments) {
				attachment := email.Attachments[index]
				if len(parts) > 2 {
					switch parts[2] {
					case "filename":
						return attachment.Filename, nil
					case "contentType":
						return attachment.ContentType, nil
					case "size":
						return attachment.Size, nil
					default:
						return nil, fmt.Errorf("unknown attachment field: %s", parts[2])
					}
				}
				return attachment, nil
			}
			return nil, fmt.Errorf("index out of range for attachments field: %d", index)
		}
		return email.Attachments, nil
	case "headers":
		if len(parts) > 1 {
			headerName := parts[1]
			if value, exists := email.Headers[headerName]; exists {
				return value, nil
			}
			return nil, fmt.Errorf("header not found: %s", headerName)
		}
		return email.Headers, nil
	default:
		return nil, fmt.Errorf("unknown email field: %s", fieldName)
	}
}

// RegisterOperator 注册操作符
func (e *ConditionEngine) RegisterOperator(operator Operator) {
	e.operators[operator.GetName()] = operator
}

// RegisterFunction 注册函数
func (e *ConditionEngine) RegisterFunction(function Function) {
	e.functions[function.GetName()] = function
}

// registerDefaultOperators 注册默认操作符
func (e *ConditionEngine) registerDefaultOperators() {
	e.RegisterOperator(&EqualOperator{})
	e.RegisterOperator(&NotEqualOperator{})
	e.RegisterOperator(&GreaterThanOperator{})
	e.RegisterOperator(&LessThanOperator{})
	e.RegisterOperator(&GreaterEqualOperator{})
	e.RegisterOperator(&LessEqualOperator{})
	e.RegisterOperator(&ContainsOperator{})
	e.RegisterOperator(&NotContainsOperator{})
	e.RegisterOperator(&StartsWithOperator{})
	e.RegisterOperator(&EndsWithOperator{})
	e.RegisterOperator(&MatchesOperator{engine: e})
	e.RegisterOperator(&InOperator{})
	e.RegisterOperator(&NotInOperator{})
}

// registerDefaultFunctions 注册默认函数
func (e *ConditionEngine) registerDefaultFunctions() {
	e.RegisterFunction(&LenFunction{})
	e.RegisterFunction(&UpperFunction{})
	e.RegisterFunction(&LowerFunction{})
	e.RegisterFunction(&TrimFunction{})
	e.RegisterFunction(&DateFunction{})
	e.RegisterFunction(&IsEmptyFunction{})
	e.RegisterFunction(&IsNotEmptyFunction{})
}

// 基本操作符实现

// EqualOperator 相等操作符
type EqualOperator struct{}

func (o *EqualOperator) Evaluate(left, right interface{}) (bool, error) {
	return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right), nil
}

func (o *EqualOperator) GetName() string { return "equals" }

// NotEqualOperator 不等操作符
type NotEqualOperator struct{}

func (o *NotEqualOperator) Evaluate(left, right interface{}) (bool, error) {
	return fmt.Sprintf("%v", left) != fmt.Sprintf("%v", right), nil
}

func (o *NotEqualOperator) GetName() string { return "not_equals" }

// GreaterThanOperator 大于操作符
type GreaterThanOperator struct{}

func (o *GreaterThanOperator) Evaluate(left, right interface{}) (bool, error) {
	leftNum, err := o.toNumber(left)
	if err != nil {
		return false, err
	}
	rightNum, err := o.toNumber(right)
	if err != nil {
		return false, err
	}
	return leftNum > rightNum, nil
}

func (o *GreaterThanOperator) toNumber(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert to number: %v", value)
	}
}

func (o *GreaterThanOperator) GetName() string { return "greater_than" }

// LessThanOperator 小于操作符
type LessThanOperator struct{}

func (o *LessThanOperator) Evaluate(left, right interface{}) (bool, error) {
	leftNum, err := o.toNumber(left)
	if err != nil {
		return false, err
	}
	rightNum, err := o.toNumber(right)
	if err != nil {
		return false, err
	}
	return leftNum < rightNum, nil
}

func (o *LessThanOperator) toNumber(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert to number: %v", value)
	}
}

func (o *LessThanOperator) GetName() string { return "less_than" }

// GreaterEqualOperator 大于等于操作符
type GreaterEqualOperator struct{}

func (o *GreaterEqualOperator) Evaluate(left, right interface{}) (bool, error) {
	leftNum, err := o.toNumber(left)
	if err != nil {
		return false, err
	}
	rightNum, err := o.toNumber(right)
	if err != nil {
		return false, err
	}
	return leftNum >= rightNum, nil
}

func (o *GreaterEqualOperator) toNumber(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert to number: %v", value)
	}
}

func (o *GreaterEqualOperator) GetName() string { return "greater_equal" }

// LessEqualOperator 小于等于操作符
type LessEqualOperator struct{}

func (o *LessEqualOperator) Evaluate(left, right interface{}) (bool, error) {
	leftNum, err := o.toNumber(left)
	if err != nil {
		return false, err
	}
	rightNum, err := o.toNumber(right)
	if err != nil {
		return false, err
	}
	return leftNum <= rightNum, nil
}

func (o *LessEqualOperator) toNumber(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert to number: %v", value)
	}
}

func (o *LessEqualOperator) GetName() string { return "less_equal" }

// ContainsOperator 包含操作符
type ContainsOperator struct{}

func (o *ContainsOperator) Evaluate(left, right interface{}) (bool, error) {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	return strings.Contains(leftStr, rightStr), nil
}

func (o *ContainsOperator) GetName() string { return "contains" }

// NotContainsOperator 不包含操作符
type NotContainsOperator struct{}

func (o *NotContainsOperator) Evaluate(left, right interface{}) (bool, error) {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	return !strings.Contains(leftStr, rightStr), nil
}

func (o *NotContainsOperator) GetName() string { return "not_contains" }

// StartsWithOperator 开始于操作符
type StartsWithOperator struct{}

func (o *StartsWithOperator) Evaluate(left, right interface{}) (bool, error) {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	return strings.HasPrefix(leftStr, rightStr), nil
}

func (o *StartsWithOperator) GetName() string { return "starts_with" }

// EndsWithOperator 结束于操作符
type EndsWithOperator struct{}

func (o *EndsWithOperator) Evaluate(left, right interface{}) (bool, error) {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	return strings.HasSuffix(leftStr, rightStr), nil
}

func (o *EndsWithOperator) GetName() string { return "ends_with" }

// MatchesOperator 匹配操作符（正则表达式）
type MatchesOperator struct {
	engine *ConditionEngine
}

func (o *MatchesOperator) Evaluate(left, right interface{}) (bool, error) {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)

	var regex *regexp.Regexp
	var err error

	// 尝试从缓存获取正则表达式
	o.engine.cacheMutex.RLock()
	cachedRegex, found := o.engine.regexCache[rightStr]
	o.engine.cacheMutex.RUnlock()

	if found {
		regex = cachedRegex
	} else {
		// 编译正则表达式
		regex, err = regexp.Compile(rightStr)
		if err != nil {
			return false, fmt.Errorf("invalid regex: %v", err)
		}

		// 缓存正则表达式
		o.engine.cacheMutex.Lock()
		o.engine.regexCache[rightStr] = regex
		o.engine.cacheMutex.Unlock()
	}

	return regex.MatchString(leftStr), nil
}

func (o *MatchesOperator) GetName() string { return "matches" }

// InOperator 在列表中操作符
type InOperator struct{}

func (o *InOperator) Evaluate(left, right interface{}) (bool, error) {
	leftStr := fmt.Sprintf("%v", left)

	// 右侧应该是一个数组
	switch rightValue := right.(type) {
	case []interface{}:
		for _, item := range rightValue {
			if fmt.Sprintf("%v", item) == leftStr {
				return true, nil
			}
		}
		return false, nil
	case []string:
		for _, item := range rightValue {
			if item == leftStr {
				return true, nil
			}
		}
		return false, nil
	default:
		// 尝试将右侧值转换为字符串并按逗号分割
		rightStr := fmt.Sprintf("%v", right)
		items := strings.Split(rightStr, ",")
		for _, item := range items {
			if strings.TrimSpace(item) == leftStr {
				return true, nil
			}
		}
		return false, nil
	}
}

func (o *InOperator) GetName() string { return "in" }

// NotInOperator 不在列表中操作符
type NotInOperator struct{}

func (o *NotInOperator) Evaluate(left, right interface{}) (bool, error) {
	inOp := &InOperator{}
	result, err := inOp.Evaluate(left, right)
	if err != nil {
		return false, err
	}
	return !result, nil
}

func (o *NotInOperator) GetName() string { return "not_in" }

// 基本函数实现

// LenFunction 长度函数
type LenFunction struct{}

func (f *LenFunction) Execute(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("len function requires 1 argument")
	}

	switch v := args[0].(type) {
	case string:
		return len(v), nil
	case []interface{}:
		return len(v), nil
	case []string:
		return len(v), nil
	case []int:
		return len(v), nil
	case []float64:
		return len(v), nil
	case map[string]interface{}:
		return len(v), nil
	default:
		// 尝试转换为字符串
		str := fmt.Sprintf("%v", v)
		return len(str), nil
	}
}

func (f *LenFunction) GetName() string  { return "len" }
func (f *LenFunction) GetArgCount() int { return 1 }

// UpperFunction 大写函数
type UpperFunction struct{}

func (f *UpperFunction) Execute(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("upper function requires 1 argument")
	}

	str := fmt.Sprintf("%v", args[0])
	return strings.ToUpper(str), nil
}

func (f *UpperFunction) GetName() string  { return "upper" }
func (f *UpperFunction) GetArgCount() int { return 1 }

// LowerFunction 小写函数
type LowerFunction struct{}

func (f *LowerFunction) Execute(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("lower function requires 1 argument")
	}

	str := fmt.Sprintf("%v", args[0])
	return strings.ToLower(str), nil
}

func (f *LowerFunction) GetName() string  { return "lower" }
func (f *LowerFunction) GetArgCount() int { return 1 }

// TrimFunction 去除空格函数
type TrimFunction struct{}

func (f *TrimFunction) Execute(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("trim function requires 1 argument")
	}

	str := fmt.Sprintf("%v", args[0])
	return strings.TrimSpace(str), nil
}

func (f *TrimFunction) GetName() string  { return "trim" }
func (f *TrimFunction) GetArgCount() int { return 1 }

// DateFunction 解析日期函数
type DateFunction struct{}

func (f *DateFunction) Execute(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("date function requires 1 argument")
	}

	str := fmt.Sprintf("%v", args[0])

	// 尝试多种日期格式
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006/01/02",
		"2006/01/02 15:04:05",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			return t, nil
		}
	}

	return nil, fmt.Errorf("cannot parse date: %s", str)
}

func (f *DateFunction) GetName() string  { return "date" }
func (f *DateFunction) GetArgCount() int { return 1 }

// IsEmptyFunction 是否为空函数
type IsEmptyFunction struct{}

func (f *IsEmptyFunction) Execute(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("isEmpty function requires 1 argument")
	}

	switch v := args[0].(type) {
	case nil:
		return true, nil
	case string:
		return v == "", nil
	case []interface{}:
		return len(v) == 0, nil
	case []string:
		return len(v) == 0, nil
	case map[string]interface{}:
		return len(v) == 0, nil
	default:
		// 尝试转换为字符串
		str := fmt.Sprintf("%v", v)
		return str == "", nil
	}
}

func (f *IsEmptyFunction) GetName() string  { return "isEmpty" }
func (f *IsEmptyFunction) GetArgCount() int { return 1 }

// IsNotEmptyFunction 是否不为空函数
type IsNotEmptyFunction struct{}

func (f *IsNotEmptyFunction) Execute(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("isNotEmpty function requires 1 argument")
	}

	emptyFunc := &IsEmptyFunction{}
	result, err := emptyFunc.Execute(args)
	if err != nil {
		return nil, err
	}

	if isEmpty, ok := result.(bool); ok {
		return !isEmpty, nil
	}

	return false, nil
}

func (f *IsNotEmptyFunction) GetName() string  { return "isNotEmpty" }
func (f *IsNotEmptyFunction) GetArgCount() int { return 1 }

// stringPtr 返回字符串的指针
func stringPtr(s string) *string {
	return &s
}
