'use client'

import React from 'react'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CheckCircle, XCircle, Clock, ArrowRight, Eye, Code, Zap } from 'lucide-react'

interface Action {
    id: string
    pluginId: string
    pluginName: string
    config: Record<string, any>
    enabled: boolean
    executionOrder: number
}

interface ExecutionResult {
    results: Array<{
        success: boolean
        result?: any
        error?: string
        duration_ms: number
        plugin_info?: any
    }>
    summary: {
        total_actions: number
        successful_actions: number
        failed_actions: number
        total_duration: number
    }
}

interface ActionResultsPanelProps {
    actions: Action[]
    selectedActionId?: string
    executionResult: ExecutionResult | null
    isExecuting: boolean
    testData: Record<string, any>
}

export function ActionResultsPanel({
    actions,
    selectedActionId,
    executionResult,
    isExecuting,
    testData
}: ActionResultsPanelProps) {
    const selectedActionIndex = selectedActionId 
        ? actions.findIndex(a => a.id === selectedActionId)
        : -1

    const selectedActionResult = executionResult && selectedActionIndex >= 0 
        ? executionResult.results[selectedActionIndex]
        : null

    const renderExecutionFlow = () => {
        if (!executionResult) return null

        return (
            <div className="space-y-4">
                <h4 className="font-medium flex items-center gap-2">
                    <Zap className="h-4 w-4" />
                    执行流程
                </h4>
                
                {/* 执行摘要 */}
                <Card className="p-4">
                    <div className="grid grid-cols-4 gap-4 text-center">
                        <div>
                            <div className="text-2xl font-bold text-blue-600">
                                {executionResult.summary.total_actions}
                            </div>
                            <div className="text-xs text-gray-500">总动作数</div>
                        </div>
                        <div>
                            <div className="text-2xl font-bold text-green-600">
                                {executionResult.summary.successful_actions}
                            </div>
                            <div className="text-xs text-gray-500">成功</div>
                        </div>
                        <div>
                            <div className="text-2xl font-bold text-red-600">
                                {executionResult.summary.failed_actions}
                            </div>
                            <div className="text-xs text-gray-500">失败</div>
                        </div>
                        <div>
                            <div className="text-2xl font-bold text-purple-600">
                                {executionResult.summary.total_duration}ms
                            </div>
                            <div className="text-xs text-gray-500">总耗时</div>
                        </div>
                    </div>
                </Card>

                {/* 执行链 */}
                <div className="space-y-3">
                    {actions.filter(a => a.enabled).map((action, index) => {
                        const result = executionResult.results[index]
                        const isSelected = action.id === selectedActionId
                        
                        return (
                            <div key={action.id} className="flex items-center gap-3">
                                <Card className={`
                                    flex-1 p-3 cursor-pointer transition-all
                                    ${isSelected ? 'ring-2 ring-blue-500 bg-blue-50' : 'hover:shadow-md'}
                                    ${result?.success ? 'border-green-200' : result?.success === false ? 'border-red-200' : ''}
                                `}>
                                    <div className="flex items-center justify-between">
                                        <div className="flex items-center gap-3">
                                            <Badge variant="outline" className="text-xs">
                                                #{action.executionOrder}
                                            </Badge>
                                            <div>
                                                <h5 className="font-medium text-sm">{action.pluginName}</h5>
                                                <p className="text-xs text-gray-500">
                                                    {getActionSummary(action)}
                                                </p>
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            {result ? (
                                                <>
                                                    {result.success ? (
                                                        <CheckCircle className="h-4 w-4 text-green-500" />
                                                    ) : (
                                                        <XCircle className="h-4 w-4 text-red-500" />
                                                    )}
                                                    <Badge variant="secondary" className="text-xs">
                                                        {result.duration_ms}ms
                                                    </Badge>
                                                </>
                                            ) : isExecuting ? (
                                                <Clock className="h-4 w-4 text-blue-500 animate-spin" />
                                            ) : null}
                                        </div>
                                    </div>
                                    
                                    {result?.error && (
                                        <div className="mt-2 p-2 bg-red-50 rounded text-xs text-red-600">
                                            {result.error}
                                        </div>
                                    )}
                                </Card>
                                
                                {index < actions.filter(a => a.enabled).length - 1 && (
                                    <ArrowRight className="h-4 w-4 text-gray-400" />
                                )}
                            </div>
                        )
                    })}
                </div>
            </div>
        )
    }

    const renderSelectedActionResult = () => {
        if (!selectedActionResult) {
            return (
                <div className="text-center py-12 text-gray-500">
                    <div className="text-4xl mb-4">👁️</div>
                    <p>选择一个动作查看详细结果</p>
                    <p className="text-sm mt-2">执行动作后可以看到每个步骤的输入输出</p>
                </div>
            )
        }

        return (
            <div className="space-y-4">
                <h4 className="font-medium flex items-center gap-2">
                    <Eye className="h-4 w-4" />
                    动作结果详情
                </h4>

                {/* 执行状态 */}
                <Card className="p-4">
                    <div className="flex items-center justify-between mb-3">
                        <h5 className="font-medium">执行状态</h5>
                        {selectedActionResult.success ? (
                            <Badge className="bg-green-100 text-green-700">
                                <CheckCircle className="h-3 w-3 mr-1" />
                                成功
                            </Badge>
                        ) : (
                            <Badge className="bg-red-100 text-red-700">
                                <XCircle className="h-3 w-3 mr-1" />
                                失败
                            </Badge>
                        )}
                    </div>
                    
                    <div className="grid grid-cols-2 gap-4 text-sm">
                        <div>
                            <span className="text-gray-500">执行时间:</span>
                            <span className="ml-2 font-mono">{selectedActionResult.duration_ms}ms</span>
                        </div>
                        <div>
                            <span className="text-gray-500">插件版本:</span>
                            <span className="ml-2">{selectedActionResult.plugin_info?.version || 'N/A'}</span>
                        </div>
                    </div>

                    {selectedActionResult.error && (
                        <div className="mt-3 p-3 bg-red-50 rounded">
                            <h6 className="font-medium text-red-700 mb-1">错误信息</h6>
                            <p className="text-sm text-red-600">{selectedActionResult.error}</p>
                        </div>
                    )}
                </Card>

                {/* 转换结果 */}
                {selectedActionResult.result && (
                    <Card className="p-4">
                        <h5 className="font-medium mb-3">转换结果</h5>
                        
                        {selectedActionResult.result.original_value && selectedActionResult.result.new_value && (
                            <div className="space-y-3 mb-4">
                                <div>
                                    <Label className="text-xs text-gray-500">转换前</Label>
                                    <div className="p-2 bg-gray-50 rounded text-sm font-mono">
                                        {selectedActionResult.result.original_value}
                                    </div>
                                </div>
                                <div className="flex justify-center">
                                    <ArrowRight className="h-4 w-4 text-gray-400" />
                                </div>
                                <div>
                                    <Label className="text-xs text-gray-500">转换后</Label>
                                    <div className="p-2 bg-green-50 rounded text-sm font-mono">
                                        {selectedActionResult.result.new_value}
                                    </div>
                                </div>
                            </div>
                        )}

                        <div className="grid grid-cols-2 gap-4 text-sm">
                            <div>
                                <span className="text-gray-500">目标字段:</span>
                                <span className="ml-2 font-mono">
                                    {selectedActionResult.result.transformed_field || 'N/A'}
                                </span>
                            </div>
                            <div>
                                <span className="text-gray-500">转换类型:</span>
                                <span className="ml-2">
                                    {selectedActionResult.result.transform_type || 'N/A'}
                                </span>
                            </div>
                        </div>
                    </Card>
                )}

                {/* 完整邮件对象 */}
                {selectedActionResult.result?.transformed_email && (
                    <Card className="p-4">
                        <h5 className="font-medium mb-3">完整邮件对象</h5>
                        <pre className="text-xs bg-gray-50 p-3 rounded overflow-x-auto max-h-64">
                            {JSON.stringify(selectedActionResult.result.transformed_email, null, 2)}
                        </pre>
                    </Card>
                )}
            </div>
        )
    }

    const getActionSummary = (action: Action) => {
        const { config } = action
        switch (action.pluginId) {
            case 'email_transform_action':
                const transformType = config.transform_type || 'template'
                const targetField = config.target_field || 'subject'
                return `${transformType} → ${targetField}`
            case 'email_forward_action':
                return `转发到 ${config.to_address || '未配置'}`
            default:
                return '未配置'
        }
    }

    return (
        <div className="p-6">
            <Tabs defaultValue="flow" className="w-full">
                <TabsList className="grid w-full grid-cols-3">
                    <TabsTrigger value="flow" className="flex items-center gap-2">
                        <Zap className="h-4 w-4" />
                        执行流程
                    </TabsTrigger>
                    <TabsTrigger value="detail" className="flex items-center gap-2">
                        <Eye className="h-4 w-4" />
                        详细结果
                    </TabsTrigger>
                    <TabsTrigger value="data" className="flex items-center gap-2">
                        <Code className="h-4 w-4" />
                        原始数据
                    </TabsTrigger>
                </TabsList>

                <TabsContent value="flow" className="mt-6">
                    {isExecuting ? (
                        <div className="text-center py-12">
                            <Clock className="h-8 w-8 text-blue-500 animate-spin mx-auto mb-4" />
                            <p className="text-gray-500">正在执行动作链...</p>
                        </div>
                    ) : executionResult ? (
                        renderExecutionFlow()
                    ) : (
                        <div className="text-center py-12 text-gray-500">
                            <div className="text-4xl mb-4">🚀</div>
                            <p>点击"执行动作"开始测试</p>
                            <p className="text-sm mt-2">动作将按顺序执行，每个动作的输出会传递给下一个动作</p>
                        </div>
                    )}
                </TabsContent>

                <TabsContent value="detail" className="mt-6">
                    {renderSelectedActionResult()}
                </TabsContent>

                <TabsContent value="data" className="mt-6">
                    <div className="space-y-4">
                        <h4 className="font-medium flex items-center gap-2">
                            <Code className="h-4 w-4" />
                            原始数据
                        </h4>
                        
                        <Card className="p-4">
                            <h5 className="font-medium mb-3">测试数据</h5>
                            <pre className="text-xs bg-gray-50 p-3 rounded overflow-x-auto max-h-64">
                                {JSON.stringify(testData, null, 2)}
                            </pre>
                        </Card>

                        {executionResult && (
                            <Card className="p-4">
                                <h5 className="font-medium mb-3">完整执行结果</h5>
                                <pre className="text-xs bg-gray-50 p-3 rounded overflow-x-auto max-h-64">
                                    {JSON.stringify(executionResult, null, 2)}
                                </pre>
                            </Card>
                        )}
                    </div>
                </TabsContent>
            </Tabs>
        </div>
    )
}