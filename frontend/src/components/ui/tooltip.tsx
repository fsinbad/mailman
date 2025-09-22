'use client'

import React, { useState, useEffect } from 'react'

// Simple tooltip implementation that matches the expected API
export const TooltipProvider = ({ children }: { children: React.ReactNode }) => {
  return <>{children}</>
}

export function Tooltip({ children }: { children: React.ReactNode }) {
  const [isVisible, setIsVisible] = useState(false)
  const [showTimeout, setShowTimeout] = useState<NodeJS.Timeout | null>(null)

  const handleMouseEnter = () => {
    const timeout = setTimeout(() => {
      setIsVisible(true)
    }, 500)
    setShowTimeout(timeout)
  }

  const handleMouseLeave = () => {
    if (showTimeout) {
      clearTimeout(showTimeout)
      setShowTimeout(null)
    }
    setIsVisible(false)
  }

  useEffect(() => {
    return () => {
      if (showTimeout) {
        clearTimeout(showTimeout)
      }
    }
  }, [showTimeout])

  let trigger: React.ReactNode = null
  let content: React.ReactNode = null

  React.Children.forEach(children, (child) => {
    if (React.isValidElement(child)) {
      if (child.type === TooltipTrigger) {
        trigger = child.props.children
      } else if (child.type === TooltipContent) {
        content = child.props.children
      }
    }
  })

  return (
    <div className="relative inline-block">
      <div
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
      >
        {trigger}
      </div>
      
      {isVisible && content && (
        <div className="absolute z-50 px-3 py-2 text-sm text-white bg-gray-900 rounded-md shadow-lg whitespace-nowrap bottom-full left-1/2 -translate-x-1/2 mb-2 max-w-xs">
          {content}
          <div className="absolute top-full left-1/2 -translate-x-1/2 w-0 h-0 border-4 border-l-transparent border-r-transparent border-b-transparent border-t-gray-900" />
        </div>
      )}
    </div>
  )
}

export const TooltipTrigger = ({ children, asChild }: { children: React.ReactNode, asChild?: boolean }) => {
  return <>{children}</>
}

export const TooltipContent = ({ children, className }: { children: React.ReactNode, className?: string }) => {
  return <>{children}</>
}