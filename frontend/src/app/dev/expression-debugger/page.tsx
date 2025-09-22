'use client'

import { useState } from 'react'
import { ExpressionDebugger } from '@/components/expression-builder/expression-debugger'

export default function ExpressionDebuggerPage() {
    const [expressions, setExpressions] = useState<any[]>([])

    return (
        <div className="h-screen">
            <ExpressionDebugger
                expressions={expressions}
                onChange={setExpressions}
            />
        </div>
    )
}