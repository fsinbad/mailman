'use client'

import React, { useState, useCallback, useEffect } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { apiClient } from '@/lib/api-client'
import { ActionPipeline } from '@/components/action-debugger/action-pipeline'
import { ActionConfigPanel } from '@/components/action-debugger/action-config-panel'
import { Zap } from 'lucide-react'

interface Action {
    id: string
    pluginId: string
    pluginName: string
    config: Record<string, any>
    enabled: boolean
    executionOrder: number
}

interface ActionSectionProps {
    actions: Action[]
    onChange: (actions: Action[]) => void
    testData: Record<string, any>
}

export function ActionSection({ actions, onChange, testData }: ActionSectionProps) {
    const [selectedActionId, setSelectedActionId] = useState<string>()
    const [availablePlugins, setAvailablePlugins] = useState<Array<{
        id: string
        name: string
        description: string
        requiredConfig: string[]
        supportedEventTypes: string[]
    }>>([])

    // 获取可用的动作插件
    const fetchAvailablePlugins = useCallback(async () => {
        try {
            const response = await apiClient.get('/plugins/ui/schemas', {
                params: { type: 'action' }
            })
            // 转换UI schemas为插件列表，过滤出动作插件
            const formattedPlugins = Object.keys(response)
                .filter(pluginId => {
                    const plugin = response[pluginId]
                    // 过滤出动作插件（排除条件插件如builtin）
                    return plugin.info.type === 'action' ||
                        (plugin.info.type !== 'condition' && pluginId !== 'builtin')
                })
                .map((pluginId) => ({
                    id: pluginId,
                    name: response[pluginId].info.name || pluginId,
                    description: response[pluginId].info.description || '',
                    requiredConfig: response[pluginId].schema?.fields?.map((field: any) => field.name) || [],
                    supportedEventTypes: ['email_received'] // 默认支持邮件接收事件
                }))

            setAvailablePlugins(formattedPlugins)
        } catch (error) {
            console.error('获取动作插件失败:', error)
        }
    }, [])

    // 获取插件默认配置
    const getDefaultConfigForPlugin = (pluginId: string): Record<string, any> => {
        switch (pluginId) {
            case 'email_transform_action':
                return {
                    target_field: 'subject',
                    transform_type: 'template'
                }
            case 'email_forward_action':
                return {
                    to_address: '',
                    subject_prefix: ''
                }
            case 'email_label_action':
                return {
                    operation: 'add',
                    labels: []
                }
            case 'email_delete_action':
                return {
                    permanent: false
                }
            default:
                return {}
        }
    }

    // 添加新动作
    const handleAddAction = useCallback((pluginId: string) => {
        const plugin = availablePlugins.find(p => p.id === pluginId)
        if (plugin) {
            // 应用插件默认配置
            const defaultConfig = getDefaultConfigForPlugin(pluginId)

            const newAction: Action = {
                id: Date.now().toString(),
                pluginId,
                pluginName: plugin.name,
                config: defaultConfig,
                enabled: true,
                executionOrder: actions.length + 1
            }
            const newActions = [...actions, newAction]
            onChange(newActions)
            setSelectedActionId(newAction.id)
        }
    }, [actions, availablePlugins, onChange])

    // 更新动作配置
    const handleActionConfigChange = useCallback((actionId: string, config: Record<string, any>) => {
        const updatedActions = actions.map(action =>
            action.id === actionId ? { ...action, config } : action
        )
        onChange(updatedActions)
    }, [actions, onChange])

    // 获取选中的动作
    const selectedAction = selectedActionId ? actions.find(a => a.id === selectedActionId) : null

    // 初始化时获取可用插件
    useEffect(() => {
        fetchAvailablePlugins()
    }, [fetchAvailablePlugins])

    // 自动选中第一个动作
    useEffect(() => {
        if (actions.length > 0 && !selectedActionId) {
            setSelectedActionId(actions[0].id)
        }
    }, [actions, selectedActionId])

    return (
        <div className="space-y-4">
            <div className="flex items-center gap-2">
                <Zap className="h-5 w-5 text-green-500" />
                <h3 className="text-lg font-semibold">动作配置</h3>
                <Badge variant="secondary" className="text-xs">
                    {actions.length} 个动作
                </Badge>
            </div>

            {/* 动作流水线 */}
            <Card className="overflow-hidden">
                <ActionPipeline
                    actions={actions}
                    selectedActionId={selectedActionId}
                    availablePlugins={availablePlugins}
                    onActionsChange={onChange}
                    onActionSelect={setSelectedActionId}
                    onAddAction={handleAddAction}
                    onExecute={() => { }} // 在父组件中统一执行
                    isExecuting={false}
                />
            </Card>

            {/* 动作配置面板 */}
            {selectedAction ? (
                <Card className="p-4">
                    <h4 className="font-medium mb-4">动作详细配置</h4>
                    <ActionConfigPanel
                        action={selectedAction}
                        availablePlugins={availablePlugins}
                        onChange={(config) => handleActionConfigChange(selectedAction.id, config)}
                    />
                </Card>
            ) : actions.length > 0 ? (
                <Card className="p-8 text-center border-dashed">
                    <div className="text-gray-500">
                        <Zap className="h-8 w-8 mx-auto mb-3 opacity-50" />
                        <p className="text-sm mb-2">选择一个动作进行配置</p>
                        <p className="text-xs">点击上方流水线中的动作卡片</p>
                    </div>
                </Card>
            ) : (
                <Card className="p-8 text-center border-dashed">
                    <div className="text-gray-500">
                        <Zap className="h-8 w-8 mx-auto mb-3 opacity-50" />
                        <p className="text-sm mb-2">暂无动作</p>
                        <p className="text-xs">添加动作来处理通过过滤器的邮件</p>
                    </div>
                </Card>
            )}

            <div className="text-xs text-gray-500 p-3 bg-green-50 rounded">
                <p><strong>提示：</strong></p>
                <ul className="mt-1 space-y-1">
                    <li>• 动作只有在过滤器通过后才会执行</li>
                    <li>• 动作按照执行顺序依次运行</li>
                    <li>• 每个动作的输出会传递给下一个动作</li>
                </ul>
            </div>
        </div>
    )
}