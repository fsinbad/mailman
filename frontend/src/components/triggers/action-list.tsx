'use client'

import React from 'react'
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
  verticalListSortingStrategy
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { GripVertical, Trash2, Plus, ChevronDown, ChevronUp } from 'lucide-react'
import { TriggerActionConfig } from '@/types'

interface SortableActionItemProps {
  action: TriggerActionConfig
  onToggleEnabled: (enabled: boolean) => void
  onEdit: () => void
  onDelete: () => void
  isRemovable: boolean
}

function SortableActionItem({ 
  action, 
  onToggleEnabled, 
  onEdit, 
  onDelete, 
  isRemovable 
}: SortableActionItemProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: action.order.toString() })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  // 获取动作配置的简短摘要
  const getActionSummary = () => {
    try {
      const config = JSON.parse(action.config)
      
      switch (action.type) {
        case 'modify_content':
          return `修改内容: ${config.subject_prefix ? `添加前缀 "${config.subject_prefix}"` : ''}${config.add_tag ? `, 添加标签 "${config.add_tag}"` : ''}${config.mark_as_read ? ', 标记为已读' : ''}`
        case 'smtp':
          return `发送邮件: 发送至 ${config.to || '未指定'}`
        default:
          return '自定义动作'
      }
    } catch (e) {
      return '配置解析错误'
    }
  }

  return (
    <Card 
      ref={setNodeRef} 
      style={style}
      className={`mb-3 ${!action.enabled ? 'opacity-70' : ''} ${isDragging ? 'z-10 shadow-lg' : ''}`}
    >
      <CardContent className="p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div 
              {...attributes} 
              {...listeners}
              className="cursor-grab hover:cursor-grabbing p-1"
            >
              <GripVertical className="h-5 w-5 text-gray-400" />
            </div>
            <Badge variant="outline" className="mr-2">
              #{action.order + 1}
            </Badge>
            <h3 className="font-medium">{action.name}</h3>
          </div>
          
          <div className="flex items-center gap-2">
            <Switch
              checked={action.enabled}
              onCheckedChange={onToggleEnabled}
              aria-label={action.enabled ? '禁用动作' : '启用动作'}
            />
            
            {isRemovable && (
              <Button 
                variant="ghost" 
                size="sm" 
                onClick={onDelete}
                className="h-8 w-8 p-0 text-red-500 hover:text-red-700 hover:bg-red-50"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>
        
        <div className="mt-2 text-sm text-gray-600">
          <p>{action.description || getActionSummary()}</p>
        </div>
        
        <div className="mt-3 flex justify-end gap-2">
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={(e) => {
              e.stopPropagation()
              // 将动作配置编码为URL参数并导航到测试页面
              const actionParam = encodeURIComponent(JSON.stringify(action))
              window.open(`/triggers/actions/test?action=${actionParam}`, '_blank')
            }}
            className="text-green-600 hover:text-green-800 hover:bg-green-50"
          >
            测试
          </Button>
          <Button 
            variant="outline" 
            size="sm" 
            onClick={onEdit}
            className="text-blue-600 hover:text-blue-800 hover:bg-blue-50"
          >
            配置
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

interface ActionListProps {
  actions: TriggerActionConfig[]
  onChange: (actions: TriggerActionConfig[]) => void
  onEditAction: (index: number) => void
}

export function ActionList({ actions, onChange, onEditAction }: ActionListProps) {
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  )

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    
    if (over && active.id !== over.id) {
      const oldIndex = parseInt(active.id.toString())
      const newIndex = parseInt(over.id.toString())
      
      // 重新排序动作
      const reorderedActions = arrayMove(
        [...actions], 
        actions.findIndex(a => a.order === oldIndex),
        actions.findIndex(a => a.order === newIndex)
      )
      
      // 更新顺序号
      const updatedActions = reorderedActions.map((action, index) => ({
        ...action,
        order: index
      }))
      
      onChange(updatedActions)
    }
  }

  const handleToggleEnabled = (index: number, enabled: boolean) => {
    const updatedActions = actions.map((action, i) => 
      i === index ? { ...action, enabled } : action
    )
    onChange(updatedActions)
  }

  const handleDeleteAction = (index: number) => {
    // 删除动作并重新排序
    const filteredActions = actions.filter((_, i) => i !== index)
    const updatedActions = filteredActions.map((action, index) => ({
      ...action,
      order: index
    }))
    onChange(updatedActions)
  }

  const handleAddAction = () => {
    // 创建新动作
    const newAction: TriggerActionConfig = {
      type: 'modify_content',
      name: `动作 ${actions.length + 1}`,
      description: '',
      config: '{}',
      enabled: true,
      order: actions.length
    }
    
    onChange([...actions, newAction])
    // 自动打开编辑界面
    onEditAction(actions.length)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">动作列表</h3>
        <Badge variant="secondary">{actions.length} 个动作</Badge>
      </div>
      
      {actions.length === 0 ? (
        <div className="flex flex-col items-center justify-center p-8 border-2 border-dashed border-gray-300 rounded-lg">
          <p className="text-gray-500 mb-4">还没有添加任何动作</p>
          <Button onClick={handleAddAction}>
            <Plus className="h-4 w-4 mr-2" />
            添加动作
          </Button>
        </div>
      ) : (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={actions.map(action => action.order.toString())}
            strategy={verticalListSortingStrategy}
          >
            <div className="space-y-2">
              {actions.map((action, index) => (
                <SortableActionItem
                  key={`action-${action.order}`}
                  action={action}
                  onToggleEnabled={(enabled) => handleToggleEnabled(index, enabled)}
                  onEdit={() => onEditAction(index)}
                  onDelete={() => handleDeleteAction(index)}
                  isRemovable={actions.length > 1}
                />
              ))}
            </div>
          </SortableContext>
        </DndContext>
      )}
      
      {actions.length > 0 && (
        <div className="flex justify-center mt-4">
          <Button 
            variant="outline" 
            onClick={handleAddAction}
            className="w-full"
          >
            <Plus className="h-4 w-4 mr-2" />
            添加动作
          </Button>
        </div>
      )}
    </div>
  )
}