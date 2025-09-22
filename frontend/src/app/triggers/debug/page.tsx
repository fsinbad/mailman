'use client'

import { useState, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink } from '@/components/ui/breadcrumb'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { AlertCircle, ArrowLeft, Home } from 'lucide-react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { TriggerDebugger } from '@/components/triggers/trigger-debugger'
import { triggerService } from '@/services/trigger.service'
import { EmailTrigger } from '@/types'

export default function TriggerDebugPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const triggerId = searchParams.get('id')
  
  const [trigger, setTrigger] = useState<EmailTrigger | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  
  useEffect(() => {
    async function loadTrigger() {
      if (!triggerId) {
        setError('未提供触发器ID')
        setLoading(false)
        return
      }
      
      try {
        const id = parseInt(triggerId)
        const triggerData = await triggerService.getTrigger(id)
        setTrigger(triggerData)
      } catch (err: any) {
        console.error('加载触发器失败:', err)
        setError(err.message || '加载触发器失败')
      } finally {
        setLoading(false)
      }
    }
    
    loadTrigger()
  }, [triggerId])
  
  const handleBack = () => {
    router.back()
  }
  
  return (
    <div className="container py-6 space-y-6">
      <div className="flex justify-between items-center">
        <Breadcrumb>
          <BreadcrumbItem>
            <BreadcrumbLink href="/triggers">
              <Home className="h-4 w-4 mr-1" />
              触发器
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbItem>
            <BreadcrumbLink>调试器</BreadcrumbLink>
          </BreadcrumbItem>
        </Breadcrumb>
        
        <Button variant="outline" size="sm" onClick={handleBack}>
          <ArrowLeft className="h-4 w-4 mr-1" />
          返回
        </Button>
      </div>
      
      {loading ? (
        <Card>
          <CardContent className="py-6">
            <div className="space-y-4">
              <Skeleton className="h-8 w-1/3" />
              <Skeleton className="h-4 w-1/2" />
              <div className="space-y-2">
                <Skeleton className="h-20 w-full" />
                <Skeleton className="h-20 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            </div>
          </CardContent>
        </Card>
      ) : error ? (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>加载失败</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : trigger ? (
        <TriggerDebugger trigger={trigger} />
      ) : (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>未找到触发器</AlertTitle>
          <AlertDescription>无法加载指定的触发器</AlertDescription>
        </Alert>
      )}
    </div>
  )
}