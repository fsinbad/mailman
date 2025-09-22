'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { TriggerLogs } from '@/components/triggers/trigger-logs'
import { TriggerExecutionLog } from '@/types'
import { ArrowLeft, BarChart3, Download } from 'lucide-react'

export default function LogsPage() {
  const router = useRouter()
  
  // 处理查看日志详情
  const handleViewDetails = (log: TriggerExecutionLog) => {
    router.push(`/triggers/logs/${log.id}`)
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
          <h1 className="text-2xl font-bold">触发器执行日志</h1>
        </div>
        
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => router.push('/triggers/stats')}>
            <BarChart3 className="h-4 w-4 mr-1" />
            统计分析
          </Button>
        </div>
      </div>
      
      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex flex-col">
              <span className="text-sm text-gray-600 dark:text-gray-400">总执行次数</span>
              <span className="text-2xl font-bold">-</span>
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="p-4">
            <div className="flex flex-col">
              <span className="text-sm text-gray-600 dark:text-gray-400">成功次数</span>
              <span className="text-2xl font-bold text-green-600">-</span>
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="p-4">
            <div className="flex flex-col">
              <span className="text-sm text-gray-600 dark:text-gray-400">失败次数</span>
              <span className="text-2xl font-bold text-red-600">-</span>
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="p-4">
            <div className="flex flex-col">
              <span className="text-sm text-gray-600 dark:text-gray-400">平均执行时间</span>
              <span className="text-2xl font-bold text-blue-600">-</span>
            </div>
          </CardContent>
        </Card>
      </div>
      
      {/* 日志列表 */}
      <Card>
        <CardHeader>
          <CardTitle>执行日志列表</CardTitle>
        </CardHeader>
        <CardContent>
          <TriggerLogs 
            showFilters={true}
            showPagination={true}
            showExport={true}
            onViewDetails={handleViewDetails}
          />
        </CardContent>
      </Card>
    </div>
  )
}