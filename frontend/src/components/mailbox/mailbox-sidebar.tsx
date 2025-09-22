'use client'

import React from 'react'
import { ChevronLeft, ChevronRight, Mail, Inbox, Settings, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import { EmailAccount } from '@/types'

interface MailboxSidebarProps {
    accounts: EmailAccount[]
    selectedAccount: EmailAccount | null
    onSelectAccount: (account: EmailAccount) => void
    collapsed: boolean
    onToggleCollapse: () => void
    loading: boolean
}

export default function MailboxSidebar({
    accounts,
    selectedAccount,
    onSelectAccount,
    collapsed,
    onToggleCollapse,
    loading
}: MailboxSidebarProps) {

    // 获取邮箱提供商图标
    const getProviderIcon = (provider: string) => {
        switch (provider?.toLowerCase()) {
            case 'gmail':
                return '📧'
            case 'outlook':
                return '📮'
            case 'yahoo':
                return '📬'
            default:
                return '📭'
        }
    }

    // 获取邮箱提供商颜色
    const getProviderColor = (provider: string) => {
        switch (provider?.toLowerCase()) {
            case 'gmail':
                return 'text-red-600 dark:text-red-400'
            case 'outlook':
                return 'text-blue-600 dark:text-blue-400'
            case 'yahoo':
                return 'text-purple-600 dark:text-purple-400'
            default:
                return 'text-gray-600 dark:text-gray-400'
        }
    }

    return (
        <div className="h-full flex flex-col bg-white dark:bg-gray-800">
            {/* 顶部标题栏 */}
            <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
                {!collapsed && (
                    <h2 className="font-semibold text-gray-900 dark:text-white">邮箱账户</h2>
                )}
                <button
                    onClick={onToggleCollapse}
                    className="p-1 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-500 dark:text-gray-400"
                    title={collapsed ? "展开侧边栏" : "折叠侧边栏"}
                >
                    {collapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
                </button>
            </div>

            {/* 邮箱列表 */}
            <div className="flex-1 overflow-y-auto">
                {loading ? (
                    <div className="p-4 text-center">
                        <div className="inline-flex items-center gap-2 text-gray-500 dark:text-gray-400">
                            <RefreshCw className="h-4 w-4 animate-spin" />
                            {!collapsed && <span className="text-sm">加载中...</span>}
                        </div>
                    </div>
                ) : accounts.length === 0 ? (
                    <div className="p-4 text-center">
                        <div className="text-gray-500 dark:text-gray-400">
                            <Inbox className="h-8 w-8 mx-auto mb-2 opacity-50" />
                            {!collapsed && (
                                <p className="text-sm">暂无邮箱账户</p>
                            )}
                        </div>
                    </div>
                ) : (
                    <div className="p-2">
                        {accounts.map((account) => (
                            <div
                                key={account.id}
                                onClick={() => onSelectAccount(account)}
                                className={cn(
                                    "rounded-lg p-3 mb-2 cursor-pointer transition-all duration-200 hover:bg-gray-100 dark:hover:bg-gray-700",
                                    selectedAccount?.id === account.id
                                        ? "bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-700"
                                        : "border border-transparent"
                                )}
                            >
                                <div className="flex items-center gap-3">
                                    {/* 提供商图标 */}
                                    <div className="text-lg shrink-0">
                                        {getProviderIcon(account.mailProvider?.type || 'custom')}
                                    </div>

                                    {!collapsed && (
                                        <div className="flex-1 min-w-0">
                                            {/* 邮箱地址 */}
                                            <p className={cn(
                                                "font-medium text-sm truncate",
                                                selectedAccount?.id === account.id
                                                    ? "text-blue-900 dark:text-blue-200"
                                                    : "text-gray-900 dark:text-gray-100"
                                            )}>
                                                {account.emailAddress}
                                            </p>

                                            {/* 提供商和状态 */}
                                            <div className="flex items-center gap-2 mt-1">
                                                <span className={cn(
                                                    "text-xs font-medium",
                                                    getProviderColor(account.mailProvider?.type || 'custom')
                                                )}>
                                                    {account.mailProvider?.type?.toUpperCase() || 'CUSTOM'}
                                                </span>

                                                {/* 状态指示器 */}
                                                <div className="flex items-center gap-1">
                                                    <div className={cn(
                                                        "w-2 h-2 rounded-full",
                                                        account.isVerified
                                                            ? "bg-green-400 dark:bg-green-500"
                                                            : "bg-red-400 dark:bg-red-500"
                                                    )} />
                                                    <span className="text-xs text-gray-500 dark:text-gray-400">
                                                        {account.isVerified ? '已验证' : '未验证'}
                                                    </span>
                                                </div>
                                            </div>
                                        </div>
                                    )}
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            {/* 底部操作区域 */}
            {!collapsed && (
                <div className="p-4 border-t border-gray-200 dark:border-gray-700">
                    <div className="text-xs text-gray-500 dark:text-gray-400 text-center">
                        共 {accounts.length} 个邮箱账户
                    </div>
                </div>
            )}
        </div>
    )
}