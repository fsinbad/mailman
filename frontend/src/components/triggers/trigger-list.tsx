'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectItem } from '@/components/ui/select'
import {
  Search,
  Filter,
  Play,
  Pause,
  Settings,
  BarChart3,
  AlertCircle,
  CheckCircle,
  Clock,
  Trash2,
  Plus,
  Zap,
  Bug
} from 'lucide-react'
import { triggerService } from '@/services/trigger.service'
import { EmailTrigger, PaginationParams } from '@/types'
export interface TriggerListProps {
  onEdit?: (trigger: EmailTrigger) => void
  onView?: (trigger: EmailTrigger) => void
  onDelete?: (trigger: EmailTrigger) => void
  onStatusChange?: (trigger: EmailTrigger, enabled: boolean) => void
  onDebug?: (trigger: EmailTrigger) => void
}

export function TriggerList({ onEdit, onView, onDelete, onStatusChange, onDebug }: TriggerListProps) {
  const [triggers, setTriggers] = useState<EmailTrigger[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [limit, setLimit] = useState(10)
  const [isLoading, setIsLoading] = useState(true)
  const [searchTerm, setSearchTerm] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')

  // 加载触发器列表
  useEffect(() => {
    loadTriggers()
  }, [page, limit, searchTerm, statusFilter])

  const loadTriggers = async () => {
    try {
      setIsLoading(true)
      const params: PaginationParams = {
        page,
        limit,
        search: searchTerm || undefined
      }
      
      const response = await triggerService.getTriggers(params)
      setTriggers(response.data)
      setTotal(response.total)
    } catch (error) {
      console.error('加载触发器列表失败:', error)
    } finally {
      setIsLoading(false)
    }
  }
  // 处理状态变更
  const handleStatusChange = async (trigger: EmailTrigger) => {
    try {
      if (trigger.status === 'enabled') {
        await triggerService.disableTrigger(trigger.id)
      } else {
        await triggerService.enableTrigger(trigger.id)
      }
      
      // 如果有外部处理函数，调用它
      if (onStatusChange) {
        onStatusChange(trigger, trigger.status !== 'enabled')
      }
      
      // 重新加载数据
      loadTriggers()
    } catch (error) {
      console.error('更改触发器状态失败:', error)
    }
  }

  // 处理删除
  const handleDelete = async (trigger: EmailTrigger) => {
    if (window.confirm(`确定要删除触发器 "${trigger.name}" 吗？`)) {
      try {
        await triggerService.deleteTrigger(trigger.id)
        
        // 如果有外部处理函数，调用它
        if (onDelete) {
          onDelete(trigger)
        }
        
        // 重新加载数据
        loadTriggers()
      } catch (error) {
        console.error('删除触发器失败:', error)
      }
    }
  }

  // 获取状态图标
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'enabled':
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'disabled':
        return <Clock className="h-4 w-4 text-gray-500" />
      default:
        return <AlertCircle className="h-4 w-4 text-red-500" />
    }
  }

  // 获取状态文本
  const getStatusText = (status: string) => {
    switch (status) {
      case 'enabled':
        return '运行中'
      case 'disabled':
        return '已停用'
      default:
        return '错误'
    }
  }  // 处理搜索
  const handleSearch = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchTerm(e.target.value)
    setPage(1) // 重置到第一页
  }

  // 处理状态过滤
  const handleStatusFilter = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setStatusFilter(e.target.value)
    setPage(1) // 重置到第一页
  }

  return (
    <div className="space-y-4">
      {/* 搜索和过滤 */}
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
                <Input
                  placeholder="搜索触发器名称或描述..."
                  value={searchTerm}
                  onChange={handleSearch}
                  className="pl-10"
                />
              </div>
            </div>
            <div className="flex gap-2">
              <Select value={statusFilter} onValueChange={handleStatusFilter}>
                <SelectItem value="all">所有状态</SelectItem>
                <SelectItem value="enabled">运行中</SelectItem>
                <SelectItem value="disabled">已停用</SelectItem>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>   
   {/* 触发器列表 */}
      {isLoading ? (
        <Card>
          <CardContent className="p-12 text-center">
            <div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full mx-auto"></div>
            <p className="mt-4 text-gray-600">加载中...</p>
          </CardContent>
        </Card>
      ) : triggers.length > 0 ? (
        <div className="grid gap-4">
          {triggers.map((trigger) => (
            <Card key={trigger.id} className="hover:shadow-lg transition-shadow">
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3">
                      <CardTitle className="text-lg">{trigger.name}</CardTitle>
                      <div className="flex items-center gap-2">
                        {getStatusIcon(trigger.status)}
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          {getStatusText(trigger.status)}
                        </span>
                      </div>
                    </div>
                    {trigger.description && (
                      <p className="mt-2 text-gray-600 dark:text-gray-400">
                        {trigger.description}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge className="bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                      触发器
                    </Badge>
                    <Badge variant="outline">
                      ID: {trigger.id}
                    </Badge>
                  </div>
                </div>
              </CardHeader> 
             <CardContent>
                <div className="space-y-4">
                  {/* 触发条件 */}
                  <div>
                    <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      触发条件
                    </h4>
                    <div className="bg-blue-50 dark:bg-blue-900/20 p-3 rounded-lg">
                      <p className="text-sm text-blue-800 dark:text-blue-200">
                        {trigger.condition?.type === 'js' ? 'JavaScript 条件' : '模板条件'}
                      </p>
                    </div>
                  </div>

                  {/* 执行动作 */}
                  <div>
                    <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      执行动作 ({trigger.actions?.length || 0})
                    </h4>
                    <div className="space-y-2">
                      {trigger.actions?.map((action, index) => (
                        <div key={index} className="bg-green-50 dark:bg-green-900/20 p-2 rounded">
                          <p className="text-sm text-green-800 dark:text-green-200">
                            {action.name} ({action.type}) {action.enabled ? '(已启用)' : '(已禁用)'}
                          </p>
                        </div>
                      ))}
                    </div>
                  </div>          
        {/* 统计信息 */}
                  <div className="flex items-center justify-between pt-2 border-t border-gray-200 dark:border-gray-700">
                    <div className="flex items-center gap-4 text-sm text-gray-600 dark:text-gray-400">
                      <span>执行次数: {trigger.total_executions || 0}</span>
                      <span>成功次数: {trigger.success_executions || 0}</span>
                      <span>创建时间: {new Date(trigger.created_at).toLocaleDateString()}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      {onEdit && (
                        <Button variant="outline" size="sm" onClick={() => onEdit(trigger)}>
                          <Settings className="h-4 w-4 mr-1" />
                          编辑
                        </Button>
                      )}
                      {onView && (
                        <Button variant="outline" size="sm" onClick={() => onView(trigger)}>
                          <BarChart3 className="h-4 w-4 mr-1" />
                          详情
                        </Button>
                      )}
                      {onDebug && (
                        <Button variant="outline" size="sm" onClick={() => onDebug(trigger)}>
                          <Bug className="h-4 w-4 mr-1" />
                          调试
                        </Button>
                      )}
                      <Button
                        variant="outline"
                        size="sm"
                        className={trigger.status === 'enabled' ? 'text-red-600' : 'text-green-600'}
                        onClick={() => handleStatusChange(trigger)}
                      >
                        {trigger.status === 'enabled' ? (
                          <>
                            <Pause className="h-4 w-4 mr-1" />
                            禁用
                          </>
                        ) : (
                          <>
                            <Play className="h-4 w-4 mr-1" />
                            启用
                          </>
                        )}
                      </Button>      
                {onDelete && (
                        <Button 
                          variant="outline" 
                          size="sm" 
                          className="text-red-600"
                          onClick={() => handleDelete(trigger)}
                        >
                          <Trash2 className="h-4 w-4 mr-1" />
                          删除
                        </Button>
                      )}
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <Card>
          <CardContent className="p-12 text-center">
            <Search className="h-12 w-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
              未找到匹配的触发器
            </h3>
            <p className="text-gray-600 dark:text-gray-400 mb-4">
              尝试调整搜索条件或创建新的触发器规则
            </p>
          </CardContent>
        </Card>
      )} 
     {/* 分页控制 */}
      {total > 0 && (
        <div className="flex justify-between items-center mt-4">
          <div className="text-sm text-gray-600">
            共 {total} 条记录，当前第 {page} / {Math.ceil(total / limit)} 页
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage(page - 1)}
            >
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= Math.ceil(total / limit)}
              onClick={() => setPage(page + 1)}
            >
              下一页
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}