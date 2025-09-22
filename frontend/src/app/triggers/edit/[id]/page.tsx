'use client'

import { TriggerForm } from '@/components/triggers/trigger-form'
import { useParams } from 'next/navigation'

export default function EditTriggerPage() {
  const params = useParams()
  const triggerId = parseInt(params.id as string)
  
  return (
    <div className="space-y-6 p-6">
      <h1 className="text-2xl font-bold text-gray-900 dark:text-white">编辑触发器</h1>
      <p className="text-gray-600 dark:text-gray-400">
        修改触发器的条件和动作
      </p>
      
      <TriggerForm triggerId={triggerId} />
    </div>
  )
}