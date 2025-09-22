'use client'

import { useState, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink } from '@/components/ui/breadcrumb'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { AlertCircle, ArrowLeft, Home } from 'lucide-react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { ErrorDiagnostics } from '@/components/triggers/error-diagnostics'
import { triggerService } from '@/services/trigger.service'
import { TriggerExecutionLog } from '@/types'

export default function DiagnosticsPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const logId = searchParams.get('logId')
  const triggerId = searchParams.get('triggerId')
  const errorParam = searchParams.get('error')
  
  const [log, setLog] = useState<TriggerExecutionLog | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  
  useEffect(() => {
    if (logId) {
      loadLog(parseInt(logId))
    }
  }, [logId])
  
  const loadLog = async (id: number) => {
    try {
      setLoading(true)
      setError(null)
      
      const logData = await triggerService.getTriggerLog(id)
      setLog(logData)
    } catch (err: any) {
      console.error('加载日志失败:', err)
      setError(err.message || '加载日志失败')
    } finally {
      setLoading(false)
    }
  }
  
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
          {triggerId && (
            <BreadcrumbItem>
              <BreadcrumbLink href={`/triggers/${triggerId}`}>
                触发器详情
              </BreadcrumbLink>
            </BreadcrumbItem>
          )}
          {logId && (
            <BreadcrumbItem>
              <BreadcrumbLink href={`/triggers/logs/${logId}`}>
                执行日志
              </BreadcrumbLink>
            </BreadcrumbItem>
          )}
          <BreadcrumbItem>
            <BreadcrumbLink>错误诊断</BreadcrumbLink>
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
      ) : (
        <ErrorDiagnostics 
          triggerId={triggerId ? parseInt(triggerId) : undefined}
          logId={logId ? parseInt(logId) : undefined}
          error={errorParam || log?.error || undefined}
        />
      )}
    </div>
  )
}