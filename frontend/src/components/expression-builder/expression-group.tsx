import React, { useState, useEffect } from 'react'
import { Plus, Trash2, ChevronDown, Puzzle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ExpressionCondition } from './expression-condition'
import { PluginCondition } from './plugin-condition'
import { Badge } from '@/components/ui/badge'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
    DropdownMenuSeparator,
    DropdownMenuLabel
} from '@/components/ui/dropdown-menu'
import { apiClient } from '@/lib/api-client'

interface ExpressionGroupProps {
    expression: any
    onChange: (expression: any) => void
    onDelete?: () => void
    testData?: Record<string, any>
    isRoot?: boolean
}

export function ExpressionGroup({
    expression,
    onChange,
    onDelete,
    testData = {},
    isRoot = false
}: ExpressionGroupProps) {
    const [availablePlugins, setAvailablePlugins] = useState<any[]>([])

    useEffect(() => {
        fetchAvailablePlugins()
    }, [])

    const fetchAvailablePlugins = async () => {
        try {
            const data = await apiClient.get('/plugins/ui/schemas', {
                params: { type: 'condition' }
            })
            const plugins = Object.entries(data).map(([id, plugin]: [string, any]) => ({
                id,
                name: plugin.info.name,
                description: plugin.info.description
            }))
            setAvailablePlugins(plugins)
        } catch (error) {
            console.error('Failed to fetch plugins:', error)
        }
    }
    const handleOperatorChange = (operator: string) => {
        onChange({
            ...expression,
            operator
        })
    }

    const handleAddCondition = (pluginId?: string) => {
        const newCondition = pluginId ? {
            id: Date.now().toString(),
            type: 'plugin',
            pluginId,
            fields: {},
            not: false
        } : {
            id: Date.now().toString(),
            type: 'condition',
            field: '',
            operator: 'equals',
            value: ''
        }

        onChange({
            ...expression,
            conditions: [...(expression.conditions || []), newCondition]
        })
    }

    const handleAddGroup = () => {
        const newGroup = {
            id: Date.now().toString(),
            type: 'group',
            operator: 'and',
            conditions: []
        }

        onChange({
            ...expression,
            conditions: [...(expression.conditions || []), newGroup]
        })
    }

    const handleUpdateCondition = (index: number, updatedCondition: any) => {
        const newConditions = [...(expression.conditions || [])]
        newConditions[index] = updatedCondition
        onChange({
            ...expression,
            conditions: newConditions
        })
    }

    const handleDeleteCondition = (index: number) => {
        const newConditions = (expression.conditions || []).filter((_: any, i: number) => i !== index)
        onChange({
            ...expression,
            conditions: newConditions
        })
    }

    const operatorStyles = {
        and: {
            border: 'border-blue-200',
            bg: 'bg-blue-50/50',
            hover: 'hover:bg-blue-50',
            badge: 'bg-blue-100 text-blue-700 border-blue-200',
            connector: 'bg-blue-400'
        },
        or: {
            border: 'border-purple-200',
            bg: 'bg-purple-50/50',
            hover: 'hover:bg-purple-50',
            badge: 'bg-purple-100 text-purple-700 border-purple-200',
            connector: 'bg-purple-400'
        },
        not: {
            border: 'border-red-200',
            bg: 'bg-red-50/50',
            hover: 'hover:bg-red-50',
            badge: 'bg-red-100 text-red-700 border-red-200',
            connector: 'bg-red-400'
        }
    }

    const currentStyle = operatorStyles[expression.operator as keyof typeof operatorStyles] || operatorStyles.and

    return (
        <div className={`rounded-lg border ${currentStyle.border} ${currentStyle.bg} p-3 transition-all`}>
            {/* 组头部 - 更紧凑的设计 */}
            <div className="flex items-center justify-between gap-2 mb-2">
                <div className="flex items-center gap-2">
                    <Select value={expression.operator} onValueChange={handleOperatorChange}>
                        <SelectTrigger className="h-8 w-24 bg-white border-gray-200 text-sm font-medium">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="and">
                                <span className="font-medium">并且</span>
                            </SelectItem>
                            <SelectItem value="or">
                                <span className="font-medium">或者</span>
                            </SelectItem>
                            <SelectItem value="not">
                                <span className="font-medium">非</span>
                            </SelectItem>
                        </SelectContent>
                    </Select>
                    {(expression.conditions || []).length > 0 && (
                        <Badge variant="outline" className={`text-xs px-2 py-0.5 ${currentStyle.badge}`}>
                            {(expression.conditions || []).length} 个条件
                        </Badge>
                    )}
                </div>
                {!isRoot && (
                    <Button
                        onClick={onDelete}
                        variant="ghost"
                        size="sm"
                        className="h-7 w-7 p-0 hover:bg-red-50 hover:text-red-600"
                    >
                        <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                )}
            </div>

            {/* 条件列表 - 更紧凑的间距 */}
            {(expression.conditions || []).length > 0 && (
                <div className="space-y-2 ml-6 relative">
                    {/* 连接线 */}
                    <div className={`absolute left-[-16px] top-2 bottom-2 w-0.5 ${currentStyle.connector}`} />

                    {(expression.conditions || []).map((condition: any, index: number) => (
                        <div key={condition.id || index} className="relative">
                            {/* 连接点 */}
                            <div className={`absolute left-[-20px] top-4 w-2 h-2 rounded-full ${currentStyle.connector}`} />

                            {condition.type === 'group' ? (
                                <ExpressionGroup
                                    expression={condition}
                                    onChange={(updated) => handleUpdateCondition(index, updated)}
                                    onDelete={() => handleDeleteCondition(index)}
                                    testData={testData}
                                />
                            ) : condition.type === 'plugin' ? (
                                <PluginCondition
                                    condition={condition}
                                    pluginId={condition.pluginId}
                                    onChange={(updated) => handleUpdateCondition(index, updated)}
                                    onDelete={() => handleDeleteCondition(index)}
                                    testData={testData}
                                />
                            ) : (
                                <ExpressionCondition
                                    condition={condition}
                                    onChange={(updated: any) => handleUpdateCondition(index, updated)}
                                    onDelete={() => handleDeleteCondition(index)}
                                    testData={testData}
                                />
                            )}
                        </div>
                    ))}
                </div>
            )}

            {/* 添加按钮 - 更紧凑的设计 */}
            <div className="flex gap-1.5 mt-2">
                <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                        <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 text-xs bg-white hover:bg-gray-50 border border-gray-200"
                        >
                            <Plus className="h-3 w-3 mr-1" />
                            添加条件
                        </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="start" className="w-48">
                        <DropdownMenuItem onClick={() => handleAddCondition()}>
                            <span className="font-medium">内置条件</span>
                        </DropdownMenuItem>
                        {availablePlugins.length > 0 && (
                            <>
                                <DropdownMenuSeparator />
                                <DropdownMenuLabel className="text-xs">插件条件</DropdownMenuLabel>
                                {availablePlugins.map(plugin => (
                                    <DropdownMenuItem
                                        key={plugin.id}
                                        onClick={() => handleAddCondition(plugin.id)}
                                        className="flex items-start gap-2"
                                    >
                                        <Puzzle className="h-3 w-3 mt-0.5 text-gray-500" />
                                        <div className="flex-1">
                                            <div className="font-medium text-sm">{plugin.name}</div>
                                            {plugin.description && (
                                                <div className="text-xs text-gray-500">{plugin.description}</div>
                                            )}
                                        </div>
                                    </DropdownMenuItem>
                                ))}
                            </>
                        )}
                    </DropdownMenuContent>
                </DropdownMenu>
                <Button
                    onClick={handleAddGroup}
                    variant="ghost"
                    size="sm"
                    className="h-7 text-xs bg-white hover:bg-gray-50 border border-gray-200"
                >
                    <Plus className="h-3 w-3 mr-1" />
                    添加条件组
                </Button>
            </div>
        </div>
    )
}