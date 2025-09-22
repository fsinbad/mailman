'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Loader2, RefreshCw, Info, ExternalLink, Shield, Clock, AlertCircle } from 'lucide-react'
import { toast } from 'sonner'
import { apiClient } from '@/lib/api-client'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'

interface PluginInfo {
    id: string;
    name: string;
    version: string;
    description: string;
    type: string;
    author?: string;
    website?: string;
    license?: string;
    status: string;
    loaded_at: string;
    last_used?: string;
    usage_count: number;
    config_schema?: any;
    default_config?: any;
    dependencies?: string[];
    min_version?: string;
    max_version?: string;
    permissions?: string[];
    sandbox?: boolean;
    avg_execution_time?: number;
    max_execution_time?: number;
    error_rate?: number;
}

export default function PluginsTab() {
    const [plugins, setPlugins] = useState<PluginInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [refreshing, setRefreshing] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [selectedPlugin, setSelectedPlugin] = useState<PluginInfo | null>(null);
    const [detailsOpen, setDetailsOpen] = useState(false);

    const fetchPlugins = async () => {
        try {
            const data = await apiClient.get<PluginInfo[]>('/plugins');
            setPlugins(data);
            setError(null);
        } catch (err: any) {
            const errorMessage = err.message || '获取插件列表失败';
            setError(errorMessage);
            toast.error('获取插件列表失败: ' + errorMessage);
        }
    };

    useEffect(() => {
        const loadPlugins = async () => {
            setLoading(true);
            await fetchPlugins();
            setLoading(false);
        };
        loadPlugins();
    }, []);

    const handleRefresh = async () => {
        setRefreshing(true);
        await fetchPlugins();
        setRefreshing(false);
        toast.success('插件列表已刷新');
    };

    const handlePluginClick = (plugin: PluginInfo) => {
        setSelectedPlugin(plugin);
        setDetailsOpen(true);
    };

    const formatDate = (dateString: string) => {
        if (!dateString || dateString === '0001-01-01T00:00:00Z') {
            return '未使用';
        }
        try {
            return new Date(dateString).toLocaleString('zh-CN');
        } catch {
            return dateString;
        }
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <Loader2 className="h-8 w-8 animate-spin text-primary" />
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <Card>
                <CardHeader className="flex flex-row items-center justify-between">
                    <CardTitle>插件管理</CardTitle>
                    <Button
                        onClick={handleRefresh}
                        disabled={refreshing}
                        size="sm"
                        variant="outline"
                    >
                        {refreshing ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                            <RefreshCw className="h-4 w-4" />
                        )}
                        <span className="ml-2">刷新</span>
                    </Button>
                </CardHeader>
                <CardContent>
                    {error ? (
                        <div className="text-center py-8 text-red-500">
                            <p>错误: {error}</p>
                            <Button
                                onClick={handleRefresh}
                                className="mt-4"
                                variant="outline"
                            >
                                重试
                            </Button>
                        </div>
                    ) : plugins.length === 0 ? (
                        <div className="text-center py-8 text-muted-foreground">
                            <p>暂无插件</p>
                        </div>
                    ) : (
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
                                    <TableRow
                                        key={plugin.id}
                                        className="cursor-pointer hover:bg-muted/50"
                                        onClick={() => handlePluginClick(plugin)}
                                    >
                                        <TableCell className="font-mono text-sm">
                                            {plugin.id}
                                        </TableCell>
                                        <TableCell className="font-medium">
                                            {plugin.name}
                                        </TableCell>
                                        <TableCell>
                                            <Badge variant="outline">{plugin.version}</Badge>
                                        </TableCell>
                                        <TableCell>
                                            <Badge>{plugin.type}</Badge>
                                        </TableCell>
                                        <TableCell className="text-muted-foreground">
                                            {plugin.description}
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    )}
                </CardContent>
            </Card>

            {/* 插件详情对话框 */}
            <Dialog open={detailsOpen} onOpenChange={setDetailsOpen}>
                <DialogContent className="max-w-3xl max-h-[80vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle>{selectedPlugin?.name}</DialogTitle>
                        <DialogDescription>
                            {selectedPlugin?.description}
                        </DialogDescription>
                    </DialogHeader>

                    {selectedPlugin && (
                        <div className="space-y-6 mt-4">
                            {/* 基本信息 */}
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <h4 className="text-sm font-semibold mb-2">基本信息</h4>
                                    <div className="space-y-2">
                                        <div className="flex items-center gap-2">
                                            <span className="text-sm text-muted-foreground">ID:</span>
                                            <code className="text-sm">{selectedPlugin.id}</code>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            <span className="text-sm text-muted-foreground">版本:</span>
                                            <Badge variant="outline">{selectedPlugin.version}</Badge>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            <span className="text-sm text-muted-foreground">类型:</span>
                                            <Badge>{selectedPlugin.type}</Badge>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            <span className="text-sm text-muted-foreground">状态:</span>
                                            <Badge variant={selectedPlugin.status === 'loaded' ? 'default' : 'secondary'}>
                                                {selectedPlugin.status}
                                            </Badge>
                                        </div>
                                    </div>
                                </div>

                                <div>
                                    <h4 className="text-sm font-semibold mb-2">开发信息</h4>
                                    <div className="space-y-2">
                                        {selectedPlugin.author && (
                                            <div className="flex items-center gap-2">
                                                <span className="text-sm text-muted-foreground">作者:</span>
                                                <span className="text-sm">{selectedPlugin.author}</span>
                                            </div>
                                        )}
                                        {selectedPlugin.license && (
                                            <div className="flex items-center gap-2">
                                                <span className="text-sm text-muted-foreground">许可证:</span>
                                                <span className="text-sm">{selectedPlugin.license}</span>
                                            </div>
                                        )}
                                        {selectedPlugin.website && (
                                            <div className="flex items-center gap-2">
                                                <ExternalLink className="h-3 w-3 text-muted-foreground" />
                                                <a
                                                    href={selectedPlugin.website}
                                                    target="_blank"
                                                    rel="noopener noreferrer"
                                                    className="text-sm text-blue-500 hover:underline"
                                                >
                                                    {selectedPlugin.website}
                                                </a>
                                            </div>
                                        )}
                                    </div>
                                </div>
                            </div>

                            {/* 时间信息 */}
                            <div>
                                <h4 className="text-sm font-semibold mb-2 flex items-center gap-2">
                                    <Clock className="h-4 w-4" />
                                    时间信息
                                </h4>
                                <div className="space-y-2">
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm text-muted-foreground">加载时间:</span>
                                        <span className="text-sm">{formatDate(selectedPlugin.loaded_at)}</span>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm text-muted-foreground">最后使用:</span>
                                        <span className="text-sm">{formatDate(selectedPlugin.last_used || '')}</span>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm text-muted-foreground">使用次数:</span>
                                        <span className="text-sm">{selectedPlugin.usage_count}</span>
                                    </div>
                                </div>
                            </div>

                            {/* 性能指标 */}
                            <div>
                                <h4 className="text-sm font-semibold mb-2 flex items-center gap-2">
                                    <AlertCircle className="h-4 w-4" />
                                    性能指标
                                </h4>
                                <div className="space-y-2">
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm text-muted-foreground">平均执行时间:</span>
                                        <span className="text-sm">{selectedPlugin.avg_execution_time || 0} ms</span>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm text-muted-foreground">最大执行时间:</span>
                                        <span className="text-sm">{selectedPlugin.max_execution_time || 0} ms</span>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <span className="text-sm text-muted-foreground">错误率:</span>
                                        <span className="text-sm">{(selectedPlugin.error_rate || 0) * 100}%</span>
                                    </div>
                                </div>
                            </div>

                            {/* 权限 */}
                            {selectedPlugin.permissions && selectedPlugin.permissions.length > 0 && (
                                <div>
                                    <h4 className="text-sm font-semibold mb-2 flex items-center gap-2">
                                        <Shield className="h-4 w-4" />
                                        权限
                                    </h4>
                                    <div className="flex gap-2 flex-wrap">
                                        {selectedPlugin.permissions.map((permission) => (
                                            <Badge key={permission} variant="secondary">
                                                {permission}
                                            </Badge>
                                        ))}
                                    </div>
                                </div>
                            )}

                            {/* 配置模式 */}
                            {selectedPlugin.config_schema && (
                                <div>
                                    <h4 className="text-sm font-semibold mb-2">配置模式</h4>
                                    <pre className="bg-muted p-3 rounded-md text-sm overflow-x-auto">
                                        {JSON.stringify(selectedPlugin.config_schema, null, 2)}
                                    </pre>
                                </div>
                            )}

                            {/* 默认配置 */}
                            {selectedPlugin.default_config && (
                                <div>
                                    <h4 className="text-sm font-semibold mb-2">默认配置</h4>
                                    <pre className="bg-muted p-3 rounded-md text-sm overflow-x-auto">
                                        {JSON.stringify(selectedPlugin.default_config, null, 2)}
                                    </pre>
                                </div>
                            )}
                        </div>
                    )}
                </DialogContent>
            </Dialog>
        </div>
    );
}
