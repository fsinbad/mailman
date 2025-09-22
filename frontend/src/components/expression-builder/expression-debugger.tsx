'use client'

import { useState, useCallback } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Plus, Play, Code, Layers, TestTube } from 'lucide-react'
import { ExpressionGroup } from './expression-group'
import { ExpressionResult } from './expression-result'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { apiClient } from '@/lib/api-client'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

interface Expression {
    id?: string
    type: 'group' | 'condition'
    operator?: 'and' | 'or' | 'not'
    field?: string
    value?: any
    conditions?: Expression[]
    not?: boolean
}

interface ExpressionDebuggerProps {
    expressions: Expression[]
    onChange: (expressions: Expression[]) => void
}

export function ExpressionDebugger({ expressions, onChange }: ExpressionDebuggerProps) {
    const [testData, setTestData] = useState<Record<string, any>>({
        email: {
            from: 'test@example.com',
            subject: '测试邮件',
            body: '这是一封测试邮件'
        },
        user: {
            role: 'admin',
            level: 5
        }
    })
    const [testDataInput, setTestDataInput] = useState(JSON.stringify({
        email: {
            from: 'test@example.com',
            subject: '测试邮件',
            body: '这是一封测试邮件'
        },
        user: {
            role: 'admin',
            level: 5
        }
    }, null, 2))
    const [evaluationResult, setEvaluationResult] = useState<{
        result: boolean
        details?: any
        error?: string
    } | null>(null)
    const [isEvaluating, setIsEvaluating] = useState(false)

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

    // 添加根条件组
    const handleAddRootGroup = () => {
        const newGroup: Expression = {
            id: Date.now().toString(),
            type: 'group',
            operator: 'and',
            conditions: []
        }
        onChange([...expressions, newGroup])
    }

    // 评估表达式
    const handleEvaluate = async () => {
        try {
            setIsEvaluating(true)
            setEvaluationResult(null)

            // 将前端的表达式格式转换为后端期望的格式
            const convertExpression = (expr: Expression): any => {
                if (expr.type === 'group') {
                    const group: any = {
                        type: expr.operator,
                        conditions: (expr.conditions || []).map(convertExpression)
                    }

                    // 处理 NOT 操作符
                    if (expr.operator === 'not' && expr.conditions && expr.conditions.length > 0) {
                        return {
                            type: 'not',
                            left: convertExpression(expr.conditions[0])
                        }
                    }

                    return group
                }

                // 处理条件
                if (expr.type === 'condition') {
                    const comparison: any = {
                        type: 'comparison',
                        field: expr.field || '',
                        operator: expr.operator || 'equals',
                        value: expr.value || ''
                    }

                    // 如果条件有 NOT 标记，包装在 NOT 操作符中
                    if (expr.not) {
                        return {
                            type: 'not',
                            left: comparison
                        }
                    }

                    return comparison
                }
                return expr
            }

            // 构建完整的表达式对象
            const expression = expressions.length === 1
                ? convertExpression(expressions[0])
                : convertExpression({
                    type: 'group',
                    operator: 'and',
                    conditions: expressions
                })

            // 调用后端API评估表达式
            const response = await apiClient.post('/triggers/evaluate-expression', {
                expression,
                data: testData
            })

            setEvaluationResult({
                result: response.result,
                details: response.error ? { error: response.error } : {}
            })
        } catch (error: any) {
            setEvaluationResult({
                result: false,
                error: error.message || '评估表达式时发生错误'
            })
        } finally {
            setIsEvaluating(false)
        }
    }

    // 更新表达式
    const handleUpdateExpression = (index: number, updatedExpression: Expression) => {
        const newExpressions = [...expressions]
        newExpressions[index] = updatedExpression
        onChange(newExpressions)
    }

    // 删除表达式
    const handleDeleteExpression = (index: number) => {
        const newExpressions = expressions.filter((_, i) => i !== index)
        onChange(newExpressions)
    }

    return (
        <div className="h-full flex flex-col bg-gray-50 dark:bg-gray-900">
            {/* 头部 */}
            <div className="bg-white dark:bg-gray-800 border-b px-6 py-4">
                <div className="flex items-center justify-between">
                    <div>
                        <h2 className="text-2xl font-bold flex items-center gap-2">
                            <TestTube className="h-6 w-6 text-blue-500" />
                            表达式调试器
                        </h2>
                        <p className="text-sm text-gray-500 mt-1">
                            构建和测试复杂的条件表达式
                        </p>
                    </div>
                    <div className="flex gap-2">
                        <Button onClick={handleAddRootGroup} variant="outline">
                            <Plus className="h-4 w-4 mr-2" />
                            添加条件组
                        </Button>
                        <Button
                            onClick={handleEvaluate}
                            disabled={isEvaluating || expressions.length === 0}
                            className="bg-blue-500 hover:bg-blue-600"
                        >
                            <Play className="h-4 w-4 mr-2" />
                            {isEvaluating ? '评估中...' : '运行测试'}
                        </Button>
                    </div>
                </div>
            </div>

            {/* 主内容区 */}
            <div className="flex-1 overflow-hidden">
                <div className="h-full grid grid-cols-1 lg:grid-cols-2 gap-6 p-6">
                    {/* 左侧 - 表达式构建器 */}
                    <div className="overflow-y-auto">
                        <Card className="p-6">
                            <div className="flex items-center gap-2 mb-4">
                                <Layers className="h-5 w-5 text-gray-500" />
                                <h3 className="text-lg font-semibold">条件表达式</h3>
                            </div>

                            <div className="space-y-4">
                                {expressions.length === 0 ? (
                                    <div className="text-center py-12 bg-gray-50 dark:bg-gray-800 rounded-lg border-2 border-dashed border-gray-300 dark:border-gray-600">
                                        <Layers className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                                        <p className="text-gray-500 mb-4">还没有添加任何表达式</p>
                                        <Button onClick={handleAddRootGroup} variant="outline">
                                            <Plus className="h-4 w-4 mr-2" />
                                            添加第一个条件组
                                        </Button>
                                    </div>
                                ) : (
                                    expressions.map((expression, index) => (
                                        <div key={expression.id || index} className="relative">
                                            <ExpressionGroup
                                                expression={expression}
                                                onChange={(updated: Expression) => handleUpdateExpression(index, updated)}
                                                onDelete={() => handleDeleteExpression(index)}
                                                testData={testData}
                                                isRoot={true}
                                            />
                                        </div>
                                    ))
                                )}
                            </div>
                        </Card>
                    </div>

                    {/* 右侧 - 测试数据和结果 */}
                    <div className="space-y-6 overflow-y-auto">
                        {/* 测试数据输入 */}
                        <Card className="p-6">
                            <div className="flex items-center gap-2 mb-4">
                                <Code className="h-5 w-5 text-gray-500" />
                                <h3 className="text-lg font-semibold">测试数据</h3>
                            </div>

                            <Tabs defaultValue="json" className="w-full">
                                <TabsList className="grid w-full grid-cols-2">
                                    <TabsTrigger value="json">JSON 编辑器</TabsTrigger>
                                    <TabsTrigger value="preview">数据预览</TabsTrigger>
                                </TabsList>

                                <TabsContent value="json" className="mt-4">
                                    <Textarea
                                        value={testDataInput}
                                        onChange={(e) => handleTestDataChange(e.target.value)}
                                        placeholder='{"field1": "value1", "field2": 123}'
                                        className="font-mono text-sm h-48"
                                    />
                                    {testDataInput && testDataInput !== '{}' && (
                                        <p className="text-xs text-gray-500 mt-2">
                                            提示：输入 JSON 格式的测试数据
                                        </p>
                                    )}
                                </TabsContent>

                                <TabsContent value="preview" className="mt-4">
                                    <div className="bg-gray-50 dark:bg-gray-800 rounded-lg p-4 h-48 overflow-auto">
                                        {Object.keys(testData).length > 0 ? (
                                            <pre className="text-sm">
                                                {JSON.stringify(testData, null, 2)}
                                            </pre>
                                        ) : (
                                            <p className="text-gray-500 text-center py-8">
                                                没有测试数据
                                            </p>
                                        )}
                                    </div>
                                </TabsContent>
                            </Tabs>
                        </Card>

                        {/* 评估结果 */}
                        {evaluationResult && (
                            <ExpressionResult
                                result={evaluationResult.result}
                                details={evaluationResult.details}
                                error={evaluationResult.error}
                                onClose={() => setEvaluationResult(null)}
                            />
                        )}
                    </div>
                </div>
            </div>
        </div>
    )
}