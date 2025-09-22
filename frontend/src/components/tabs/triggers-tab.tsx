'use client'

import { useState } from 'react'
import { TriggerPageManager } from '@/components/triggers/trigger-page-manager'

interface TriggersTabProps {
    tabId: string
}

export function TriggersTab({ tabId }: TriggersTabProps) {
    return (
        <div className="h-full">
            <TriggerPageManager activeTab={tabId} />
        </div>
    )
}

