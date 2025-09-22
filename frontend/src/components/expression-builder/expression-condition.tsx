import React, { useState, useMemo } from 'react'
import { Trash2, ToggleLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'

interface ExpressionConditionProps {
    condition: any
    onChange: (condition: any) => void
    onDelete: () => void
    testData?: Record<string, any>
}

export function ExpressionCondition({
    condition,
    onChange,
    onDelete,
    testData = {}
}: ExpressionConditionProps) {
    // 递归提取testData中的所有字段路径
    const extractFieldPaths = (obj: any, prefix: string = ''): string[] => {
        const paths: string[] = []

        if (obj && typeof obj === 'object' && !Array.isArray(obj)) {
            for (const [key, value] of Object.entries(obj)) {
                const currentPath = prefix ? `${prefix}.${key}` : key
                paths.push(currentPath)

                if (value && typeof value === 'object' && !Array.isArray(value)) {
                    paths.push(...extractFieldPaths(value, currentPath))
                }
            }
        }

        return paths
    }

    // 获取所有可用的字段路径
    const availableFields = useMemo(() => {
        return extractFieldPaths(testData).sort()
    }, [testData])

    // 过滤字段建议
    const fieldSuggestions = useMemo(() => {
        if (!condition.field) return availableFields

        const query = condition.field.toLowerCase()
        return availableFields.filter(field =>
            field.toLowerCase().includes(query)
        )
    }, [availableFields, condition.field])

    const handleFieldChange = (field: string) => {
        onChange({
            ...condition,
            field
        })
    }

    const handleOperatorChange = (operator: string) => {
        onChange({
            ...condition,
            operator
        })
    }

    const handleValueChange = (value: string) => {
        onChange({
            ...condition,
            value
        })
    }

    const toggleNot = () => {
        onChange({
            ...condition,
            not: !condition.not
        })
    }

    // 获取字段的当前值（如果存在）
    const currentFieldValue = condition.field ?
        condition.field.split('.').reduce((acc: any, key: string) => acc?.[key], testData) :
        undefined

    // 操作符选项
    const operators = [
        { value: 'equals', label: '等于' },
        { value: 'not_equals', label: '不等于' },
        { value: 'contains', label: '包含' },
        { value: 'not_contains', label: '不包含' },
        { value: 'starts_with', label: '开头是' },
        { value: 'ends_with', label: '结尾是' },
        { value: 'greater_than', label: '大于' },
        { value: 'less_than', label: '小于' },
        { value: 'in', label: '在列表中' },
        { value: 'not_in', label: '不在列表中' }
    ]

    return (
        <div className={`flex items-center gap-2 p-2 rounded-md bg-white border transition-all ${condition.not ? 'border-red-300 bg-red-50' : 'border-gray-200'
            }`}>
            {/* NOT 标记 */}
            {condition.not && (
                <Badge className="bg-red-100 text-red-700 border-red-200 text-xs px-1.5 py-0">
                    NOT
                </Badge>
            )}

            {/* 字段选择 */}
            <div className="flex-1 min-w-0">
                <div className="relative">
                    <Input
                        value={condition.field || ''}
                        onChange={(e) => handleFieldChange(e.target.value)}
                        placeholder="选择字段 (开始输入以查看建议)"
                        className="h-8 text-sm"
                        list="field-suggestions"
                        autoComplete="off"
                    />
                    <datalist id="field-suggestions">
                        {fieldSuggestions.map((field, index) => (
                            <option key={index} value={field} />
                        ))}
                    </datalist>

                    {/* 显示建议计数 */}
                    {condition.field && fieldSuggestions.length > 0 && (
                        <div className="absolute right-2 top-1/2 transform -translate-y-1/2 text-xs text-gray-400 pointer-events-none">
                            {fieldSuggestions.length} 个建议
                        </div>
                    )}
                </div>

                {currentFieldValue !== undefined && (
                    <div className="text-xs text-gray-500 mt-0.5 truncate">
                        当前值: {JSON.stringify(currentFieldValue)}
                    </div>
                )}

                {/* 显示前几个建议作为快速选择 */}
                {condition.field && fieldSuggestions.length > 0 && fieldSuggestions.length <= 5 && (
                    <div className="flex flex-wrap gap-1 mt-1">
                        {fieldSuggestions.map((field, index) => (
                            <button
                                key={index}
                                onClick={() => handleFieldChange(field)}
                                className="text-xs bg-blue-50 hover:bg-blue-100 text-blue-700 px-2 py-1 rounded border border-blue-200 transition-colors"
                            >
                                {field}
                            </button>
                        ))}
                    </div>
                )}
            </div>

            {/* 操作符选择 */}
            <Select value={condition.operator || 'equals'} onValueChange={handleOperatorChange}>
                <SelectTrigger className="w-32 h-8 text-sm">
                    <SelectValue />
                </SelectTrigger>
                <SelectContent>
                    {operators.map(op => (
                        <SelectItem key={op.value} value={op.value}>
                            {op.label}
                        </SelectItem>
                    ))}
                </SelectContent>
            </Select>

            {/* 值输入 */}
            <Input
                value={condition.value || ''}
                onChange={(e) => handleValueChange(e.target.value)}
                placeholder="输入值"
                className="flex-1 min-w-0 h-8 text-sm"
            />

            {/* 操作按钮 */}
            <div className="flex items-center gap-1">
                {/* NOT 切换按钮 */}
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

                {/* 删除按钮 */}
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
    )
}