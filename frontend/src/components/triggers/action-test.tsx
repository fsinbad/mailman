'use client'

import React, { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Loader2, CheckCircle, AlertCircle } from 'lucide-react'
import { TriggerActionConfig } from '@/types'
import { triggerService } from '@/services/trigger.service'

interface ActionTestProps {
  action: TriggerActionConfig
}

export function ActionTest({ action }: ActionTestProps) {
  const [testData, setTestData] = useState<string>(`{
  "ID": 12345,
  "MessageID": "<test-message-id@example.com>",
  "AccountID": 1,
  "Subject": "测试邮件主题",
  "From": ["sender@example.com"],
  "To": ["recipient@example.com"],
  "Date": "${new Date().toISOString()}",
  "Body": "这是一封测试邮件的正文内容。",
  "HTMLBody": "<p>这是一封测试邮件的HTML正文内容。</p>",
  "MailboxName": "INBOX",
  "Flags": ["\\\\Seen"],
  "Size": 1024
}`)
  const [isLoading, setIsLoading] = useState(false)
  const [result, setResult] = useState<any>(null)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<boolean>(false)

  const handleTest = async () => {
    try {
      setIsLoading(true)
      setError(null)
      setResult(null)
      setSuccess(false)

      // 解析测试数据
      const emailData = JSON.parse(testData)
      
      // 调用API测试动作
      const response = await triggerService.testTriggerAction(action, emailData)
      
      setResult(response.result)
      setSuccess(!response.error)
      if (response.error) {
        setError(response.error)
      }
    } catch (err: any) {
      setError(err.message || '测试动作时发生错误')
      setSuccess(false)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <span>动作测试</span>
          <span className="text-sm font-normal text-gray-500">
            ({action.name} - {action.type})
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <label className="text-sm font-medium">测试数据 (JSON)</label>
          <Textarea
            value={testData}
            onChange={(e) => setTestData(e.target.value)}
            rows={10}
            className="font-mono text-sm"
            placeholder="输入测试邮件数据 (JSON 格式)"
          />
          <p className="text-xs text-gray-500">
            提供一个邮件对象的 JSON 数据，用于测试动作执行效果
          </p>
        </div>

        <Button 
          onClick={handleTest} 
          disabled={isLoading}
          className="w-full"
        >
          {isLoading ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              测试中...
            </>
          ) : (
            '测试动作'
          )}
        </Button>

        {error && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>测试失败</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {success && (
          <Alert variant="default" className="bg-green-50 border-green-200 text-green-800">
            <CheckCircle className="h-4 w-4 text-green-600" />
            <AlertTitle>测试成功</AlertTitle>
            <AlertDescription>动作执行成功</AlertDescription>
          </Alert>
        )}

        {result && (
          <div className="space-y-2">
            <h4 className="text-sm font-medium">执行结果:</h4>
            <div className="bg-gray-50 p-3 rounded-md overflow-auto max-h-60">
              <pre className="text-xs">{JSON.stringify(result, null, 2)}</pre>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}