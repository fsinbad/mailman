'use client'

import React, { useState, useEffect } from 'react'
import {
    Activity,
    Clock,
    Users,
    AlertTriangle,
    CheckCircle,
    Wifi,
    WifiOff,
    RefreshCw,
    BarChart3
} from 'lucide-react'
import { cn } from '@/lib/utils'

// 数据类型定义
interface QueueMetrics {
    queue_length: number
    queue_capacity: number
    usage_rate: number
    skipped_syncs: number
    active_accounts: number
    worker_count: number
}

interface PerAccountStats {
    active_syncers: number
    total_syncers: number
    total_syncs: number
    total_errors: number
    concurrent_limit: number
    current_concurrent: number
    start_time: string
}

interface AccountSyncStatus {
    account_id: number
    account_email: string
    is_running: boolean
    last_sync_time: string
    next_sync_time: string
    sync_interval: number
    sync_count: number
    error_count: number
    last_error?: string
    last_error_time?: string
}

interface SyncManagerStats {
    optimized_manager?: QueueMetrics
    per_account_manager?: PerAccountStats
    email_scheduler?: any
}

interface NotificationStats {
    connected_clients: number
    total_notifications: number
    history_count: number
    max_history: number
}

// 统计卡片组件
interface StatsCardProps {
    title: string
    value: string | number
    icon: React.ReactNode
    color: 'blue' | 'green' | 'yellow' | 'red' | 'purple'
    subtitle?: string
}

function StatsCard({ title, value, icon, color, subtitle }: StatsCardProps) {
    const colorClasses = {
        blue: 'bg-blue-500 text-blue-100',
        green: 'bg-green-500 text-green-100',
        yellow: 'bg-yellow-500 text-yellow-100',
        red: 'bg-red-500 text-red-100',
        purple: 'bg-purple-500 text-purple-100'
    }

    return (
        <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow-md">
            <div className="flex items-center justify-between">
                <div>
                    <p className="text-sm font-medium text-gray-600 dark:text-gray-400">{title}</p>
                    <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{value}</p>
                    {subtitle && (
                        <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">{subtitle}</p>
                    )}
                </div>
                <div className={cn("p-3 rounded-full", colorClasses[color])}>
                    {icon}
                </div>
            </div>
        </div>
    )
}

// 进度条组件
interface ProgressBarProps {
    label: string
    current: number
    max: number
    color: 'blue' | 'green' | 'yellow' | 'red'
}

function ProgressBar({ label, current, max, color }: ProgressBarProps) {
    const percentage = max > 0 ? (current / max) * 100 : 0

    const colorClasses = {
        blue: 'bg-blue-500',
        green: 'bg-green-500',
        yellow: 'bg-yellow-500',
        red: 'bg-red-500'
    }

    return (
        <div className="mb-4">
            <div className="flex justify-between text-sm text-gray-600 dark:text-gray-400 mb-1">
                <span>{label}</span>
                <span>{current} / {max} ({percentage.toFixed(1)}%)</span>
            </div>
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                <div
                    className={cn("h-2 rounded-full transition-all duration-300", colorClasses[color])}
                    style={{ width: `${Math.min(percentage, 100)}%` }}
                />
            </div>
        </div>
    )
}

// 账户状态列表组件
interface AccountStatusListProps {
    statuses: AccountSyncStatus[]
    onRefresh: () => void
}

function AccountStatusList({ statuses, onRefresh }: AccountStatusListProps) {
    return (
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md">
            <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">账户同步状态</h3>
                <button
                    onClick={onRefresh}
                    className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
                >
                    <RefreshCw className="w-4 h-4" />
                </button>
            </div>

            <div className="max-h-96 overflow-y-auto">
                {statuses.map((status) => (
                    <div key={status.account_id} className="p-4 border-b border-gray-100 dark:border-gray-700 last:border-b-0">
                        <div className="flex items-center justify-between">
                            <div className="flex-1">
                                <div className="flex items-center space-x-2">
                                    <span className="font-medium text-gray-900 dark:text-gray-100">
                                        {status.account_email}
                                    </span>
                                    <div className={cn(
                                        "w-2 h-2 rounded-full",
                                        status.is_running ? "bg-green-500" : "bg-gray-400"
                                    )} />
                                </div>

                                <div className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                                    <span>同步间隔: {status.sync_interval}秒</span>
                                    <span className="mx-2">•</span>
                                    <span>成功次数: {status.sync_count}</span>
                                    {status.error_count > 0 && (
                                        <>
                                            <span className="mx-2">•</span>
                                            <span className="text-red-500">错误次数: {status.error_count}</span>
                                        </>
                                    )}
                                </div>

                                <div className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                                    上次同步: {new Date(status.last_sync_time).toLocaleString()}
                                    <span className="mx-2">•</span>
                                    下次同步: {new Date(status.next_sync_time).toLocaleString()}
                                </div>

                                {status.last_error && (
                                    <div className="text-xs text-red-500 mt-1 p-2 bg-red-50 dark:bg-red-900/20 rounded">
                                        错误: {status.last_error}
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    )
}

// 主监控Dashboard组件
export default function SyncMonitorDashboard() {
    const [queueMetrics, setQueueMetrics] = useState<QueueMetrics | null>(null)
    const [syncStats, setSyncStats] = useState<SyncManagerStats | null>(null)
    const [accountStatuses, setAccountStatuses] = useState<AccountSyncStatus[]>([])
    const [notificationStats, setNotificationStats] = useState<NotificationStats | null>(null)
    const [loading, setLoading] = useState(true)
    const [lastUpdate, setLastUpdate] = useState<Date | null>(null)
    const [autoRefresh, setAutoRefresh] = useState(true)

    // 获取监控数据
    const fetchMonitoringData = async () => {
        try {
            const [queueRes, statsRes, statusRes, notifRes] = await Promise.all([
                fetch('/api/sync/queue-metrics'),
                fetch('/api/sync/manager-stats'),
                fetch('/api/sync/account-status'),
                fetch('/api/notifications/stats')
            ])

            if (queueRes.ok) {
                const queueData = await queueRes.json()
                setQueueMetrics(queueData.data)
            }

            if (statsRes.ok) {
                const statsData = await statsRes.json()
                setSyncStats(statsData.data)
            }

            if (statusRes.ok) {
                const statusData = await statusRes.json()
                setAccountStatuses(statusData.data || [])
            }

            if (notifRes.ok) {
                const notifData = await notifRes.json()
                setNotificationStats(notifData.data)
            }

            setLastUpdate(new Date())
        } catch (error) {
            console.error('Failed to fetch monitoring data:', error)
        } finally {
            setLoading(false)
        }
    }

    // 自动刷新
    useEffect(() => {
        fetchMonitoringData()

        let interval: NodeJS.Timeout | null = null
        if (autoRefresh) {
            interval = setInterval(fetchMonitoringData, 10000) // 每10秒刷新
        }

        return () => {
            if (interval) clearInterval(interval)
        }
    }, [autoRefresh])

    if (loading) {
        return (
            <div className="p-8 flex items-center justify-center">
                <div className="text-center">
                    <RefreshCw className="w-8 h-8 animate-spin text-blue-500 mx-auto mb-2" />
                    <p className="text-gray-600 dark:text-gray-400">加载监控数据...</p>
                </div>
            </div>
        )
    }

    const queueUsageColor: 'blue' | 'green' | 'yellow' | 'red' | 'purple' = queueMetrics
        ? queueMetrics.usage_rate > 80 ? 'red'
            : queueMetrics.usage_rate > 60 ? 'yellow'
                : 'green'
        : 'blue'

    return (
        <div className="p-6 space-y-6 bg-gray-50 dark:bg-gray-900 min-h-screen">
            {/* 页面标题 */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">同步监控Dashboard</h1>
                    <p className="text-gray-600 dark:text-gray-400">实时邮件同步系统监控</p>
                </div>

                <div className="flex items-center space-x-4">
                    <div className="flex items-center space-x-2">
                        <button
                            onClick={() => setAutoRefresh(!autoRefresh)}
                            className={cn(
                                "flex items-center space-x-2 px-3 py-2 rounded-md text-sm font-medium",
                                autoRefresh
                                    ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300"
                                    : "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300"
                            )}
                        >
                            {autoRefresh ? <Wifi className="w-4 h-4" /> : <WifiOff className="w-4 h-4" />}
                            <span>{autoRefresh ? '自动刷新' : '手动刷新'}</span>
                        </button>

                        <button
                            onClick={fetchMonitoringData}
                            className="flex items-center space-x-2 px-3 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md text-sm font-medium"
                        >
                            <RefreshCw className="w-4 h-4" />
                            <span>刷新</span>
                        </button>
                    </div>

                    {lastUpdate && (
                        <p className="text-sm text-gray-500 dark:text-gray-400">
                            最后更新: {lastUpdate.toLocaleTimeString()}
                        </p>
                    )}
                </div>
            </div>

            {/* 概览统计卡片 */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                {queueMetrics && (
                    <StatsCard
                        title="队列使用率"
                        value={`${queueMetrics.usage_rate.toFixed(1)}%`}
                        icon={<BarChart3 className="w-5 h-5" />}
                        color={queueUsageColor}
                        subtitle={`${queueMetrics.queue_length}/${queueMetrics.queue_capacity}`}
                    />
                )}

                {syncStats?.per_account_manager && (
                    <StatsCard
                        title="活跃同步器"
                        value={syncStats.per_account_manager.active_syncers}
                        icon={<Activity className="w-5 h-5" />}
                        color="blue"
                        subtitle={`总计: ${syncStats.per_account_manager.total_syncers}`}
                    />
                )}

                {syncStats?.per_account_manager && (
                    <StatsCard
                        title="总同步次数"
                        value={syncStats.per_account_manager.total_syncs}
                        icon={<CheckCircle className="w-5 h-5" />}
                        color="green"
                        subtitle={`错误: ${syncStats.per_account_manager.total_errors}`}
                    />
                )}

                {notificationStats && (
                    <StatsCard
                        title="WebSocket连接"
                        value={notificationStats.connected_clients}
                        icon={<Users className="w-5 h-5" />}
                        color="purple"
                        subtitle={`通知: ${notificationStats.total_notifications}`}
                    />
                )}
            </div>

            {/* 详细监控面板 */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* 队列监控 */}
                {queueMetrics && (
                    <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow-md">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">队列监控</h3>

                        <ProgressBar
                            label="队列使用率"
                            current={queueMetrics.queue_length}
                            max={queueMetrics.queue_capacity}
                            color={queueMetrics.usage_rate > 80 ? 'red' : queueMetrics.usage_rate > 60 ? 'yellow' : 'green'}
                        />

                        <div className="grid grid-cols-2 gap-4 mt-4">
                            <div className="text-center p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                                <p className="text-lg font-bold text-gray-900 dark:text-gray-100">
                                    {queueMetrics.worker_count}
                                </p>
                                <p className="text-sm text-gray-600 dark:text-gray-400">Worker数量</p>
                            </div>

                            <div className="text-center p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                                <p className="text-lg font-bold text-red-600 dark:text-red-400">
                                    {queueMetrics.skipped_syncs}
                                </p>
                                <p className="text-sm text-gray-600 dark:text-gray-400">跳过同步</p>
                            </div>
                        </div>
                    </div>
                )}

                {/* 系统性能 */}
                {syncStats?.per_account_manager && (
                    <div className="bg-white dark:bg-gray-800 rounded-lg p-6 shadow-md">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">系统性能</h3>

                        <ProgressBar
                            label="并发使用率"
                            current={syncStats.per_account_manager.current_concurrent}
                            max={syncStats.per_account_manager.concurrent_limit}
                            color={
                                syncStats.per_account_manager.current_concurrent / syncStats.per_account_manager.concurrent_limit > 0.8
                                    ? 'red' : 'green'
                            }
                        />

                        <div className="grid grid-cols-2 gap-4 mt-4">
                            <div className="text-center p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                                <p className="text-lg font-bold text-green-600 dark:text-green-400">
                                    {(syncStats.per_account_manager.total_syncs - syncStats.per_account_manager.total_errors)}
                                </p>
                                <p className="text-sm text-gray-600 dark:text-gray-400">成功同步</p>
                            </div>

                            <div className="text-center p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
                                <p className="text-lg font-bold text-blue-600 dark:text-blue-400">
                                    {Math.round((new Date().getTime() - new Date(syncStats.per_account_manager.start_time).getTime()) / (1000 * 60 * 60))}h
                                </p>
                                <p className="text-sm text-gray-600 dark:text-gray-400">运行时间</p>
                            </div>
                        </div>
                    </div>
                )}
            </div>

            {/* 账户详细状态 */}
            <AccountStatusList
                statuses={accountStatuses}
                onRefresh={() => fetchMonitoringData()}
            />

            {/* 警告和建议 */}
            {queueMetrics && queueMetrics.usage_rate > 80 && (
                <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
                    <div className="flex items-center space-x-2">
                        <AlertTriangle className="w-5 h-5 text-yellow-600 dark:text-yellow-400" />
                        <h4 className="font-medium text-yellow-800 dark:text-yellow-200">性能警告</h4>
                    </div>
                    <p className="text-yellow-700 dark:text-yellow-300 mt-2">
                        队列使用率过高 ({queueMetrics.usage_rate.toFixed(1)}%)，建议：
                    </p>
                    <ul className="list-disc list-inside text-yellow-700 dark:text-yellow-300 mt-1 text-sm">
                        <li>检查是否有账户同步失败导致任务堆积</li>
                        <li>考虑增加Worker数量或优化同步间隔</li>
                        <li>检查网络连接和邮件服务器响应时间</li>
                    </ul>
                </div>
            )}

            {queueMetrics && queueMetrics.skipped_syncs > 0 && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                    <div className="flex items-center space-x-2">
                        <AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400" />
                        <h4 className="font-medium text-red-800 dark:text-red-200">同步跳过警告</h4>
                    </div>
                    <p className="text-red-700 dark:text-red-300 mt-2">
                        已有 {queueMetrics.skipped_syncs} 次同步被跳过，这可能导致邮件延迟。
                        建议立即检查系统负载和网络状况。
                    </p>
                </div>
            )}
        </div>
    )
}