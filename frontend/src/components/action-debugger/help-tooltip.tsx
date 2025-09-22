'use client'

import React from 'react'
import { HelpCircle } from 'lucide-react'
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from '@/components/ui/tooltip'

interface HelpTooltipProps {
    content: string
    children?: React.ReactNode
}

export function HelpTooltip({ content, children }: HelpTooltipProps) {
    return (
        <TooltipProvider>
            <Tooltip>
                <TooltipTrigger asChild>
                    {children || (
                        <HelpCircle className="h-4 w-4 text-gray-400 hover:text-gray-600 cursor-help" />
                    )}
                </TooltipTrigger>
                <TooltipContent>
                    <div className="max-w-xs">
                        <p className="text-sm">{content}</p>
                    </div>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
    )
}