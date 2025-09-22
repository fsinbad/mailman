'use client'

import React, { useState, useEffect, useRef } from 'react'
import { X, Mail, Check } from 'lucide-react'
import { cn } from '@/lib/utils'

// 通知类型定义
interface EmailNotification {
    type: string
    account_id: number
    account_email: string
    email_count: number
    subject?: string
    from?: string
    timestamp: string
}

// 通知项组件属性
interface NotificationItemProps {
    notification: EmailNotification
    onDismiss: () => void
    onClick: () => void
}

// 单个通知项组件
function NotificationItem({ notification, onDismiss, onClick }: NotificationItemProps) {
    const [isVisible, setIsVisible] = useState(false)
    const [isExiting, setIsExiting] = useState(false)

    useEffect(() => {
        // 进场动画
        const timer = setTimeout(() => setIsVisible(true), 100)
        return () => clearTimeout(timer)
    }, [])

    const handleDismiss = () => {
        setIsExiting(true)
        setTimeout(onDismiss, 300) // 等待退场动画完成
    }

    const handleClick = () => {
        onClick()
        handleDismiss()
    }

    return (
        <div
            className={cn(
                "transform transition-all duration-300 ease-in-out",
                "bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700",
                "p-4 mb-3 cursor-pointer hover:shadow-xl",
                "max-w-sm mx-auto",
                isVisible && !isExiting ? "translate-x-0 opacity-100" : "translate-x-full opacity-0"
            )}
            onClick={handleClick}
        >
            <div className="flex items-start justify-between">
                <div className="flex items-start space-x-3">
                    <div className="flex-shrink-0">
                        <div className="w-8 h-8 bg-blue-100 dark:bg-blue-900 rounded-full flex items-center justify-center">
                            <Mail className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                        </div>
                    </div>

                    <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium text-gray-900 dark:text-gray-100">
                            {notification.account_email}
                        </div>
                        <div className="text-sm text-gray-600 dark:text-gray-400">
                            收到 {notification.email_count} 封新邮件
                        </div>
                        {notification.subject && (
                            <div className="text-xs text-gray-500 dark:text-gray-500 truncate mt-1">
                                {notification.subject}
                            </div>
                        )}
                        <div className="text-xs text-gray-400 mt-1">
                            {new Date(notification.timestamp).toLocaleTimeString()}
                        </div>
                    </div>
                </div>

                <button
                    onClick={(e) => {
                        e.stopPropagation()
                        handleDismiss()
                    }}
                    className="flex-shrink-0 ml-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                >
                    <X className="w-4 h-4" />
                </button>
            </div>

            <div className="mt-3 flex justify-end">
                <button
                    className="text-xs bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded-md transition-colors"
                    onClick={(e) => {
                        e.stopPropagation()
                        handleClick()
                    }}
                >
                    点击查看
                </button>
            </div>
        </div>
    )
}

// 通知容器组件属性
interface EmailNotificationToastProps {
    onNotificationClick?: (accountId: number, accountEmail: string) => void
}

// 主通知容器组件
export default function EmailNotificationToast({ onNotificationClick }: EmailNotificationToastProps) {
    const [notifications, setNotifications] = useState<EmailNotification[]>([])
    const wsRef = useRef<WebSocket | null>(null)
    const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
    const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected'>('connecting')

    // WebSocket连接管理
    const connectWebSocket = () => {
        try {
            // 确定WebSocket URL
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
            const wsUrl = `${protocol}//${window.location.host}/api/ws/notifications`

            const ws = new WebSocket(wsUrl)
            wsRef.current = ws

            ws.onopen = () => {
                console.log('[EmailNotificationToast] WebSocket connected')
                setConnectionStatus('connected')

                // 清除重连定时器
                if (reconnectTimeoutRef.current) {
                    clearTimeout(reconnectTimeoutRef.current)
                    reconnectTimeoutRef.current = null
                }
            }

            ws.onmessage = (event) => {
                try {
                    const notification: EmailNotification = JSON.parse(event.data)

                    if (notification.type === 'new_email') {
                        console.log('[EmailNotificationToast] Received notification:', notification)

                        // 添加到通知列表（最多显示5个）
                        setNotifications(prev => {
                            const newNotifications = [notification, ...prev.slice(0, 4)]
                            return newNotifications
                        })

                        // 3秒后自动移除
                        setTimeout(() => {
                            setNotifications(prev =>
                                prev.filter(n => n.timestamp !== notification.timestamp)
                            )
                        }, 3000)
                    }
                } catch (error) {
                    console.error('[EmailNotificationToast] Failed to parse notification:', error)
                }
            }

            ws.onclose = (event) => {
                console.log('[EmailNotificationToast] WebSocket closed:', event.code, event.reason)
                setConnectionStatus('disconnected')
                wsRef.current = null

                // 如果不是手动关闭，尝试重连
                if (event.code !== 1000) {
                    scheduleReconnect()
                }
            }

            ws.onerror = (error) => {
                console.error('[EmailNotificationToast] WebSocket error:', error)
                setConnectionStatus('disconnected')
            }

        } catch (error) {
            console.error('[EmailNotificationToast] Failed to create WebSocket:', error)
            setConnectionStatus('disconnected')
            scheduleReconnect()
        }
    }

    // 安排重连
    const scheduleReconnect = () => {
        if (reconnectTimeoutRef.current) return

        console.log('[EmailNotificationToast] Scheduling reconnect in 5 seconds...')
        reconnectTimeoutRef.current = setTimeout(() => {
            console.log('[EmailNotificationToast] Attempting to reconnect...')
            setConnectionStatus('connecting')
            connectWebSocket()
        }, 5000)
    }

    // 断开连接
    const disconnect = () => {
        if (reconnectTimeoutRef.current) {
            clearTimeout(reconnectTimeoutRef.current)
            reconnectTimeoutRef.current = null
        }

        if (wsRef.current) {
            wsRef.current.close(1000, 'User disconnect')
            wsRef.current = null
        }
    }

    // 处理通知点击
    const handleNotificationClick = (notification: EmailNotification) => {
        console.log('[EmailNotificationToast] Notification clicked:', notification.account_email)

        if (onNotificationClick) {
            onNotificationClick(notification.account_id, notification.account_email)
        }
    }

    // 手动移除通知
    const handleDismissNotification = (timestamp: string) => {
        setNotifications(prev => prev.filter(n => n.timestamp !== timestamp))
    }

    // 组件生命周期
    useEffect(() => {
        connectWebSocket()

        return () => {
            disconnect()
        }
    }, [])

    // 连接状态指示器
    const renderConnectionStatus = () => {
        switch (connectionStatus) {
            case 'connecting':
                return (
                    <div className="fixed top-4 right-4 bg-yellow-100 text-yellow-800 px-3 py-1 rounded-md text-xs z-40">
                        正在连接通知服务...
                    </div>
                )
            case 'disconnected':
                return (
                    <div className="fixed top-4 right-4 bg-red-100 text-red-800 px-3 py-1 rounded-md text-xs z-40">
                        通知服务已断开
                    </div>
                )
            case 'connected':
                return null
        }
    }

    return (
        <>
            {/* 连接状态指示器 */}
            {renderConnectionStatus()}

            {/* 通知容器 */}
            <div className="fixed top-4 right-4 z-50 space-y-2">
                {notifications.map((notification) => (
                    <NotificationItem
                        key={notification.timestamp}
                        notification={notification}
                        onDismiss={() => handleDismissNotification(notification.timestamp)}
                        onClick={() => handleNotificationClick(notification)}
                    />
                ))}
            </div>
        </>
    )
}

// WebSocket状态Hook（可选，供其他组件使用）
export function useWebSocketConnection() {
    const [isConnected, setIsConnected] = useState(false)
    const [lastMessage, setLastMessage] = useState<EmailNotification | null>(null)

    useEffect(() => {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const wsUrl = `${protocol}//${window.location.host}/api/ws/notifications`

        const ws = new WebSocket(wsUrl)

        ws.onopen = () => setIsConnected(true)
        ws.onclose = () => setIsConnected(false)
        ws.onmessage = (event) => {
            try {
                const notification: EmailNotification = JSON.parse(event.data)
                setLastMessage(notification)
            } catch (error) {
                console.error('Failed to parse WebSocket message:', error)
            }
        }

        return () => {
            ws.close()
        }
    }, [])

    return { isConnected, lastMessage }
}