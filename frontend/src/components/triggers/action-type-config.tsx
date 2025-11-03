'use client'

import React, { useState, useEffect } from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { Checkbox } from '@/components/ui/checkbox'
import { TriggerActionType } from '@/types'
import { Card, CardContent } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { AlertCircle } from 'lucide-react'

interface ActionTypeConfigProps {
  actionType: TriggerActionType
  config: Record<string, any>
  onChange: (config: Record<string, any>) => void
}

export function ActionTypeConfig({ actionType, config, onChange }: ActionTypeConfigProps) {
  const [errors, setErrors] = useState<Record<string, string>>({})

  // 验证配置
  useEffect(() => {
    const newErrors: Record<string, string> = {}

    if (actionType === 'smtp') {
      if (!config.to) {
        newErrors.to = '收件人是必填项'
      } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(config.to)) {
        newErrors.to = '请输入有效的邮箱地址'
      }

      if (!config.subject) {
        newErrors.subject = '主题是必填项'
      }

      if (!config.body) {
        newErrors.body = '邮件内容是必填项'
      }
    }

    setErrors(newErrors)
  }, [actionType, config])

  // 处理字段变更
  const handleFieldChange = (field: string, value: any) => {
    onChange({
      ...config,
      [field]: value
    })
  }

  // 根据动作类型渲染不同的配置表单
  const renderConfigForm = () => {
    switch (actionType) {
      case 'modify_content':
        return (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="subject-prefix">主题前缀</Label>
              <Input
                id="subject-prefix"
                value={config.subject_prefix || ''}
                onChange={(e) => handleFieldChange('subject_prefix', e.target.value)}
                placeholder="例如: [处理]"
              />
              <p className="text-xs text-gray-500">添加到邮件主题前的文本</p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="add-tag">添加标签</Label>
              <Input
                id="add-tag"
                value={config.add_tag || ''}
                onChange={(e) => handleFieldChange('add_tag', e.target.value)}
                placeholder="例如: 已处理"
              />
              <p className="text-xs text-gray-500">添加到邮件的标签</p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="move-to-folder">移动到文件夹</Label>
              <Input
                id="move-to-folder"
                value={config.move_to_folder || ''}
                onChange={(e) => handleFieldChange('move_to_folder', e.target.value)}
                placeholder="例如: 已处理"
              />
              <p className="text-xs text-gray-500">将邮件移动到指定文件夹</p>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="mark-as-read"
                checked={!!config.mark_as_read}
                onChange={(e) => handleFieldChange('mark_as_read', e.target.checked)}
              />
              <Label htmlFor="mark-as-read">标记为已读</Label>
            </div>
          </div>
        )

      case 'smtp':
        return (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="to-address">收件人 <span className="text-red-500">*</span></Label>
              <Input
                id="to-address"
                value={config.to || ''}
                onChange={(e) => handleFieldChange('to', e.target.value)}
                placeholder="recipient@example.com"
                className={errors.to ? 'border-red-500' : ''}
              />
              {errors.to && <p className="text-xs text-red-500">{errors.to}</p>}
            </div>

            <div className="space-y-2">
              <Label htmlFor="cc-address">抄送</Label>
              <Input
                id="cc-address"
                value={config.cc || ''}
                onChange={(e) => handleFieldChange('cc', e.target.value)}
                placeholder="cc@example.com"
              />
              <p className="text-xs text-gray-500">可选，多个地址用逗号分隔</p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="email-subject">邮件主题 <span className="text-red-500">*</span></Label>
              <Input
                id="email-subject"
                value={config.subject || ''}
                onChange={(e) => handleFieldChange('subject', e.target.value)}
                placeholder="自动回复: {{.Email.Subject}}"
                className={errors.subject ? 'border-red-500' : ''}
              />
              {errors.subject && <p className="text-xs text-red-500">{errors.subject}</p>}
              <p className="text-xs text-gray-500">支持模板变量，如 {"{{.Email.Subject}}"}</p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="email-body">邮件内容 <span className="text-red-500">*</span></Label>
              <Textarea
                id="email-body"
                value={config.body || ''}
                onChange={(e) => handleFieldChange('body', e.target.value)}
                placeholder="您好，这是一封自动回复邮件。"
                rows={5}
                className={errors.body ? 'border-red-500' : ''}
              />
              {errors.body && <p className="text-xs text-red-500">{errors.body}</p>}
              <p className="text-xs text-gray-500">
                支持模板变量，如 {"{{.Email.From}}"}, {"{{.Email.Subject}}"}, {"{{.Email.Body}}"}
              </p>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="include-original"
                checked={!!config.include_original}
                onChange={(e) => handleFieldChange('include_original', e.target.checked)}
              />
              <Label htmlFor="include-original">包含原始邮件</Label>
            </div>
          </div>
        )

      default:
        return (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              未知的动作类型: {actionType}
            </AlertDescription>
          </Alert>
        )
    }
  }

  return (
    <Card>
      <CardContent className="pt-6">
        {renderConfigForm()}
      </CardContent>
    </Card>
  )
}