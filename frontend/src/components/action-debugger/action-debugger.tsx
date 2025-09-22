'use client'

import { useState, useCallback } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Plus, Play, Code, Layers, Settings, TestTube } from 'lucide-react'
import { ActionGroup } from './action-group'
import { ActionResult } from './action-result'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { apiClient } from '@/lib/api-client'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

interface Action {
    id?: string
    pluginId: string
    pluginName: string
    config: Record<string, any>
    enabled: boolean
    executionOrder: number
}

interface ActionDebuggerProps {
    actions: Action[]
    onChange: (actions: Action[]) => void
}

export function ActionDebugger({ actions, onChange }: ActionDebuggerProps) {
    const [testData, setTestData] = useState<Record<string, any>>({
        event: {
            type: 'email_received',
            id: 'test-event-123',
            timestamp: new Date().toISOString(),
            data: {
                from: 'test@example.com',
                to: 'user@example.com',
                subject: '测试邮件',
                body: '这是一封测试邮件，用于调试动作插件',
                messageId: 'test-message-123',
                size: 1024,
                attachments: []
            }
        },
        context: {
            userId: 'user-123',
            accountId: 'account-456',
            triggerId: 'trigger-789'
        }
    })
    const [testDataInput, setTestDataInput] = useState(JSON.stringify({
        event: {
            type: 'email_received',
            id: 'test-event-123',
            timestamp: new Date().toISOString(),
            data: {
                from: 'test@example.com',
                to: 'user@example.com',
                subject: '测试邮件',
                body: '这是一封测试邮件，用于调试动作插件',
                messageId: 'test-message-123',
                size: 1024,
                attachments: []
            }
        },
        context: {
            userId: 'user-123',
            accountId: 'account-456',
            triggerId: 'trigger-789'
        }
    }, null, 2))
    const [executionResult, setExecutionResult] = useState<{
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
    } | null>(null)
    const [isExecuting, setIsExecuting] = useState(false)
    const [availablePlugins, setAvailablePlugins] = useState<Array<{
        id: string
        name: string
        description: string
        requiredConfig: string[]
        supportedEventTypes: string[]
    }>>([])

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

    // 获取可用的动作插件
    const fetchAvailablePlugins = useCallback(async () => {
        try {
            const response = await apiClient.get('/plugins')
            // 过滤出动作插件
            const actionPlugins = (response || []).filter((plugin: any) => plugin.type === 'action')
            const formattedPlugins = actionPlugins.map((plugin: any) => ({
                id: plugin.id,
                name: plugin.name,
                description: plugin.description || '',
                requiredConfig: Object.keys(plugin.config_schema || {}),
                supportedEventTypes: ['email_received'] // 默认支持邮件接收事件
            }))
            setAvailablePlugins(formattedPlugins)
        } catch (error) {
            console.error('获取动作插件失败:', error)
        }
    }, [])

    // 添加新动作
    const handleAddAction = (pluginId: string, pluginName: string) => {
        const newAction: Action = {
            id: Date.now().toString(),
            pluginId,
            pluginName,
            config: {},
            enabled: true,
            executionOrder: actions.length + 1
        }
        onChange([...actions, newAction])
    }

    // 执行动作
    const handleExecute = async () => {
        try {
            setIsExecuting(true)
            setExecutionResult(null)

            // 过滤启用的动作并按执行顺序排序
            const enabledActions = actions
                .filter(action => action.enabled)
                .sort((a, b) => a.executionOrder - b.executionOrder)

            if (enabledActions.length === 0) {
                setExecutionResult({
                    results: [],
                    error: '没有启用的动作可执行'
                })
                return
            }

            // 构建执行请求
            const executeRequest = {
                actions: enabledActions.map(action => ({
                    plugin_id: action.pluginId,
                    config: action.config
                })),
                data: testData
            }

            // 调用后端API执行动作
            const response = await apiClient.post('/triggers/execute-actions', executeRequest)

            setExecutionResult({
                results: response.results || [],
                totalTime: response.totalTime,
                error: response.error
            })
        } catch (error: any) {
            setExecutionResult({
                results: [],
                error: error.message || '执行动作时发生错误'
            })
        } finally {
            setIsExecuting(false)
        }
    }

    // 更新动作
    const handleUpdateAction = (index: number, updatedAction: Action) => {
        const newActions = [...actions]
        newActions[index] = updatedAction
        onChange(newActions)
    }

    // 删除动作
    const handleDeleteAction = (index: number) => {
        const newActions = actions.filter((_, i) => i !== index)
        onChange(newActions)
    }

    // 移动动作顺序
    const handleMoveAction = (index: number, direction: 'up' | 'down') => {
        const newActions = [...actions]
        const targetIndex = direction === 'up' ? index - 1 : index + 1

        if (targetIndex >= 0 && targetIndex < actions.length) {
            [newActions[index], newActions[targetIndex]] = [newActions[targetIndex], newActions[index]]

            // 更新执行顺序
            newActions.forEach((action, i) => {
                action.executionOrder = i + 1
            })

            onChange(newActions)
        }
    }

    // 初始化时获取可用插件
    useState(() => {
        fetchAvailablePlugins()
    })

    return (
        <div className="h-full flex flex-col bg-gray-50 dark:bg-gray-900">
            {/* 头部 */}
            <div className="bg-white dark:bg-gray-800 border-b px-6 py-4">
                <div className="flex items-center justify-between">
                    <div>
                        <h2 className="text-2xl font-bold flex items-center gap-2">
                            <Settings className="h-6 w-6 text-green-500" />
                            动作调试器
                        </h2>
                        <p className="text-sm text-gray-500 mt-1">
                            配置和测试动作插件的执行效果
                        </p>
                    </div>
                    <div className="flex gap-2">
                        <Button onClick={fetchAvailablePlugins} variant="outline">
                            <Plus className="h-4 w-4 mr-2" />
                            刷新插件
                        </Button>
                        <Button
                            onClick={handleExecute}
                            disabled={isExecuting || actions.length === 0}
                            className="bg-green-500 hover:bg-green-600"
                        >
                            <Play className="h-4 w-4 mr-2" />
                            {isExecuting ? '执行中...' : '执行动作'}
                        </Button>
                    </div>
                </div>
            </div>

            {/* 主内容区 */}
            <div className="flex-1 overflow-hidden">
                <div className="h-full grid grid-cols-1 lg:grid-cols-2 gap-6 p-6">
                    {/* 左侧 - 动作配置 */}
                    <div className="overflow-y-auto">
                        <Card className="p-6">
                            <div className="flex items-center gap-2 mb-4">
                                <Layers className="h-5 w-5 text-gray-500" />
                                <h3 className="text-lg font-semibold">动作配置</h3>
                            </div>

                            <div className="space-y-4">
                                {actions.length === 0 ? (
                                    <div className="text-center py-12 bg-gray-50 dark:bg-gray-800 rounded-lg border-2 border-dashed border-gray-300 dark:border-gray-600">
                                        <Settings className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                                        <p className="text-gray-500 mb-4">还没有添加任何动作</p>
                                        <div className="space-y-2">
                                            {availablePlugins.map((plugin) => (
                                                <Button
                                                    key={plugin.id}
                                                    onClick={() => handleAddAction(plugin.id, plugin.name)}
                                                    variant="outline"
                                                    className="mx-1"
                                                >
                                                    <Plus className="h-4 w-4 mr-2" />
                                                    {plugin.name}
                                                </Button>
                                            ))}
                                        </div>
                                    </div>
                                ) : (
                                    <>
                                        {/* 可用插件选择 */}
                                        <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
                                            <h4 className="text-sm font-medium mb-2">添加动作插件:</h4>
                                            <div className="flex flex-wrap gap-2">
                                                {availablePlugins.map((plugin) => (
                                                    <Button
                                                        key={plugin.id}
                                                        onClick={() => handleAddAction(plugin.id, plugin.name)}
                                                        variant="outline"
                                                        size="sm"
                                                    >
                                                        <Plus className="h-3 w-3 mr-1" />
                                                        {plugin.name}
                                                    </Button>
                                                ))}
                                            </div>
                                        </div>

                                        {/* 动作列表 */}
                                        {actions.map((action, index) => (
                                            <div key={action.id || index} className="relative">
                                                <ActionGroup
                                                    action={action}
                                                    index={index}
                                                    totalCount={actions.length}
                                                    availablePlugins={availablePlugins}
                                                    onChange={(updated: Action) => handleUpdateAction(index, updated)}
                                                    onDelete={() => handleDeleteAction(index)}
                                                    onMove={(direction: 'up' | 'down') => handleMoveAction(index, direction)}
                                                />
                                            </div>
                                        ))}
                                    </>
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
                                        placeholder='{"event": {...}, "context": {...}}'
                                        className="font-mono text-sm h-48"
                                    />
                                    <p className="text-xs text-gray-500 mt-2">
                                        提示：输入包含 event 和 context 的 JSON 测试数据
                                    </p>
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

                        {/* 执行结果 */}
                        {executionResult && (
                            <ActionResult
                                result={executionResult}
                                onClose={() => setExecutionResult(null)}
                            />
                        )}
                    </div>
                </div>
            </div>
        </div>
    )
}