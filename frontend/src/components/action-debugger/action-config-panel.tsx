'use client'

import React, { useState, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { apiClient } from '@/lib/api-client'
import { HelpTooltip } from './help-tooltip'

interface Action {
    id: string
    pluginId: string
    pluginName: string
    config: Record<string, any>
    enabled: boolean
    executionOrder: number
}

interface UIField {
    name: string
    label: string
    type: 'text' | 'number' | 'select' | 'multi_select' | 'textarea' | 'dynamic' | 'boolean' | 'date' | 'time' | 'json' | 'code'
    description?: string
    placeholder?: string
    required?: boolean
    pattern?: string
    min?: number | null
    max?: number | null
    default?: any
    width?: string
    hidden?: boolean
    disabled?: boolean
    options?: Array<{
        value: string
        label: string
        description?: string
        icon?: string
        color?: string
    }>
    options_api?: string
    show_if?: Record<string, any>
    depends_on?: string | null
}

interface UISchema {
    fields: UIField[]
    help_text?: string
    examples?: Array<{
        title: string
        description: string
        expression: any
    }>
}

interface PluginData {
    info: {
        id: string
        name: string
        description: string
    }
    schema: UISchema
}

interface ActionConfigPanelProps {
    action: Action
    availablePlugins: Array<{
        id: string
        name: string
        description: string
    }>
    onChange: (config: Record<string, any>) => void
}

export function ActionConfigPanel({ action, availablePlugins, onChange }: ActionConfigPanelProps) {
    const [pluginData, setPluginData] = useState<PluginData | null>(null)
    const [loading, setLoading] = useState(true)
    const [dynamicOptions, setDynamicOptions] = useState<Record<string, any[]>>({})

    // 获取插件UI架构
    useEffect(() => {
        if (action.pluginId) {
            fetchPluginSchema()
        }
    }, [action.pluginId])

    const fetchPluginSchema = async () => {
        try {
            setLoading(true)
            const data = await apiClient.get('/plugins/ui/schemas', {
                params: { type: 'action' }
            })

            // 查找对应的插件数据
            const plugin = data[action.pluginId]
            if (plugin) {
                setPluginData(plugin)
            }
        } catch (error) {
            console.error('获取插件配置架构失败:', error)
        } finally {
            setLoading(false)
        }
    }

    // 获取动态选项
    const fetchDynamicOptions = async (field: UIField, query: string = '') => {
        if (field.type !== 'dynamic' || !field.options_api) return

        try {
            const data = await apiClient.get(field.options_api, {
                params: { query }
            })
            setDynamicOptions(prev => ({
                ...prev,
                [field.name]: data.options || []
            }))
        } catch (error) {
            console.error('Error fetching dynamic options:', error)
        }
    }

    const handleFieldChange = (fieldName: string, value: any) => {
        const newConfig = {
            ...action.config,
            [fieldName]: value
        }
        onChange(newConfig)
    }

    // 检查字段是否应该显示
    const shouldShowField = (field: UIField): boolean => {
        if (field.hidden) return false
        if (!field.show_if) return true

        // 检查show_if条件
        for (const [fieldName, expectedValues] of Object.entries(field.show_if)) {
            const currentValue = action.config?.[fieldName]
            if (Array.isArray(expectedValues)) {
                if (!expectedValues.includes(currentValue)) {
                    return false
                }
            } else {
                if (currentValue !== expectedValues) {
                    return false
                }
            }
        }
        return true
    }

    const renderField = (field: UIField) => {
        const value = action.config?.[field.name] ?? field.default ?? ''

        const fieldComponent = (() => {
            switch (field.type) {
                case 'boolean':
                    return (
                        <div className="flex items-center space-x-2">
                            <Switch
                                checked={!!value}
                                onCheckedChange={(checked) => handleFieldChange(field.name, checked)}
                                disabled={field.disabled}
                            />
                            <span className="text-sm text-gray-600">
                                {value ? '是' : '否'}
                            </span>
                        </div>
                    )

                case 'select':
                    return (
                        <Select
                            value={String(value)}
                            onValueChange={(v) => handleFieldChange(field.name, v)}
                            disabled={field.disabled}
                        >
                            <SelectTrigger className="h-8 text-sm">
                                <SelectValue placeholder={field.placeholder || `选择${field.label}`} />
                            </SelectTrigger>
                            <SelectContent>
                                {field.options?.map(option => (
                                    <SelectItem key={option.value} value={String(option.value)}>
                                        <div className="flex items-center gap-2">
                                            {option.icon && <span className="text-sm">{option.icon}</span>}
                                            <span style={{ color: option.color }}>{option.label}</span>
                                        </div>
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    )

                case 'multi_select':
                    return (
                        <div className="space-y-2">
                            {field.options?.map((option: any) => (
                                <div key={option.value} className="flex items-center space-x-2">
                                    <input
                                        type="checkbox"
                                        id={`${field.name}-${option.value}`}
                                        checked={Array.isArray(value) && value.includes(option.value)}
                                        onChange={(e) => {
                                            const currentArray = Array.isArray(value) ? value : []
                                            if (e.target.checked) {
                                                handleFieldChange(field.name, [...currentArray, option.value])
                                            } else {
                                                handleFieldChange(field.name, currentArray.filter(v => v !== option.value))
                                            }
                                        }}
                                        className="rounded border-gray-300"
                                        disabled={field.disabled}
                                    />
                                    <label htmlFor={`${field.name}-${option.value}`} className="text-sm">
                                        {option.label}
                                    </label>
                                </div>
                            ))}
                        </div>
                    )

                case 'dynamic':
                    return (
                        <Select
                            value={String(value)}
                            onValueChange={(v) => handleFieldChange(field.name, v)}
                            onOpenChange={(open) => {
                                if (open) fetchDynamicOptions(field)
                            }}
                            disabled={field.disabled}
                        >
                            <SelectTrigger className="h-8 text-sm">
                                <SelectValue placeholder={field.placeholder || `选择${field.label}`} />
                            </SelectTrigger>
                            <SelectContent>
                                {(dynamicOptions[field.name] || []).map((option: any) => (
                                    <SelectItem
                                        key={option.value}
                                        value={String(option.value)}
                                    >
                                        {option.label}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    )

                case 'textarea':
                case 'code':
                    return (
                        <Textarea
                            value={String(value)}
                            onChange={(e) => handleFieldChange(field.name, e.target.value)}
                            placeholder={field.placeholder}
                            className={`min-h-[80px] text-sm resize-none ${field.type === 'code' ? 'font-mono' : ''}`}
                            rows={field.type === 'code' ? 4 : 3}
                            disabled={field.disabled}
                        />
                    )

                case 'number':
                    return (
                        <Input
                            type="number"
                            value={String(value)}
                            onChange={(e) => handleFieldChange(field.name, e.target.value ? parseFloat(e.target.value) : '')}
                            placeholder={field.placeholder}
                            className="h-8 text-sm"
                            min={field.min || undefined}
                            max={field.max || undefined}
                            disabled={field.disabled}
                        />
                    )

                case 'date':
                    return (
                        <Input
                            type="date"
                            value={String(value)}
                            onChange={(e) => handleFieldChange(field.name, e.target.value)}
                            className="h-8 text-sm"
                            disabled={field.disabled}
                        />
                    )

                case 'time':
                    return (
                        <Input
                            type="time"
                            value={String(value)}
                            onChange={(e) => handleFieldChange(field.name, e.target.value)}
                            className="h-8 text-sm"
                            disabled={field.disabled}
                        />
                    )

                case 'json':
                    return (
                        <Textarea
                            value={typeof value === 'string' ? value : JSON.stringify(value, null, 2)}
                            onChange={(e) => {
                                try {
                                    const parsed = JSON.parse(e.target.value)
                                    handleFieldChange(field.name, parsed)
                                } catch {
                                    handleFieldChange(field.name, e.target.value)
                                }
                            }}
                            placeholder={field.placeholder || '输入JSON格式数据'}
                            className="min-h-[100px] font-mono text-sm"
                            disabled={field.disabled}
                        />
                    )

                default:
                    // 默认使用文本输入
                    return (
                        <Input
                            value={String(value)}
                            onChange={(e) => handleFieldChange(field.name, e.target.value)}
                            placeholder={field.placeholder}
                            className="h-8 text-sm"
                            pattern={field.pattern || undefined}
                            disabled={field.disabled}
                        />
                    )
            }
        })()

        return (
            <div key={field.name} className="space-y-2">
                <div className="flex items-center gap-2">
                    <Label htmlFor={`field-${field.name}`} className="text-sm font-medium">
                        {field.label}
                        {field.required && <span className="text-red-500">*</span>}
                    </Label>
                    {field.description && (
                        <HelpTooltip content={field.description} />
                    )}
                </div>
                {fieldComponent}
            </div>
        )
    }

    // 根据字段宽度计算网格类
    const getFieldClass = (width?: string) => {
        switch (width) {
            case '1/4': return 'col-span-3'
            case '1/3': return 'col-span-4'
            case '1/2':
            case 'half': return 'col-span-6'
            case '2/3': return 'col-span-8'
            case '3/4': return 'col-span-9'
            case 'full': return 'col-span-12'
            default: return 'col-span-12'
        }
    }

    if (loading) {
        return (
            <Card className="p-6">
                <div className="flex items-center justify-center p-8 bg-gray-50 rounded-md">
                    <span className="text-sm text-gray-500">加载插件配置...</span>
                </div>
            </Card>
        )
    }

    if (!pluginData) {
        return (
            <Card className="p-6">
                <div className="text-center py-8 text-gray-500">
                    <div className="text-4xl mb-4">⚠️</div>
                    <p>无法加载插件配置</p>
                    <p className="text-sm mt-2">插件ID: {action.pluginId}</p>
                </div>
            </Card>
        )
    }

    return (
        <div className="space-y-6">
            {/* 插件信息 */}
            <Card className="p-4">
                <div className="flex items-start gap-3">
                    <div className="flex-1">
                        <h3 className="font-semibold text-lg">{pluginData.info.name}</h3>
                        <p className="text-sm text-gray-600 mt-1">{pluginData.info.description}</p>
                        <Badge variant="secondary" className="mt-2 text-xs">
                            {action.pluginId}
                        </Badge>
                    </div>
                </div>
            </Card>

            {/* 配置字段 */}
            <Card className="p-4">
                <h4 className="font-medium mb-4 flex items-center gap-2">
                    ⚙️ 配置参数
                    {pluginData.schema.help_text && (
                        <HelpTooltip content={pluginData.schema.help_text} />
                    )}
                </h4>
                
                <div className="grid grid-cols-12 gap-4">
                    {pluginData.schema.fields.filter(shouldShowField).map(field => (
                        <div key={field.name} className={getFieldClass(field.width)}>
                            {renderField(field)}
                        </div>
                    ))}
                </div>
            </Card>

            {/* 配置示例 */}
            {pluginData.schema.examples && pluginData.schema.examples.length > 0 && (
                <Card className="p-4">
                    <h4 className="font-medium mb-4 flex items-center gap-2">
                        💡 配置示例
                        <HelpTooltip content="点击示例可以快速应用配置" />
                    </h4>
                    <div className="space-y-3">
                        {pluginData.schema.examples.map((example, index) => (
                            <div key={index} className="border rounded-lg p-3 hover:bg-gray-50 cursor-pointer"
                                 onClick={() => onChange(example.expression)}>
                                <h5 className="font-medium text-sm">{example.title}</h5>
                                <p className="text-xs text-gray-600 mt-1">{example.description}</p>
                                <pre className="text-xs bg-gray-100 p-2 rounded mt-2 overflow-x-auto">
                                    {JSON.stringify(example.expression, null, 2)}
                                </pre>
                            </div>
                        ))}
                    </div>
                </Card>
            )}

            {/* 当前配置预览 */}
            {Object.keys(action.config).length > 0 && (
                <Card className="p-4">
                    <h4 className="font-medium mb-4">📋 当前配置</h4>
                    <pre className="text-xs bg-gray-100 p-3 rounded overflow-x-auto">
                        {JSON.stringify(action.config, null, 2)}
                    </pre>
                </Card>
            )}
        </div>
    )
}