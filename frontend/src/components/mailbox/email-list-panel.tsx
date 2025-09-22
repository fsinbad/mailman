'use client'

import React, { useState, useEffect } from 'react'
import { Search, RefreshCw, Mail, Paperclip, ChevronDown, Filter, Play, Pause } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Email, EmailAccount } from '@/types'
import { formatDate, truncate } from '@/lib/utils'

interface EmailListPanelProps {
    emails: Email[]
    selectedEmail: Email | null
    onSelectEmail: (email: Email) => void
    onSearch: (query: string) => void
    onRefresh: () => Promise<void>
    loading: boolean
    selectedAccount: EmailAccount | null
    autoSyncEnabled: boolean
    onToggleAutoSync: () => void
    isRefreshing: boolean
}

export default function EmailListPanel({
    emails,
    selectedEmail,
    onSelectEmail,
    onSearch,
    onRefresh,
    loading,
    selectedAccount,
    autoSyncEnabled,
    onToggleAutoSync,
    isRefreshing
}: EmailListPanelProps) {
    const [searchQuery, setSearchQuery] = useState('')
    const [sortBy, setSortBy] = useState<'date' | 'from' | 'subject'>('date')
    const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')

    // 处理搜索
    const handleSearch = (e: React.FormEvent) => {
        e.preventDefault()
        onSearch(searchQuery)
    }

    // 处理搜索输入变化
    const handleSearchInputChange = (value: string) => {
        setSearchQuery(value)
        // 实时搜索（防抖处理）
        if (searchTimeoutRef.current) {
            clearTimeout(searchTimeoutRef.current)
        }
        searchTimeoutRef.current = setTimeout(() => {
            onSearch(value)
        }, 500)
    }

    const searchTimeoutRef = React.useRef<NodeJS.Timeout | null>(null)

    // 清理定时器
    useEffect(() => {
        return () => {
            if (searchTimeoutRef.current) {
                clearTimeout(searchTimeoutRef.current)
            }
        }
    }, [])

    // 排序邮件
    const sortedEmails = [...emails].sort((a, b) => {
        let comparison = 0

        switch (sortBy) {
            case 'date':
                comparison = new Date(a.Date).getTime() - new Date(b.Date).getTime()
                break
            case 'from':
                const fromA = Array.isArray(a.From) ? a.From[0] : a.From || ''
                const fromB = Array.isArray(b.From) ? b.From[0] : b.From || ''
                comparison = fromA.localeCompare(fromB)
                break
            case 'subject':
                comparison = (a.Subject || '').localeCompare(b.Subject || '')
                break
        }

        return sortOrder === 'desc' ? -comparison : comparison
    })

    return (
        <div className="h-full flex flex-col">
            {/* 顶部工具栏 */}
            <div className="border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
                {/* 标题栏 */}
                <div className="px-4 py-3 flex items-center justify-between">
                    <h3 className="font-semibold text-gray-900 dark:text-white">
                        {selectedAccount
                            ? `${selectedAccount.emailAddress} 的邮件`
                            : '邮件列表'
                        }
                    </h3>
                    <div className="flex items-center gap-3">
                        <span className="text-sm text-gray-500 dark:text-gray-400">
                            {emails.length} 封邮件
                        </span>

                        {/* 自动同步切换按钮 */}
                        <button
                            onClick={onToggleAutoSync}
                            className={cn(
                                "flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-colors",
                                autoSyncEnabled
                                    ? "bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-300"
                                    : "bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-400"
                            )}
                            title={autoSyncEnabled ? "关闭自动同步" : "开启自动同步"}
                        >
                            {autoSyncEnabled ? (
                                <Pause className="h-3 w-3" />
                            ) : (
                                <Play className="h-3 w-3" />
                            )}
                            <span className="hidden sm:inline">
                                {autoSyncEnabled ? "自动同步" : "手动模式"}
                            </span>
                        </button>

                        {/* 手动刷新按钮 */}
                        <button
                            onClick={onRefresh}
                            disabled={loading || isRefreshing}
                            className="p-1 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-500 dark:text-gray-400 disabled:opacity-50 disabled:cursor-not-allowed"
                            title="刷新邮件"
                        >
                            <RefreshCw className={cn(
                                "h-4 w-4",
                                (loading || isRefreshing) && "animate-spin"
                            )} />
                        </button>
                    </div>
                </div>

                {/* 搜索栏 */}
                <div className="px-4 pb-3">
                    <form onSubmit={handleSearch} className="relative">
                        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                        <input
                            type="text"
                            placeholder="搜索邮件内容、发件人、主题..."
                            value={searchQuery}
                            onChange={(e) => handleSearchInputChange(e.target.value)}
                            className="w-full rounded-lg border border-gray-300 bg-white py-2 pl-10 pr-4 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                        />
                    </form>
                </div>

                {/* 排序控制 */}
                <div className="px-4 pb-3 flex items-center gap-4">
                    <div className="flex items-center gap-2">
                        <span className="text-sm text-gray-500 dark:text-gray-400">排序:</span>
                        <select
                            value={sortBy}
                            onChange={(e) => setSortBy(e.target.value as 'date' | 'from' | 'subject')}
                            className="text-sm border-gray-300 dark:border-gray-600 rounded-md dark:bg-gray-700"
                        >
                            <option value="date">日期</option>
                            <option value="from">发件人</option>
                            <option value="subject">主题</option>
                        </select>
                        <button
                            onClick={() => setSortOrder(sortOrder === 'desc' ? 'asc' : 'desc')}
                            className="text-sm text-blue-600 dark:text-blue-400 hover:underline"
                        >
                            {sortOrder === 'desc' ? '↓ 降序' : '↑ 升序'}
                        </button>
                    </div>
                </div>
            </div>

            {/* 邮件列表 */}
            <div className="flex-1 overflow-y-auto">
                {loading ? (
                    <div className="h-full flex items-center justify-center">
                        <div className="text-center">
                            <RefreshCw className="h-8 w-8 animate-spin text-gray-400 mx-auto mb-4" />
                            <p className="text-gray-500 dark:text-gray-400">加载邮件中...</p>
                        </div>
                    </div>
                ) : emails.length === 0 ? (
                    <div className="h-full flex flex-col">
                        {/* 空状态标题区域 */}
                        <div className="flex-1 flex items-center justify-center min-h-0">
                            <div className="text-center max-w-md px-4">
                                <Mail className="h-16 w-16 text-gray-300 dark:text-gray-600 mx-auto mb-6" />
                                <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                                    {selectedAccount ? '邮箱暂无邮件' : '选择邮箱账户'}
                                </h3>
                                <p className="text-gray-500 dark:text-gray-400 mb-6">
                                    {selectedAccount
                                        ? `当前 ${selectedAccount.emailAddress} 邮箱中没有邮件，或者所有邮件都已同步完成。`
                                        : '请从左侧选择一个邮箱账户来查看邮件。'
                                    }
                                </p>
                                {selectedAccount && (
                                    <div className="space-y-2">
                                        <button
                                            onClick={onRefresh}
                                            disabled={loading || isRefreshing}
                                            className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                                        >
                                            {(loading || isRefreshing) && <RefreshCw className="h-4 w-4 animate-spin" />}
                                            刷新邮件
                                        </button>
                                        <button
                                            onClick={onToggleAutoSync}
                                            className={cn(
                                                "w-full px-4 py-2 rounded-lg transition-colors flex items-center justify-center gap-2",
                                                autoSyncEnabled
                                                    ? "bg-green-600 text-white hover:bg-green-700"
                                                    : "border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
                                            )}
                                        >
                                            {autoSyncEnabled ? (
                                                <>
                                                    <Pause className="h-4 w-4" />
                                                    关闭自动同步
                                                </>
                                            ) : (
                                                <>
                                                    <Play className="h-4 w-4" />
                                                    开启自动同步
                                                </>
                                            )}
                                        </button>
                                    </div>
                                )}
                            </div>
                        </div>

                        {/* 底部提示区域 */}
                        <div className="p-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
                            <div className="text-center">
                                <p className="text-xs text-gray-500 dark:text-gray-400">
                                    {selectedAccount ? '尝试刷新或检查网络连接' : '开始管理您的邮件'}
                                </p>
                            </div>
                        </div>
                    </div>
                ) : (
                    <div className="divide-y divide-gray-200 dark:divide-gray-700">
                        {sortedEmails.map((email) => (
                            <div
                                key={email.ID}
                                onClick={() => onSelectEmail(email)}
                                className={cn(
                                    "p-4 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors",
                                    selectedEmail?.ID === email.ID
                                        ? "bg-blue-50 dark:bg-blue-900/30 border-r-4 border-blue-500"
                                        : ""
                                )}
                            >
                                <div className="flex items-start gap-3">
                                    {/* 邮件图标 */}
                                    <div className={cn(
                                        "w-3 h-3 rounded-full mt-2 shrink-0",
                                        selectedEmail?.ID === email.ID
                                            ? "bg-blue-500"
                                            : "bg-gray-400 dark:bg-gray-500"
                                    )} />

                                    {/* 邮件信息 */}
                                    <div className="flex-1 min-w-0">
                                        {/* 第一行：发件人和时间 */}
                                        <div className="flex items-center justify-between mb-1">
                                            <span className={cn(
                                                "font-medium text-sm truncate",
                                                selectedEmail?.ID === email.ID
                                                    ? "text-blue-900 dark:text-blue-200"
                                                    : "text-gray-900 dark:text-gray-100"
                                            )}>
                                                {Array.isArray(email.From) ? email.From[0] : email.From || '未知发件人'}
                                            </span>
                                            <span className="text-xs text-gray-500 dark:text-gray-400 ml-2 shrink-0">
                                                {formatDate(email.Date)}
                                            </span>
                                        </div>

                                        {/* 第二行：主题和附件 */}
                                        <div className="flex items-center gap-2 mb-1">
                                            <span className={cn(
                                                "text-sm truncate",
                                                selectedEmail?.ID === email.ID
                                                    ? "text-blue-800 dark:text-blue-300"
                                                    : "text-gray-700 dark:text-gray-300"
                                            )}>
                                                {email.Subject || '(无主题)'}
                                            </span>
                                            {email.Attachments && email.Attachments.length > 0 && (
                                                <Paperclip className="h-3 w-3 text-gray-400" />
                                            )}
                                        </div>

                                        {/* 第三行：邮件预览 */}
                                        <p className="text-xs text-gray-500 dark:text-gray-400 truncate">
                                            {truncate(email.Body || '', 80)}
                                        </p>
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            {/* 底部状态栏 */}
            <div className="border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 px-4 py-2">
                <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
                    <span>
                        {selectedAccount && `当前: ${selectedAccount.emailAddress}`}
                    </span>
                    <span>
                        {emails.length > 0 && `显示 ${emails.length} 封邮件`}
                    </span>
                </div>
            </div>
        </div>
    )
}
