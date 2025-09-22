'use client'

import { cn } from '@/lib/utils'
import {
    LayoutDashboard,
    Mail,
    UserCog,
    Settings,
    ChevronLeft,
    ChevronRight,
    LogOut,
    Moon,
    Sun,
    FileText,
    Bot,
    Inbox,
    Bell,
    RefreshCw,
    Zap,
    Key,
    Sparkles,
    PlusCircle,
    BarChart3,
    TestTube,
    ChevronDown,
    ChevronUp,
    PlayCircle,
    Puzzle,
    Code2,
    Bug,
    FlaskConical,
    type LucideIcon,
} from 'lucide-react'
import { useState } from 'react'
import { useTheme } from '@/components/theme-provider'

// 菜单项接口定义
interface MenuItem {
    name: string
    id: string
    icon: LucideIcon
    description?: string
}

interface MenuGroup {
    title: string
    icon: LucideIcon
    items: MenuItem[]
    expanded: boolean
    setExpanded: (expanded: boolean) => void
}

const navigation: MenuItem[] = [
    { name: '仪表板', id: 'dashboard', icon: LayoutDashboard },
    { name: '邮箱账户管理', id: 'accounts', icon: UserCog },
    { name: '邮件管理', id: 'emails', icon: Mail },
    { name: '经典邮件管理器', id: 'classic-mailbox', icon: Inbox },
    { name: '同步配置', id: 'sync-config', icon: RefreshCw },
    { name: '取件', id: 'mail-pickup', icon: Inbox },
    // 根据需求隐藏订阅管理菜单项
    // { name: '订阅管理', id: 'subscriptions', icon: Bell },
    { name: '取件模板', id: 'pickup', icon: FileText },
    { name: 'OAuth2 配置', id: 'oauth2-config', icon: Key },
    { name: 'AI 配置', id: 'ai-config', icon: Bot },
    { name: '访问令牌', id: 'user-sessions', icon: Settings },
    { name: '系统配置', id: 'system-config', icon: Settings },
    // 根据需求隐藏设置菜单项
    // { name: '设置', id: 'settings', icon: Settings },
]

// 高级触发器菜单组 - 简化版本
const triggerNavigation: MenuItem[] = [
    { name: '触发器管理', id: 'triggers', icon: Zap, description: '管理所有触发器规则' },
    { name: '创建触发器', id: 'trigger-create', icon: PlusCircle, description: '多步骤触发器创建向导' },
    { name: '高级调试器', id: 'trigger-advanced-debug', icon: TestTube, description: '过滤动作触发器调试工具' },
]

// 插件管理菜单组
const pluginNavigation: MenuItem[] = [
    { name: '插件列表', id: 'plugins', icon: Puzzle, description: '管理所有插件' },
]

// 开发者模式菜单组
const developerNavigation: MenuItem[] = [
    { name: '表达式调试器', id: 'expression-debugger', icon: Bug, description: '调试条件表达式' },
    { name: '动作调试器', id: 'action-debugger', icon: PlayCircle, description: '调试动作插件' },
    { name: '过滤动作触发器', id: 'filter-action-trigger', icon: Zap, description: '完整触发器调试' },
    { name: '组件测试', id: 'component-test', icon: FlaskConical, description: '测试UI组件' },
]

interface SidebarProps {
    activeTab: string
    onTabChange: (tab: string) => void
}

// 统一的菜单项组件
function MenuItemComponent({
    item,
    isActive,
    collapsed,
    onClick,
    isSubItem = false
}: {
    item: MenuItem
    isActive: boolean
    collapsed: boolean
    onClick: () => void
    isSubItem?: boolean
}) {
    return (
        <button
            onClick={onClick}
            className={cn(
                'group flex w-full items-center rounded-lg py-2 text-sm font-medium transition-colors',
                isActive
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/20 dark:text-primary-400'
                    : 'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800',
                collapsed && 'justify-center',
                // 根据是否为子菜单项调整padding
                isSubItem ? 'px-6' : 'px-3'
            )}
            title={collapsed ? item.name : undefined}
        >
            <item.icon
                className={cn(
                    'h-5 w-5 shrink-0',
                    isActive
                        ? 'text-primary-600 dark:text-primary-400'
                        : 'text-gray-400 group-hover:text-gray-500 dark:text-gray-500 dark:group-hover:text-gray-400'
                )}
            />
            {!collapsed && (
                <div className="ml-3 flex-1 text-left">
                    <div className="font-medium">{item.name}</div>
                    {item.description && (
                        <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                            {item.description}
                        </div>
                    )}
                </div>
            )}
        </button>
    )
}

// 统一的菜单组组件
function MenuGroupComponent({
    group,
    collapsed,
    activeTab,
    onTabChange
}: {
    group: MenuGroup
    collapsed: boolean
    activeTab: string
    onTabChange: (tab: string) => void
}) {
    return (
        <div className="pt-6">
            {/* 分组标题 - 使用分隔线样式 */}
            <div className="px-3 mb-3">
                <div className="flex items-center mb-2">
                    <div className="flex-1 border-t border-gray-200 dark:border-gray-700"></div>
                </div>
                <button
                    onClick={() => group.setExpanded(!group.expanded)}
                    className={cn(
                        'flex w-full items-center text-xs font-semibold uppercase tracking-wide text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 py-1',
                        collapsed && 'justify-center'
                    )}
                >
                    <group.icon className="h-4 w-4 mr-2 text-gray-500 dark:text-gray-400" />
                    {!collapsed && (
                        <>
                            <span className="flex-1 text-left">{group.title}</span>
                            <div className="transition-transform duration-200 ease-in-out">
                                {group.expanded ? (
                                    <ChevronUp className="h-3 w-3" />
                                ) : (
                                    <ChevronDown className="h-3 w-3" />
                                )}
                            </div>
                        </>
                    )}
                </button>
            </div>

            {/* 菜单项 */}
            <div
                className={cn(
                    'overflow-hidden transition-all duration-200 ease-in-out',
                    (group.expanded || collapsed) ? 'max-h-screen opacity-100' : 'max-h-0 opacity-0'
                )}
            >
                <div className="space-y-1">
                    {group.items.map((item) => (
                        <MenuItemComponent
                            key={item.id}
                            item={item}
                            isActive={activeTab === item.id}
                            collapsed={collapsed}
                            onClick={() => onTabChange && onTabChange(item.id)}
                            isSubItem={true}
                        />
                    ))}
                </div>
            </div>
        </div>
    )
}

export function Sidebar({ activeTab, onTabChange }: SidebarProps) {
    const [collapsed, setCollapsed] = useState(false)
    const [triggerMenuExpanded, setTriggerMenuExpanded] = useState(true)
    const [pluginMenuExpanded, setPluginMenuExpanded] = useState(true)
    const [developerMenuExpanded, setDeveloperMenuExpanded] = useState(true)
    const { theme, setTheme } = useTheme()

    // 菜单组配置
    const menuGroups: MenuGroup[] = [
        {
            title: '高级触发器',
            icon: Zap,
            items: triggerNavigation,
            expanded: triggerMenuExpanded,
            setExpanded: setTriggerMenuExpanded,
        },
        {
            title: '插件管理',
            icon: Puzzle,
            items: pluginNavigation,
            expanded: pluginMenuExpanded,
            setExpanded: setPluginMenuExpanded,
        },
        {
            title: '开发者模式',
            icon: Code2,
            items: developerNavigation,
            expanded: developerMenuExpanded,
            setExpanded: setDeveloperMenuExpanded,
        },
    ]

    return (
        <div
            className={cn(
                'relative flex h-screen flex-col border-r border-gray-200 bg-white transition-all duration-300 dark:border-gray-800 dark:bg-gray-900',
                collapsed ? 'w-16' : 'w-64'
            )}
        >
            {/* Logo */}
            <div className="flex-shrink-0 flex h-16 items-center justify-between border-b border-gray-200 px-4 dark:border-gray-800">
                <div
                    className={cn(
                        'flex items-center space-x-2 text-lg font-semibold',
                        collapsed && 'justify-center'
                    )}
                >
                    <Mail className="h-6 w-6 text-primary-600" />
                    {!collapsed && <span>邮箱管理系统</span>}
                </div>
            </div>

            {/* Navigation */}
            <nav className="flex-1 min-h-0 space-y-1 px-2 py-4 overflow-y-auto">
                {/* 常规菜单 */}
                {navigation.map((item) => (
                    <MenuItemComponent
                        key={item.id}
                        item={item}
                        isActive={activeTab === item.id}
                        collapsed={collapsed}
                        onClick={() => onTabChange && onTabChange(item.id)}
                    />
                ))}

                {/* 菜单组 */}
                {menuGroups.map((group) => (
                    <MenuGroupComponent
                        key={group.title}
                        group={group}
                        collapsed={collapsed}
                        activeTab={activeTab}
                        onTabChange={onTabChange}
                    />
                ))}
            </nav>

            {/* Bottom section */}
            <div className="flex-shrink-0 border-t border-gray-200 p-4 dark:border-gray-800">
                {/* Theme toggle */}
                <button
                    onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
                    className={cn(
                        'mb-2 flex w-full items-center rounded-lg px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800',
                        collapsed && 'justify-center'
                    )}
                >
                    {theme === 'dark' ? (
                        <Sun className="h-5 w-5" />
                    ) : (
                        <Moon className="h-5 w-5" />
                    )}
                    {!collapsed && <span className="ml-3">切换主题</span>}
                </button>

                {/* Logout */}
                <button
                    className={cn(
                        'flex w-full items-center rounded-lg px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800',
                        collapsed && 'justify-center'
                    )}
                >
                    <LogOut className="h-5 w-5" />
                    {!collapsed && <span className="ml-3">退出登录</span>}
                </button>
            </div>

            {/* Collapse toggle */}
            <button
                onClick={() => setCollapsed(!collapsed)}
                className="absolute -right-3 top-20 flex h-6 w-6 items-center justify-center rounded-full border border-gray-300 bg-white text-gray-600 transition-colors hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700"
            >
                {collapsed ? (
                    <ChevronRight className="h-4 w-4" />
                ) : (
                    <ChevronLeft className="h-4 w-4" />
                )}
            </button>
        </div>
    )
}