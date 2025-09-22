package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"mailman/internal/config"
	"mailman/internal/database"
	"mailman/internal/models"
	"mailman/internal/repository"
)

// This script migrates old trigger data to the new email trigger v2 format

func main() {
	log.Println("Starting trigger migration...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.NewDatabase(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create repositories
	oldTriggerRepo := repository.NewTriggerRepository(db)
	newTriggerRepo := repository.NewEmailTriggerV2Repository(db)

	// Get all old triggers
	oldTriggers, err := oldTriggerRepo.GetAll()
	if err != nil {
		log.Fatalf("Failed to get old triggers: %v", err)
	}

	log.Printf("Found %d old triggers to migrate", len(oldTriggers))

	// Migrate each trigger
	for _, oldTrigger := range oldTriggers {
		log.Printf("Migrating trigger: %s (ID: %d)", oldTrigger.Name, oldTrigger.ID)

		// Convert old trigger to new format
		newTrigger, err := convertTrigger(oldTrigger)
		if err != nil {
			log.Printf("Failed to convert trigger %d: %v", oldTrigger.ID, err)
			continue
		}

		// Create new trigger
		if err := newTriggerRepo.Create(newTrigger); err != nil {
			log.Printf("Failed to create new trigger %d: %v", oldTrigger.ID, err)
			continue
		}

		log.Printf("Successfully migrated trigger %d to new ID %d", oldTrigger.ID, newTrigger.ID)
	}

	log.Println("Trigger migration completed")
}

// convertTrigger converts an old trigger to the new format
func convertTrigger(oldTrigger *models.Trigger) (*models.EmailTriggerV2, error) {
	// Create new trigger with basic properties
	newTrigger := &models.EmailTriggerV2{
		Name:        oldTrigger.Name,
		Description: oldTrigger.Description,
		Enabled:     oldTrigger.Enabled,
	}

	// Convert expressions
	expressions, err := convertExpressions(oldTrigger.Conditions)
	if err != nil {
		return nil, fmt.Errorf("failed to convert expressions: %w", err)
	}
	newTrigger.Expressions = expressions

	// Convert actions
	actions, err := convertActions(oldTrigger.Actions)
	if err != nil {
		return nil, fmt.Errorf("failed to convert actions: %w", err)
	}
	newTrigger.Actions = actions

	// Copy statistics if available
	if oldTrigger.TotalExecutions > 0 {
		newTrigger.TotalExecutions = oldTrigger.TotalExecutions
		newTrigger.SuccessExecutions = oldTrigger.SuccessExecutions
	}

	if oldTrigger.LastExecutedAt != nil {
		newTrigger.LastExecutedAt = oldTrigger.LastExecutedAt
	}

	if oldTrigger.LastError != "" {
		newTrigger.LastError = oldTrigger.LastError
	}

	return newTrigger, nil
}

// convertExpressions converts old conditions to new expressions
func convertExpressions(conditions json.RawMessage) ([]models.TriggerExpression, error) {
	if len(conditions) == 0 {
		// Create a default "always true" expression
		return []models.TriggerExpression{
			{
				ID:   "root",
				Type: models.TriggerExpressionTypeGroup,
				Operator: (*models.TriggerOperator)(func() *string {
					s := string(models.TriggerOperatorAnd)
					return &s
				}()),
				Conditions: []models.TriggerExpression{},
			},
		}, nil
	}

	// Parse old conditions
	var oldConditions interface{}
	if err := json.Unmarshal(conditions, &oldConditions); err != nil {
		return nil, fmt.Errorf("failed to parse old conditions: %w", err)
	}

	// Convert to new format
	// This is a simplified conversion that assumes a specific structure
	// In a real implementation, you would need to handle all possible structures
	
	// For this example, we'll create a simple AND group with all conditions
	expressions := []models.TriggerExpression{}
	
	// Handle different old condition formats
	switch c := oldConditions.(type) {
	case map[string]interface{}:
		// Single condition or complex structure
		if condType, ok := c["type"].(string); ok {
			if condType == "group" {
				// It's a group, try to convert directly
				return convertGroupExpression(c)
			} else {
				// Single condition
				expr, err := convertSingleExpression(c)
				if err != nil {
					return nil, err
				}
				expressions = append(expressions, expr)
			}
		}
	case []interface{}:
		// Array of conditions
		for _, item := range c {
			if condMap, ok := item.(map[string]interface{}); ok {
				expr, err := convertSingleExpression(condMap)
				if err != nil {
					return nil, err
				}
				expressions = append(expressions, expr)
			}
		}
	}

	// Create root group if needed
	if len(expressions) > 0 {
		return []models.TriggerExpression{
			{
				ID:   "root",
				Type: models.TriggerExpressionTypeGroup,
				Operator: (*models.TriggerOperator)(func() *string {
					s := string(models.TriggerOperatorAnd)
					return &s
				}()),
				Conditions: expressions,
			},
		}, nil
	}

	// Fallback to empty expression list
	return expressions, nil
}

// convertGroupExpression converts a group expression
func convertGroupExpression(group map[string]interface{}) ([]models.TriggerExpression, error) {
	// Extract operator
	operator := models.TriggerOperatorAnd
	if op, ok := group["operator"].(string); ok {
		switch op {
		case "and":
			operator = models.TriggerOperatorAnd
		case "or":
			operator = models.TriggerOperatorOr
		case "not":
			operator = models.TriggerOperatorNot
		}
	}

	// Extract conditions
	var subExpressions []models.TriggerExpression
	if conditions, ok := group["conditions"].([]interface{}); ok {
		for _, cond := range conditions {
			if condMap, ok := cond.(map[string]interface{}); ok {
				if condType, ok := condMap["type"].(string); ok && condType == "group" {
					// Recursive group
					subGroupExpr, err := convertGroupExpression(condMap)
					if err != nil {
						return nil, err
					}
					subExpressions = append(subExpressions, subGroupExpr[0])
				} else {
					// Single condition
					expr, err := convertSingleExpression(condMap)
					if err != nil {
						return nil, err
					}
					subExpressions = append(subExpressions, expr)
				}
			}
		}
	}

	// Create group expression
	return []models.TriggerExpression{
		{
			ID:   fmt.Sprintf("group_%d", time.Now().UnixNano()),
			Type: models.TriggerExpressionTypeGroup,
			Operator: (*models.TriggerOperator)(func() *string {
				s := string(operator)
				return &s
			}()),
			Conditions: subExpressions,
		},
	}, nil
}

// convertSingleExpression converts a single condition to an expression
func convertSingleExpression(condition map[string]interface{}) (models.TriggerExpression, error) {
	// Extract field
	field, _ := condition["field"].(string)
	
	// Extract operator
	operator := "equals"
	if op, ok := condition["operator"].(string); ok {
		operator = op
	}
	
	// Extract value
	value := condition["value"]
	
	// Extract not flag
	not := false
	if notVal, ok := condition["not"].(bool); ok {
		not = notVal
	}
	
	// Create expression
	return models.TriggerExpression{
		ID:   fmt.Sprintf("cond_%d", time.Now().UnixNano()),
		Type: models.TriggerExpressionTypeCondition,
		Field: func() *string {
			return &field
		}(),
		Operator: (*models.TriggerOperator)(func() *string {
			return &operator
		}()),
		Value: value,
		Not: func() *bool {
			if not {
				return &not
			}
			return nil
		}(),
	}, nil
}

// convertActions converts old actions to new actions
func convertActions(actions json.RawMessage) ([]models.TriggerAction, error) {
	if len(actions) == 0 {
		return []models.TriggerAction{}, nil
	}

	// Parse old actions
	var oldActions []map[string]interface{}
	if err := json.Unmarshal(actions, &oldActions); err != nil {
		return nil, fmt.Errorf("failed to parse old actions: %w", err)
	}

	// Convert to new format
	newActions := make([]models.TriggerAction, 0, len(oldActions))
	for i, oldAction := range oldActions {
		// Extract plugin ID
		pluginID, _ := oldAction["type"].(string)
		if pluginID == "" {
			pluginID = "unknown"
		}

		// Extract plugin name
		pluginName, _ := oldAction["name"].(string)
		if pluginName == "" {
			pluginName = pluginID
		}

		// Extract config
		config := make(map[string]interface{})
		if cfg, ok := oldAction["config"].(map[string]interface{}); ok {
			config = cfg
		}

		// Create new action
		newAction := models.TriggerAction{
			ID:            fmt.Sprintf("action_%d", time.Now().UnixNano()),
			PluginID:      pluginID,
			PluginName:    pluginName,
			Config:        config,
			Enabled:       true, // Default to enabled
			ExecutionOrder: i,
		}

		newActions = append(newActions, newAction)
	}

	return newActions, nil
}