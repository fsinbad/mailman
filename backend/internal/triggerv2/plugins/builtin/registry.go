package builtin

import (
	"fmt"
	"mailman/internal/triggerv2/plugins"
)

// GetBuiltinPlugins 获取所有内置插件
func GetBuiltinPlugins() []plugins.Plugin {
	return []plugins.Plugin{
		// 现有的内置插件
		NewEmailConditionPlugin(),
		NewEmailFilterPlugin(),
		NewNotificationActionPlugin(),

		// 新增的邮件条件插件
		NewEmailAccountSetPlugin(),
		NewEmailPrefixPlugin(),
		NewEmailSuffixPlugin(),
		NewEmailTimeRangePlugin(),
		NewEmailSizePlugin(),

		// 新增的邮件动作插件
		NewEmailForwardActionPlugin(),
		NewEmailDeleteActionPlugin(),
		NewEmailLabelActionPlugin(),
		NewEmailTransformActionPlugin(),
	}
}

// RegisterBuiltinPlugins 注册所有内置插件到管理器
func RegisterBuiltinPlugins(manager plugins.PluginManager) error {
	plugins := GetBuiltinPlugins()

	for _, plugin := range plugins {
		if err := manager.RegisterPlugin(plugin); err != nil {
			return err
		}
	}

	return nil
}

// GetBuiltinPluginByID 根据ID获取内置插件
func GetBuiltinPluginByID(id string) plugins.Plugin {
	pluginMap := map[string]func() plugins.Plugin{
		"builtin.email_condition":   func() plugins.Plugin { return NewEmailConditionPlugin() },
		"email_filter":              func() plugins.Plugin { return NewEmailFilterPlugin() },
		"notification_action":       func() plugins.Plugin { return NewNotificationActionPlugin() },
		"builtin.email_account_set": func() plugins.Plugin { return NewEmailAccountSetPlugin() },
		"builtin.email_prefix":      func() plugins.Plugin { return NewEmailPrefixPlugin() },
		"builtin.email_suffix":      func() plugins.Plugin { return NewEmailSuffixPlugin() },
		"builtin.email_time_range":  func() plugins.Plugin { return NewEmailTimeRangePlugin() },
		"builtin.email_size":        func() plugins.Plugin { return NewEmailSizePlugin() },
		"email_forward_action":      func() plugins.Plugin { return NewEmailForwardActionPlugin() },
		"email_delete_action":       func() plugins.Plugin { return NewEmailDeleteActionPlugin() },
		"email_label_action":        func() plugins.Plugin { return NewEmailLabelActionPlugin() },
		"email_transform_action":    func() plugins.Plugin { return NewEmailTransformActionPlugin() },
	}

	if factory, exists := pluginMap[id]; exists {
		return factory()
	}

	return nil
}

// GetBuiltinPluginIDs 获取所有内置插件的ID列表
func GetBuiltinPluginIDs() []string {
	return []string{
		"builtin.email_condition",
		"email_filter",
		"notification_action",
		"builtin.email_account_set",
		"builtin.email_prefix",
		"builtin.email_suffix",
		"builtin.email_time_range",
		"builtin.email_size",
		"email_forward_action",
		"email_delete_action",
		"email_label_action",
		"email_transform_action",
	}
}

// ValidateBuiltinPluginConfig 验证内置插件配置
func ValidateBuiltinPluginConfig(pluginID string, config map[string]interface{}) error {
	plugin := GetBuiltinPluginByID(pluginID)
	if plugin == nil {
		return fmt.Errorf("plugin not found: %s", pluginID)
	}

	return plugin.ValidateConfig(config)
}

// GetBuiltinPluginInfo 获取所有内置插件信息
func GetBuiltinPluginInfo() []*plugins.PluginInfo {
	var infos []*plugins.PluginInfo

	for _, plugin := range GetBuiltinPlugins() {
		infos = append(infos, plugin.GetInfo())
	}

	return infos
}

// IsBuiltinPlugin 检查是否为内置插件
func IsBuiltinPlugin(pluginID string) bool {
	for _, id := range GetBuiltinPluginIDs() {
		if id == pluginID {
			return true
		}
	}
	return false
}
