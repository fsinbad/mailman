'use client'

import React, { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { TriggerActionConfig, TriggerActionType } from '@/types'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ActionTypeConfig } from './action-type-config'

interface ActionConfigDialogProps {
  action: TriggerActionConfig
  isOpen: boolean
  onClose: () => void
  onSave: (action: TriggerActionConfig) => void
}

export function ActionConfigDialog({
  action,
  isOpen,
  onClose,
  onSave
}: ActionConfigDialogProps) {
  const [editedAction, setEditedAction] = useState<TriggerActionConfig>({ ...action })
  const [configTemplate, setConfigTemplate] = useState('')
  const [activeTab, setActiveTab] = useState<string>('basic')
  const [configError, setConfigError] = useState<string | null>(null)

  // 当动作改变时重置状态
  useEffect(() => {
    setEditedAction({ ...action })
    setConfigError(null)
    
    // 根据动作类型设置配置模板
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
  }, [action, isOpen])

  // 使用模板
  const useTemplate = () => {
    setEditedAction({
      ...editedAction,
      config: configTemplate
    })
  }
  
  // 验证JSON配置
  const validateConfig = (config: string): boolean => {
    try {
      JSON.parse(config)
      setConfigError(null)
      return true
    } catch (e) {
      setConfigError('配置必须是有效的JSON格式')
      return false
    }
  }
  
  // 保存动作
  const handleSave = () => {
    // 验证配置
    if (!validateConfig(editedAction.config)) {
      return
    }
    
    onSave(editedAction)
    onClose()
  }
  
  // 处理动作类型变更
  const handleTypeChange = (type: TriggerActionType) => {
    setEditedAction({
      ...editedAction,
      type,
      // 重置配置为空对象
      config: '{}'
    })
    
    // 切换到高级配置选项卡
    setActiveTab('advanced')
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>配置动作</DialogTitle>
          <DialogDescription>
            设置动作的基本信息和执行参数
          </DialogDescription>
        </DialogHeader>
        
        <Tabs value={activeTab} onValueChange={setActiveTab} className="mt-4">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="basic">基本信息</TabsTrigger>
            <TabsTrigger value="advanced">高级配置</TabsTrigger>
          </TabsList>
          
          <TabsContent value="basic" className="space-y-4 pt-4">
            <div className="space-y-2">
              <Label htmlFor="action-name">动作名称</Label>
              <Input
                id="action-name"
                value={editedAction.name}
                onChange={(e) => setEditedAction({ ...editedAction, name: e.target.value })}
                placeholder="输入动作名称"
                required
              />
            </div>
            
            <div className="space-y-2">
              <Label htmlFor="action-description">描述（可选）</Label>
              <Textarea
                id="action-description"
                value={editedAction.description || ''}
                onChange={(e) => setEditedAction({ ...editedAction, description: e.target.value })}
                placeholder="输入动作描述"
                rows={3}
              />
            </div>
            
            <div className="space-y-2">
              <Label htmlFor="action-type">动作类型</Label>
              <select
                id="action-type"
                className="w-full p-2 border rounded"
                value={editedAction.type}
                onChange={(e) => handleTypeChange(e.target.value as TriggerActionType)}
              >
                <option value="modify_content">修改内容</option>
                <option value="smtp">发送邮件</option>
              </select>
              <p className="text-sm text-gray-500">
                {editedAction.type === 'modify_content' 
                  ? "修改邮件内容，如添加前缀、标记已读等" 
                  : "发送邮件通知，支持模板变量"}
              </p>
            </div>
            
            <div className="flex items-center justify-between">
              <Label htmlFor="action-enabled">启用此动作</Label>
              <Switch
                id="action-enabled"
                checked={editedAction.enabled}
                onCheckedChange={(checked) => setEditedAction({ ...editedAction, enabled: checked })}
              />
            </div>
          </TabsContent>
          
          <TabsContent value="advanced" className="space-y-4 pt-4">
            <Tabs defaultValue="visual">
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="visual">可视化配置</TabsTrigger>
                <TabsTrigger value="json">JSON 配置</TabsTrigger>
              </TabsList>
              
              <TabsContent value="visual" className="pt-4">
                <ActionTypeConfig 
                  actionType={editedAction.type}
                  config={
                    // 尝试解析JSON配置
                    (() => {
                      try {
                        return JSON.parse(editedAction.config)
                      } catch (e) {
                        return {}
                      }
                    })()
                  }
                  onChange={(newConfig) => {
                    setEditedAction({
                      ...editedAction,
                      config: JSON.stringify(newConfig, null, 2)
                    })
                  }}
                />
              </TabsContent>
              
              <TabsContent value="json" className="pt-4">
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
                    value={editedAction.config}
                    onChange={(e) => setEditedAction({ ...editedAction, config: e.target.value })}
                    rows={10}
                    className="font-mono"
                    required
                  />
                  {configError && (
                    <p className="text-sm text-red-500">{configError}</p>
                  )}
                  <p className="text-sm text-gray-500">
                    {editedAction.type === 'modify_content' 
                      ? "配置邮件内容修改选项，如添加前缀、标记已读等" 
                      : "配置要发送的邮件，支持模板变量"}
                  </p>
                </div>
                
                {editedAction.type === 'modify_content' && (
                  <div className="bg-blue-50 p-3 rounded-md mt-4">
                    <h4 className="text-sm font-medium text-blue-800 mb-2">可用配置选项:</h4>
                    <ul className="text-xs text-blue-700 space-y-1 list-disc pl-4">
                      <li><code>subject_prefix</code>: 添加主题前缀</li>
                      <li><code>add_tag</code>: 添加标签</li>
                      <li><code>mark_as_read</code>: 标记为已读 (true/false)</li>
                      <li><code>move_to_folder</code>: 移动到文件夹</li>
                    </ul>
                  </div>
                )}
                
                {editedAction.type === 'smtp' && (
                  <div className="bg-blue-50 p-3 rounded-md mt-4">
                    <h4 className="text-sm font-medium text-blue-800 mb-2">可用配置选项:</h4>
                    <ul className="text-xs text-blue-700 space-y-1 list-disc pl-4">
                      <li><code>to</code>: 收件人邮箱</li>
                      <li><code>cc</code>: 抄送邮箱</li>
                      <li><code>subject</code>: 邮件主题</li>
                      <li><code>body</code>: 邮件内容</li>
                      <li><code>include_original</code>: 包含原始邮件 (true/false)</li>
                    </ul>
                    <p className="text-xs text-blue-700 mt-2">
                      支持的模板变量: <code>{'{{.Email.Subject}}'}</code>, <code>{'{{.Email.From}}'}</code>, <code>{'{{.Email.To}}'}</code>, <code>{'{{.Email.Body}}'}</code>
                    </p>
                  </div>
                )}
              </TabsContent>
            </Tabs>
          </TabsContent>
        </Tabs>
        
        <DialogFooter>
          <div className="flex items-center gap-2 mr-auto">
            <Button 
              variant="secondary" 
              onClick={() => {
                // 验证配置
                if (!validateConfig(editedAction.config)) {
                  return
                }
                
                // 将动作配置编码为URL参数并导航到测试页面
                const actionParam = encodeURIComponent(JSON.stringify(editedAction))
                window.open(`/triggers/actions/test?action=${actionParam}`, '_blank')
              }}
            >
              测试
            </Button>
          </div>
          <Button variant="outline" onClick={onClose}>取消</Button>
          <Button onClick={handleSave}>保存</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}