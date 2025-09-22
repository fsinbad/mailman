'use client'

import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { CheckCircle, XCircle, X } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ExpressionResultProps {
    result: boolean
    details: any
    error?: string
    onClose: () => void
}

export function ExpressionResult({ result, details, error, onClose }: ExpressionResultProps) {
    return (
        <Card className={cn(
            'relative mt-4 p-4',
            error ? 'border-red-500 bg-red-50 dark:bg-red-950' :
                result ? 'border-green-500 bg-green-50 dark:bg-green-950' :
                    'border-red-500 bg-red-50 dark:bg-red-950'
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
            <div className="flex items-center gap-3 mb-3">
                {error ? (
                    <>
                        <XCircle className="h-6 w-6 text-red-600" />
                        <h3 className="text-lg font-semibold text-red-800 dark:text-red-200">
                            评估失败
                        </h3>
                    </>
                ) : result ? (
                    <>
                        <CheckCircle className="h-6 w-6 text-green-600" />
                        <h3 className="text-lg font-semibold text-green-800 dark:text-green-200">
                            条件满足
                        </h3>
                    </>
                ) : (
                    <>
                        <XCircle className="h-6 w-6 text-red-600" />
                        <h3 className="text-lg font-semibold text-red-800 dark:text-red-200">
                            条件不满足
                        </h3>
                    </>
                )}
            </div>

            {/* 错误信息 */}
            {error && (
                <div className="mb-3">
                    <p className="text-red-700 dark:text-red-300">{error}</p>
                </div>
            )}

            {/* 详细信息 */}
            {details && Object.keys(details).length > 0 && (
                <div className="space-y-2">
                    <h4 className="text-sm font-semibold text-gray-700 dark:text-gray-300">
                        评估详情：
                    </h4>
                    <pre className="p-3 bg-white dark:bg-gray-900 rounded border text-xs overflow-auto">
                        {JSON.stringify(details, null, 2)}
                    </pre>
                </div>
            )}
        </Card>
    )
}