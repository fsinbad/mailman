'use client'

import { useState } from 'react'
import dynamic from 'next/dynamic'

// 动态导入触发器页面组件 - 简化版本
const TriggerList = dynamic(() => import('@/app/triggers/page'), {
    loading: () => <div className="flex items-center justify-center h-64">
        <div className="text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-2 text-gray-600">加载中...</p>
        </div>
    </div>
})

const MultiStepTriggerCreator = dynamic(() => import('@/app/triggers/create/page'), {
    loading: () => <div className="flex items-center justify-center h-64">
        <div className="text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-2 text-gray-600">加载中...</p>
        </div>
    </div>
})

const TriggerAdvancedDebug = dynamic(() => import('@/app/triggers/advanced-debug/page'), {
    loading: () => <div className="flex items-center justify-center h-64">
        <div className="text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-2 text-gray-600">加载中...</p>
        </div>
    </div>
})

interface TriggerPageManagerProps {
    activeTab: string
}

export function TriggerPageManager({ activeTab }: TriggerPageManagerProps) {
    const renderContent = () => {
        switch (activeTab) {
            case 'triggers':
                return <TriggerList />
            case 'trigger-create':
                return <MultiStepTriggerCreator />
            case 'trigger-advanced-debug':
                return <TriggerAdvancedDebug />
            default:
                return <TriggerList />
        }
    }

    return (
        <div className="w-full h-full">
            {renderContent()}
        </div>
    )
}