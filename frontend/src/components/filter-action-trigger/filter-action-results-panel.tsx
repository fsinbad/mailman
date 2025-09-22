'use client'

import React from 'react'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CheckCircle, XCircle, Clock, ArrowRight, Eye, Code, Zap, Filter, AlertTriangle } from 'lucide-react'

interface FilterActionResultsPanelProps {
    filters: any[]
    actions: any[]
    executionResult: any
    isExecuting: boolean
    testData: Record<string, any>
}

export function FilterActionResultsPanel({
    filters,
    actions,
    executionResult,
    isExecuting,
    testData
}: FilterActionResultsPanelProps) {

    const renderExecutionFlow = () => {
        if (!executionResult) return null

        const { filterResults, actionResults, overallSuccess, totalDuration, error } = executionResult

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
                            <div className={`text-2xl font-bold ${overallSuccess ? 'text-green-600' : 'text-red-600'}`}>
                                {overallSuccess ? '✓' : '✗'}
                            </div>
                            <div className="text-xs text-gray-500">整体结果</div>
                        </div>
                        <div>
                            <div className="text-2xl font-bold text-blue-600">
                                {filters.length}
                            </div>
                            <div className="text-xs text-gray-500">过滤器数</div>
                        </div>
                        <div>
                            <div className="text-2xl font-bold text-green-600">
                                {actions.length}
                            </div>
                            <div className="text-xs text-gray-500">动作数</div>
                        </div>
                        <div>
                            <div className="text-2xl font-bold text-purple-600">
                                {totalDuration}ms
                            </div>
                            <div className="text-xs text-gray-500">总耗时</div>
                        </div>
                    </div>

                    {error && (
                        <div className="mt-4 p-3 bg-red-50 rounded border border-red-200">
                            <div className="flex items-center gap-2 text-red-700 text-sm">
                                <AlertTriangle className="h-4 w-4" />
                                <span className="font-medium">执行错误</span>
                            </div>
                            <p className="text-red-600 text-xs mt-1">{error}</p>
                        </div>
                    )}
                </Card>

                {/* 执行阶段 */}
                <div className="space-y-3">
                    {/* 过滤器阶段 */}
                    <Card className="p-4">
                        <div className="flex items-center justify-between mb-3">
                            <div className="flex items-center gap-2">
                                <Filter className="h-4 w-4 text-blue-500" />
                                <h5 className="font-medium">过滤器阶段</h5>
                                <Badge variant="outline" className="text-xs">
                                    第1步
                                </Badge>
                            </div>
                            {filterResults.length > 0 && (
                                <div className="flex items-center gap-2">
                                    {filterResults[0].success ? (
                                        filterResults[0].result ? (
                                            <CheckCircle className="h-4 w-4 text-green-500" />
                                        ) : (
                                            <XCircle className="h-4 w-4 text-orange-500" />
                                        )
                                    ) : (
                                        <XCircle className="h-4 w-4 text-red-500" />
                                    )}
                                    <Badge variant="secondary" className="text-xs">
                                        {filterResults[0].duration_ms || 0}ms
                                    </Badge>
                                </div>
                            )}
                        </div>

                        {filters.length === 0 ? (
                            <p className="text-sm text-gray-500">未配置过滤器，直接执行动作</p>
                        ) : filterResults.length > 0 ? (
                            <div className="space-y-2">
                                <div className="flex items-center gap-2">
                                    <span className="text-sm text-gray-600">过滤结果:</span>
                                    {filterResults[0].success ? (
                                        filterResults[0].result ? (
                                            <Badge className="bg-green-100 text-green-700">通过</Badge>
                                        ) : (
                                            <Badge className="bg-orange-100 text-orange-700">未通过</Badge>
                                        )
                                    ) : (
                                        <Badge className="bg-red-100 text-red-700">执行失败</Badge>
                                    )}
                                </div>

                                {filterResults[0].error && (
                                    <div className="text-xs text-red-600 bg-red-50 p-2 rounded">
                                        {filterResults[0].error}
                                    </div>
                                )}

                                {filterResults[0].evaluation && (
                                    <div className="text-xs text-gray-500">
                                        详细评估: {JSON.stringify(filterResults[0].evaluation.details || {}, null, 2)}
                                    </div>
                                )}
                            </div>
                        ) : (
                            <p className="text-sm text-gray-500">等待执行...</p>
                        )}
                    </Card>

                    {/* 箭头 */}
                    <div className="flex justify-center">
                        <ArrowRight className="h-5 w-5 text-gray-400" />
                    </div>

                    {/* 动作阶段 */}
                    <Card className="p-4">
                        <div className="flex items-center justify-between mb-3">
                            <div className="flex items-center gap-2">
                                <Zap className="h-4 w-4 text-green-500" />
                                <h5 className="font-medium">动作阶段</h5>
                                <Badge variant="outline" className="text-xs">
                                    第2步
                                </Badge>
                            </div>
                            {actionResults.length > 0 && (
                                <div className="flex items-center gap-2">
                                    {actionResults.every((r: any) => r.success) ? (
                                        <CheckCircle className="h-4 w-4 text-green-500" />
                                    ) : (
                                        <XCircle className="h-4 w-4 text-red-500" />
                                    )}
                                    <Badge variant="secondary" className="text-xs">
                                        {actionResults.reduce((sum: number, r: any) => sum + (r.duration_ms || 0), 0)}ms
                                    </Badge>
                                </div>
                            )}
                        </div>

                        {actions.length === 0 ? (
                            <p className="text-sm text-gray-500">未配置动作</p>
                        ) : actionResults.length > 0 ? (
                            <div className="space-y-2">
                                <div className="grid grid-cols-3 gap-4 text-sm">
                                    <div className="text-center">
                                        <div className="text-lg font-bold text-blue-600">
                                            {actionResults.length}
                                        </div>
                                        <div className="text-xs text-gray-500">总动作</div>
                                    </div>
                                    <div className="text-center">
                                        <div className="text-lg font-bold text-green-600">
                                            {actionResults.filter((r: any) => r.success).length}
                                        </div>
                                        <div className="text-xs text-gray-500">成功</div>
                                    </div>
                                    <div className="text-center">
                                        <div className="text-lg font-bold text-red-600">
                                            {actionResults.filter((r: any) => !r.success).length}
                                        </div>
                                        <div className="text-xs text-gray-500">失败</div>
                                    </div>
                                </div>

                                {/* 每个动作的结果 */}
                                <div className="space-y-1">
                                    {actionResults.map((result: any, index: number) => (
                                        <div key={index} className="flex items-center justify-between p-2 bg-gray-50 rounded text-xs">
                                            <span>动作 {index + 1}</span>
                                            <div className="flex items-center gap-2">
                                                {result.success ? (
                                                    <CheckCircle className="h-3 w-3 text-green-500" />
                                                ) : (
                                                    <XCircle className="h-3 w-3 text-red-500" />
                                                )}
                                                <span>{result.duration_ms || 0}ms</span>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        ) : (
                            <p className="text-sm text-gray-500">
                                {filterResults.length > 0 && !filterResults[0].result
                                    ? '过滤器未通过，跳过动作执行'
                                    : '等待执行...'}
                            </p>
                        )}
                    </Card>
                </div>
            </div>
        )
    }

    return (
        <div className="p-6">
            <Tabs defaultValue="flow" className="w-full">
                <TabsList className="grid w-full grid-cols-2">
                    <TabsTrigger value="flow" className="flex items-center gap-2">
                        <Zap className="h-4 w-4" />
                        执行流程
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
                            <p className="text-gray-500">正在执行触发器...</p>
                        </div>
                    ) : executionResult ? (
                        renderExecutionFlow()
                    ) : (
                        <div className="text-center py-12 text-gray-500">
                            <div className="text-4xl mb-4">🚀</div>
                            <p>点击"运行触发器"开始测试</p>
                            <p className="text-sm mt-2">
                                将依次执行过滤器和动作，展示完整的触发器执行流程
                            </p>
                        </div>
                    )}
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