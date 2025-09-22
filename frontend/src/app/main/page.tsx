'use client'

import { useState, useEffect, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/contexts/auth-context'
import { Sidebar } from '@/components/layout/sidebar'
import { Header } from '@/components/layout/header'
import { TabManager } from '@/components/layout/tab-manager'
import { Loader2 } from 'lucide-react'
import DashboardTab from '@/components/tabs/dashboard-tab'
import AccountsTab from '@/components/tabs/accounts-tab'
import EmailsTab from '@/components/tabs/emails-tab'
import MailPickupTab from '@/components/tabs/mail-pickup-tab'
import PickupTab from '@/components/tabs/pickup-tab'
// 根据需求隐藏设置功能
// import SettingsTab from '@/components/tabs/settings-tab'
import { AIConfigTab } from '@/components/tabs/ai-config-tab'
// 根据需求隐藏订阅管理功能
// import { SubscriptionsTab } from '@/components/tabs/subscriptions-tab'
import SyncConfigTab from '@/components/tabs/sync-config-tab'
import UserSessionsTab from '@/components/tabs/user-sessions-tab'
import { TriggersTab } from '@/components/tabs/triggers-tab'
import OAuth2ConfigTab from '@/components/tabs/oauth2-config-tab'
import PluginsTab from '@/components/tabs/plugins-tab'
import SystemConfigTab from '@/components/tabs/system-config-tab'
import { cn } from '@/lib/utils'
import { registerTabCallback, unregisterTabCallback } from '@/lib/tab-utils'

interface TabContent {
    [key: string]: React.ReactNode
}

export default function MainPage() {
    const [activeTab, setActiveTab] = useState('dashboard')
    const [openTabs, setOpenTabs] = useState<string[]>(['dashboard'])
    const [tabContents, setTabContents] = useState<TabContent>({})
    const { isAuthenticated, isLoading } = useAuth()
    const router = useRouter()

    // 存储待处理的Tab数据
    const [pendingTabData, setPendingTabData] = useState<{ [key: string]: any }>({})

    // 如果未登录，重定向到登录页
    useEffect(() => {
        if (!isLoading && !isAuthenticated) {
            router.push('/login')
        }
    }, [isAuthenticated, isLoading, router])

    // 初始化tab内容
    useEffect(() => {
        const initializeTab = (tabId: string) => {
            if (!tabContents[tabId]) {
                let content: React.ReactNode
                switch (tabId) {
                    case 'dashboard':
                        content = <DashboardTab key={tabId} />
                        break
                    case 'accounts':
                        content = <AccountsTab key={tabId} />
                        break
                    case 'emails':
                        content = <EmailsTab key={tabId} />
                        break
                    case 'mail-pickup':
                        content = <MailPickupTab key={tabId} />
                        break
                    // 根据需求隐藏订阅管理功能
                    // case 'subscriptions':
                    //     content = <SubscriptionsTab key={tabId} />
                    //     break
                    case 'sync-config':
                        content = <SyncConfigTab key={tabId} />
                        break
                    case 'pickup':
                        content = <PickupTab key={tabId} />
                        break
                    case 'triggers':
                    case 'trigger-demo':
                    case 'trigger-create':
                    case 'trigger-templates':
                    case 'trigger-stats':
                    case 'trigger-test':
                        content = <TriggersTab key={tabId} tabId={tabId} />
                        break
                    case 'oauth2-config':
                        content = <OAuth2ConfigTab key={tabId} />
                        break
                    // 根据需求隐藏设置功能
                    // case 'settings':
                    //     content = <SettingsTab key={tabId} />
                    //     break
                    case 'ai-config':
                        content = <AIConfigTab key={tabId} />
                        break
                    case 'user-sessions':
                        content = <UserSessionsTab key={tabId} />
                        break
                    case 'plugins':
                        content = <PluginsTab key={tabId} />
                        break
                    case 'system-config':
                        content = <SystemConfigTab key={tabId} />
                        break
                    case 'classic-mailbox':
                        const ClassicMailboxView = require('@/components/mailbox/classic-mailbox-view').default
                        content = <ClassicMailboxView key={tabId} />
                        break
                    case 'expression-debugger':
                        const ExpressionDebuggerPage = require('@/app/dev/expression-debugger/page').default
                        content = <ExpressionDebuggerPage key={tabId} />
                        break
                    case 'action-debugger':
                        const ActionDebuggerPage = require('@/app/dev/action-debugger/page').default
                        content = <ActionDebuggerPage key={tabId} />
                        break
                    case 'filter-action-trigger':
                        const FilterActionTriggerPage = require('@/app/dev/filter-action-trigger/page').default
                        content = <FilterActionTriggerPage key={tabId} />
                        break
                    default:
                        content = <DashboardTab key={tabId} />
                }
                setTabContents(prev => ({ ...prev, [tabId]: content }))
            }
        }

        openTabs.forEach(initializeTab)
    }, [openTabs, tabContents])

    // 处理tab切换 - 使用 useCallback 避免重复创建
    const handleTabChange = useCallback((tabId: string) => {
        console.log('[MainPage] handleTabChange 被调用，切换到:', tabId);
        setActiveTab(tabId)
        // 如果tab不在打开列表中，添加它
        if (!openTabs.includes(tabId)) {
            setOpenTabs(prev => [...prev, tabId])
        }
    }, [openTabs])

    // 监听 switchTab 事件
    useEffect(() => {
        const handleSwitchTab = (event: CustomEvent) => {
            console.log('[MainPage] 收到 switchTab 事件:', event.detail);
            const { tab, data } = event.detail;
            if (!tab) return;

            // 处理额外的数据
            if (data) {
                console.log(`[MainPage] 存储Tab ${tab}的数据:`, data);

                // 移除旧的全局变量，避免冲突 - 仅使用pendingTabData
                if ((window as any).__switchTabData) {
                    console.log('[MainPage] 清除旧的全局变量 __switchTabData');
                    delete (window as any).__switchTabData;
                }

                // 使用时间戳标记数据，确保能追踪调用
                const dataWithTimestamp = {
                    ...data,
                    __timestamp: new Date().getTime(),
                    __processed: false
                };

                // 存储到本地状态，以便随时可用
                setPendingTabData(prev => ({
                    ...prev,
                    [tab]: dataWithTimestamp
                }));

                // 检查Tab是否已注册回调
                if ((window as any).__tabCallbacks?.[tab]?.onReady) {
                    try {
                        console.log(`[MainPage] 发现Tab ${tab}已注册回调，直接调用`);
                        (window as any).__tabCallbacks[tab].onReady(dataWithTimestamp);

                        // 标记为已处理
                        setPendingTabData(prev => ({
                            ...prev,
                            [tab]: {
                                ...prev[tab],
                                __processed: true
                            }
                        }));
                    } catch (error) {
                        console.error(`[MainPage] 调用Tab ${tab}回调出错:`, error);
                    }
                } else {
                    console.log(`[MainPage] Tab ${tab}尚未注册回调，数据将在Tab准备好时传递`);
                }
            }

            // 切换到目标Tab
            handleTabChange(tab);
        }

        window.addEventListener('switchTab', handleSwitchTab as EventListener);
        return () => {
            window.removeEventListener('switchTab', handleSwitchTab as EventListener);
        }
    }, [handleTabChange]);

    // 监听Tab回调注册
    useEffect(() => {
        const handleTabCallbackRegistered = (event: CustomEvent) => {
            const { tabId, callbackName } = event.detail;

            console.log(`[MainPage] 收到Tab ${tabId}回调注册事件:`, callbackName);

            // 如果有待处理的数据且数据未被处理过，才调用回调
            if (callbackName === 'onReady' && pendingTabData[tabId]) {
                // 检查数据是否已被处理过
                if (pendingTabData[tabId].__processed) {
                    console.log(`[MainPage] Tab ${tabId}的数据已被处理过，跳过重复调用`);
                    return;
                }

                try {
                    console.log(`[MainPage] 发现Tab ${tabId}有待处理数据，调用新注册的回调`);
                    (window as any).__tabCallbacks[tabId].onReady(pendingTabData[tabId]);

                    // 标记为已处理
                    setPendingTabData(prev => ({
                        ...prev,
                        [tabId]: {
                            ...prev[tabId],
                            __processed: true
                        }
                    }));

                    // 延迟清理数据
                    setTimeout(() => {
                        setPendingTabData(prev => {
                            if (!prev[tabId]) return prev;

                            const newData = { ...prev };
                            delete newData[tabId];
                            return newData;
                        });
                    }, 5000);
                } catch (error) {
                    console.error(`[MainPage] 调用新注册的Tab ${tabId}回调出错:`, error);
                }
            }
        };

        window.addEventListener('tabCallbackRegistered', handleTabCallbackRegistered as EventListener);
        return () => {
            window.removeEventListener('tabCallbackRegistered', handleTabCallbackRegistered as EventListener);
        };
    }, [pendingTabData]);

    // 如果正在检查认证状态，显示加载器
    if (isLoading) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <Loader2 className="h-8 w-8 animate-spin text-primary" />
            </div>
        )
    }

    // 如果未登录，不显示内容（会被重定向）
    if (!isAuthenticated) {
        return null
    }

    // 处理tab关闭
    const handleTabClose = (tabId: string) => {
        const newOpenTabs = openTabs.filter(id => id !== tabId)
        setOpenTabs(newOpenTabs)

        // 清理关闭的tab内容
        const newTabContents = { ...tabContents }
        delete newTabContents[tabId]
        setTabContents(newTabContents)

        // 如果关闭的是当前激活的tab，切换到最后一个打开的tab
        if (activeTab === tabId && newOpenTabs.length > 0) {
            setActiveTab(newOpenTabs[newOpenTabs.length - 1])
        }
    }

    // 处理tab打开
    const handleTabOpen = (tabId: string) => {
        if (!openTabs.includes(tabId)) {
            setOpenTabs([...openTabs, tabId])
        }
        setActiveTab(tabId)
    }

    return (
        <div className="flex h-screen bg-gray-50 dark:bg-gray-900">
            {/* 侧边栏 */}
            <Sidebar activeTab={activeTab} onTabChange={handleTabChange} />

            {/* 主内容区 */}
            <div className="flex flex-1 flex-col overflow-hidden">
                {/* 顶部导航栏 */}
                <Header />

                {/* Tab管理器 */}
                <TabManager
                    activeTab={activeTab}
                    openTabs={openTabs}
                    onTabChange={handleTabChange}
                    onTabClose={handleTabClose}
                    onTabOpen={handleTabOpen}
                />

                {/* Tab内容 */}
                <main className="flex-1 overflow-hidden bg-gray-50 dark:bg-gray-900">
                    <div className="relative h-full">
                        {openTabs.map((tabId) => (
                            <div
                                key={tabId}
                                className={cn(
                                    'absolute inset-0 overflow-y-auto transition-all duration-300',
                                    activeTab === tabId
                                        ? 'opacity-100 translate-x-0 z-10'
                                        : 'opacity-0 translate-x-4 pointer-events-none z-0'
                                )}
                            >
                                <div className="container mx-auto px-6 py-8">
                                    {tabContents[tabId]}
                                </div>
                            </div>
                        ))}
                    </div>
                </main>
            </div>
        </div>
    )
}
