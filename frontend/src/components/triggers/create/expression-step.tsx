'use client'

import React, { useState, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
    Filter,
    Play,
    CheckCircle,
    XCircle,
    AlertCircle,
    Eye,
    X
} from 'lucide-react'
import { FilterSection } from '@/components/filter-action-trigger/filter-section'

interface ExpressionStepProps {
    data: any
    onDataChange: (key: string, value: any) => void
    onNext: () => void
    onPrevious: () => void
}

interface TestResult {
    passed: boolean
    message: string
    details?: any
}

export default function ExpressionStep({ data, onDataChange, onNext, onPrevious }: ExpressionStepProps) {
    const [isTestingExpression, setIsTestingExpression] = useState(false)
    const [testResult, setTestResult] = useState<TestResult | null>(null)
    const [showDataPreview, setShowDataPreview] = useState(false)

    const expressions = data.expressions || []
    const emailData = data.emailData?.sampleData || {}

    // 更新表达式
    const handleExpressionsChange = useCallback((newExpressions: any[]) => {
        onDataChange('expressions', newExpressions)
        // 清除之前的测试结果
        setTestResult(null)
    }, [onDataChange])

    // 测试表达式
    const testExpressions = useCallback(async () => {
        if (!emailData || Object.keys(emailData).length === 0) {
            setTestResult({
                passed: false,
                message: '请先在第一步选择邮件数据'
            })
            return
        }

        if (expressions.length === 0) {
            setTestResult({
                passed: false,
                message: '请先添加过滤表达式'
            })
            return
        }

        setIsTestingExpression(true)
        setTestResult(null)

        try {
            const response = await fetch('/api/expressions/test', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    expressions,
                    testData: emailData
                })
            })

            if (response.ok) {
                const result = await response.json()
                setTestResult({
                    passed: result.passed,
                    message: result.passed ? '表达式测试通过' : '表达式测试失败',
                    details: result.details
                })
            } else {
                setTestResult({
                    passed: false,
                    message: '表达式测试失败'
                })
            }
        } catch (error) {
            console.error('测试表达式失败:', error)
            setTestResult({
                passed: false,
                message: '测试过程中发生错误'
            })
        } finally {
            setIsTestingExpression(false)
        }
    }, [expressions, emailData])

    // 检查是否可以继续下一步
    const canProceed = () => {
        return expressions.length > 0 && testResult?.passed
    }

    return (
        <div className="relative">
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <Filter className="h-5 w-5" />
                        第二步：配置过滤表达式
                    </CardTitle>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                        配置邮件过滤条件并测试匹配结果
                    </p>
                </CardHeader>
                <CardContent className="space-y-6">

                    {/* 过滤器配置 */}
                    <div className="space-y-4">
                        <FilterSection
                            filters={expressions}
                            onChange={handleExpressionsChange}
                            testData={emailData}
                        />
                    </div>

                    {/* 测试按钮和结果 */}
                    <div className="space-y-4">
                        <div className="flex items-center gap-4">
                            <Button
                                onClick={testExpressions}
                                disabled={isTestingExpression || expressions.length === 0}
                                variant="outline"
                                size="sm"
                            >
                                <Play className={`h-4 w-4 mr-2 ${isTestingExpression ? 'animate-spin' : ''}`} />
                                {isTestingExpression ? '测试中...' : '测试表达式'}
                            </Button>

                            {expressions.length > 0 && (
                                <Badge variant="secondary" className="text-xs">
                                    {expressions.length} 个过滤条件
                                </Badge>
                            )}
                        </div>

                        {/* 测试结果 */}
                        {testResult && (
                            <div className={`p-4 rounded-lg ${testResult.passed
                                ? 'bg-green-50 dark:bg-green-900/20'
                                : 'bg-red-50 dark:bg-red-900/20'
                                }`}>
                                <div className={`flex items-center gap-2 ${testResult.passed
                                    ? 'text-green-700 dark:text-green-300'
                                    : 'text-red-700 dark:text-red-300'
                                    }`}>
                                    {testResult.passed ? (
                                        <CheckCircle className="h-4 w-4" />
                                    ) : (
                                        <XCircle className="h-4 w-4" />
                                    )}
                                    <span className="font-medium">{testResult.message}</span>
                                </div>
                                {testResult.details && (
                                    <div className="mt-2 text-sm opacity-80">
                                        <pre className="whitespace-pre-wrap">
                                            {JSON.stringify(testResult.details, null, 2)}
                                        </pre>
                                    </div>
                                )}
                            </div>
                        )}
                    </div>

                    {/* 提示信息 */}
                    {expressions.length === 0 && (
                        <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                            <div className="flex items-center gap-2 text-blue-700 dark:text-blue-300">
                                <AlertCircle className="h-4 w-4" />
                                <span className="font-medium">提示</span>
                            </div>
                            <p className="text-sm text-blue-600 dark:text-blue-400 mt-1">
                                点击"添加过滤器"开始配置邮件过滤条件。您可以基于邮件的主题、发件人、收件人、内容等属性创建过滤规则。
                            </p>
                        </div>
                    )}

                    {expressions.length > 0 && !testResult && (
                        <div className="p-4 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg">
                            <div className="flex items-center gap-2 text-yellow-700 dark:text-yellow-300">
                                <AlertCircle className="h-4 w-4" />
                                <span className="font-medium">请测试表达式</span>
                            </div>
                            <p className="text-sm text-yellow-600 dark:text-yellow-400 mt-1">
                                配置完成后，请点击"测试表达式"验证过滤条件是否正确。
                            </p>
                        </div>
                    )}

                </CardContent>
            </Card>

            {/* 悬浮数据预览按钮 */}
            {emailData && Object.keys(emailData).length > 0 && (
                <Button
                    onClick={() => setShowDataPreview(true)}
                    className="fixed bottom-6 right-6 z-50 rounded-full shadow-lg"
                    size="sm"
                >
                    <Eye className="h-4 w-4 mr-2" />
                    数据预览
                </Button>
            )}

            {/* 悬浮数据预览模态框 */}
            {showDataPreview && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-2xl w-full mx-4 max-h-96 overflow-y-auto">
                        <div className="flex items-center justify-between mb-4">
                            <h3 className="text-lg font-semibold">邮件数据预览</h3>
                            <Button
                                onClick={() => setShowDataPreview(false)}
                                variant="ghost"
                                size="sm"
                            >
                                <X className="h-4 w-4" />
                            </Button>
                        </div>
                        <div className="space-y-4">
                            <div className="flex items-center gap-2">
                                <Badge variant="outline" className="text-xs">
                                    {data.emailData?.source === 'api' ? '从API获取' : '手动输入'}
                                </Badge>
                            </div>
                            <div className="p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                                <pre className="text-sm whitespace-pre-wrap">
                                    {JSON.stringify(emailData, null, 2)}
                                </pre>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    )
}