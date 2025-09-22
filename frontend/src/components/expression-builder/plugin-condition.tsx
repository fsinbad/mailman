import React, { useState, useEffect } from 'react'
import { Trash2, ToggleLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { apiClient } from '@/lib/api-client'

// 递归提取JSON对象中的所有字段路径
function extractFieldPaths(obj: any, prefix: string = '', paths: Set<string> = new Set()): string[] {
    if (obj === null || obj === undefined) return Array.from(paths)

    if (typeof obj === 'object' && !Array.isArray(obj)) {
        Object.keys(obj).forEach(key => {
            const currentPath = prefix ? `${prefix}.${key}` : key
            paths.add(currentPath)
            extractFieldPaths(obj[key], currentPath, paths)
        })
    } else if (Array.isArray(obj) && obj.length > 0) {
        // 对于数组，分析第一个元素的结构
        const firstElement = obj[0]
        if (typeof firstElement === 'object' && firstElement !== null) {
            Object.keys(firstElement).forEach(key => {
                const currentPath = prefix ? `${prefix}.${key}` : key
                paths.add(currentPath)
                extractFieldPaths(firstElement[key], currentPath, paths)
            })
        }
    }

    return Array.from(paths)
}

interface PluginConditionProps {
    condition: any
    pluginId: string
    onChange: (condition: any) => void
    onDelete: () => void
    testData?: Record<string, any>
}

interface UIField {
    name: string
    label: string
    type: 'text' | 'number' | 'select' | 'textarea' | 'dynamic' | 'boolean'
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
    operators?: Array<{
        value: string
        label: string
        description?: string
        applicable_to?: string[]
    }>
    layout?: string
    allow_custom_fields?: boolean
    allow_nesting?: boolean
    max_nesting_level?: number
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

export function PluginCondition({
    condition,
    pluginId,
    onChange,
    onDelete,
    testData = {}
}: PluginConditionProps) {
    const [pluginData, setPluginData] = useState<PluginData | null>(null)
    const [loading, setLoading] = useState(true)
    const [dynamicOptions, setDynamicOptions] = useState<Record<string, any[]>>({})

    // 获取插件UI架构
    useEffect(() => {
        fetchPluginSchemas()
    }, [pluginId])

    const fetchPluginSchemas = async () => {
        try {
            setLoading(true)
            const data = await apiClient.get('/plugins/ui/schemas', {
                params: { type: 'condition' }
            })

            // 查找对应的插件数据
            const plugin = data[pluginId]
            if (plugin) {
                setPluginData(plugin)
            }
        } catch (error) {
            console.error('Error fetching plugin schemas:', error)
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

    // 验证字段值
    const validateField = async (field: UIField, value: any) => {
        try {
            await apiClient.post(
                `/plugins/${pluginId}/callbacks/validate-field`,
                { field: field.name, value }
            )
            return null
        } catch (error: any) {
            console.error('Error validating field:', error)
            return error.message || 'Validation failed'
        }
    }

    const handleFieldChange = async (fieldName: string, value: any) => {
        const newCondition = {
            ...condition,
            pluginId,
            fields: {
                ...condition.fields,
                [fieldName]: value
            }
        }
        onChange(newCondition)

        // 触发验证
        const field = pluginData?.schema.fields.find(f => f.name === fieldName)
        if (field) {
            await validateField(field, value)
        }
    }

    const toggleNot = () => {
        onChange({
            ...condition,
            not: !condition.not
        })
    }

    // 检查字段是否应该显示
    const shouldShowField = (field: UIField): boolean => {
        if (field.hidden) return false
        if (!field.show_if) return true

        // 检查show_if条件
        for (const [fieldName, expectedValue] of Object.entries(field.show_if)) {
            const currentValue = condition.fields?.[fieldName]
            if (currentValue !== expectedValue) {
                return false
            }
        }
        return true
    }

    const renderField = (field: UIField) => {
        const value = condition.fields?.[field.name] ?? field.default ?? ''

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
                        value={value}
                        onValueChange={(v) => handleFieldChange(field.name, v)}
                        disabled={field.disabled}
                    >
                        <SelectTrigger className="h-8 text-sm">
                            <SelectValue placeholder={field.placeholder || `选择${field.label}`} />
                        </SelectTrigger>
                        <SelectContent>
                            {field.options?.map(option => (
                                <SelectItem key={option.value} value={option.value}>
                                    <div className="flex items-center gap-2">
                                        {option.icon && <span className="text-sm">{option.icon}</span>}
                                        <span style={{ color: option.color }}>{option.label}</span>
                                    </div>
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                )

            case 'dynamic':
                return (
                    <Select
                        value={value}
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
                                    value={option.value}
                                >
                                    {option.label}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                )

            case 'textarea':
                return (
                    <Textarea
                        value={value}
                        onChange={(e) => handleFieldChange(field.name, e.target.value)}
                        placeholder={field.placeholder}
                        className="min-h-[60px] text-sm resize-none"
                        rows={2}
                        disabled={field.disabled}
                    />
                )

            case 'number':
                return (
                    <Input
                        type="number"
                        value={value}
                        onChange={(e) => handleFieldChange(field.name, e.target.value)}
                        placeholder={field.placeholder}
                        className="h-8 text-sm"
                        min={field.min || undefined}
                        max={field.max || undefined}
                        disabled={field.disabled}
                    />
                )

            default:
                // 只对字段名称包含 field、path、key 等关键词的字段提供智能提示
                // 这些通常是用于输入字段路径的字段，而不是用于输入值的字段
                const isFieldPathInput = field.name.toLowerCase().includes('field') ||
                    field.name.toLowerCase().includes('path') ||
                    field.name.toLowerCase().includes('key') ||
                    field.label.toLowerCase().includes('字段') ||
                    field.label.toLowerCase().includes('路径')

                if (isFieldPathInput) {
                    // 为字段路径输入添加智能提示功能
                    const fieldPaths = extractFieldPaths(testData)
                    const datalistId = `field-suggestions-${field.name}`

                    return (
                        <div className="relative" data-field={field.name}>
                            <Input
                                value={value}
                                onChange={(e) => handleFieldChange(field.name, e.target.value)}
                                placeholder={field.placeholder}
                                className="h-8 text-sm"
                                pattern={field.pattern || undefined}
                                disabled={field.disabled}
                                list={datalistId}
                            />
                            <datalist id={datalistId}>
                                {fieldPaths.map(path => (
                                    <option key={path} value={path} />
                                ))}
                            </datalist>

                            {/* 快速选择按钮 */}
                            {fieldPaths.length > 0 && (
                                <div className="absolute -right-2 top-0 bottom-0 flex flex-col justify-center">
                                    <div className="relative group">
                                        <button
                                            type="button"
                                            className="h-6 w-6 rounded-full bg-blue-500 text-white text-xs flex items-center justify-center hover:bg-blue-600 transition-colors"
                                            onClick={(e) => {
                                                e.preventDefault()
                                                const dropdown = e.currentTarget.nextElementSibling as HTMLElement
                                                dropdown.style.display = dropdown.style.display === 'block' ? 'none' : 'block'
                                            }}
                                        >
                                            ⚡
                                        </button>
                                        <div className="absolute right-0 top-7 bg-white border border-gray-300 rounded-md shadow-lg z-10 min-w-[200px] max-h-[200px] overflow-y-auto hidden">
                                            {fieldPaths.slice(0, 10).map(path => (
                                                <button
                                                    key={path}
                                                    type="button"
                                                    className="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 border-b border-gray-100 last:border-b-0"
                                                    onClick={(e) => {
                                                        handleFieldChange(field.name, path)
                                                        // 隐藏下拉菜单
                                                        const dropdown = e.currentTarget.parentElement as HTMLElement
                                                        if (dropdown) dropdown.style.display = 'none'
                                                    }}
                                                >
                                                    {path}
                                                </button>
                                            ))}
                                        </div>
                                    </div>
                                </div>
                            )}
                        </div>
                    )
                } else {
                    // 普通文本输入框，不提供智能提示
                    return (
                        <Input
                            value={value}
                            onChange={(e) => handleFieldChange(field.name, e.target.value)}
                            placeholder={field.placeholder}
                            className="h-8 text-sm"
                            pattern={field.pattern || undefined}
                            disabled={field.disabled}
                        />
                    )
                }
        }
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center p-4 bg-gray-50 rounded-md">
                <span className="text-sm text-gray-500">加载插件配置...</span>
            </div>
        )
    }

    if (!pluginData) {
        return (
            <div className="flex items-center justify-center p-4 bg-red-50 rounded-md">
                <span className="text-sm text-red-500">无法加载插件配置: {pluginId}</span>
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
            default: return 'col-span-4'
        }
    }

    // 过滤要显示的字段
    const visibleFields = pluginData.schema.fields.filter(shouldShowField)

    return (
        <div className={`p-3 rounded-md bg-white border transition-all ${condition.not ? 'border-red-300 bg-red-50' : 'border-gray-200'
            }`}>
            <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                    <Badge variant="secondary" className="text-xs">
                        {pluginData.info.name}
                    </Badge>
                    {condition.not && (
                        <Badge className="bg-red-100 text-red-700 border-red-200 text-xs px-1.5 py-0">
                            NOT
                        </Badge>
                    )}
                </div>
                <div className="flex items-center gap-1">
                    <Button
                        onClick={toggleNot}
                        variant="ghost"
                        size="sm"
                        className={`h-7 w-7 p-0 ${condition.not
                            ? 'text-red-600 hover:text-red-700 hover:bg-red-100'
                            : 'text-gray-400 hover:text-gray-600 hover:bg-gray-100'
                            }`}
                        title={condition.not ? "取消否定" : "添加否定"}
                    >
                        <ToggleLeft className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                        onClick={onDelete}
                        variant="ghost"
                        size="sm"
                        className="h-7 w-7 p-0 text-gray-400 hover:text-red-600 hover:bg-red-50"
                    >
                        <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                </div>
            </div>

            {/* 字段网格布局 */}
            <div className="grid grid-cols-12 gap-2">
                {visibleFields.map(field => (
                    <div key={field.name} className={getFieldClass(field.width)}>
                        <label className="block text-xs font-medium text-gray-700 mb-1">
                            {field.label}
                            {field.required && <span className="text-red-500">*</span>}
                        </label>
                        {renderField(field)}
                        {field.description && (
                            <p className="text-xs text-gray-500 mt-1">{field.description}</p>
                        )}
                    </div>
                ))}
            </div>

            {/* 帮助文本 */}
            {pluginData.schema.help_text && (
                <div className="mt-2 p-2 bg-blue-50 rounded text-xs text-blue-600">
                    {pluginData.schema.help_text}
                </div>
            )}
        </div>
    )
}
