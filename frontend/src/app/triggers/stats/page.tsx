'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { triggerService } from '@/services/trigger.service'
import { EmailTrigger, TriggerStatistics } from '@/types'
import { ArrowLeft, Download, RefreshCw, BarChart3, Cpu } from 'lucide-react'
import { TriggerPerformance } from '@/components/triggers/trigger-performance'

export default function StatsPage() {
  const router = useRouter()
  
  const [triggers, setTriggers] = useState<EmailTrigger[]>([])
  const [selectedTriggerId, setSelectedTriggerId] = useState<number | null>(null)
  const [startDate, setStartDate] = useState<string>('')
  const [endDate, setEndDate] = useState<string>('')
  const [isLoading, setIsLoading] = useState(true)
  const [stats, setStats] = useState<TriggerStatistics | null>(null)
  
  // 加载触发器列表
  useEffect(() => {
    loadTriggers()
  }, [])
  
  // 加载统计数据
  useEffect(() => {
    if (selectedTriggerId) {
      loadStats()
    }
  }, [selectedTriggerId, startDate, endDate])
  
  const loadTriggers = async () => {
    try {
      setIsLoading(true)
      
      const response = await triggerService.getTriggers()
      setTriggers(response.data)
      
      // 默认选择第一个触发器
      if (response.data.length > 0 && !selectedTriggerId) {
        setSelectedTriggerId(response.data[0].id)
      }
    } catch (error) {
      console.error('加载触发器列表失败:', error)
    } finally {
      setIsLoading(false)
    }
  }
  
  const loadStats = async () => {
    if (!selectedTriggerId) return
    
    try {
      setIsLoading(true)
      
      const statistics = await triggerService.getTriggerStatistics(
        selectedTriggerId,
        startDate || undefined,
        endDate || undefined
      )
      
      setStats(statistics)
    } catch (error) {
      console.error('加载统计数据失败:', error)
    } finally {
      setIsLoading(false)
    }
  }
  
  // 格式化百分比
  const formatPercentage = (value: number) => {
    return `${(value * 100).toFixed(1)}%`
  }
  
  // 格式化执行时间
  const formatExecutionTime = (ms: number) => {
    if (ms < 1000) {
      return `${ms.toFixed(0)}ms`
    } else {
      return `${(ms / 1000).toFixed(2)}s`
    }
  }
  
  // 导出统计数据
  const exportStats = () => {
    if (!stats) return
    
    const statsData = JSON.stringify(stats, null, 2)
    const blob = new Blob([statsData], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `trigger-stats-${selectedTriggerId}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }
  
  return (
    <div className="space-y-6 p-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={() => router.push('/triggers')}
          >
            <ArrowLeft className="h-4 w-4 mr-1" />
            返回触发器列表
          </Button>
          <h1 className="text-2xl font-bold">触发器统计分析</h1>
        </div>
        
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => router.push('/triggers/logs')}>
            <BarChart3 className="h-4 w-4 mr-1" />
            查看日志
          </Button>
          {stats && (
            <Button variant="outline" onClick={exportStats}>
              <Download className="h-4 w-4 mr-1" />
              导出数据
            </Button>
          )}
        </div>
      </div>
      
      {/* 过滤器 */}
      <Card>
        <CardContent className="p-4">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div>
              <Label htmlFor="triggerSelect">选择触发器</Label>
              <select
                id="triggerSelect"
                className="w-full p-2 border rounded mt-1"
                value={selectedTriggerId || ''}
                onChange={(e) => setSelectedTriggerId(parseInt(e.target.value))}
              >
                {triggers.map((trigger) => (
                  <option key={trigger.id} value={trigger.id}>
                    {trigger.name}
                  </option>
                ))}
              </select>
            </div>
            
            <div>
              <Label htmlFor="startDate">开始日期</Label>
              <Input
                id="startDate"
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                className="mt-1"
              />
            </div>
            
            <div>
              <Label htmlFor="endDate">结束日期</Label>
              <Input
                id="endDate"
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                className="mt-1"
              />
            </div>
            
            <div className="flex items-end">
              <Button 
                type="button" 
                onClick={() => {
                  setStartDate('')
                  setEndDate('')
                }}
                variant="outline"
                className="mr-2"
              >
                重置
              </Button>
              <Button type="button" onClick={loadStats}>
                <RefreshCw className="h-4 w-4 mr-2" />
                刷新
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>      

      {/* 统计数据 */}
      {isLoading ? (
        <Card>
          <CardContent className="p-12 text-center">
            <div className="animate-spin h-8 w-8 border-4 border-blue-600 border-t-transparent rounded-full mx-auto"></div>
            <p className="mt-4 text-gray-600">加载中...</p>
          </CardContent>
        </Card>
      ) : stats ? (
        <div className="space-y-6">
          {/* 主要指标 */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <Card>
              <CardContent className="p-4">
                <div className="flex flex-col">
                  <span className="text-sm text-gray-600 dark:text-gray-400">总执行次数</span>
                  <span className="text-2xl font-bold">{stats.total_executions}</span>
                </div>
              </CardContent>
            </Card>
            
            <Card>
              <CardContent className="p-4">
                <div className="flex flex-col">
                  <span className="text-sm text-gray-600 dark:text-gray-400">成功次数</span>
                  <span className="text-2xl font-bold text-green-600">{stats.success_executions}</span>
                </div>
              </CardContent>
            </Card>
            
            <Card>
              <CardContent className="p-4">
                <div className="flex flex-col">
                  <span className="text-sm text-gray-600 dark:text-gray-400">失败次数</span>
                  <span className="text-2xl font-bold text-red-600">{stats.failed_executions}</span>
                </div>
              </CardContent>
            </Card>
            
            <Card>
              <CardContent className="p-4">
                <div className="flex flex-col">
                  <span className="text-sm text-gray-600 dark:text-gray-400">平均执行时间</span>
                  <span className="text-2xl font-bold text-blue-600">
                    {formatExecutionTime(stats.avg_execution_time)}
                  </span>
                </div>
              </CardContent>
            </Card>
          </div>
          
          {/* 详细统计 */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* 成功率 */}
            <Card>
              <CardHeader>
                <CardTitle>成功率</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between mb-4">
                  <span className="text-3xl font-bold text-green-600">
                    {formatPercentage(stats.success_rate)}
                  </span>
                </div>
                <div className="h-4 bg-gray-200 rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-green-600" 
                    style={{ width: `${stats.success_rate * 100}%` }}
                  ></div>
                </div>
              </CardContent>
            </Card>
            
            {/* 执行结果分布 */}
            <Card>
              <CardHeader>
                <CardTitle>执行结果分布</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div>
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm text-green-600">成功</span>
                      <span className="text-sm font-medium">
                        {stats.success_executions} ({formatPercentage(stats.success_executions / stats.total_executions)})
                      </span>
                    </div>
                    <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                      <div 
                        className="h-full bg-green-600" 
                        style={{ width: `${stats.total_executions > 0 ? (stats.success_executions / stats.total_executions) * 100 : 0}%` }}
                      ></div>
                    </div>
                  </div>
                  
                  <div>
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm text-red-600">失败</span>
                      <span className="text-sm font-medium">
                        {stats.failed_executions} ({formatPercentage(stats.failed_executions / stats.total_executions)})
                      </span>
                    </div>
                    <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                      <div 
                        className="h-full bg-red-600" 
                        style={{ width: `${stats.total_executions > 0 ? (stats.failed_executions / stats.total_executions) * 100 : 0}%` }}
                      ></div>
                    </div>
                  </div>
                  
                  <div>
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm text-yellow-600">部分成功</span>
                      <span className="text-sm font-medium">
                        {stats.partial_executions} ({formatPercentage(stats.partial_executions / stats.total_executions)})
                      </span>
                    </div>
                    <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                      <div 
                        className="h-full bg-yellow-600" 
                        style={{ width: `${stats.total_executions > 0 ? (stats.partial_executions / stats.total_executions) * 100 : 0}%` }}
                      ></div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
          
          {/* 性能分析 */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center">
                <Cpu className="h-5 w-5 mr-2" />
                性能分析
              </CardTitle>
            </CardHeader>
            <CardContent>
              <TriggerPerformance 
                triggerId={selectedTriggerId as number}
                startDate={startDate}
                endDate={endDate}
              />
            </CardContent>
          </Card>
        </div>
      ) : (
        <Card>
          <CardContent className="p-6 text-center">
            <p className="text-gray-600">暂无统计数据</p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}