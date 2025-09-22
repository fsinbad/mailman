'use client'

import React, { useState, useEffect } from 'react'
import { Star, Paperclip, Reply, ReplyAll, Forward, Archive, Trash2, Code, Printer, Download } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Email } from '@/types'
import { formatDate } from '@/lib/utils'

interface EmailPreviewPanelProps {
    email: Email | null
    loading: boolean
}

// 使用React.memo优化性能，自定义比较函数
const EmailPreviewPanel = React.memo(function EmailPreviewPanel({
    email,
    loading
}: EmailPreviewPanelProps) {
    const [showRawContent, setShowRawContent] = useState(false)
    const [isStarred, setIsStarred] = useState(false)

    // 检查邮件是否已收藏
    useEffect(() => {
        if (email) {
            const starredEmails = JSON.parse(localStorage.getItem('starredEmails') || '[]')
            setIsStarred(starredEmails.includes(email.ID))
        }
    }, [email])

    // 切换收藏状态
    const toggleStar = () => {
        if (!email) return

        const starredEmails = JSON.parse(localStorage.getItem('starredEmails') || '[]')
        let newStarredEmails

        if (isStarred) {
            newStarredEmails = starredEmails.filter((id: number) => id !== email.ID)
        } else {
            newStarredEmails = [...starredEmails, email.ID]
        }

        localStorage.setItem('starredEmails', JSON.stringify(newStarredEmails))
        setIsStarred(!isStarred)
    }

    // 格式化邮箱地址列表
    const formatEmailList = (emails: string[] | string | undefined) => {
        if (!emails) return ''
        if (typeof emails === 'string') return emails
        return emails.join(', ')
    }

    // 打印邮件
    const handlePrint = () => {
        window.print()
    }

    if (loading) {
        return (
            <div className="h-full flex items-center justify-center bg-white dark:bg-gray-800">
                <div className="text-center">
                    <div className="w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
                    <p className="text-gray-500 dark:text-gray-400">加载中...</p>
                </div>
            </div>
        )
    }

    if (!email) {
        return (
            <div className="h-full flex flex-col bg-white dark:bg-gray-800">
                {/* 顶部占位区域 */}
                <div className="border-b border-gray-200 dark:border-gray-700 p-4">
                    <div className="flex items-center justify-between">
                        <h2 className="text-lg font-semibold text-gray-400 dark:text-gray-500">
                            邮件预览
                        </h2>
                        <div className="flex items-center gap-2 opacity-50">
                            <button className="p-2 rounded-lg text-gray-300 dark:text-gray-600" disabled>
                                <Star className="h-4 w-4" />
                            </button>
                            <button className="p-2 rounded-lg text-gray-300 dark:text-gray-600" disabled>
                                <Printer className="h-4 w-4" />
                            </button>
                            <button className="p-2 rounded-lg text-gray-300 dark:text-gray-600" disabled>
                                <Code className="h-4 w-4" />
                            </button>
                        </div>
                    </div>
                </div>

                {/* 主内容区域 */}
                <div className="flex-1 flex flex-col">
                    {/* 中心提示区域 */}
                    <div className="flex-1 flex items-center justify-center min-h-0 p-8">
                        <div className="text-center max-w-md">
                            <div className="w-20 h-20 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-6">
                                <Reply className="h-10 w-10 text-gray-400 dark:text-gray-500" />
                            </div>
                            <h3 className="text-xl font-medium text-gray-900 dark:text-white mb-2">
                                选择邮件进行预览
                            </h3>
                            <p className="text-gray-500 dark:text-gray-400 mb-6 leading-relaxed">
                                从左侧邮件列表中选择一封邮件，在这里查看详细内容、回复或转发邮件。
                            </p>

                            {/* 功能介绍卡片 */}
                            <div className="grid grid-cols-2 gap-3 text-sm">
                                <div className="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                                    <Reply className="h-5 w-5 text-blue-500 mx-auto mb-2" />
                                    <p className="text-gray-600 dark:text-gray-400">快速回复</p>
                                </div>
                                <div className="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                                    <Paperclip className="h-5 w-5 text-green-500 mx-auto mb-2" />
                                    <p className="text-gray-600 dark:text-gray-400">查看附件</p>
                                </div>
                                <div className="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                                    <Star className="h-5 w-5 text-yellow-500 mx-auto mb-2" />
                                    <p className="text-gray-600 dark:text-gray-400">收藏重要邮件</p>
                                </div>
                                <div className="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                                    <Forward className="h-5 w-5 text-purple-500 mx-auto mb-2" />
                                    <p className="text-gray-600 dark:text-gray-400">转发分享</p>
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* 底部提示区域 */}
                    <div className="border-t border-gray-200 dark:border-gray-700 p-4 bg-gray-50 dark:bg-gray-800">
                        <div className="text-center">
                            <p className="text-xs text-gray-500 dark:text-gray-400">
                                提示：使用键盘上下箭头键可快速切换邮件
                            </p>
                        </div>
                    </div>
                </div>
            </div>
        )
    }

    return (
        <div className="h-full flex flex-col bg-white dark:bg-gray-800">
            {/* 邮件头部工具栏 */}
            <div className="border-b border-gray-200 dark:border-gray-700 p-4">
                <div className="flex items-center justify-between mb-4">
                    <h2 className="text-lg font-semibold text-gray-900 dark:text-white truncate mr-4">
                        {email.Subject || '(无主题)'}
                    </h2>
                    <div className="flex items-center gap-2 shrink-0">
                        <button
                            onClick={toggleStar}
                            className={cn(
                                "p-2 rounded-lg transition-colors",
                                isStarred
                                    ? "text-yellow-500 bg-yellow-50 dark:bg-yellow-900/20"
                                    : "text-gray-400 hover:text-yellow-500 hover:bg-gray-100 dark:hover:bg-gray-700"
                            )}
                            title={isStarred ? "取消收藏" : "收藏"}
                        >
                            <Star className={cn("h-4 w-4", isStarred && "fill-current")} />
                        </button>
                        <button
                            onClick={handlePrint}
                            className="p-2 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700 dark:hover:text-gray-300"
                            title="打印"
                        >
                            <Printer className="h-4 w-4" />
                        </button>
                        <button
                            onClick={() => setShowRawContent(!showRawContent)}
                            className="p-2 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:bg-gray-700 dark:hover:text-gray-300"
                            title={showRawContent ? "显示格式化内容" : "显示原始内容"}
                        >
                            <Code className="h-4 w-4" />
                        </button>
                    </div>
                </div>

                {/* 邮件基本信息 */}
                <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                        <div className="flex items-center gap-4">
                            <span className="font-medium text-gray-900 dark:text-white">
                                发件人: {formatEmailList(email.From)}
                            </span>
                            {email.Attachments && email.Attachments.length > 0 && (
                                <div className="flex items-center gap-1 text-gray-500 dark:text-gray-400">
                                    <Paperclip className="h-4 w-4" />
                                    <span>{email.Attachments.length} 个附件</span>
                                </div>
                            )}
                        </div>
                        <span className="text-gray-500 dark:text-gray-400">
                            {formatDate(email.Date)}
                        </span>
                    </div>

                    {email.To && (
                        <div className="text-sm text-gray-600 dark:text-gray-300">
                            收件人: {formatEmailList(email.To)}
                        </div>
                    )}

                    {email.Cc && (
                        <div className="text-sm text-gray-600 dark:text-gray-300">
                            抄送: {formatEmailList(email.Cc)}
                        </div>
                    )}
                </div>

                {/* 邮件操作按钮 */}
                <div className="flex items-center gap-2 mt-4">
                    <button className="flex items-center gap-2 px-3 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors">
                        <Reply className="h-4 w-4" />
                        回复
                    </button>
                    <button className="flex items-center gap-2 px-3 py-1.5 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors">
                        <ReplyAll className="h-4 w-4" />
                        全部回复
                    </button>
                    <button className="flex items-center gap-2 px-3 py-1.5 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors">
                        <Forward className="h-4 w-4" />
                        转发
                    </button>
                    <div className="flex-1"></div>
                    <button className="p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
                        <Archive className="h-4 w-4" />
                    </button>
                    <button className="p-1.5 text-gray-400 hover:text-red-600 dark:hover:text-red-400 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
                        <Trash2 className="h-4 w-4" />
                    </button>
                </div>
            </div>

            {/* 邮件内容 */}
            <div className="flex-1 overflow-y-auto p-4">
                {showRawContent ? (
                    <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-4">
                        <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-3">原始邮件内容</h4>
                        <pre className="text-xs text-gray-700 dark:text-gray-300 whitespace-pre-wrap overflow-x-auto">
                            {email.RawMessage || '原始内容不可用'}
                        </pre>
                    </div>
                ) : (
                    <div className="prose prose-sm max-w-none dark:prose-invert">
                        {email.HTMLBody ? (
                            <div
                                className="email-content"
                                dangerouslySetInnerHTML={{ __html: email.HTMLBody }}
                            />
                        ) : (
                            <div className="whitespace-pre-wrap text-gray-700 dark:text-gray-300">
                                {email.Body || '邮件内容为空'}
                            </div>
                        )}
                    </div>
                )}

                {/* 附件列表 */}
                {email.Attachments && email.Attachments.length > 0 && (
                    <div className="mt-6 pt-4 border-t border-gray-200 dark:border-gray-700">
                        <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                            <Paperclip className="h-4 w-4" />
                            附件 ({email.Attachments.length})
                        </h4>
                        <div className="grid gap-2">
                            {email.Attachments.map((attachment, index) => (
                                <div
                                    key={index}
                                    className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-900 rounded-lg"
                                >
                                    <div className="flex items-center gap-3">
                                        <div className="w-8 h-8 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center">
                                            <Paperclip className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                                        </div>
                                        <div>
                                            <p className="text-sm font-medium text-gray-900 dark:text-white">
                                                {attachment.filename || `附件${index + 1}`}
                                            </p>
                                            <p className="text-xs text-gray-500 dark:text-gray-400">
                                                {attachment.content_type} • {Math.round((attachment.size || 0) / 1024)}KB
                                            </p>
                                        </div>
                                    </div>
                                    <button className="p-2 text-gray-400 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded-lg transition-colors">
                                        <Download className="h-4 w-4" />
                                    </button>
                                </div>
                            ))}
                        </div>
                    </div>
                )}
            </div>
        </div>
    )
}, (prevProps, nextProps) => {
    // 自定义比较函数，只有当email或loading真正变化时才重新渲染
    if (prevProps.loading !== nextProps.loading) {
        return false // props有变化，需要重新渲染
    }

    // 如果都是null或都是非null但引用相同，则无需重新渲染
    if (prevProps.email === nextProps.email) {
        return true // props相同，跳过重新渲染
    }

    // 如果一个为null另一个不为null，需要重新渲染
    if ((prevProps.email === null) !== (nextProps.email === null)) {
        return false // props有变化，需要重新渲染
    }

    // 如果都不为null，比较关键字段
    if (prevProps.email && nextProps.email) {
        return (
            prevProps.email.ID === nextProps.email.ID &&
            prevProps.email.Subject === nextProps.email.Subject &&
            prevProps.email.Body === nextProps.email.Body &&
            prevProps.email.HTMLBody === nextProps.email.HTMLBody &&
            prevProps.email.Date === nextProps.email.Date &&
            JSON.stringify(prevProps.email.Attachments) === JSON.stringify(nextProps.email.Attachments)
        )
    }

    return true // 默认跳过重新渲染
})

export default EmailPreviewPanel
