'use client'

import React from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ExpressionGroup } from '@/components/expression-builder/expression-group'
import { Plus, Filter } from 'lucide-react'

interface FilterSectionProps {
    filters: any[]
    onChange: (filters: any[]) => void
    testData: Record<string, any>
}

export function FilterSection({ filters, onChange, testData }: FilterSectionProps) {
    const handleAddRootGroup = () => {
        const newExpression = {
            id: Date.now().toString(),
            type: 'group',
            operator: 'and',
            conditions: []
        }
        onChange([...filters, newExpression])
    }

    const handleUpdateFilter = (index: number, updatedExpression: any) => {
        const newFilters = [...filters]
        newFilters[index] = updatedExpression
        onChange(newFilters)
    }

    const handleDeleteFilter = (index: number) => {
        const newFilters = filters.filter((_, i) => i !== index)
        onChange(newFilters)
    }

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                    <Filter className="h-5 w-5 text-blue-500" />
                    <h3 className="text-lg font-semibold">过滤器配置</h3>
                    <Badge variant="secondary" className="text-xs">
                        {filters.length} 个过滤器
                    </Badge>
                </div>
                <Button onClick={handleAddRootGroup} variant="outline" size="sm">
                    <Plus className="h-4 w-4 mr-2" />
                    添加过滤器
                </Button>
            </div>

            {filters.length === 0 ? (
                <Card className="p-8 text-center border-dashed">
                    <div className="text-gray-500">
                        <Filter className="h-8 w-8 mx-auto mb-3 opacity-50" />
                        <p className="text-sm mb-2">暂无过滤器</p>
                        <p className="text-xs">添加过滤器来筛选符合条件的邮件</p>
                    </div>
                </Card>
            ) : (
                <div className="space-y-4">
                    {filters.map((filter, index) => (
                        <ExpressionGroup
                            key={filter.id || index}
                            expression={filter}
                            onChange={(updated) => handleUpdateFilter(index, updated)}
                            onDelete={() => handleDeleteFilter(index)}
                            testData={testData}
                            isRoot={true}
                        />
                    ))}
                </div>
            )}

            <div className="text-xs text-gray-500 p-3 bg-blue-50 rounded">
                <p><strong>提示：</strong></p>
                <ul className="mt-1 space-y-1">
                    <li>• 过滤器将按顺序执行，所有条件都通过才会执行动作</li>
                    <li>• 支持嵌套条件组合，可以构建复杂的过滤逻辑</li>
                    <li>• 可以使用内置条件或插件条件</li>
                </ul>
            </div>
        </div>
    )
}