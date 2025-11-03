'use client'

import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Textarea } from '@/components/ui/textarea'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Loader2, CheckCircle, AlertCircle, Clock, ArrowRight } from 'lucide-react'
import { TriggerActionConfig } from '@/types'
import { actionTestService } from '@/services/action-test.service'

interface ActionDebugProps {
  action: TriggerActionConfig
}

export function ActionDebug({ action }: ActionDebugProps) {
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
  const [executionTime, setExecutionTime] = useState<number | null>(null)
  const [testHistory, setTestHistory] = useState<Array<{
    timestamp: Date
    success: boolean
    executionTime?: number
    error?: string
    result?: any
  }>>([])
  const [activeTab, setActiveTab] = useState('test')

  // 加载测试邮件模板
  const loadTemplate = (template: 'simple' | 'detailed' | 'html' | 'attachment') => {
    switch (template) {
      case 'simple':
        setTestData(`{
  "ID": 12345,
  "MessageID": "<test-message-id@example.com>",
  "AccountID": 1,
  "Subject": "测试邮件主题",
  "From": ["sender@example.com"],
  "To": ["recipient@example.com"],
  "Date": "${new Date().toISOString()}",
  "Body": "这是一封测试邮件的正文内容。",
  "MailboxName": "INBOX",
  "Size": 1024
}`)
        break
      case 'detailed':
        setTestData(`{
  "ID": 12345,
  "MessageID": "<test-message-id@example.com>",
  "AccountID": 1,
  "Subject": "重要客户询价单 - 请尽快处理",
  "From": ["important-client@example.com"],
  "To": ["sales@yourcompany.com"],
  "Cc": ["manager@yourcompany.com"],
  "Date": "${new Date().toISOString()}",
  "Body": "尊敬的销售团队，\\n\\n我们正在考虑购买贵公司的产品，请提供最新的报价单和产品规格。\\n\\n我们需要在下周五之前收到回复。\\n\\n谢谢！\\n\\n客户团队",
  "HTMLBody": "<p>尊敬的销售团队，</p><p>我们正在考虑购买贵公司的产品，请提供最新的报价单和产品规格。</p><p>我们需要在下周五之前收到回复。</p><p>谢谢！</p><p>客户团队</p>",
  "MailboxName": "INBOX",
  "Flags": [],
  "Size": 2048
}`)
        break
      case 'html':
        setTestData(`{
  "ID": 12345,
  "MessageID": "<newsletter@example.com>",
  "AccountID": 1,
  "Subject": "每周通讯 - 行业最新动态",
  "From": ["newsletter@example.com"],
  "To": ["subscriber@yourcompany.com"],
  "Date": "${new Date().toISOString()}",
  "Body": "此邮件包含HTML内容，请使用支持HTML的邮件客户端查看。",
  "HTMLBody": "<html><head><style>body{font-family:Arial,sans-serif;} .header{background-color:#4a86e8;color:white;padding:20px;} .content{padding:15px;} .footer{background-color:#f3f3f3;padding:10px;font-size:12px;}</style></head><body><div class='header'><h1>每周通讯</h1></div><div class='content'><h2>行业最新动态</h2><p>这是一封包含<strong>HTML格式</strong>的测试邮件，用于测试HTML内容的处理。</p><ul><li>新闻项目1</li><li>新闻项目2</li><li>新闻项目3</li></ul></div><div class='footer'>如需退订，请点击<a href='#'>这里</a></div></body></html>",
  "MailboxName": "INBOX",
  "Flags": ["\\\\Seen"],
  "Size": 4096
}`)
        break
      case 'attachment':
        setTestData(`{
  "ID": 12345,
  "MessageID": "<attachment@example.com>",
  "AccountID": 1,
  "Subject": "包含附件的测试邮件",
  "From": ["sender@example.com"],
  "To": ["recipient@yourcompany.com"],
  "Date": "${new Date().toISOString()}",
  "Body": "这封邮件包含附件，请查收。",
  "HTMLBody": "<p>这封邮件包含附件，请查收。</p>",
  "MailboxName": "INBOX",
  "Flags": [],
  "Size": 8192,
  "Attachments": [
    {
      "id": 1001,
      "email_id": 12345,
      "filename": "document.pdf",
      "content_type": "application/pdf",
      "size": 5242880
    },
    {
      "id": 1002,
      "email_id": 12345,
      "filename": "image.jpg",
      "content_type": "image/jpeg",
      "size": 2097152
    }
  ]
}`)
        break
    }
  }

  const handleTest = async () => {
    try {
      setIsLoading(true)
      setError(null)
      setResult(null)
      setSuccess(false)
      setExecutionTime(null)

      // 解析测试数据
      const emailData = JSON.parse(testData)
      
      // 记录开始时间
      const startTime = performance.now()

      // 调用API测试动作
      const response = await actionTestService.testAction(action, emailData)

      // 计算执行时间
      const executionTimeMs = performance.now() - startTime

      setResult(response.result)
      setSuccess(response.success)
      setExecutionTime(response.executionTime || null)

      if (response.error) {
        setError(response.error)
      }

      // 添加到测试历史
      setTestHistory(prev => [
        {
          timestamp: new Date(),
          success: !response.error,
          executionTime: executionTimeMs,
          error: response.error,
          result: response.result
        },
        ...prev.slice(0, 9) // 只保留最近10条记录
      ])
      
      // 切换到结果选项卡
      setActiveTab('result')
    } catch (err: any) {
      setError(err.message || '测试动作时发生错误')
      setSuccess(false)
      
      // 添加到测试历史
      setTestHistory(prev => [
        {
          timestamp: new Date(),
          success: false,
          error: err.message || '测试动作时发生错误'
        },
        ...prev.slice(0, 9)
      ])
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span>动作调试器</span>
          <Badge variant={action.enabled ? 'default' : 'secondary'}>
            {action.enabled ? '已启用' : '已禁用'}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList className="grid grid-cols-3">
            <TabsTrigger value="test">测试数据</TabsTrigger>
            <TabsTrigger value="result">执行结果</TabsTrigger>
            <TabsTrigger value="history">测试历史</TabsTrigger>
          </TabsList>
          
          <TabsContent value="test" className="space-y-4 pt-4">
            <div className="flex items-center justify-between">
              <Label>测试邮件数据 (JSON)</Label>
              <div className="flex items-center gap-2">
                <Button 
                  variant="outline" 
                  size="sm" 
                  onClick={() => loadTemplate('simple')}
                >
                  简单模板
                </Button>
                <Button 
                  variant="outline" 
                  size="sm" 
                  onClick={() => loadTemplate('detailed')}
                >
                  详细模板
                </Button>
                <Button 
                  variant="outline" 
                  size="sm" 
                  onClick={() => loadTemplate('html')}
                >
                  HTML模板
                </Button>
                <Button 
                  variant="outline" 
                  size="sm" 
                  onClick={() => loadTemplate('attachment')}
                >
                  附件模板
                </Button>
              </div>
            </div>
            
            <Textarea
              value={testData}
              onChange={(e) => setTestData(e.target.value)}
              rows={15}
              className="font-mono text-sm"
              placeholder="输入测试邮件数据 (JSON 格式)"
            />
            
            <div className="flex justify-between items-center">
              <p className="text-xs text-gray-500">
                提供一个邮件对象的 JSON 数据，用于测试动作执行效果
              </p>
              
              <Button 
                onClick={handleTest} 
                disabled={isLoading}
                className="bg-green-500 hover:bg-green-600"
              >
                {isLoading ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    测试中...
                  </>
                ) : (
                  '执行测试'
                )}
              </Button>
            </div>
          </TabsContent>
          
          <TabsContent value="result" className="space-y-4 pt-4">
            {isLoading ? (
              <div className="flex items-center justify-center p-12">
                <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
                <span className="ml-3 text-lg text-gray-500">执行测试中...</span>
              </div>
            ) : (
              <>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    {success ? (
                      <CheckCircle className="h-5 w-5 text-green-500" />
                    ) : (
                      <AlertCircle className="h-5 w-5 text-red-500" />
                    )}
                    <h3 className="text-lg font-medium">
                      {success ? '测试成功' : '测试失败'}
                    </h3>
                  </div>
                  
                  {executionTime !== null && (
                    <div className="flex items-center gap-1 text-gray-500">
                      <Clock className="h-4 w-4" />
                      <span className="text-sm">{executionTime} ms</span>
                    </div>
                  )}
                </div>
                
                {error && (
                  <Alert variant="destructive">
                    <AlertCircle className="h-4 w-4" />
                    <AlertTitle>错误信息</AlertTitle>
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}
                
                {result && (
                  <div className="space-y-2">
                    <Label>执行结果</Label>
                    <div className="bg-gray-50 p-4 rounded-md overflow-auto max-h-80">
                      <pre className="text-sm">{JSON.stringify(result, null, 2)}</pre>
                    </div>
                  </div>
                )}
                
                <div className="flex justify-end">
                  <Button 
                    variant="outline" 
                    onClick={() => setActiveTab('test')}
                  >
                    返回测试
                  </Button>
                </div>
              </>
            )}
          </TabsContent>
          
          <TabsContent value="history" className="space-y-4 pt-4">
            {testHistory.length === 0 ? (
              <div className="text-center p-8 text-gray-500">
                暂无测试历史记录
              </div>
            ) : (
              <div className="space-y-3">
                {testHistory.map((test, index) => (
                  <Card key={index} className={`border-l-4 ${test.success ? 'border-l-green-500' : 'border-l-red-500'}`}>
                    <CardContent className="p-4">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          {test.success ? (
                            <CheckCircle className="h-4 w-4 text-green-500" />
                          ) : (
                            <AlertCircle className="h-4 w-4 text-red-500" />
                          )}
                          <span className="font-medium">
                            {test.success ? '成功' : '失败'}
                          </span>
                        </div>
                        
                        <div className="text-sm text-gray-500">
                          {test.timestamp.toLocaleTimeString()}
                        </div>
                      </div>
                      
                      <div className="mt-2 flex items-center gap-4 text-sm">
                        {test.executionTime && (
                          <div className="flex items-center gap-1 text-gray-500">
                            <Clock className="h-3 w-3" />
                            <span>{test.executionTime} ms</span>
                          </div>
                        )}
                      </div>
                      
                      {test.error && (
                        <div className="mt-2 text-sm text-red-600">
                          {test.error}
                        </div>
                      )}
                      
                      {test.result && (
                        <div className="mt-2 flex items-center gap-1">
                          <Button 
                            variant="ghost" 
                            size="sm"
                            className="h-6 text-xs"
                            onClick={() => {
                              setResult(test.result)
                              setSuccess(test.success)
                              setError(test.error || null)
                              setExecutionTime(test.executionTime || null)
                              setActiveTab('result')
                            }}
                          >
                            查看结果
                            <ArrowRight className="ml-1 h-3 w-3" />
                          </Button>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}