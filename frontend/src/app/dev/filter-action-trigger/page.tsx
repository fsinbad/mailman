'use client'

import { useState } from 'react'
import { FilterActionTriggerDebugger } from '@/components/filter-action-trigger/filter-action-trigger-debugger'

export default function FilterActionTriggerPage() {
    const [filters, setFilters] = useState<any[]>([])
    const [actions, setActions] = useState<any[]>([])

    return (
        <div className="h-screen">
            <FilterActionTriggerDebugger
                filters={filters}
                actions={actions}
                onFiltersChange={setFilters}
                onActionsChange={setActions}
            />
        </div>
    )
}