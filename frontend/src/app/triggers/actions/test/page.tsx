'use client'

import { useState, useEffect } from 'react'
import { useSearchParams } from 'next/navigation'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { ActionTest } from '@/components/triggers/action-test'
import { ActionDebug } from '@/components/triggers/action-debug'
import { ActionConfigDialog } from '@/components/triggers/action-config-dialog'
import { TriggerActionConfig } from '@/types'
import { ArrowLeft, Settings } from 'lucide-react'
import Link from 'next/link'

export default function ActionTestPage() {
  const searchParams = useSearchParams()
  const [action, setAction] = useState<TriggerActionConfig>({
    type: 'modify_content',
    name: '测试动作',
    description: '用于测试的动作配置',
    config: '{}',
    enabled: true,
    order: 0
  })
  const [isConfigDialogOpen, setIsConfigDialogOpen] = useState(false)
  
  // 从URL参数中获取动作配置
  useEffect(() => {
    const actionParam = searchParams.get('action')
    if (actionParam) {
      try {
        const decodedAction = JSON.parse(decodeURIComponent(actionParam))
        setAction(decodedAction)
      } catch (e) {
        console.error('无法解析动作参数:', e)
      }
    }
  }, [searchParams])

  return (
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Link href="/triggers">
            <Button variant="ghost" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              返回触发器列表
            </Button>
          </Link>
          <h1 className="text-2xl font-bold">动作测试</h1>
        </div>
        
        <Button 
          variant="outline" 
          onClick={() => setIsConfigDialogOpen(true)}
        >
          <Settings className="h-4 w-4 mr-2" />
          配置动作
        </Button>
      </div>
      
      <div className="grid grid-cols-1 gap-6">
        <Card>
          <CardHeader>
            <CardTitle>当前动作配置</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div>
                <span className="font-medium">名称:</span> {action.name}
              </div>
              <div>
                <span className="font-medium">类型:</span> {action.type}
              </div>
              {action.description && (
                <div>
                  <span className="font-medium">描述:</span> {action.description}
                </div>
              )}
              <div>
                <span className="font-medium">配置:</span>
                <pre className="mt-2 bg-gray-50 p-3 rounded-md text-xs overflow-auto">
                  {action.config}
                </pre>
              </div>
            </div>
          </CardContent>
        </Card>
        
        <ActionDebug action={action} />
      </div>
      
      <ActionConfigDialog
        action={action}
        isOpen={isConfigDialogOpen}
        onClose={() => setIsConfigDialogOpen(false)}
        onSave={(updatedAction) => {
          setAction(updatedAction)
          setIsConfigDialogOpen(false)
        }}
      />
    </div>
  )
}