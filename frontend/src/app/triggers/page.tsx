'use client'

import { useState } from 'react'
import { TriggerList } from '@/components/triggers/trigger-list'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Plus, Zap } from 'lucide-react'
import { EmailTrigger } from '@/types'
import { useRouter } from 'next/navigation'

export default function TriggersPage() {
  const router = useRouter()
  
  // 处理编辑触发器
  const handleEdit = (trigger: EmailTrigger) => {
    router.push(`/triggers/edit/${trigger.id}`)
  }
  
  // 处理查看触发器详情
  const handleView = (trigger: EmailTrigger) => {
    router.push(`/triggers/${trigger.id}`)
  }
  
  // 处理调试触发器
  const handleDebug = (trigger: EmailTrigger) => {
    router.push(`/triggers/debug?id=${trigger.id}`)
  }
  
  // 处理创建新触发器
  const handleCreate = () => {
    router.push('/triggers/create')
  }

  return (
    <div className="space-y-6 p-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            <Zap className="inline-block mr-2 h-6 w-6 text-blue-600" />
            邮件触发器
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            智能邮件处理规则，让您的邮箱管理更高效
          </p>
        </div>
        <Button className="bg-blue-600 hover:bg-blue-700 text-white" onClick={handleCreate}>
          <Plus className="h-4 w-4 mr-2" />
          创建新规则
        </Button>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600 dark:text-gray-400">总规则数</p>
                <p className="text-2xl font-bold text-gray-900 dark:text-white">0</p>
              </div>
              <Zap className="h-8 w-8 text-blue-600" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600 dark:text-gray-400">运行中</p>
                <p className="text-2xl font-bold text-green-600">0</p>
              </div>
              <Zap className="h-8 w-8 text-green-600" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600 dark:text-gray-400">已处理邮件</p>
                <p className="text-2xl font-bold text-purple-600">0</p>
              </div>
              <Zap className="h-8 w-8 text-purple-600" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-600 dark:text-gray-400">成功率</p>
                <p className="text-2xl font-bold text-orange-600">0%</p>
              </div>
              <Zap className="h-8 w-8 text-orange-600" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 触发器列表 */}
      <TriggerList 
        onEdit={handleEdit}
        onView={handleView}
        onDebug={handleDebug}
      />
    </div>
  )
}