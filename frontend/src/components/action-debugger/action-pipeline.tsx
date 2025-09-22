'use client'

import React from 'react'
import { createPortal } from 'react-dom'
import {
    DndContext,
    closestCenter,
    KeyboardSensor,
    PointerSensor,
    useSensor,
    useSensors,
    DragEndEvent
} from '@dnd-kit/core'
import {
    arrayMove,
    SortableContext,
    sortableKeyboardCoordinates,
    useSortable,
    horizontalListSortingStrategy
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Trash2, GripVertical, Plus, Settings, ChevronDown } from 'lucide-react'

interface AddActionDropdownProps {
    availablePlugins: Array<{
        id: string
        name: string
        description: string
    }>
    onAddAction: (pluginId: string) => void
}

function AddActionDropdown({ availablePlugins, onAddAction }: AddActionDropdownProps) {
    const [isOpen, setIsOpen] = React.useState(false)
    const [buttonRect, setButtonRect] = React.useState<DOMRect | null>(null)
    const buttonRef = React.useRef<HTMLButtonElement>(null)
    const dropdownRef = React.useRef<HTMLDivElement>(null)

    console.log('AddActionDropdown render - availablePlugins:', availablePlugins)
    console.log('AddActionDropdown render - isOpen:', isOpen)

    // 处理点击外部关闭下拉菜单
    React.useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node) &&
                buttonRef.current && !buttonRef.current.contains(event.target as Node)) {
                console.log('点击外部，关闭下拉菜单')
                setIsOpen(false)
            }
        }

        if (isOpen) {
            document.addEventListener('mousedown', handleClickOutside)
            return () => {
                document.removeEventListener('mousedown', handleClickOutside)
            }
        }
    }, [isOpen])

    // 计算按钮位置
    const handleButtonClick = (e: React.MouseEvent) => {
        e.preventDefault()
        e.stopPropagation()

        if (buttonRef.current) {
            const rect = buttonRef.current.getBoundingClientRect()
            setButtonRect(rect)
        }

        console.log('点击添加动作按钮，当前isOpen:', isOpen)
        setIsOpen(!isOpen)
    }

    return (
        <>
            <Button
                ref={buttonRef}
                variant="outline"
                className="min-w-[120px] h-[80px] border-2 border-dashed border-gray-300 hover:border-blue-500 hover:bg-blue-50"
                onClick={handleButtonClick}
            >
                <div className="flex flex-col items-center gap-1">
                    <Plus className="h-5 w-5" />
                    <span className="text-xs">添加动作</span>
                    <ChevronDown className={`h-3 w-3 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
                </div>
            </Button>

            {/* 使用 Portal 渲染下拉菜单 */}
            {isOpen && buttonRect && createPortal(
                <div
                    ref={dropdownRef}
                    className="fixed bg-white border rounded-lg shadow-lg z-[10000] min-w-[200px]"
                    style={{
                        position: 'fixed',
                        top: buttonRect.bottom + 8,
                        left: buttonRect.left,
                        zIndex: 10000,
                        backgroundColor: 'white',
                        border: '1px solid #e5e7eb',
                        borderRadius: '8px',
                        boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)'
                    }}
                    onClick={(e) => {
                        e.preventDefault()
                        e.stopPropagation()
                        console.log('点击菜单内容，防止关闭')
                    }}
                >
                    <div className="p-4">
                        <h4 className="text-sm font-medium mb-3 text-gray-900">选择动作插件</h4>
                        {availablePlugins.length === 0 ? (
                            <div className="text-sm text-gray-500 p-2">
                                暂无可用的动作插件
                            </div>
                        ) : (
                            <div className="space-y-1">
                                {availablePlugins.map((plugin) => (
                                    <button
                                        key={plugin.id}
                                        onClick={() => {
                                            console.log('选择插件:', plugin.id)
                                            onAddAction(plugin.id)
                                            setIsOpen(false)
                                        }}
                                        className="w-full text-left p-3 hover:bg-gray-100 rounded-md text-sm transition-colors"
                                    >
                                        <div className="font-medium text-gray-900">{plugin.name}</div>
                                        <div className="text-xs text-gray-500 mt-1">{plugin.description}</div>
                                    </button>
                                ))}
                            </div>
                        )}
                    </div>
                </div>,
                document.body
            )}
        </>
    )
}

interface SortableActionCardProps {
    action: Action
    index: number
    isSelected: boolean
    isLast: boolean
    onSelect: (actionId: string) => void
    onToggleEnabled: (actionId: string, enabled: boolean) => void
    onDelete: (actionId: string) => void
    getActionSummary: (action: Action) => string
}

function SortableActionCard({
    action,
    index,
    isSelected,
    isLast,
    onSelect,
    onToggleEnabled,
    onDelete,
    getActionSummary
}: SortableActionCardProps) {
    const {
        attributes,
        listeners,
        setNodeRef,
        transform,
        transition,
        isDragging,
    } = useSortable({ id: action.id })

    const style = {
        transform: CSS.Transform.toString(transform),
        transition,
    }

    return (
        <div className="flex items-center gap-2">
            <Card
                ref={setNodeRef}
                style={style}
                className={`
                    min-w-[200px] p-3 cursor-pointer transition-all
                    ${isSelected
                        ? 'ring-2 ring-blue-500 bg-blue-50'
                        : 'hover:shadow-md'
                    }
                    ${!action.enabled ? 'opacity-60' : ''}
                    ${isDragging ? 'shadow-lg rotate-2' : ''}
                `}
                onClick={() => onSelect(action.id)}
            >
                <div className="flex items-start justify-between mb-2">
                    <div className="flex items-center gap-2">
                        <div
                            {...attributes}
                            {...listeners}
                            className="cursor-grab hover:cursor-grabbing"
                        >
                            <GripVertical className="h-4 w-4 text-gray-400" />
                        </div>
                        <Badge variant="outline" className="text-xs">
                            #{action.executionOrder}
                        </Badge>
                    </div>
                    <div className="flex items-center gap-1">
                        <Switch
                            checked={action.enabled}
                            onCheckedChange={(enabled) =>
                                onToggleEnabled(action.id, enabled)
                            }
                        />
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={(e) => {
                                e.stopPropagation()
                                onDelete(action.id)
                            }}
                            className="h-6 w-6 p-0 text-red-500 hover:text-red-700"
                        >
                            <Trash2 className="h-3 w-3" />
                        </Button>
                    </div>
                </div>
                <div>
                    <h4 className="font-medium text-sm mb-1">
                        {action.pluginName}
                    </h4>
                    <p className="text-xs text-gray-500">
                        {getActionSummary(action)}
                    </p>
                </div>
            </Card>
            {!isLast && (
                <div className="text-gray-400">→</div>
            )}
        </div>
    )
}

interface Action {
    id: string
    pluginId: string
    pluginName: string
    config: Record<string, any>
    enabled: boolean
    executionOrder: number
}

interface ActionPipelineProps {
    actions: Action[]
    selectedActionId?: string
    availablePlugins: Array<{
        id: string
        name: string
        description: string
    }>
    onActionsChange: (actions: Action[]) => void
    onActionSelect: (actionId: string) => void
    onAddAction: (pluginId: string) => void
    onExecute: () => void
    isExecuting: boolean
}

export function ActionPipeline({
    actions,
    selectedActionId,
    availablePlugins,
    onActionsChange,
    onActionSelect,
    onAddAction,
    onExecute,
    isExecuting
}: ActionPipelineProps) {
    const sensors = useSensors(
        useSensor(PointerSensor),
        useSensor(KeyboardSensor, {
            coordinateGetter: sortableKeyboardCoordinates,
        })
    )

    const handleDragEnd = (event: DragEndEvent) => {
        const { active, over } = event

        if (over && active.id !== over.id) {
            const oldIndex = actions.findIndex(action => action.id === active.id)
            const newIndex = actions.findIndex(action => action.id === over.id)

            const updatedActions = arrayMove(actions, oldIndex, newIndex).map(
                (action, index) => ({
                    ...action,
                    executionOrder: index + 1
                })
            )

            onActionsChange(updatedActions)
        }
    }

    const handleToggleEnabled = (actionId: string, enabled: boolean) => {
        const updatedActions = actions.map(action =>
            action.id === actionId ? { ...action, enabled } : action
        )
        onActionsChange(updatedActions)
    }

    const handleDeleteAction = (actionId: string) => {
        const updatedActions = actions.filter(action => action.id !== actionId)
        onActionsChange(updatedActions)
    }

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

    const handleAddActionAtStart = (pluginId: string) => {
        // 创建新动作，放在最前面
        const plugin = availablePlugins.find(p => p.id === pluginId)

        // 应用插件默认配置
        const defaultConfig = getDefaultConfigForPlugin(pluginId)

        const newAction: Action = {
            id: `action-${Date.now()}`,
            pluginId,
            pluginName: plugin?.name || pluginId,
            config: defaultConfig,
            enabled: true,
            executionOrder: 1
        }

        // 更新所有现有动作的执行顺序
        const updatedExistingActions = actions.map(action => ({
            ...action,
            executionOrder: action.executionOrder + 1
        }))

        // 合并新动作和现有动作
        const allActions = [newAction, ...updatedExistingActions]
        onActionsChange(allActions)
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
        <div className="bg-white border-b p-4">
            {/* 头部操作栏 */}
            <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                    <Settings className="h-5 w-5 text-gray-500" />
                    <h3 className="text-lg font-semibold">动作流水线</h3>
                    <Badge variant="secondary" className="text-xs">
                        {actions.length} 个动作
                    </Badge>
                </div>
                <div className="flex items-center gap-2">
                    <Button
                        onClick={onExecute}
                        disabled={isExecuting || actions.length === 0}
                        className="bg-green-500 hover:bg-green-600"
                    >
                        {isExecuting ? '执行中...' : '执行动作'}
                    </Button>
                </div>
            </div>

            {/* 动作流水线 */}
            <div className="overflow-x-auto overflow-y-visible">
                <div className="flex items-center gap-4 min-w-max py-4 px-3">
                    {actions.length === 0 ? (
                        /* 空状态：只显示一个居中的添加动作按钮 */
                        <div className="flex items-center justify-center w-full">
                            <AddActionDropdown
                                availablePlugins={availablePlugins}
                                onAddAction={onAddAction}
                            />
                        </div>
                    ) : (
                        /* 有动作时：显示前面和后面的按钮 */
                        <>
                            {/* 在最前面添加动作按钮 */}
                            <div className="flex items-center gap-2 flex-shrink-0">
                                <AddActionDropdown
                                    availablePlugins={availablePlugins}
                                    onAddAction={handleAddActionAtStart}
                                />
                                <div className="text-gray-400">→</div>
                            </div>

                            <DndContext
                                sensors={sensors}
                                collisionDetection={closestCenter}
                                onDragEnd={handleDragEnd}
                            >
                                <SortableContext
                                    items={actions.map(action => action.id)}
                                    strategy={horizontalListSortingStrategy}
                                >
                                    <div className="flex items-center gap-4">
                                        {actions.map((action, index) => (
                                            <SortableActionCard
                                                key={action.id}
                                                action={action}
                                                index={index}
                                                isSelected={selectedActionId === action.id}
                                                isLast={index === actions.length - 1}
                                                onSelect={onActionSelect}
                                                onToggleEnabled={handleToggleEnabled}
                                                onDelete={handleDeleteAction}
                                                getActionSummary={getActionSummary}
                                            />
                                        ))}
                                    </div>
                                </SortableContext>
                            </DndContext>

                            {/* 在末尾添加动作按钮 */}
                            <div className="flex items-center gap-2 ml-4 flex-shrink-0">
                                <div className="text-gray-400">→</div>
                                <AddActionDropdown
                                    availablePlugins={availablePlugins}
                                    onAddAction={onAddAction}
                                />
                            </div>
                        </>
                    )}
                </div>
            </div>
        </div>
    )
}

