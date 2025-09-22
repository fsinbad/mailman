'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { 
  AlertCircle, 
  CheckCircle2, 
  Search, 
  Code, 
  FileText, 
  MessageSquare, 
  ArrowRight,
  RefreshCw,
  HelpCircle
} from 'lucide-react'
import { TriggerExecutionLog } from '@/types'
import { triggerService } from '@/services/trigger.service'

interface ErrorDiagnosticsProps {
  triggerId?: number
  logId?: number
  error?: string
  onClose?: () => void
}

interface ErrorAnalysisResult {
  errorType: string
  errorMessage: string
  possibleCauses: string[]
  suggestedSolutions: string[]
  relatedComponents: string[]
  severity: 'low' | 'medium' | 'high' | 'critical'
  code?: string
  additionalInfo?: string
}

export function ErrorDiagnostics({ triggerId, logId, error, onClose }: ErrorDiagnosticsProps) {
  const [isLoading, setIsLoading] = useState(false)
  const [errorLog, setErrorLog] = useState<TriggerExecutionLog | null>(null)
  const [errorText, setErrorText] = useState<string>(error || '')
  const [analysisResult, setAnalysisResult] = useState<ErrorAnalysisResult | null>(null)
  const [activeTab, setActiveTab] = useState('analysis')
  
  // 加载错误日志
  useEffect(() => {
    if (logId) {
      loadErrorLog(logId)
    }
  }, [logId])
  
  // 如果提供了错误文本，直接分析
  useEffect(() => {
    if (error && error.trim() !== '') {
      setErrorText(error)
      analyzeError(error)
    }
  }, [error])
  
  // 加载错误日志
  const loadErrorLog = async (id: number) => {
    try {
      setIsLoading(true)
      const log = await triggerService.getTriggerLog(id)
      setErrorLog(log)
      
      if (log.error) {
        setErrorText(log.error)
        analyzeError(log.error)
      }
    } catch (err) {
      console.error('加载错误日志失败:', err)
    } finally {
      setIsLoading(false)
    }
  }
  
  // 分析错误
  const analyzeError = (errorMessage: string) => {
    setIsLoading(true)
    
    // 模拟API调用延迟
    setTimeout(() => {
      // 错误类型识别
      let result: ErrorAnalysisResult = {
        errorType: '未知错误',
        errorMessage: errorMessage,
        possibleCauses: [],
        suggestedSolutions: [],
        relatedComponents: [],
        severity: 'medium'
      }
      
      // 条件表达式错误
      if (errorMessage.includes('condition') || errorMessage.includes('expression') || errorMessage.includes('syntax')) {
        result = {
          errorType: '条件表达式错误',
          errorMessage: errorMessage,
          possibleCauses: [
            '条件表达式语法错误',
            '使用了不存在的字段或属性',
            '比较运算符使用不当',
            '逻辑运算符使用不当'
          ],
          suggestedSolutions: [
            '检查条件表达式的语法',
            '确认所有字段名称拼写正确',
            '确认比较运算符使用正确（例如：==, !=, >, <）',
            '检查逻辑运算符（AND, OR）的使用'
          ],
          relatedComponents: ['条件引擎', '表达式解析器'],
          severity: 'medium',
          code: errorMessage.match(/near\s+['"](.+?)['"]/)?.[1] || ''
        }
      }
      // 动作执行错误
      else if (errorMessage.includes('action') || errorMessage.includes('plugin') || errorMessage.includes('execution')) {
        result = {
          errorType: '动作执行错误',
          errorMessage: errorMessage,
          possibleCauses: [
            '动作配置参数错误',
            '动作插件不存在或已禁用',
            '动作执行超时',
            '动作执行权限不足'
          ],
          suggestedSolutions: [
            '检查动作配置参数是否正确',
            '确认动作插件已启用并正确安装',
            '检查网络连接和外部服务状态',
            '检查权限设置'
          ],
          relatedComponents: ['动作执行器', '插件管理器'],
          severity: 'high'
        }
      }
      // 邮件数据错误
      else if (errorMessage.includes('email') || errorMessage.includes('mail') || errorMessage.includes('message')) {
        result = {
          errorType: '邮件数据错误',
          errorMessage: errorMessage,
          possibleCauses: [
            '邮件格式不正确',
            '邮件数据缺失',
            '邮件过大',
            '邮件编码问题'
          ],
          suggestedSolutions: [
            '检查邮件格式是否符合标准',
            '确认邮件包含必要的字段（如发件人、收件人等）',
            '检查邮件大小是否超过系统限制',
            '检查邮件编码'
          ],
          relatedComponents: ['邮件解析器', '邮件事件处理器'],
          severity: 'medium'
        }
      }
      // 数据库错误
      else if (errorMessage.includes('database') || errorMessage.includes('DB') || errorMessage.includes('SQL')) {
        result = {
          errorType: '数据库错误',
          errorMessage: errorMessage,
          possibleCauses: [
            '数据库连接失败',
            'SQL语法错误',
            '数据库权限问题',
            '数据库资源不足'
          ],
          suggestedSolutions: [
            '检查数据库连接配置',
            '检查SQL语句语法',
            '确认数据库用户权限',
            '检查数据库服务器资源'
          ],
          relatedComponents: ['数据库连接池', '触发器存储库'],
          severity: 'high'
        }
      }
      // 权限错误
      else if (errorMessage.includes('permission') || errorMessage.includes('access') || errorMessage.includes('denied')) {
        result = {
          errorType: '权限错误',
          errorMessage: errorMessage,
          possibleCauses: [
            '用户权限不足',
            '资源访问被拒绝',
            '认证失败',
            '令牌过期'
          ],
          suggestedSolutions: [
            '检查用户权限设置',
            '确认资源访问权限',
            '重新认证',
            '更新访问令牌'
          ],
          relatedComponents: ['认证服务', '权限管理器'],
          severity: 'high'
        }
      }
      // 网络错误
      else if (errorMessage.includes('network') || errorMessage.includes('connection') || errorMessage.includes('timeout')) {
        result = {
          errorType: '网络错误',
          errorMessage: errorMessage,
          possibleCauses: [
            '网络连接中断',
            '服务不可用',
            '请求超时',
            'DNS解析失败'
          ],
          suggestedSolutions: [
            '检查网络连接',
            '确认服务状态',
            '增加超时时间',
            '检查DNS配置'
          ],
          relatedComponents: ['网络服务', 'HTTP客户端'],
          severity: 'medium'
        }
      }
      // 系统错误
      else if (errorMessage.includes('system') || errorMessage.includes('internal') || errorMessage.includes('server')) {
        result = {
          errorType: '系统错误',
          errorMessage: errorMessage,
          possibleCauses: [
            '系统资源不足',
            '内部服务错误',
            '配置错误',
            '依赖服务故障'
          ],
          suggestedSolutions: [
            '检查系统资源使用情况',
            '查看服务器日志',
            '检查系统配置',
            '检查依赖服务状态'
          ],
          relatedComponents: ['系统服务', '资源管理器'],
          severity: 'critical'
        }
      }
      
      setAnalysisResult(result)
      setIsLoading(false)
    }, 1000)
  }
  
  // 手动分析错误
  const handleAnalyzeError = () => {
    if (errorText.trim() !== '') {
      analyzeError(errorText)
    }
  }
  
  // 获取严重性标签颜色
  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'low':
        return 'bg-blue-100 text-blue-800 border-blue-200'
      case 'medium':
        return 'bg-yellow-100 text-yellow-800 border-yellow-200'
      case 'high':
        return 'bg-orange-100 text-orange-800 border-orange-200'
      case 'critical':
        return 'bg-red-100 text-red-800 border-red-200'
      default:
        return 'bg-gray-100 text-gray-800 border-gray-200'
    }
  }
  
  // 渲染错误分析结果
  const renderAnalysisResult = () => {
    if (!analysisResult) return null
    
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium">{analysisResult.errorType}</h3>
            <p className="text-sm text-gray-500">错误诊断结果</p>
          </div>
          <Badge className={`${getSeverityColor(analysisResult.severity)} border`}>
            {analysisResult.severity === 'low' && '低'}
            {analysisResult.severity === 'medium' && '中'}
            {analysisResult.severity === 'high' && '高'}
            {analysisResult.severity === 'critical' && '严重'}
            严重度
          </Badge>
        </div>
        
        <div className="p-4 bg-red-50 border border-red-200 rounded-md">
          <h4 className="text-sm font-medium text-red-800 mb-2">错误信息</h4>
          <p className="text-sm text-red-700 font-mono">{analysisResult.errorMessage}</p>
          
          {analysisResult.code && (
            <div className="mt-2 p-2 bg-gray-800 text-gray-200 rounded font-mono text-xs overflow-auto">
              <code>{analysisResult.code}</code>
            </div>
          )}
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="p-4 bg-yellow-50 border border-yellow-200 rounded-md">
            <h4 className="text-sm font-medium text-yellow-800 mb-2">可能原因</h4>
            <ul className="list-disc pl-5 space-y-1">
              {analysisResult.possibleCauses.map((cause, index) => (
                <li key={index} className="text-sm text-yellow-700">{cause}</li>
              ))}
            </ul>
          </div>
          
          <div className="p-4 bg-green-50 border border-green-200 rounded-md">
            <h4 className="text-sm font-medium text-green-800 mb-2">建议解决方案</h4>
            <ul className="list-disc pl-5 space-y-1">
              {analysisResult.suggestedSolutions.map((solution, index) => (
                <li key={index} className="text-sm text-green-700">{solution}</li>
              ))}
            </ul>
          </div>
        </div>
        
        <div className="p-4 bg-blue-50 border border-blue-200 rounded-md">
          <h4 className="text-sm font-medium text-blue-800 mb-2">相关组件</h4>
          <div className="flex flex-wrap gap-2">
            {analysisResult.relatedComponents.map((component, index) => (
              <Badge key={index} variant="outline" className="bg-blue-100 text-blue-800 border-blue-200">
                {component}
              </Badge>
            ))}
          </div>
        </div>
        
        {analysisResult.additionalInfo && (
          <div className="p-4 bg-gray-50 border border-gray-200 rounded-md">
            <h4 className="text-sm font-medium text-gray-800 mb-2">附加信息</h4>
            <p className="text-sm text-gray-700">{analysisResult.additionalInfo}</p>
          </div>
        )}
      </div>
    )
  }
  
  // 渲染常见问题解答
  const renderFAQ = () => {
    return (
      <div className="space-y-4">
        <div className="p-4 bg-blue-50 border border-blue-200 rounded-md">
          <h4 className="font-medium text-blue-800 mb-2">条件表达式常见问题</h4>
          <div className="space-y-3">
            <div>
              <h5 className="text-sm font-medium text-blue-700">Q: 为什么我的条件表达式不匹配任何邮件？</h5>
              <p className="text-sm text-blue-600 mt-1">
                A: 检查字段名称是否正确，确保使用正确的比较运算符，并验证邮件数据中是否包含您期望的值。
              </p>
            </div>
            <div>
              <h5 className="text-sm font-medium text-blue-700">Q: 如何调试复杂的条件表达式？</h5>
              <p className="text-sm text-blue-600 mt-1">
                A: 使用触发器调试器测试条件表达式，查看评估结果和详细信息，逐步简化条件直到找出问题所在。
              </p>
            </div>
          </div>
        </div>
        
        <div className="p-4 bg-green-50 border border-green-200 rounded-md">
          <h4 className="font-medium text-green-800 mb-2">动作执行常见问题</h4>
          <div className="space-y-3">
            <div>
              <h5 className="text-sm font-medium text-green-700">Q: 为什么我的动作执行失败？</h5>
              <p className="text-sm text-green-600 mt-1">
                A: 检查动作配置是否正确，确认所有必要参数都已提供，并验证外部服务或API是否可用。
              </p>
            </div>
            <div>
              <h5 className="text-sm font-medium text-green-700">Q: 如何解决动作执行超时问题？</h5>
              <p className="text-sm text-green-600 mt-1">
                A: 检查网络连接，确认外部服务响应时间，考虑增加超时设置或优化动作执行逻辑。
              </p>
            </div>
          </div>
        </div>
        
        <div className="p-4 bg-purple-50 border border-purple-200 rounded-md">
          <h4 className="font-medium text-purple-800 mb-2">系统和权限问题</h4>
          <div className="space-y-3">
            <div>
              <h5 className="text-sm font-medium text-purple-700">Q: 为什么我无法创建或编辑触发器？</h5>
              <p className="text-sm text-purple-600 mt-1">
                A: 检查您的用户权限，确认您有创建和编辑触发器的权限，或联系系统管理员获取适当的权限。
              </p>
            </div>
            <div>
              <h5 className="text-sm font-medium text-purple-700">Q: 系统资源不足怎么办？</h5>
              <p className="text-sm text-purple-600 mt-1">
                A: 优化触发器逻辑，减少不必要的操作，考虑增加系统资源或联系系统管理员。
              </p>
            </div>
          </div>
        </div>
      </div>
    )
  }
  
  // 渲染文档链接
  const renderDocumentation = () => {
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-base">条件表达式文档</CardTitle>
              <CardDescription>学习如何创建有效的条件表达式</CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2">
                <li>
                  <a href="#" className="text-blue-600 hover:underline flex items-center">
                    <FileText className="h-4 w-4 mr-2" />
                    条件表达式语法指南
                  </a>
                </li>
                <li>
                  <a href="#" className="text-blue-600 hover:underline flex items-center">
                    <Code className="h-4 w-4 mr-2" />
                    条件表达式示例
                  </a>
                </li>
                <li>
                  <a href="#" className="text-blue-600 hover:underline flex items-center">
                    <HelpCircle className="h-4 w-4 mr-2" />
                    条件表达式常见问题
                  </a>
                </li>
              </ul>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-base">动作配置文档</CardTitle>
              <CardDescription>了解如何配置和使用动作</CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2">
                <li>
                  <a href="#" className="text-blue-600 hover:underline flex items-center">
                    <FileText className="h-4 w-4 mr-2" />
                    动作类型和参数
                  </a>
                </li>
                <li>
                  <a href="#" className="text-blue-600 hover:underline flex items-center">
                    <Code className="h-4 w-4 mr-2" />
                    动作配置示例
                  </a>
                </li>
                <li>
                  <a href="#" className="text-blue-600 hover:underline flex items-center">
                    <HelpCircle className="h-4 w-4 mr-2" />
                    动作执行故障排除
                  </a>
                </li>
              </ul>
            </CardContent>
          </Card>
        </div>
        
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">系统文档</CardTitle>
            <CardDescription>系统架构和组件文档</CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="space-y-2">
              <li>
                <a href="#" className="text-blue-600 hover:underline flex items-center">
                  <FileText className="h-4 w-4 mr-2" />
                  触发器系统架构
                </a>
              </li>
              <li>
                <a href="#" className="text-blue-600 hover:underline flex items-center">
                  <Code className="h-4 w-4 mr-2" />
                  API参考
                </a>
              </li>
              <li>
                <a href="#" className="text-blue-600 hover:underline flex items-center">
                  <MessageSquare className="h-4 w-4 mr-2" />
                  联系支持
                </a>
              </li>
            </ul>
          </CardContent>
        </Card>
      </div>
    )
  }
  
  return (
    <Card className="w-full">
      <CardHeader>
        <CardTitle className="flex items-center">
          <AlertCircle className="h-5 w-5 mr-2 text-red-500" />
          错误诊断工具
        </CardTitle>
        <CardDescription>
          分析错误并提供解决方案建议
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* 错误输入 */}
        <div className="space-y-2">
          <Label htmlFor="error-text">错误信息</Label>
          <div className="flex gap-2">
            <Textarea
              id="error-text"
              value={errorText}
              onChange={(e) => setErrorText(e.target.value)}
              placeholder="输入或粘贴错误信息..."
              className="flex-1"
              rows={3}
            />
            <Button 
              onClick={handleAnalyzeError} 
              disabled={isLoading || !errorText.trim()}
              className="self-start"
            >
              {isLoading ? (
                <RefreshCw className="h-4 w-4 animate-spin" />
              ) : (
                <Search className="h-4 w-4 mr-2" />
              )}
              分析
            </Button>
          </div>
        </div>
        
        {/* 分析结果标签页 */}
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList className="grid grid-cols-3">
            <TabsTrigger value="analysis">
              <AlertCircle className="h-4 w-4 mr-2" />
              错误分析
            </TabsTrigger>
            <TabsTrigger value="faq">
              <MessageSquare className="h-4 w-4 mr-2" />
              常见问题
            </TabsTrigger>
            <TabsTrigger value="docs">
              <FileText className="h-4 w-4 mr-2" />
              相关文档
            </TabsTrigger>
          </TabsList>
          
          <TabsContent value="analysis" className="pt-4">
            {isLoading ? (
              <div className="space-y-4">
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-24 w-full" />
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Skeleton className="h-32 w-full" />
                  <Skeleton className="h-32 w-full" />
                </div>
              </div>
            ) : analysisResult ? (
              renderAnalysisResult()
            ) : (
              <Alert>
                <HelpCircle className="h-4 w-4" />
                <AlertDescription>
                  输入错误信息并点击"分析"按钮开始诊断
                </AlertDescription>
              </Alert>
            )}
          </TabsContent>
          
          <TabsContent value="faq" className="pt-4">
            {renderFAQ()}
          </TabsContent>
          
          <TabsContent value="docs" className="pt-4">
            {renderDocumentation()}
          </TabsContent>
        </Tabs>
        
        {/* 操作按钮 */}
        {onClose && (
          <div className="flex justify-end">
            <Button variant="outline" onClick={onClose}>
              关闭
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  )
}