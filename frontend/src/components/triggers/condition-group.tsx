'use client'

import { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Plus, X, ChevronDown, ChevronUp, ArrowDown, ArrowUp } from 'lucide-react'
import { ConditionItem } from './condition-item'
import { v4 as uuidv4 } from 'uuid'

// 表达式类型
export type ExpressionType = 'group' | 'condition'

// 逻辑操作符类型
export type LogicalOperatorType = 'and' | 'or' | 'not'

// 比较操作符类型
export type ComparisonOperatorType = 'equals' | 'not_equals' | 'contains' | 'not_contains' | 'starts_with' | 'ends_with' | 'matches' | 'greater_than' | 'less_than' | 'greater_equal' | 'less_equal' | 'in' | 'not_in' | 'regex' | 'not_regex'

// 操作符类型（联合类型）
export type OperatorType = LogicalOperatorType | ComparisonOperatorType

// 表达式接口
export interface Expression {
  id: string
  type: ExpressionType
  operator?: OperatorType
  field?: string
  value?: any
  conditions?: Expression[]
  not?: boolean
}

interface ConditionGroupProps {
  group: Expression
  parentId?: string
  onUpdate: (updatedGroup: Expression) => void
  onRemove?: () => void
  onAddCondition: (groupId: string) => void
  onAddGroup: (groupId: string) => void
  onRemoveExpression: (expressionId: string, groupId: string) => void
  level?: number
}

export function ConditionGroup({ 
  group, 
  parentId, 
  onUpdate, 
  onRemove, 
  onAddCondition, 
  onAddGroup, 
  onRemoveExpression,
  level = 0
}: ConditionGroupProps) {
  const [collapsed, setCollapsed] = useState(false)
  
  // 更新组操作符
  const handleOperatorChange = (value: string) => {
    onUpdate({
      ...group,
      operator: value as OperatorType
    })
  }
  
  // 更新组取反状态
  const handleNotChange = (checked: boolean) => {
    onUpdate({
      ...group,
      not: checked
    })
  }
  
  // 更新子条件
  const handleConditionUpdate = (updatedCondition: Expression) => {
    const updatedConditions = group.conditions?.map(condition => 
      condition.id === updatedCondition.id ? updatedCondition : condition
    )
    
    onUpdate({
      ...group,
      conditions: updatedConditions
    })
  }
  
  // 移动条件项
  const moveCondition = (id: string, direction: 'up' | 'down') => {
    if (!group.conditions) return
    
    const index = group.conditions.findIndex(c => c.id === id)
    if (index === -1) return
    
    const newConditions = [...group.conditions]
    
    if (direction === 'up' && index > 0) {
      // 向上移动
      [newConditions[index], newConditions[index - 1]] = [newConditions[index - 1], newConditions[index]]
    } else if (direction === 'down' && index < newConditions.length - 1) {
      // 向下移动
      [newConditions[index], newConditions[index + 1]] = [newConditions[index + 1], newConditions[index]]
    } else {
      return // 无法移动
    }
    
    onUpdate({
      ...group,
      conditions: newConditions
    })
  }
  
  // 获取条件项的位置信息
  const getItemPosition = (id: string) => {
    if (!group.conditions) return { isFirst: true, isLast: true }
    
    const index = group.conditions.findIndex(c => c.id === id)
    return {
      isFirst: index === 0,
      isLast: index === group.conditions.length - 1
    }
  }
  
  // 根据操作符获取描述文本
  const getOperatorDescription = (operator?: OperatorType, not?: boolean) => {
    if (not) {
      return '不满足以下条件'
    }
    
    switch (operator) {
      case 'and':
        return '满足所有条件'
      case 'or':
        return '满足任一条件'
      case 'not':
        return '不满足以下条件'
      default:
        return '满足条件'
    }
  }
  
  // 获取嵌套级别的样式
  const getNestingStyles = () => {
    const borderColor = level % 3 === 0 
      ? 'border-blue-200' 
      : level % 3 === 1 
        ? 'border-green-200' 
        : 'border-amber-200'
    
    const bgColor = level % 3 === 0 
      ? 'bg-blue-50' 
      : level % 3 === 1 
        ? 'bg-green-50' 
        : 'bg-amber-50'
    
    return {
      borderColor,
      bgColor
    }
  }
  
  const { borderColor, bgColor } = getNestingStyles()
  
  return (
    <Card className={`mb-4 border-2 border-dashed ${borderColor}`}>
      <CardContent className={`p-4 ${collapsed ? '' : bgColor}`}>
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setCollapsed(!collapsed)}
              className="p-1 h-8 w-8"
            >
              {collapsed ? <ChevronDown className="h-4 w-4" /> : <ChevronUp className="h-4 w-4" />}
            </Button>
            
            <Label>当</Label>
            <Select 
              value={group.operator} 
              onValueChange={handleOperatorChange}
            >
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="选择操作符" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="and">满足所有条件 (AND)</SelectItem>
                <SelectItem value="or">满足任一条件 (OR)</SelectItem>
                <SelectItem value="not">不满足条件 (NOT)</SelectItem>
              </SelectContent>
            </Select>
            
            <div className="flex items-center ml-4">
              <Label htmlFor={`not-${group.id}`} className="mr-2">取反</Label>
              <input
                id={`not-${group.id}`}
                type="checkbox"
                checked={group.not === true}
                onChange={(e) => handleNotChange(e.target.checked)}
                className="h-4 w-4"
              />
            </div>
          </div>
          
          <div className="flex items-center gap-2">
            {parentId && onRemove && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={onRemove}
                className="text-red-600"
              >
                <X className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>
        
        {!collapsed && (
          <>
            <div className="pl-4 border-l-2 border-gray-200">
              {group.conditions?.map((condition, index) => {
                const { isFirst, isLast } = getItemPosition(condition.id)
                
                return condition.type === 'condition' ? (
                  <div key={condition.id} className="relative">
                    <ConditionItem
                      condition={condition}
                      onUpdate={handleConditionUpdate}
                      onRemove={() => onRemoveExpression(condition.id, group.id)}
                    />
                    <div className="absolute right-0 top-1/2 transform -translate-y-1/2 flex flex-col gap-1 mr-12">
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => moveCondition(condition.id, 'up')}
                        disabled={isFirst}
                        className="p-1 h-6 w-6"
                      >
                        <ArrowUp className="h-3 w-3" />
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => moveCondition(condition.id, 'down')}
                        disabled={isLast}
                        className="p-1 h-6 w-6"
                      >
                        <ArrowDown className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div key={condition.id} className="relative">
                    <ConditionGroup
                      group={condition}
                      parentId={group.id}
                      onUpdate={handleConditionUpdate}
                      onRemove={() => onRemoveExpression(condition.id, group.id)}
                      onAddCondition={onAddCondition}
                      onAddGroup={onAddGroup}
                      onRemoveExpression={onRemoveExpression}
                      level={level + 1}
                    />
                    <div className="absolute right-0 top-8 flex flex-col gap-1 mr-2">
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => moveCondition(condition.id, 'up')}
                        disabled={isFirst}
                        className="p-1 h-6 w-6"
                      >
                        <ArrowUp className="h-3 w-3" />
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => moveCondition(condition.id, 'down')}
                        disabled={isLast}
                        className="p-1 h-6 w-6"
                      >
                        <ArrowDown className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>
                )
              })}
            </div>
            
            <div className="flex gap-2 mt-4">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => onAddCondition(group.id)}
              >
                <Plus className="h-4 w-4 mr-2" />
                添加条件
              </Button>
              
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => onAddGroup(group.id)}
              >
                <Plus className="h-4 w-4 mr-2" />
                添加条件组
              </Button>
            </div>
          </>
        )}
        
        {collapsed && (
          <div className="mt-2 text-sm text-gray-500">
            {group.conditions?.length || 0} 个条件 - {getOperatorDescription(group.operator, group.not)}
          </div>
        )}
      </CardContent>
    </Card>
  )
}