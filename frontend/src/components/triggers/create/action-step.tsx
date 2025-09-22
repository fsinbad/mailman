'use client'

import React, { useState, useCallback } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
    Zap,
    Play,
    CheckCircle,
    XCircle,
    AlertCircle,
    Eye,
    X
} from 'lucide-react'
import { ActionSection } from '@/components/filter-action-trigger/action-section'

interface ActionStepProps {
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

export default function ActionStep({ data, onDataChange, onNext, onPrevious }: ActionStepProps) {
    const [isTestingActions, setIsTestingActions] = useState(false)
    const [testResult, setTestResult] = useState<TestResult | null>(null)
    const [showDataPreview, setShowDataPreview] = useState(false)

    const actions = data.actions || []
    const emailData = data.emailData?.sampleData || {}
    const expressions = data.expressions || []

    // 更新动作
    const handleActionsChange = useCallback((newActions: any[]) => {
        onDataChange('actions', newActions)
        // 清除之前的测试结果
        setTestResult(null)
    }, [onDataChange])

    // 测试动作
    const testActions = useCallback(async () => {
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
                message: '请先在第二步配置过滤表达式'
            })
            return
        }

        if (actions.length === 0) {
            setTestResult({
                passed: false,
                message: '请先添加动作配置'
            })
            return
        }

        setIsTestingActions(true)
        setTestResult(null)

        try {
            const response = await fetch('/api/actions/test', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    actions,
                    testData: emailData,
                    expressions
                })
            })

            if (response.ok) {
                const result = await response.json()
                setTestResult({
                    passed: result.passed,
                    message: result.passed ? '动作测试通过' : '动作测试失败',
                    details: result.details
                })
            } else {
                setTestResult({
                    passed: false,
                    message: '动作测试失败'
                })
            }
        } catch (error) {
            console.error('测试动作失败:', error)
            setTestResult({
                passed: false,
                message: '测试过程中发生错误'
            })
        } finally {
            setIsTestingActions(false)
        }
    }, [actions, emailData, expressions])

    // 检查是否可以继续下一步
    const canProceed = () => {
        return actions.length > 0 && testResult?.passed
    }

    return (
        <>
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <Zap className="h-5 w-5" />
                        第三步：配置动作行为
                    </CardTitle>
                    <p className="text-sm text-gray-600 dark:text-gray-400">
                        配置触发器执行的动作并测试执行效果
                    </p>
                </CardHeader>
                <CardContent className="relative space-y-6">
                    {/* 悬浮数据预览按钮 */}
                    <div className="fixed bottom-6 right-6 z-10">
                        <button
                            onClick={() => setShowDataPreview(true)}
                            className="bg-blue-500 hover:bg-blue-600 text-white p-3 rounded-full shadow-lg"
                            title="预览邮件数据"
                        >
                            <Eye className="h-5 w-5" />
                        </button>
                    </div>

                    {/* 前置条件检查 */}
                    <div className="space-y-2">
                        <h4 className="font-medium">前置条件检查</h4>
                        <div className="grid grid-cols-2 gap-4">
                            <div className={`p-3 rounded-lg ${emailData && Object.keys(emailData).length > 0
                                ? 'bg-green-50 dark:bg-green-900/20'
                                : 'bg-red-50 dark:bg-red-900/20'
                                }`}>
                                <div className={`flex items-center gap-2 ${emailData && Object.keys(emailData).length > 0
                                    ? 'text-green-700 dark:text-green-300'
                                    : 'text-red-700 dark:text-red-300'
                                    }`}>
                                    {emailData && Object.keys(emailData).length > 0 ? (
                                        <CheckCircle className="h-4 w-4" />
                                    ) : (
                                        <XCircle className="h-4 w-4" />
                                    )}
                                    <span className="text-sm font-medium">邮件数据</span>
                                </div>
                                <p className="text-xs mt-1 opacity-80">
                                    {emailData && Object.keys(emailData).length > 0 ? '已配置' : '未配置'}
                                </p>
                            </div>

                            <div className={`p-3 rounded-lg ${expressions.length > 0
                                ? 'bg-green-50 dark:bg-green-900/20'
                                : 'bg-red-50 dark:bg-red-900/20'
                                }`}>
                                <div className={`flex items-center gap-2 ${expressions.length > 0
                                    ? 'text-green-700 dark:text-green-300'
                                    : 'text-red-700 dark:text-red-300'
                                    }`}>
                                    {expressions.length > 0 ? (
                                        <CheckCircle className="h-4 w-4" />
                                    ) : (
                                        <XCircle className="h-4 w-4" />
                                    )}
                                    <span className="text-sm font-medium">过滤表达式</span>
                                </div>
                                <p className="text-xs mt-1 opacity-80">
                                    {expressions.length > 0 ? `${expressions.length} 个条件` : '未配置'}
                                </p>
                            </div>
                        </div>
                    </div>

                    {/* 动作配置 */}
                    <div className="space-y-4">
                        <ActionSection
                            actions={actions}
                            onChange={handleActionsChange}
                            testData={emailData}
                        />
                    </div>

                    {/* 测试按钮和结果 */}
                    <div className="space-y-4">
                        <div className="flex items-center gap-4">
                            <Button
                                onClick={testActions}
                                disabled={isTestingActions || actions.length === 0}
                                variant="outline"
                                size="sm"
                            >
                                <Play className={`h-4 w-4 mr-2 ${isTestingActions ? 'animate-spin' : ''}`} />
                                {isTestingActions ? '测试中...' : '测试动作'}
                            </Button>

                            {actions.length > 0 && (
                                <Badge variant="secondary" className="text-xs">
                                    {actions.length} 个动作
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
                    {actions.length === 0 && (
                        <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                            <div className="flex items-center gap-2 text-blue-700 dark:text-blue-300">
                                <AlertCircle className="h-4 w-4" />
                                <span className="font-medium">配置动作</span>
                            </div>
                            <p className="text-sm text-blue-600 dark:text-blue-400 mt-1">
                                请配置当邮件匹配过滤条件时要执行的动作。您可以添加多个动作，它们会按顺序执行。
                            </p>
                        </div>
                    )}

                    {actions.length > 0 && !testResult && (
                        <div className="p-4 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg">
                            <div className="flex items-center gap-2 text-yellow-700 dark:text-yellow-300">
                                <AlertCircle className="h-4 w-4" />
                                <span className="font-medium">请测试动作</span>
                            </div>
                            <p className="text-sm text-yellow-600 dark:text-yellow-400 mt-1">
                                配置完成后，请点击"测试动作"验证执行效果。
                            </p>
                        </div>
                    )}

                </CardContent>
            </Card>

            {/* 数据预览模态框 */}
            {showDataPreview && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onClick={() => setShowDataPreview(false)}>
                    <div className="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[80vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
                        <div className="flex justify-between items-center mb-4">
                            <h3 className="text-lg font-semibold">邮件数据预览</h3>
                            <button
                                onClick={() => setShowDataPreview(false)}
                                className="text-gray-500 hover:text-gray-700"
                            >
                                <X className="h-5 w-5" />
                            </button>
                        </div>
                        <div className="space-y-4">
                            {emailData && Object.keys(emailData).length > 0 ? (
                                <div className="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
                                    <h4 className="font-medium mb-2">邮件信息</h4>
                                    <div className="space-y-2 text-sm">
                                        <div><strong>主题:</strong> {emailData.subject || '无'}</div>
                                        <div><strong>发件人:</strong> {emailData.from || '无'}</div>
                                        <div><strong>收件人:</strong> {emailData.to || '无'}</div>
                                        <div><strong>日期:</strong> {emailData.date || '无'}</div>
                                        <div><strong>内容:</strong> {emailData.textContent ? emailData.textContent.substring(0, 200) + '...' : '无'}</div>
                                    </div>
                                </div>
                            ) : (
                                <div className="text-center py-8 text-gray-500">
                                    <p>暂无邮件数据</p>
                                    <p className="text-sm mt-2">请先在第一步选择或输入邮件数据</p>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            )}
        </>
    )
}