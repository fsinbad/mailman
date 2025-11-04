'use client'

import { useEffect, useState } from 'react'
import { Settings, Save, RotateCcw, AlertCircle, CheckCircle, Info } from 'lucide-react'
import { systemConfigService, SystemConfig } from '@/services/system-config.service'
import { ThemeTest } from '@/components/theme-test'
import { cn } from '@/lib/utils'

export default function SystemConfigTab() {
    const [configs, setConfigs] = useState<SystemConfig[]>([])
    const [loading, setLoading] = useState(true)
    const [saving, setSaving] = useState<string | null>(null)
    const [searchQuery, setSearchQuery] = useState('')
    const [selectedCategory, setSelectedCategory] = useState<string>('all')
    const [modifiedConfigs, setModifiedConfigs] = useState<Record<string, any>>({})

    useEffect(() => {
        loadConfigs()
    }, [])

    const loadConfigs = async () => {
        try {
            setLoading(true)
            const data = await systemConfigService.getAllConfigs()
            setConfigs(data)
        } catch (error) {
            console.error('Failed to load system configs:', error)
        } finally {
            setLoading(false)
        }
    }

    const handleConfigChange = (key: string, value: any) => {
        setModifiedConfigs(prev => ({
            ...prev,
            [key]: value
        }))
    }

    const handleSaveConfig = async (config: SystemConfig) => {
        const modifiedValue = modifiedConfigs[config.key]
        if (modifiedValue === undefined) return

        // 验证配置值
        const validation = systemConfigService.validateConfigValue(config, modifiedValue)
        if (!validation.valid) {
            alert(`配置值无效: ${validation.error}`)
            return
        }

        try {
            setSaving(config.key)
            await systemConfigService.updateConfigValue(config.key, modifiedValue)

            // 更新本地状态
            setConfigs(prev => prev.map(c =>
                c.key === config.key
                    ? { ...c, current_value: modifiedValue }
                    : c
            ))

            // 清除修改状态
            setModifiedConfigs(prev => {
                const newModified = { ...prev }
                delete newModified[config.key]
                return newModified
            })

        } catch (error) {
            console.error('Failed to save config:', error)
            alert('保存配置失败')
        } finally {
            setSaving(null)
        }
    }

    const handleResetConfig = async (config: SystemConfig) => {
        if (!confirm(`确定要重置"${config.name}"为默认值吗？`)) return

        try {
            setSaving(config.key)
            const resetConfig = await systemConfigService.resetConfigToDefault(config.key)

            // 更新本地状态
            setConfigs(prev => prev.map(c =>
                c.key === config.key ? resetConfig : c
            ))

            // 清除修改状态
            setModifiedConfigs(prev => {
                const newModified = { ...prev }
                delete newModified[config.key]
                return newModified
            })

        } catch (error) {
            console.error('Failed to reset config:', error)
            alert('重置配置失败')
        } finally {
            setSaving(null)
        }
    }

    // 获取当前显示值
    const getCurrentValue = (config: SystemConfig) => {
        return modifiedConfigs[config.key] !== undefined
            ? modifiedConfigs[config.key]
            : config.current_value
    }

    // 检查配置是否被修改
    const isConfigModified = (config: SystemConfig) => {
        return modifiedConfigs[config.key] !== undefined
    }

    // 过滤配置
    const filteredConfigs = (configs || []).filter(config => {
        const matchesSearch = config.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            config.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
            config.key.toLowerCase().includes(searchQuery.toLowerCase())

        const matchesCategory = selectedCategory === 'all' || config.category === selectedCategory

        return matchesSearch && matchesCategory && config.is_visible
    })

    // 按分类分组
    const groupedConfigs = systemConfigService.groupConfigsByCategory(filteredConfigs)
    const categories = Object.keys(groupedConfigs).sort()

    // 获取所有可用分类
    const allCategories = Array.from(new Set((configs || []).map(c => c.category).filter(Boolean)))

    // 渲染配置输入组件
    const renderConfigInput = (config: SystemConfig) => {
        const currentValue = getCurrentValue(config)
        const isModified = isConfigModified(config)

        switch (config.value_type) {
            case 'boolean':
                return (
                    <div className="flex items-center space-x-3">
                        <label className="flex items-center space-x-2 cursor-pointer">
                            <input
                                type="checkbox"
                                checked={currentValue === true}
                                onChange={(e) => handleConfigChange(config.key, e.target.checked)}
                                disabled={!config.is_editable}
                                className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                            />
                            <span className="text-sm font-medium text-gray-900 dark:text-white">
                                {currentValue ? '启用' : '禁用'}
                            </span>
                        </label>
                    </div>
                )

            case 'string':
                return (
                    <input
                        type="text"
                        value={currentValue || ''}
                        onChange={(e) => handleConfigChange(config.key, e.target.value)}
                        disabled={!config.is_editable}
                        className="w-full rounded-lg border border-gray-300 bg-white py-2 px-3 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white disabled:opacity-50"
                        placeholder={config.default_value?.toString() || ''}
                    />
                )

            case 'number':
                return (
                    <input
                        type="number"
                        value={currentValue || 0}
                        onChange={(e) => handleConfigChange(config.key, parseInt(e.target.value))}
                        disabled={!config.is_editable}
                        className="w-full rounded-lg border border-gray-300 bg-white py-2 px-3 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white disabled:opacity-50"
                    />
                )

            case 'float':
                return (
                    <input
                        type="number"
                        step="0.01"
                        value={currentValue || 0}
                        onChange={(e) => handleConfigChange(config.key, parseFloat(e.target.value))}
                        disabled={!config.is_editable}
                        className="w-full rounded-lg border border-gray-300 bg-white py-2 px-3 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white disabled:opacity-50"
                    />
                )

            case 'json':
                return (
                    <textarea
                        value={JSON.stringify(currentValue, null, 2) || ''}
                        onChange={(e) => {
                            try {
                                const parsedValue = JSON.parse(e.target.value)
                                handleConfigChange(config.key, parsedValue)
                            } catch {
                                // 允许输入不完整的JSON，等用户完成编辑
                            }
                        }}
                        disabled={!config.is_editable}
                        rows={4}
                        className="w-full rounded-lg border border-gray-300 bg-white py-2 px-3 text-sm font-mono focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white disabled:opacity-50"
                        placeholder="请输入有效的JSON格式"
                    />
                )

            default:
                return (
                    <div className="text-sm text-gray-500 dark:text-gray-400">
                        不支持的配置类型: {config.value_type}
                    </div>
                )
        }
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center py-20">
                <div className="text-center">
                    <div className="mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-4 border-primary-600 border-t-transparent"></div>
                    <p className="text-gray-500 dark:text-gray-400">加载中...</p>
                </div>
            </div>
        )
    }

    return (
        <div className="space-y-6">
            {/* 页面标题 */}
            <div className="flex items-center space-x-3">
                <Settings className="h-6 w-6 text-primary-600" />
                <h1 className="text-2xl font-bold text-gray-900 dark:text-white">系统配置</h1>
            </div>

            {/* 主题测试组件 - 用于调试主题切换 */}
            <ThemeTest />

            {/* 搜索和筛选 */}
            <div className="flex items-center justify-between">
                <div className="w-96">
                    <input
                        type="text"
                        placeholder="搜索配置项..."
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="w-full rounded-lg border border-gray-300 bg-white py-2 pl-4 pr-4 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                    />
                </div>
                <div className="flex items-center space-x-3">
                    <select
                        value={selectedCategory}
                        onChange={(e) => setSelectedCategory(e.target.value)}
                        className="rounded-lg border border-gray-300 bg-white py-2 px-3 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                    >
                        <option value="all">所有分类</option>
                        {allCategories.map(category => (
                            <option key={category} value={category}>
                                {systemConfigService.getCategoryDisplayName(category)}
                            </option>
                        ))}
                    </select>
                </div>
            </div>

            {/* 配置列表 */}
            {Object.keys(modifiedConfigs).length > 0 && (
                <div className="rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 p-4">
                    <div className="flex items-center space-x-2">
                        <AlertCircle className="h-5 w-5 text-yellow-600 dark:text-yellow-400" />
                        <p className="text-sm font-medium text-yellow-800 dark:text-yellow-200">
                            您有 {Object.keys(modifiedConfigs).length} 项未保存的配置更改
                        </p>
                    </div>
                </div>
            )}

            {selectedCategory === 'all' && categories.length > 1 ? (
                // 按分类显示
                <div className="space-y-8">
                    {categories.map(category => (
                        <div key={category} className="space-y-4">
                            <h2 className="text-lg font-semibold text-gray-900 dark:text-white border-b border-gray-200 dark:border-gray-700 pb-2">
                                {systemConfigService.getCategoryDisplayName(category)}
                            </h2>
                            <div className="grid gap-6">
                                {groupedConfigs[category].map(config => (
                                    <ConfigItem
                                        key={config.key}
                                        config={config}
                                        currentValue={getCurrentValue(config)}
                                        isModified={isConfigModified(config)}
                                        isSaving={saving === config.key}
                                        onSave={() => handleSaveConfig(config)}
                                        onReset={() => handleResetConfig(config)}
                                        renderInput={() => renderConfigInput(config)}
                                    />
                                ))}
                            </div>
                        </div>
                    ))}
                </div>
            ) : (
                // 单一分类或搜索结果
                <div className="grid gap-6">
                    {filteredConfigs.map(config => (
                        <ConfigItem
                            key={config.key}
                            config={config}
                            currentValue={getCurrentValue(config)}
                            isModified={isConfigModified(config)}
                            isSaving={saving === config.key}
                            onSave={() => handleSaveConfig(config)}
                            onReset={() => handleResetConfig(config)}
                            renderInput={() => renderConfigInput(config)}
                        />
                    ))}
                </div>
            )}

            {filteredConfigs.length === 0 && !loading && (
                <div className="text-center py-12">
                    <Settings className="mx-auto h-12 w-12 text-gray-400" />
                    <h3 className="mt-4 text-lg font-medium text-gray-900 dark:text-white">没有找到配置项</h3>
                    <p className="mt-2 text-gray-500 dark:text-gray-400">
                        {searchQuery ? '尝试其他搜索词' : '当前分类下没有可用的配置项'}
                    </p>
                </div>
            )}
        </div>
    )
}

// 配置项组件
interface ConfigItemProps {
    config: SystemConfig
    currentValue: any
    isModified: boolean
    isSaving: boolean
    onSave: () => void
    onReset: () => void
    renderInput: () => React.ReactNode
}

function ConfigItem({ config, currentValue, isModified, isSaving, onSave, onReset, renderInput }: ConfigItemProps) {
    return (
        <div className={cn(
            "rounded-lg border bg-white p-6 transition-all dark:bg-gray-800",
            isModified
                ? "border-primary-500 bg-primary-50 dark:border-primary-400 dark:bg-primary-900/20"
                : "border-gray-200 dark:border-gray-700"
        )}>
            <div className="flex items-start justify-between">
                <div className="flex-1 space-y-4">
                    {/* 配置信息 */}
                    <div>
                        <div className="flex items-center space-x-3">
                            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                                {config.name}
                            </h3>
                            {isModified && (
                                <span className="inline-flex items-center rounded-full bg-yellow-100 px-2.5 py-0.5 text-xs font-medium text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400">
                                    已修改
                                </span>
                            )}
                            {!config.is_editable && (
                                <span className="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-400">
                                    只读
                                </span>
                            )}
                        </div>

                        <p className="mt-1 text-sm text-gray-600 dark:text-gray-300">
                            {config.description}
                        </p>

                        <div className="mt-2 flex items-center space-x-4 text-xs text-gray-500 dark:text-gray-400">
                            <span>键名: <code className="bg-gray-100 dark:bg-gray-700 px-1 rounded">{config.key}</code></span>
                            <span>类型: {config.value_type}</span>
                            <span>默认值: {systemConfigService.formatConfigValue({ ...config, current_value: config.default_value })}</span>
                        </div>
                    </div>

                    {/* 配置输入 */}
                    <div className="space-y-2">
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                            当前值
                        </label>
                        {renderInput()}
                    </div>
                </div>

                {/* 操作按钮 */}
                {config.is_editable && (
                    <div className="flex items-center space-x-2 ml-6">
                        {isModified && (
                            <button
                                onClick={onSave}
                                disabled={isSaving}
                                className="flex items-center space-x-1 rounded-lg bg-primary-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-700 disabled:opacity-50"
                            >
                                {isSaving ? (
                                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                                ) : (
                                    <Save className="h-4 w-4" />
                                )}
                                <span>保存</span>
                            </button>
                        )}

                        <button
                            onClick={onReset}
                            disabled={isSaving}
                            className="flex items-center space-x-1 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700 disabled:opacity-50"
                            title="重置为默认值"
                        >
                            <RotateCcw className="h-4 w-4" />
                            <span>重置</span>
                        </button>
                    </div>
                )}
            </div>

            {/* 修改提示 */}
            {isModified && (
                <div className="mt-4 pt-4 border-t border-primary-200 dark:border-primary-800">
                    <div className="flex items-center space-x-2 text-sm text-primary-700 dark:text-primary-300">
                        <Info className="h-4 w-4" />
                        <span>配置已修改但未保存，点击"保存"按钮确认更改</span>
                    </div>
                </div>
            )}
        </div>
    )
}