'use client'

import { useState, useEffect } from 'react'
import { cn } from '@/lib/utils'
import { Dropdown } from '@/components/ui/dropdown'
import {
    LayoutDashboard,
    Mail,
    UserCog,
    Settings,
    X,
    FileText,
    Bot,
    Inbox,
    Bell,
    Key,
    Plus,
    RefreshCw,
    Zap,
    PlusCircle,
    Sparkles,
    BarChart3,
    TestTube,
    PlayCircle,
    Bug,
} from 'lucide-react'

interface Tab {
    id: string
    name: string
    icon: React.ComponentType<{ className?: string }>
}

const tabConfig: Tab[] = [
    { id: 'dashboard', name: '仪表板', icon: LayoutDashboard },
    { id: 'accounts', name: '邮箱账户管理', icon: UserCog },
    { id: 'emails', name: '邮件管理', icon: Mail },
    { id: 'sync-config', name: '同步配置', icon: RefreshCw },
    { id: 'mail-pickup', name: '取件', icon: Inbox },
    // 根据需求隐藏订阅管理菜单项
    // { id: 'subscriptions', name: '订阅管理', icon: Bell },
    { id: 'pickup', name: '取件模板', icon: FileText },
    // 高级触发器菜单组
    { id: 'trigger-demo', name: '系统演示', icon: PlayCircle },
    { id: 'triggers', name: '触发器管理', icon: Zap },
    { id: 'trigger-create', name: '创建新规则', icon: PlusCircle },
    { id: 'trigger-templates', name: '规则模板', icon: Sparkles },
    { id: 'trigger-stats', name: '执行统计', icon: BarChart3 },
    { id: 'trigger-test', name: '测试调试', icon: TestTube },
    { id: 'oauth2-config', name: 'OAuth2 配置', icon: Key },
    { id: 'ai-config', name: 'AI 配置', icon: Bot },
    { id: 'user-sessions', name: '访问令牌', icon: Settings },
    // { id: 'settings', name: '设置', icon: Settings },
    // 开发者模式菜单项
    { id: 'expression-debugger', name: '表达式调试器', icon: Bug },
    { id: 'action-debugger', name: '动作调试器', icon: PlayCircle },
    { id: 'filter-action-trigger', name: '过滤动作触发器', icon: Zap },
]

interface TabManagerProps {
    activeTab: string
    openTabs: string[]
    onTabChange: (tabId: string) => void
    onTabClose: (tabId: string) => void
    onTabOpen: (tabId: string) => void
}

export function TabManager({
    activeTab,
    openTabs,
    onTabChange,
    onTabClose,
    onTabOpen,
}: TabManagerProps) {
    const getTabInfo = (tabId: string) => {
        return tabConfig.find(tab => tab.id === tabId)
    }

    const availableTabs = tabConfig.filter(tab => !openTabs.includes(tab.id))

    // 监听切换到邮箱账户管理页面的事件
    useEffect(() => {
        const handleSwitchToAccountsTab = (event: CustomEvent) => {
            // 如果邮箱账户管理tab未打开，则打开它
            if (!openTabs.includes('accounts')) {
                onTabOpen('accounts')
            }
            // 切换到邮箱账户管理tab
            onTabChange('accounts')

            // 通知邮箱账户管理页面进行过滤
            const filterEvent = new CustomEvent('filterAccountsByProvider', {
                detail: event.detail
            })
            window.dispatchEvent(filterEvent)
        }

        window.addEventListener('switchToAccountsTab', handleSwitchToAccountsTab as EventListener)

        return () => {
            window.removeEventListener('switchToAccountsTab', handleSwitchToAccountsTab as EventListener)
        }
    }, [openTabs, onTabOpen, onTabChange])

    return (
        <div className="flex h-14 items-center border-b border-gray-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900">
            {/* 左侧间距，避免与侧边栏按钮重叠 */}
            <div className="w-4 flex-shrink-0" />

            {/* Tab列表 */}
            <div className="flex flex-1 items-center overflow-x-auto scrollbar-hide">
                {openTabs.map((tabId) => {
                    const tab = getTabInfo(tabId)
                    if (!tab) return null

                    const Icon = tab.icon
                    const isActive = activeTab === tabId

                    return (
                        <div
                            key={tabId}
                            className={cn(
                                'group relative flex h-12 min-w-[140px] max-w-[200px] cursor-pointer items-center rounded-t-lg px-4 mx-1 transition-all duration-200',
                                isActive
                                    ? 'bg-gradient-to-t from-gray-50 to-white text-primary-600 shadow-sm border-t border-l border-r border-gray-200 dark:from-gray-800 dark:to-gray-700 dark:text-primary-400 dark:border-gray-700'
                                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'
                            )}
                            onClick={() => onTabChange(tabId)}
                        >
                            <Icon className="mr-2 h-4 w-4 shrink-0 transition-transform duration-200 group-hover:scale-110" />
                            <span className="flex-1 truncate text-sm font-medium">
                                {tab.name}
                            </span>
                            {openTabs.length > 1 && (
                                <button
                                    onClick={(e) => {
                                        e.stopPropagation()
                                        onTabClose(tabId)
                                    }}
                                    className="ml-2 rounded-full p-0.5 opacity-0 transition-all duration-200 hover:bg-gray-300 group-hover:opacity-100 dark:hover:bg-gray-600"
                                >
                                    <X className="h-3 w-3" />
                                </button>
                            )}
                            {isActive && (
                                <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary-600 dark:bg-primary-400 animate-pulse" />
                            )}
                        </div>
                    )
                })}
            </div>

            {/* 添加新tab的下拉菜单 */}
            {availableTabs.length > 0 && (
                <Dropdown
                    trigger={
                        <button className="flex h-10 items-center justify-center rounded-lg px-3 mx-2 text-gray-600 hover:bg-gray-100 transition-colors duration-200 dark:text-gray-400 dark:hover:bg-gray-800">
                            <Plus className="h-5 w-5 transition-transform duration-200 hover:rotate-90" />
                        </button>
                    }
                    className="w-48"
                >
                    {availableTabs.map((tab) => {
                        const Icon = tab.icon
                        return (
                            <button
                                key={tab.id}
                                onClick={() => onTabOpen(tab.id)}
                                className="flex w-full items-center px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
                            >
                                <Icon className="mr-2 h-4 w-4" />
                                {tab.name}
                            </button>
                        )
                    })}
                </Dropdown>
            )}
        </div>
    )
}
