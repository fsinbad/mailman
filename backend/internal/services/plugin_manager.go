package services

import (
	"fmt"
	"log"
	"time"

	"mailman/internal/models"
)

// PluginManager manages and executes plugins for trigger actions
type PluginManager struct {
	// Dependencies can be added here as needed
	plugins map[string]Plugin
}

// Plugin represents a plugin that can execute actions
type Plugin interface {
	Execute(config map[string]interface{}, context *PluginContext) (*PluginResult, error)
	GetName() string
	GetDescription() string
	GetConfigSchema() map[string]interface{}
}

// PluginContext contains the data needed for plugin execution
type PluginContext struct {
	Email models.Email
	// Additional context data can be added here
}

// PluginResult contains the result of a plugin execution
type PluginResult struct {
	Success bool
	Data    interface{}
	Error   string
}

// NewPluginManager creates a new PluginManager
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]Plugin),
	}
}

// RegisterPlugin registers a plugin with the manager
func (m *PluginManager) RegisterPlugin(pluginID string, plugin Plugin) {
	m.plugins[pluginID] = plugin
	log.Printf("[PluginManager] Registered plugin: %s (%s)", pluginID, plugin.GetName())
}

// GetPlugin gets a plugin by ID
func (m *PluginManager) GetPlugin(pluginID string) (Plugin, error) {
	plugin, exists := m.plugins[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginID)
	}
	return plugin, nil
}

// ExecuteAction executes a trigger action
func (m *PluginManager) ExecuteAction(action models.TriggerAction, context *PluginContext) (*models.ActionExecutionResult, error) {
	log.Printf("[PluginManager] Executing action: %s (Plugin: %s)", action.ID, action.PluginID)
	
	// Skip disabled actions
	if !action.Enabled {
		log.Printf("[PluginManager] Skipping disabled action: %s", action.ID)
		return &models.ActionExecutionResult{
			ActionID:   action.ID,
			PluginID:   action.PluginID,
			PluginName: action.PluginName,
			Success:    false,
			StartTime:  time.Now(),
			EndTime:    time.Now(),
			Duration:   0,
			Error:      "Action is disabled",
		}, nil
	}
	
	// Get the plugin
	plugin, err := m.GetPlugin(action.PluginID)
	if err != nil {
		return &models.ActionExecutionResult{
			ActionID:   action.ID,
			PluginID:   action.PluginID,
			PluginName: action.PluginName,
			Success:    false,
			StartTime:  time.Now(),
			EndTime:    time.Now(),
			Duration:   0,
			Error:      fmt.Sprintf("Plugin not found: %s", err),
		}, err
	}
	
	// Execute the plugin
	startTime := time.Now()
	result, err := plugin.Execute(action.Config, context)
	endTime := time.Now()
	
	// Create the execution result
	executionResult := &models.ActionExecutionResult{
		ActionID:   action.ID,
		PluginID:   action.PluginID,
		PluginName: action.PluginName,
		StartTime:  startTime,
		EndTime:    endTime,
		Duration:   endTime.Sub(startTime).Milliseconds(),
	}
	
	if err != nil {
		executionResult.Success = false
		executionResult.Error = err.Error()
		log.Printf("[PluginManager] Action execution failed: %s - %v", action.ID, err)
	} else if result == nil {
		executionResult.Success = false
		executionResult.Error = "Plugin returned nil result"
		log.Printf("[PluginManager] Action execution failed: %s - Plugin returned nil result", action.ID)
	} else {
		executionResult.Success = result.Success
		executionResult.Result = result.Data
		if !result.Success {
			executionResult.Error = result.Error
			log.Printf("[PluginManager] Action execution failed: %s - %s", action.ID, result.Error)
		} else {
			log.Printf("[PluginManager] Action execution succeeded: %s", action.ID)
		}
	}
	
	return executionResult, nil
}

// ExecuteActions executes multiple trigger actions in order
func (m *PluginManager) ExecuteActions(actions []models.TriggerAction, context *PluginContext) (models.ActionExecutionResults, error) {
	log.Printf("[PluginManager] Executing %d actions", len(actions))
	
	results := make(models.ActionExecutionResults, 0, len(actions))
	var firstError error
	
	// Sort actions by execution order
	// In a real implementation, we would sort the actions here
	
	// Execute each action in order
	for _, action := range actions {
		result, err := m.ExecuteAction(action, context)
		results = append(results, *result)
		
		if err != nil && firstError == nil {
			firstError = err
		}
	}
	
	return results, firstError
}