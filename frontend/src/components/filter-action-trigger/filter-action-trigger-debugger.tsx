'use client'

import React, { useState, useCallback, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { apiClient } from '@/lib/api-client'
import { FilterSection } from './filter-section'
import { ActionSection } from './action-section'
import { FilterActionResultsPanel } from './filter-action-results-panel'
import { Play, TestTube } from 'lucide-react'

interface FilterActionTriggerDebuggerProps {
    filters: any[]
    actions: any[]
    onFiltersChange: (filters: any[]) => void
    onActionsChange: (actions: any[]) => void
}

interface TriggerExecutionResult {
    filterResults: any[]
    actionResults: any[]
    overallSuccess: boolean
    totalDuration: number
    error?: string
}

export function FilterActionTriggerDebugger({
    filters,
    actions,
    onFiltersChange,
    onActionsChange
}: FilterActionTriggerDebuggerProps) {
    const [testData, setTestData] = useState<Record<string, any>>({
        event: {
            type: 'email_received',
            id: 'test-event-123',
            timestamp: new Date().toISOString(),
            data: {
                from: 'test@example.com',
                to: 'user@example.com',
                subject: '测试邮件主题',
                body: '这是一封测试邮件，用于调试过滤器和动作',
                messageId: 'test-message-123',
                size: 1024,
                attachments: []
            }
        },
        user: {
            role: 'admin',
            level: 5,
            id: 'user-123'
        },
        context: {
            userId: 'user-123',
            accountId: 'account-456',
            triggerId: 'trigger-789'
        }
    })
    const [testDataInput, setTestDataInput] = useState('')
    const [executionResult, setExecutionResult] = useState<TriggerExecutionResult | null>(null)
    const [isExecuting, setIsExecuting] = useState(false)

    // 初始化测试数据输入
    useEffect(() => {
        setTestDataInput(JSON.stringify(testData, null, 2))
    }, [])

    // 解析测试数据
    const handleTestDataChange = useCallback((value: string) => {
        setTestDataInput(value)
        try {
            const parsed = JSON.parse(value)
            setTestData(parsed)
        } catch {
            // 忽略解析错误
        }
    }, [])

    // 转换前端表达式格式为后端期望格式
    const convertExpressionForBackend = (expression: any): any => {
        if (!expression) return null

        const converted = {
            Type: expression.operator || expression.type,
            Conditions: []
        }

        if (expression.conditions && Array.isArray(expression.conditions)) {
            converted.Conditions = expression.conditions.map((condition: any) => {
                if (condition.type === 'group') {
                    // 递归转换嵌套的条件组
                    return convertExpressionForBackend(condition)
                } else if (condition.type === 'plugin') {
                    // 插件条件
                    return {
                        Type: 'plugin',
                        PluginId: condition.pluginId,
                        Fields: condition.fields,
                        Not: condition.not || false
                    }
                } else if (condition.type === 'condition') {
                    // 普通条件
                    return {
                        Type: 'comparison',
                        Field: condition.field,
                        Operator: condition.operator,
                        Value: condition.value
                    }
                }
                return condition
            })
        }

        return converted
    }

    // 执行完整的触发器测试
    const handleExecuteTrigger = async () => {
        try {
            setIsExecuting(true)
            setExecutionResult(null)

            // 如果没有过滤器或动作，直接返回
            if (filters.length === 0 && actions.length === 0) {
                setExecutionResult({
                    filterResults: [],
                    actionResults: [],
                    overallSuccess: true,
                    totalDuration: 0
                })
                return
            }

            const startTime = Date.now()

            // 第一步：执行过滤器
            let filterResults: any[] = []
            let filterPassed = true

            if (filters.length > 0) {
                try {
                    // 构建过滤器表达式
                    const filterExpression = {
                        operator: 'and',
                        conditions: filters
                    }

                    // 转换为后端格式
                    const backendExpression = convertExpressionForBackend(filterExpression)

                    const filterResponse = await apiClient.post('/triggers/evaluate-expression', {
                        expression: backendExpression,
                        data: testData
                    })

                    filterResults = [{
                        success: true,
                        result: filterResponse.result,
                        duration_ms: filterResponse.duration_ms || 0,
                        evaluation: filterResponse
                    }]

                    filterPassed = filterResponse.result === true
                } catch (error: any) {
                    filterResults = [{
                        success: false,
                        error: error.message || '过滤器执行失败',
                        duration_ms: 0
                    }]
                    filterPassed = false
                }
            }

            // 第二步：如果过滤器通过，执行动作
            let actionResults: any[] = []

            if (filterPassed && actions.length > 0) {
                try {
                    // 过滤启用的动作并按执行顺序排序
                    const enabledActions = actions
                        .filter(action => action.enabled)
                        .sort((a, b) => a.executionOrder - b.executionOrder)

                    if (enabledActions.length > 0) {
                        const executeRequest = {
                            actions: enabledActions.map(action => ({
                                plugin_id: action.pluginId,
                                config: action.config
                            })),
                            data: testData
                        }

                        const actionResponse = await apiClient.post('/triggers/execute-actions', executeRequest)
                        actionResults = actionResponse.results || []
                    }
                } catch (error: any) {
                    actionResults = [{
                        success: false,
                        error: error.message || '动作执行失败',
                        duration_ms: 0
                    }]
                }
            }

            const totalDuration = Date.now() - startTime

            setExecutionResult({
                filterResults,
                actionResults,
                overallSuccess: filterPassed && (actionResults.length === 0 || actionResults.every(r => r.success)),
                totalDuration
            })

        } catch (error: any) {
            setExecutionResult({
                filterResults: [],
                actionResults: [],
                overallSuccess: false,
                totalDuration: 0,
                error: error.message || '执行失败'
            })
        } finally {
            setIsExecuting(false)
        }
    }

    return (
        <div className="h-full flex flex-col bg-gray-50 dark:bg-gray-900">
            {/* 头部 */}
            <div className="bg-white dark:bg-gray-800 border-b px-6 py-4">
                <div className="flex items-center justify-between">
                    <div>
                        <h2 className="text-2xl font-bold flex items-center gap-2">
                            <TestTube className="h-6 w-6 text-blue-500" />
                            过滤动作触发器调试器
                        </h2>
                        <p className="text-sm text-gray-500 mt-1">
                            配置和测试完整的邮件触发器：过滤器 + 动作
                        </p>
                    </div>
                    <div className="flex items-center gap-2">
                        <Badge variant="outline" className="text-sm">
                            {filters.length} 个过滤器
                        </Badge>
                        <Badge variant="outline" className="text-sm">
                            {actions.length} 个动作
                        </Badge>
                        <Button
                            onClick={handleExecuteTrigger}
                            disabled={isExecuting || (filters.length === 0 && actions.length === 0)}
                            className="bg-blue-500 hover:bg-blue-600"
                        >
                            <Play className="h-4 w-4 mr-2" />
                            {isExecuting ? '执行中...' : '运行触发器'}
                        </Button>
                    </div>
                </div>
            </div>

            {/* 主要内容区域 */}
            <div className="flex-1 flex overflow-hidden">
                {/* 左侧配置面板 */}
                <div className="w-1/2 border-r bg-white overflow-y-auto">
                    <div className="p-6">
                        <Tabs defaultValue="filters" className="w-full">
                            <TabsList className="grid w-full grid-cols-3">
                                <TabsTrigger value="filters">
                                    🔍 过滤器配置
                                </TabsTrigger>
                                <TabsTrigger value="actions">
                                    ⚡ 动作配置
                                </TabsTrigger>
                                <TabsTrigger value="data">
                                    📝 测试数据
                                </TabsTrigger>
                            </TabsList>

                            <TabsContent value="filters" className="mt-6">
                                <FilterSection
                                    filters={filters}
                                    onChange={onFiltersChange}
                                    testData={testData}
                                />
                            </TabsContent>

                            <TabsContent value="actions" className="mt-6">
                                <ActionSection
                                    actions={actions}
                                    onChange={onActionsChange}
                                    testData={testData}
                                />
                            </TabsContent>

                            <TabsContent value="data" className="mt-6">
                                <div className="space-y-4">
                                    <div className="flex items-center gap-2">
                                        <Label htmlFor="test-data">测试数据 (JSON)</Label>
                                    </div>
                                    <Textarea
                                        id="test-data"
                                        value={testDataInput}
                                        onChange={(e) => handleTestDataChange(e.target.value)}
                                        placeholder='{"email": {...}, "user": {...}, "context": {...}}'
                                        className="font-mono text-sm h-96"
                                    />
                                    <div className="text-xs text-gray-500">
                                        <p>支持的字段：email.subject, from, to, body 等</p>
                                        <p>过滤器先执行，通过后才会执行动作</p>
                                    </div>
                                </div>
                            </TabsContent>
                        </Tabs>
                    </div>
                </div>

                {/* 右侧结果面板 */}
                <div className="w-1/2 bg-white overflow-y-auto">
                    <FilterActionResultsPanel
                        filters={filters}
                        actions={actions}
                        executionResult={executionResult}
                        isExecuting={isExecuting}
                        testData={testData}
                    />
                </div>
            </div>
        </div>
    )
}