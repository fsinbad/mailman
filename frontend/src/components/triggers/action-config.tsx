'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { TriggerActionConfig } from '@/types'

interface ActionConfigProps {
  action: TriggerActionConfig
  onChange: (action: TriggerActionConfig) => void
  onRemove: () => void
  isRemovable: boolean
}

export function ActionConfig({ action, onChange, onRemove, isRemovable }: ActionConfigProps) {
  const [configTemplate, setConfigTemplate] = useState('')
  
  // 根据动作类型设置配置模板
  useEffect(() => {
    if (action.type === 'modify_content') {
      setConfigTemplate(JSON.stringify({
        subject_prefix: "[处理]",
        add_tag: "已处理",
        mark_as_read: true
      }, null, 2))
    } else if (action.type === 'smtp') {
      setConfigTemplate(JSON.stringify({
        to: "recipient@example.com",
        subject: "自动回复: {{.Email.Subject}}",
        body: "您好，\n\n这是一封自动回复邮件。\n\n原始邮件主题: {{.Email.Subject}}\n发件人: {{.Email.From}}\n\n此致，\n自动回复系统"
      }, null, 2))
    }
  }, [action.type])
  
  // 使用模板
  const useTemplate = () => {
    onChange({
      ...action,
      config: configTemplate
    })
  }
  
  return (
    <Card className="border border-gray-200">
      <CardContent className="p-4 space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="font-medium">动作配置</h3>
          {isRemovable && (
            <Button 
              type="button" 
              variant="outline" 
              size="sm"
              onClick={onRemove}
              className="text-red-600"
            >
              删除
            </Button>
          )}
        </div>
        
        <div className="space-y-2">
          <Label htmlFor="action-name">动作名称</Label>
          <Input
            id="action-name"
            value={action.name}
            onChange={(e) => onChange({ ...action, name: e.target.value })}
            placeholder="输入动作名称"
            required
          />
        </div>
        
        <div className="space-y-2">
          <Label htmlFor="action-description">描述（可选）</Label>
          <Input
            id="action-description"
            value={action.description || ''}
            onChange={(e) => onChange({ ...action, description: e.target.value })}
            placeholder="输入动作描述"
          />
        </div>
        
        <div className="space-y-2">
          <Label htmlFor="action-type">动作类型</Label>
          <select
            id="action-type"
            className="w-full p-2 border rounded"
            value={action.type}
            onChange={(e) => onChange({ ...action, type: e.target.value as any })}
          >
            <option value="modify_content">修改内容</option>
            <option value="smtp">发送邮件</option>
          </select>
        </div>
        
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <Label htmlFor="action-config">配置 (JSON)</Label>
            <Button 
              type="button" 
              variant="outline" 
              size="sm"
              onClick={useTemplate}
            >
              使用模板
            </Button>
          </div>
          <Textarea
            id="action-config"
            value={action.config}
            onChange={(e) => onChange({ ...action, config: e.target.value })}
            rows={8}
            className="font-mono"
            required
          />
          <p className="text-sm text-gray-500">
            {action.type === 'modify_content' 
              ? "配置邮件内容修改选项，如添加前缀、标记已读等" 
              : "配置要发送的邮件，支持模板变量"}
          </p>
        </div>
        
        <div className="flex items-center justify-between">
          <Label htmlFor="action-enabled">启用此动作</Label>
          <Switch
            id="action-enabled"
            checked={action.enabled}
            onCheckedChange={(checked) => onChange({ ...action, enabled: checked })}
          />
        </div>
      </CardContent>
    </Card>
  )
}