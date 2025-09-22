'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { apiClient } from '@/lib/api-client'
import { toast } from 'sonner'

interface PluginInfo {
    id: string;
    name: string;
    version: string;
    description: string;
    type: string;
}

export default function PluginsPage() {
    const [plugins, setPlugins] = useState<PluginInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        async function fetchPlugins() {
            try {
                const data = await apiClient.get<PluginInfo[]>('/plugins');
                setPlugins(data);
                setError(null);
            } catch (err: any) {
                const errorMessage = err.message || '获取插件列表失败';
                setError(errorMessage);
                toast.error('获取插件列表失败: ' + errorMessage);
            } finally {
                setLoading(false);
            }
        }

        fetchPlugins();
    }, []);

    if (loading) {
        return <div>Loading...</div>;
    }

    if (error) {
        return <div>Error: {error}</div>;
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>插件管理</CardTitle>
            </CardHeader>
            <CardContent>
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>ID</TableHead>
                            <TableHead>名称</TableHead>
                            <TableHead>版本</TableHead>
                            <TableHead>类型</TableHead>
                            <TableHead>描述</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {plugins.map((plugin) => (
                            <TableRow key={plugin.id}>
                                <TableCell>{plugin.id}</TableCell>
                                <TableCell>{plugin.name}</TableCell>
                                <TableCell>
                                    <Badge variant="outline">{plugin.version}</Badge>
                                </TableCell>
                                <TableCell>
                                    <Badge>{plugin.type}</Badge>
                                </TableCell>
                                <TableCell>{plugin.description}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </CardContent>
        </Card>
    );
}