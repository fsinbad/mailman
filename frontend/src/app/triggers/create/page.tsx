'use client'

import { TriggerForm } from '@/components/triggers/trigger-form'

export default function CreateTriggerPage() {
  return (
    <div className="space-y-6 p-6">
      <h1 className="text-2xl font-bold text-gray-900 dark:text-white">创建新触发器</h1>
      <p className="text-gray-600 dark:text-gray-400">
        创建一个新的邮件触发器，设置条件和动作
      </p>
      
      <TriggerForm />
    </div>
  )
}