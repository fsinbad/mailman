'use client'

import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { CheckCircle, XCircle, X, Clock, AlertCircle, Settings } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'

interface ActionResultProps {
    result: {
        results: Array<{
            pluginId: string
            pluginName: string
            success: boolean
            result?: any
            error?: string
            executionTime?: number
        }>
        totalTime?: number
        error?: string
    }
    onClose: () => void
}

export function ActionResult({ result, onClose }: ActionResultProps) {
    const { results, totalTime, error } = result
    const hasError = error || results.some(r => !r.success)
    const allSuccess = !error && results.every(r => r.success)

    return (
        <Card className={cn(
            'relative mt-4 p-4',
            error ? 'border-red-500 bg-red-50 dark:bg-red-950' :
                allSuccess ? 'border-green-500 bg-green-50 dark:bg-green-950' :
                    'border-yellow-500 bg-yellow-50 dark:bg-yellow-950'
        )}>
            {/* 关闭按钮 */}
            <Button
                size="sm"
                variant="ghost"
                className="absolute right-2 top-2"
                onClick={onClose}
            >
                <X className="h-4 w-4" />
            </Button>

            {/* 结果标题 */}
            <div className="flex items-center gap-3 mb-4">
                {error ? (
                    <>
                        <XCircle className="h-6 w-6 text-red-600" />
                        <h3 className="text-lg font-semibold text-red-800 dark:text-red-200">
                            执行失败
                        </h3>
                    </>
                ) : allSuccess ? (
                    <>
                        <CheckCircle className="h-6 w-6 text-green-600" />
                        <h3 className="text-lg font-semibold text-green-800 dark:text-green-200">
                            执行成功
                        </h3>
                    </>
                ) : (
                    <>
                        <AlertCircle className="h-6 w-6 text-yellow-600" />
                        <h3 className="text-lg font-semibold text-yellow-800 dark:text-yellow-200">
                            部分成功
                        </h3>
                    </>
                )}
                {totalTime && (
                    <div className="flex items-center gap-1 text-sm text-gray-600">
                        <Clock className="h-4 w-4" />
                        <span>{totalTime}ms</span>
                    </div>
                )}
            </div>

            {/* 总体错误信息 */}
            {error && (
                <div className="mb-4 p-3 bg-red-100 dark:bg-red-900 rounded-lg">
                    <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
                </div>
            )}

            {/* 执行统计 */}
            <div className="flex items-center gap-2 mb-4">
                <Badge variant="outline" className="text-gray-600">
                    总数: {results.length}
                </Badge>
                <Badge variant="outline" className="text-green-600">
                    成功: {results.filter(r => r.success).length}
                </Badge>
                <Badge variant="outline" className="text-red-600">
                    失败: {results.filter(r => !r.success).length}
                </Badge>
            </div>

            {/* 各个动作的执行结果 */}
            <div className="space-y-3">
                {results.map((actionResult, index) => (
                    <div
                        key={`${actionResult.pluginId}-${index}`}
                        className={cn(
                            'p-3 rounded-lg border',
                            actionResult.success
                                ? 'border-green-200 bg-green-50 dark:bg-green-900/20'
                                : 'border-red-200 bg-red-50 dark:bg-red-900/20'
                        )}
                    >
                        <div className="flex items-center justify-between mb-2">
                            <div className="flex items-center gap-2">
                                <Settings className="h-4 w-4 text-gray-500" />
                                <span className="font-medium">{actionResult.pluginName}</span>
                                <Badge variant="outline" className="text-xs">
                                    {actionResult.pluginId}
                                </Badge>
                            </div>
                            <div className="flex items-center gap-2">
                                {actionResult.success ? (
                                    <CheckCircle className="h-4 w-4 text-green-600" />
                                ) : (
                                    <XCircle className="h-4 w-4 text-red-600" />
                                )}
                                {actionResult.executionTime && (
                                    <Badge variant="outline" className="text-xs">
                                        {actionResult.executionTime}ms
                                    </Badge>
                                )}
                            </div>
                        </div>

                        {/* 错误信息 */}
                        {actionResult.error && (
                            <div className="mb-2 p-2 bg-red-100 dark:bg-red-900 rounded text-sm">
                                <p className="text-red-800 dark:text-red-200">{actionResult.error}</p>
                            </div>
                        )}

                        {/* 执行结果 */}
                        {actionResult.result && (
                            <div className="bg-gray-50 dark:bg-gray-800 rounded p-2">
                                <Label className="text-xs text-gray-500 block mb-1">执行结果:</Label>
                                <pre className="text-xs text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
                                    {typeof actionResult.result === 'string'
                                        ? actionResult.result
                                        : JSON.stringify(actionResult.result, null, 2)}
                                </pre>
                            </div>
                        )}
                    </div>
                ))}
            </div>

            {/* 如果没有结果 */}
            {results.length === 0 && !error && (
                <div className="text-center py-8 text-gray-500">
                    <Settings className="h-12 w-12 mx-auto mb-2 text-gray-400" />
                    <p>没有动作执行结果</p>
                </div>
            )}
        </Card>
    )
}