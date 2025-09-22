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

// OptimizedConditionEngine is an optimized version of the ConditionEngine
// with improved performance and resource utilization
type OptimizedConditionEngine struct {
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
	// 并行评估设置
	maxParallelEvaluations int
	// 评估信号量
	evalSemaphore chan struct{}
}

// NewOptimizedConditionEngine creates a new OptimizedConditionEngine
func NewOptimizedConditionEngine(maxParallelEvaluations int) *OptimizedConditionEngine {
	if maxParallelEvaluations <= 0 {
		maxParallelEvaluations = 10 // 默认最大并行评估数
	}

	engine := &OptimizedConditionEngine{
		operators: make(map[string]Operator),
		functions: make(map[string]Function),
		// 创建缓存，默认过期时间5分钟，每10分钟清理一次过期项
		resultCache:            cache.New(5*time.Minute, 10*time.Minute),
		fieldValueCache:        cache.New(5*time.Minute, 10*time.Minute),
		regexCache:             make(map[string]*regexp.Regexp),
		maxParallelEvaluations: maxParallelEvaluations,
		evalSemaphore:          make(chan struct{}, maxParallelEvaluations),
	}

	// 注册默认操作符
	engine.registerDefaultOperators()

	// 注册默认函数
	engine.registerDefaultFunctions()

	return engine
}

// Evaluate evaluates a trigger expression against an evaluation context
// with optimized performance
func (e *OptimizedConditionEngine) Evaluate(expression models.TriggerExpression, context *EvaluationContext) (bool, models.JSONMap, error) {
	log.Printf("[OptimizedConditionEngine] Evaluating expression: %s", expression.ID)

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
		log.Printf("[OptimizedConditionEngine] Cache hit for expression: %s", expression.ID)
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
func (e *OptimizedConditionEngine) generateCacheKey(expression models.TriggerExpression, context *EvaluationContext) string {
	// 使用表达式ID和邮件ID作为缓存键
	return fmt.Sprintf("opt:%s:%d", expression.ID, context.Email.ID)
}

// evaluateCondition 评估单一条件表达式
func (e *OptimizedConditionEngine) evaluateCondition(expression models.TriggerExpression, context *EvaluationContext) (bool, models.JSONMap, error) {
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
			"operator":     operator,
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
			"operator":     operator,
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
		"result":       strconv.FormatBool(result),
		"field":        *expression.Field,
		"operator":     operator,
		"fieldValue":   fmt.Sprintf("%v", fieldValue),
		"value":        fmt.Sprintf("%v", expression.Value),
		"not":          strconv.FormatBool(expression.Not != nil && *expression.Not),
	}, nil
}

// evaluateGroup 评估条件组表达式，使用并行评估优化
func (e *OptimizedConditionEngine) evaluateGroup(expression models.TriggerExpression, context *EvaluationContext) (bool, models.JSONMap, error) {
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

	// 对于AND和OR操作符，可以使用短路评估
	if operator == models.TriggerOperatorAnd || operator == models.TriggerOperatorOr {
		return e.evaluateGroupWithShortCircuit(expression, context, operator)
	}

	// 对于其他操作符，使用标准评估
	return e.evaluateGroupStandard(expression, context, operator)
}

// evaluateGroupWithShortCircuit 使用短路评估优化条件组
func (e *OptimizedConditionEngine) evaluateGroupWithShortCircuit(expression models.TriggerExpression, context *EvaluationContext, operator models.TriggerOperator) (bool, models.JSONMap, error) {
	results := make([]models.JSONMap, 0, len(expression.Conditions))

	// 对于AND操作符，一个条件为假则整个表达式为假
	// 对于OR操作符，一个条件为真则整个表达式为真
	shortCircuitValue := operator == models.TriggerOperatorOr

	for _, condition := range expression.Conditions {
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

		results = append(results, evalDetails)

		// 短路评估
		if result == shortCircuitValue {
			message := "OR condition succeeded"
			if operator == models.TriggerOperatorAnd {
				message = "AND condition failed"
			}

			finalResult := shortCircuitValue

			// 处理否定条件
			if expression.Not != nil && *expression.Not {
				finalResult = !finalResult
				message = fmt.Sprintf("NOT (%s)", message)
			}

			return finalResult, models.JSONMap{
				"expressionId": expression.ID,
				"type":         string(expression.Type),
				"result":       strconv.FormatBool(finalResult),
				"operator":     string(operator),
				"message":      message,
				"not":          strconv.FormatBool(expression.Not != nil && *expression.Not),
			}, nil
		}
	}

	// 如果没有短路，则所有条件都已评估
	var finalResult bool
	var message string

	if operator == models.TriggerOperatorAnd {
		finalResult = true
		message = "All AND conditions succeeded"
	} else {
		finalResult = false
		message = "All OR conditions failed"
	}

	// 处理否定条件
	if expression.Not != nil && *expression.Not {
		finalResult = !finalResult
		message = fmt.Sprintf("NOT (%s)", message)
	}

	return finalResult, models.JSONMap{
		"expressionId": expression.ID,
		"type":         string(expression.Type),
		"result":       strconv.FormatBool(finalResult),
		"operator":     string(operator),
		"message":      message,
		"not":          strconv.FormatBool(expression.Not != nil && *expression.Not),
	}, nil
}

// evaluateGroupStandard 标准评估条件组
func (e *OptimizedConditionEngine) evaluateGroupStandard(expression models.TriggerExpression, context *EvaluationContext, operator models.TriggerOperator) (bool, models.JSONMap, error) {
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
	}

	// 根据操作符计算最终结果
	var finalResult bool
	var message string

	switch operator {
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
		"result":       strconv.FormatBool(finalResult),
		"operator":     string(operator),
		"message":      message,
		"not":          strconv.FormatBool(expression.Not != nil && *expression.Not),
	}, nil
}

// EvaluateExpressions evaluates a list of trigger expressions against an evaluation context
// with optimized parallel evaluation
func (e *OptimizedConditionEngine) EvaluateExpressions(expressions []models.TriggerExpression, context *EvaluationContext) (bool, models.JSONMap, error) {
	log.Printf("[OptimizedConditionEngine] Evaluating %d expressions", len(expressions))

	if len(expressions) == 0 {
		return true, models.JSONMap{"result": "true", "message": "No expressions to evaluate"}, nil
	}

	// 创建一个根级别的 AND 组表达式
	rootExpression := models.TriggerExpression{
		ID:         "root",
		Type:       models.TriggerExpressionTypeGroup,
		Operator:   &[]models.TriggerOperator{models.TriggerOperatorAnd}[0],
		Conditions: expressions,
	}

	// 评估根表达式
	result, evalDetails, err := e.Evaluate(rootExpression, context)
	if err != nil {
		return false, nil, fmt.Errorf("failed to evaluate expressions: %w", err)
	}

	return result, evalDetails, nil
}

// getFieldValue 从评估上下文中获取字段值，使用缓存优化
func (e *OptimizedConditionEngine) getFieldValue(fieldName string, context *EvaluationContext) (interface{}, error) {
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
func (e *OptimizedConditionEngine) getEmailFieldValue(parts []string, email models.Email) (interface{}, error) {
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
func (e *OptimizedConditionEngine) RegisterOperator(operator Operator) {
	e.operators[operator.GetName()] = operator
}

// RegisterFunction 注册函数
func (e *OptimizedConditionEngine) RegisterFunction(function Function) {
	e.functions[function.GetName()] = function
}

// registerDefaultOperators 注册默认操作符
func (e *OptimizedConditionEngine) registerDefaultOperators() {
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
	e.RegisterOperator(&MatchesOperator{engine: nil}) // 需要在使用前设置engine
	e.RegisterOperator(&InOperator{})
	e.RegisterOperator(&NotInOperator{})
}

// registerDefaultFunctions 注册默认函数
func (e *OptimizedConditionEngine) registerDefaultFunctions() {
	e.RegisterFunction(&LenFunction{})
	e.RegisterFunction(&UpperFunction{})
	e.RegisterFunction(&LowerFunction{})
	e.RegisterFunction(&TrimFunction{})
	e.RegisterFunction(&DateFunction{})
	e.RegisterFunction(&IsEmptyFunction{})
	e.RegisterFunction(&IsNotEmptyFunction{})
}

// ClearCache 清除所有缓存
func (e *OptimizedConditionEngine) ClearCache() {
	e.cacheMutex.Lock()
	defer e.cacheMutex.Unlock()

	e.resultCache.Flush()
	e.fieldValueCache.Flush()
	e.regexCache = make(map[string]*regexp.Regexp)
}

// GetCacheStats 获取缓存统计信息
func (e *OptimizedConditionEngine) GetCacheStats() map[string]interface{} {
	e.cacheMutex.RLock()
	defer e.cacheMutex.RUnlock()

	resultItems := e.resultCache.ItemCount()
	fieldItems := e.fieldValueCache.ItemCount()
	regexItems := len(e.regexCache)

	return map[string]interface{}{
		"resultCacheItems":     resultItems,
		"fieldValueCacheItems": fieldItems,
		"regexCacheItems":      regexItems,
		"totalCacheItems":      resultItems + fieldItems + regexItems,
	}
}
