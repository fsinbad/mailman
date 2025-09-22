'use client'

import { useState } from 'react'
import { ActionDebuggerEnhanced } from '@/components/action-debugger/action-debugger-enhanced'

export default function ActionDebuggerPage() {
    const [actions, setActions] = useState<any[]>([])

    return (
        <div className="h-screen">
            <ActionDebuggerEnhanced
                actions={actions}
                onChange={setActions}
            />
        </div>
    )
}