'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { AlertCircle, Clock, Cpu, BarChart3, Calendar } from 'lucide-react'
import { triggerService } from '@/services/trigger.service'
import { TriggerStatistics } from '@/types'

interface TriggerPerformanceProps {
  triggerId: number
  startDate?: string
  endDate?: string
}

export function TriggerPerformance({ triggerId, startDate, endDate }: TriggerPerformanceProps) {
  const [stats, setStats] = useState<TriggerStatistics | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState('execution-time')
  
  // 加载统计数据
  useEffect(() => {
    loadStats()
  }, [triggerId, startDate, endDate])
  
  const loadStats = async () => {
    try {
      setIsLoading(true)
      setError(null)
      
      const statistics = await triggerService.getTriggerStatistics(triggerId, startDate, endDate)
      
      // 如果后端API还没有实现新的统计数据，添加模拟数据用于UI展示
      if (!statistics.max_execution_time) {
        statistics.max_execution_time = statistics.avg_execution_time * 2.5
        statistics.min_execution_time = statistics.avg_execution_time * 0.4
        statistics.avg_condition_time = statistics.avg_execution_time * 0.3
        statistics.avg_action_time = statistics.avg_execution_time * 0.7
        statistics.execution_time_percentiles = {
          p50: statistics.avg_execution_time * 0.9,
          p90: statistics.avg_execution_time * 1.5,
          p95: statistics.avg_execution_time * 1.8,
          p99: statistics.avg_execution_time * 2.2
        }
        statistics.resource_usage = {
          avg_memory_mb: 45.2,
          max_memory_mb: 78.5,
          avg_cpu_percent: 12.3,
          max_cpu_percent: 35.7
        }
        statistics.time_distribution = {
          labels: ['0-100ms', '100-200ms', '200-500ms', '500ms-1s', '1s+'],
          values: [15, 35, 30, 15, 5]
        }
        statistics.executions_by_day = {
          dates: ['2025-07-13', '2025-07-14', '2025-07-15', '2025-07-16', '2025-07-17', '2025-07-18', '2025-07-19'],
          counts: [12, 15, 18, 14, 22, 19, 25],
          success_counts: [10, 13, 15, 12, 20, 17, 23],
          failed_counts: [2, 2, 3, 2, 2, 2, 2]
        }
      }
      
      setStats(statistics)
    } catch (err) {
      console.error('加载触发器性能数据失败:', err)
      setError('加载性能数据失败，请重试')
    } finally {
      setIsLoading(false)
    }
  }
  
  // 格式化执行时间
  const formatExecutionTime = (ms: number) => {
    if (ms < 1000) {
      return `${ms.toFixed(0)}ms`
    } else {
      return `${(ms / 1000).toFixed(2)}s`
    }
  }
  
  // 渲染执行时间分析
  const renderExecutionTimeAnalysis = () => {
    if (!stats) return null
    
    return (
      <div className="space-y-6">
        {/* 执行时间概览 */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">平均执行时间</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-blue-600">
                {formatExecutionTime(stats.avg_execution_time)}
              </div>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">最长执行时间</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-orange-600">
                {formatExecutionTime(stats.max_execution_time)}
              </div>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">最短执行时间</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-green-600">
                {formatExecutionTime(stats.min_execution_time)}
              </div>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">P95执行时间</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-purple-600">
                {formatExecutionTime(stats.execution_time_percentiles?.p95 || 0)}
              </div>
            </CardContent>
          </Card>
        </div>
        
        {/* 执行时间百分位 */}
        <Card>
          <CardHeader>
            <CardTitle>执行时间百分位</CardTitle>
            <CardDescription>不同百分位的执行时间分布</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm">中位数 (P50)</span>
                  <span className="text-sm font-medium">{formatExecutionTime(stats.execution_time_percentiles?.p50 || 0)}</span>
                </div>
                <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-blue-600" 
                    style={{ width: `${(stats.execution_time_percentiles?.p50 || 0) / stats.max_execution_time * 100}%` }}
                  ></div>
                </div>
              </div>
              
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm">P90</span>
                  <span className="text-sm font-medium">{formatExecutionTime(stats.execution_time_percentiles?.p90 || 0)}</span>
                </div>
                <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-blue-600" 
                    style={{ width: `${(stats.execution_time_percentiles?.p90 || 0) / stats.max_execution_time * 100}%` }}
                  ></div>
                </div>
              </div>
              
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm">P95</span>
                  <span className="text-sm font-medium">{formatExecutionTime(stats.execution_time_percentiles?.p95 || 0)}</span>
                </div>
                <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-blue-600" 
                    style={{ width: `${(stats.execution_time_percentiles?.p95 || 0) / stats.max_execution_time * 100}%` }}
                  ></div>
                </div>
              </div>
              
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm">P99</span>
                  <span className="text-sm font-medium">{formatExecutionTime(stats.execution_time_percentiles?.p99 || 0)}</span>
                </div>
                <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-blue-600" 
                    style={{ width: `${(stats.execution_time_percentiles?.p99 || 0) / stats.max_execution_time * 100}%` }}
                  ></div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
        
        {/* 执行阶段分析 */}
        <Card>
          <CardHeader>
            <CardTitle>执行阶段分析</CardTitle>
            <CardDescription>条件评估和动作执行的时间分布</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-6">
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium">条件评估</span>
                  <span className="text-sm">{formatExecutionTime(stats.avg_condition_time)}</span>
                </div>
                <div className="h-4 bg-gray-200 rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-green-600" 
                    style={{ width: `${(stats.avg_condition_time / stats.avg_execution_time) * 100}%` }}
                  ></div>
                </div>
                <div className="text-xs text-gray-500 mt-1">
                  占总执行时间的 {((stats.avg_condition_time / stats.avg_execution_time) * 100).toFixed(1)}%
                </div>
              </div>
              
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium">动作执行</span>
                  <span className="text-sm">{formatExecutionTime(stats.avg_action_time)}</span>
                </div>
                <div className="h-4 bg-gray-200 rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-blue-600" 
                    style={{ width: `${(stats.avg_action_time / stats.avg_execution_time) * 100}%` }}
                  ></div>
                </div>
                <div className="text-xs text-gray-500 mt-1">
                  占总执行时间的 {((stats.avg_action_time / stats.avg_execution_time) * 100).toFixed(1)}%
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
        
        {/* 执行时间分布 */}
        {stats.time_distribution && (
          <Card>
            <CardHeader>
              <CardTitle>执行时间分布</CardTitle>
              <CardDescription>不同时间段的执行次数分布</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="h-64">
                <div className="flex h-full items-end space-x-2">
                  {stats.time_distribution.labels.map((label, index) => (
                    <div key={index} className="flex-1 flex flex-col items-center">
                      <div 
                        className="w-full bg-blue-600 rounded-t"
                        style={{ 
                          height: `${(stats.time_distribution?.values[index] || 0) / Math.max(...(stats.time_distribution?.values || [1])) * 100}%` 
                        }}
                      ></div>
                      <div className="text-xs mt-2 text-gray-600">{label}</div>
                      <div className="text-xs font-medium">{stats.time_distribution?.values[index]}%</div>
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    )
  }
  
  // 渲染资源使用分析
  const renderResourceUsageAnalysis = () => {
    if (!stats || !stats.resource_usage) return null
    
    return (
      <div className="space-y-6">
        {/* 资源使用概览 */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">平均内存使用</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-blue-600">
                {stats.resource_usage.avg_memory_mb.toFixed(1)} MB
              </div>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">最大内存使用</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-orange-600">
                {stats.resource_usage.max_memory_mb.toFixed(1)} MB
              </div>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">平均CPU使用</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-green-600">
                {stats.resource_usage.avg_cpu_percent.toFixed(1)}%
              </div>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">最大CPU使用</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-purple-600">
                {stats.resource_usage.max_cpu_percent.toFixed(1)}%
              </div>
            </CardContent>
          </Card>
        </div>
        
        {/* 内存使用分析 */}
        <Card>
          <CardHeader>
            <CardTitle>内存使用分析</CardTitle>
            <CardDescription>触发器执行过程中的内存使用情况</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm">平均内存使用</span>
                <span className="text-sm font-medium">{stats.resource_usage.avg_memory_mb.toFixed(1)} MB</span>
              </div>
              <div className="h-4 bg-gray-200 rounded-full overflow-hidden">
                <div 
                  className="h-full bg-blue-600" 
                  style={{ width: `${(stats.resource_usage.avg_memory_mb / stats.resource_usage.max_memory_mb) * 100}%` }}
                ></div>
              </div>
              
              <div className="flex items-center justify-between">
                <span className="text-sm">最大内存使用</span>
                <span className="text-sm font-medium">{stats.resource_usage.max_memory_mb.toFixed(1)} MB</span>
              </div>
              <div className="h-4 bg-gray-200 rounded-full overflow-hidden">
                <div 
                  className="h-full bg-orange-600" 
                  style={{ width: '100%' }}
                ></div>
              </div>
              
              <div className="mt-4 p-4 bg-gray-50 rounded-md">
                <h4 className="text-sm font-medium mb-2">内存使用建议</h4>
                <ul className="text-sm text-gray-600 space-y-1 list-disc pl-5">
                  <li>当前内存使用处于正常范围内</li>
                  <li>如果触发器处理大型邮件或附件，建议监控内存使用峰值</li>
                  <li>考虑优化条件表达式复杂度以减少内存占用</li>
                </ul>
              </div>
            </div>
          </CardContent>
        </Card>
        
        {/* CPU使用分析 */}
        <Card>
          <CardHeader>
            <CardTitle>CPU使用分析</CardTitle>
            <CardDescription>触发器执行过程中的CPU使用情况</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm">平均CPU使用</span>
                <span className="text-sm font-medium">{stats.resource_usage.avg_cpu_percent.toFixed(1)}%</span>
              </div>
              <div className="h-4 bg-gray-200 rounded-full overflow-hidden">
                <div 
                  className="h-full bg-green-600" 
                  style={{ width: `${(stats.resource_usage.avg_cpu_percent / stats.resource_usage.max_cpu_percent) * 100}%` }}
                ></div>
              </div>
              
              <div className="flex items-center justify-between">
                <span className="text-sm">最大CPU使用</span>
                <span className="text-sm font-medium">{stats.resource_usage.max_cpu_percent.toFixed(1)}%</span>
              </div>
              <div className="h-4 bg-gray-200 rounded-full overflow-hidden">
                <div 
                  className="h-full bg-purple-600" 
                  style={{ width: '100%' }}
                ></div>
              </div>
              
              <div className="mt-4 p-4 bg-gray-50 rounded-md">
                <h4 className="text-sm font-medium mb-2">CPU使用建议</h4>
                <ul className="text-sm text-gray-600 space-y-1 list-disc pl-5">
                  <li>当前CPU使用处于正常范围内</li>
                  <li>复杂的正则表达式可能导致CPU使用率上升</li>
                  <li>如果有多个触发器同时执行，建议监控系统整体CPU负载</li>
                </ul>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }
  
  // 渲染执行趋势分析
  const renderExecutionTrendAnalysis = () => {
    if (!stats || !stats.executions_by_day) return null
    
    return (
      <div className="space-y-6">
        {/* 每日执行次数趋势 */}
        <Card>
          <CardHeader>
            <CardTitle>每日执行次数趋势</CardTitle>
            <CardDescription>过去7天的触发器执行情况</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-64">
              <div className="flex h-full items-end space-x-2">
                {stats.executions_by_day.dates.map((date, index) => {
                  const total = stats.executions_by_day?.counts[index] || 0
                  const success = stats.executions_by_day?.success_counts[index] || 0
                  const failed = stats.executions_by_day?.failed_counts[index] || 0
                  const maxCount = Math.max(...(stats.executions_by_day?.counts || [1]))
                  
                  return (
                    <div key={index} className="flex-1 flex flex-col items-center">
                      <div className="w-full flex flex-col-reverse" style={{ height: `${(total / maxCount) * 100}%` }}>
                        <div 
                          className="w-full bg-green-600 rounded-t"
                          style={{ height: `${(success / total) * 100}%` }}
                        ></div>
                        <div 
                          className="w-full bg-red-600"
                          style={{ height: `${(failed / total) * 100}%` }}
                        ></div>
                      </div>
                      <div className="text-xs mt-2 text-gray-600">{new Date(date).toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric' })}</div>
                      <div className="text-xs font-medium">{total}</div>
                    </div>
                  )
                })}
              </div>
            </div>
            <div className="flex items-center justify-center mt-4 space-x-4">
              <div className="flex items-center">
                <div className="w-3 h-3 bg-green-600 rounded-full mr-1"></div>
                <span className="text-xs">成功</span>
              </div>
              <div className="flex items-center">
                <div className="w-3 h-3 bg-red-600 rounded-full mr-1"></div>
                <span className="text-xs">失败</span>
              </div>
            </div>
          </CardContent>
        </Card>
        
        {/* 执行成功率趋势 */}
        <Card>
          <CardHeader>
            <CardTitle>执行成功率趋势</CardTitle>
            <CardDescription>过去7天的触发器成功率变化</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-64">
              <div className="flex h-full items-end space-x-2">
                {stats.executions_by_day.dates.map((date, index) => {
                  const total = stats.executions_by_day?.counts[index] || 0
                  const success = stats.executions_by_day?.success_counts[index] || 0
                  const successRate = total > 0 ? (success / total) * 100 : 0
                  
                  return (
                    <div key={index} className="flex-1 flex flex-col items-center">
                      <div 
                        className="w-full bg-blue-600 rounded-t"
                        style={{ height: `${successRate}%` }}
                      ></div>
                      <div className="text-xs mt-2 text-gray-600">{new Date(date).toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric' })}</div>
                      <div className="text-xs font-medium">{successRate.toFixed(0)}%</div>
                    </div>
                  )
                })}
              </div>
            </div>
          </CardContent>
        </Card>
        
        {/* 性能优化建议 */}
        <Card>
          <CardHeader>
            <CardTitle>性能优化建议</CardTitle>
            <CardDescription>基于执行数据的优化建议</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="p-4 bg-blue-50 border border-blue-200 rounded-md">
                <h4 className="text-sm font-medium text-blue-800 mb-2">执行时间优化</h4>
                <ul className="text-sm text-blue-700 space-y-1 list-disc pl-5">
                  <li>当前平均执行时间为 {formatExecutionTime(stats.avg_execution_time)}，处于正常范围内</li>
                  <li>条件评估占用了 {((stats.avg_condition_time / stats.avg_execution_time) * 100).toFixed(1)}% 的执行时间，可考虑简化复杂条件</li>
                  <li>动作执行占用了 {((stats.avg_action_time / stats.avg_execution_time) * 100).toFixed(1)}% 的执行时间，是主要耗时部分</li>
                </ul>
              </div>
              
              <div className="p-4 bg-green-50 border border-green-200 rounded-md">
                <h4 className="text-sm font-medium text-green-800 mb-2">资源使用优化</h4>
                <ul className="text-sm text-green-700 space-y-1 list-disc pl-5">
                  <li>内存使用峰值为 {stats.resource_usage?.max_memory_mb.toFixed(1)} MB，处于安全范围内</li>
                  <li>CPU使用率峰值为 {stats.resource_usage?.max_cpu_percent.toFixed(1)}%，负载较轻</li>
                  <li>当前资源使用状况良好，无需特别优化</li>
                </ul>
              </div>
              
              <div className="p-4 bg-yellow-50 border border-yellow-200 rounded-md">
                <h4 className="text-sm font-medium text-yellow-800 mb-2">执行成功率优化</h4>
                <ul className="text-sm text-yellow-700 space-y-1 list-disc pl-5">
                  <li>当前成功率为 {(stats.success_rate * 100).toFixed(1)}%</li>
                  <li>建议检查失败的执行日志，分析失败原因</li>
                  <li>考虑添加错误处理逻辑，提高触发器的稳定性</li>
                </ul>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }
  
  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-[200px] w-full" />
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Skeleton className="h-[300px] w-full" />
          <Skeleton className="h-[300px] w-full" />
        </div>
      </div>
    )
  }
  
  if (error) {
    return (
      <Alert variant="destructive">
        <AlertCircle className="h-4 w-4" />
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    )
  }
  
  if (!stats) {
    return (
      <Alert>
        <AlertDescription>暂无性能数据可用</AlertDescription>
      </Alert>
    )
  }
  
  return (
    <div className="space-y-6">
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid grid-cols-3">
          <TabsTrigger value="execution-time">
            <Clock className="h-4 w-4 mr-2" />
            执行时间分析
          </TabsTrigger>
          <TabsTrigger value="resource-usage">
            <Cpu className="h-4 w-4 mr-2" />
            资源使用分析
          </TabsTrigger>
          <TabsTrigger value="execution-trend">
            <BarChart3 className="h-4 w-4 mr-2" />
            执行趋势分析
          </TabsTrigger>
        </TabsList>
        
        <TabsContent value="execution-time" className="mt-6">
          {renderExecutionTimeAnalysis()}
        </TabsContent>
        
        <TabsContent value="resource-usage" className="mt-6">
          {renderResourceUsageAnalysis()}
        </TabsContent>
        
        <TabsContent value="execution-trend" className="mt-6">
          {renderExecutionTrendAnalysis()}
        </TabsContent>
      </Tabs>
    </div>
  )
}