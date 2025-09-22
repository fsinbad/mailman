import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { FilterActionTriggerDebugger } from "@/components/filter-action-trigger/filter-action-trigger-debugger";

export default function AdvancedDebugPage() {
    const [filters, setFilters] = useState<any[]>([]);
    const [actions, setActions] = useState<any[]>([]);

    const handleFiltersChange = (newFilters: any[]) => {
        setFilters(newFilters);
    };

    const handleActionsChange = (newActions: any[]) => {
        setActions(newActions);
    };

    return (
        <div className="container mx-auto p-6">
            <Card>
                <CardHeader>
                    <CardTitle>高级调试器</CardTitle>
                </CardHeader>
                <CardContent>
                    <FilterActionTriggerDebugger
                        filters={filters}
                        actions={actions}
                        onFiltersChange={handleFiltersChange}
                        onActionsChange={handleActionsChange}
                    />
                </CardContent>
            </Card>
        </div>
    );
}
