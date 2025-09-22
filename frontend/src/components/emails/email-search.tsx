'use client'

import { useState, useCallback } from 'react'
import { Search, Calendar, User, FileText, Paperclip, Check } from 'lucide-react'
import { emailService } from '@/services/email.service'
import { Email } from '@/types'
import { formatDate } from '@/lib/utils'

interface EmailSearchProps {
    mode?: 'browse' | 'select'
    onSelect?: (email: Email) => void
    selectedEmail?: Email | null
    maxHeight?: string
}

export default function EmailSearch({
    mode = 'browse',
    onSelect,
    selectedEmail,
    maxHeight = 'h-full'
}: EmailSearchProps) {
    const [searchParams, setSearchParams] = useState({
        keyword: '',
        from_query: '',
        to_query: '',
        cc_query: '',
        subject_query: '',
        body_query: '',
        start_date: '',
        end_date: '',
        mailbox: '',
        has_attachments: false,
        account_id: ''
    })

    const [searchResults, setSearchResults] = useState<Email[]>([])
    const [loading, setLoading] = useState(false)
    const [totalCount, setTotalCount] = useState(0)
    const [hasSearched, setHasSearched] = useState(false)

    const handleSearch = useCallback(async () => {
        setLoading(true)
        try {
            const response = await emailService.searchEmails({
                ...searchParams,
                limit: 20
            })
            setSearchResults(response.emails || [])
            setTotalCount(response.total || 0)
            setHasSearched(true)
        } catch (error) {
            console.error('Search failed:', error)
            setSearchResults([])
            setTotalCount(0)
            setHasSearched(true)
        } finally {
            setLoading(false)
        }
    }, [searchParams])

    const handleEmailClick = useCallback((email: Email) => {
        if (mode === 'select' && onSelect) {
            onSelect(email)
        }
    }, [mode, onSelect])

    const handleKeywordSearch = useCallback(async () => {
        if (searchParams.keyword.trim()) {
            await handleSearch()
        }
    }, [searchParams.keyword, handleSearch])

    const handleResetSearch = useCallback(() => {
        setSearchParams({
            keyword: '',
            from_query: '',
            to_query: '',
            cc_query: '',
            subject_query: '',
            body_query: '',
            start_date: '',
            end_date: '',
            mailbox: '',
            has_attachments: false,
            account_id: ''
        })
        setSearchResults([])
        setTotalCount(0)
        setHasSearched(false)
    }, [])

    return (
        <div className={`flex ${maxHeight}`}>
            {/* 搜索表单区 */}
            <div className="w-96 border-r border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800">
                <h2 className="mb-6 text-xl font-semibold text-gray-900 dark:text-white">
                    {mode === 'select' ? '选择邮件' : '高级邮件搜索'}
                </h2>

                <div className="space-y-4">
                    {/* 全局关键词搜索 */}
                    <div>
                        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                            关键词
                        </label>
                        <div className="relative">
                            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                            <input
                                type="text"
                                value={searchParams.keyword}
                                onChange={(e) => setSearchParams({ ...searchParams, keyword: e.target.value })}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        handleKeywordSearch()
                                    }
                                }}
                                placeholder="在所有字段中搜索..."
                                className="w-full rounded-lg border border-gray-300 bg-white py-2 pl-9 pr-3 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                            />
                        </div>
                    </div>

                    {/* 发件人搜索 */}
                    <div>
                        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                            发件人
                        </label>
                        <div className="relative">
                            <User className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                            <input
                                type="text"
                                value={searchParams.from_query}
                                onChange={(e) => setSearchParams({ ...searchParams, from_query: e.target.value })}
                                placeholder="发件人邮箱或姓名"
                                className="w-full rounded-lg border border-gray-300 bg-white py-2 pl-9 pr-3 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                            />
                        </div>
                    </div>

                    {/* 收件人搜索 */}
                    <div>
                        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                            收件人
                        </label>
                        <div className="relative">
                            <User className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                            <input
                                type="text"
                                value={searchParams.to_query}
                                onChange={(e) => setSearchParams({ ...searchParams, to_query: e.target.value })}
                                placeholder="收件人邮箱或姓名"
                                className="w-full rounded-lg border border-gray-300 bg-white py-2 pl-9 pr-3 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                            />
                        </div>
                    </div>

                    {/* 主题搜索 */}
                    <div>
                        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                            主题
                        </label>
                        <div className="relative">
                            <FileText className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                            <input
                                type="text"
                                value={searchParams.subject_query}
                                onChange={(e) => setSearchParams({ ...searchParams, subject_query: e.target.value })}
                                placeholder="邮件主题关键词"
                                className="w-full rounded-lg border border-gray-300 bg-white py-2 pl-9 pr-3 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                            />
                        </div>
                    </div>

                    {/* 日期范围 */}
                    <div>
                        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                            日期范围
                        </label>
                        <div className="space-y-2">
                            <div className="relative">
                                <Calendar className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                                <input
                                    type="date"
                                    value={searchParams.start_date}
                                    onChange={(e) => setSearchParams({ ...searchParams, start_date: e.target.value })}
                                    className="w-full rounded-lg border border-gray-300 bg-white py-2 pl-9 pr-3 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                                />
                            </div>
                            <div className="relative">
                                <Calendar className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                                <input
                                    type="date"
                                    value={searchParams.end_date}
                                    onChange={(e) => setSearchParams({ ...searchParams, end_date: e.target.value })}
                                    className="w-full rounded-lg border border-gray-300 bg-white py-2 pl-9 pr-3 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                                />
                            </div>
                        </div>
                    </div>

                    {/* 邮箱文件夹 */}
                    <div>
                        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                            邮箱文件夹
                        </label>
                        <select
                            value={searchParams.mailbox}
                            onChange={(e) => setSearchParams({ ...searchParams, mailbox: e.target.value })}
                            className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                        >
                            <option value="">所有文件夹</option>
                            <option value="INBOX">收件箱</option>
                            <option value="Sent">已发送</option>
                            <option value="Drafts">草稿</option>
                            <option value="Trash">垃圾箱</option>
                        </select>
                    </div>

                    {/* 附件筛选 */}
                    <div className="flex items-center space-x-2">
                        <input
                            type="checkbox"
                            id="hasAttachments"
                            checked={searchParams.has_attachments}
                            onChange={(e) => setSearchParams({ ...searchParams, has_attachments: e.target.checked })}
                            className="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                        />
                        <label htmlFor="hasAttachments" className="flex items-center text-sm text-gray-700 dark:text-gray-300">
                            <Paperclip className="mr-1 h-4 w-4" />
                            仅显示有附件的邮件
                        </label>
                    </div>

                    {/* 操作按钮 */}
                    <div className="pt-4 space-y-2">
                        <button
                            onClick={handleSearch}
                            disabled={loading}
                            className="w-full rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-700 disabled:bg-gray-400"
                        >
                            {loading ? '搜索中...' : '开始搜索'}
                        </button>

                        {hasSearched && (
                            <button
                                onClick={handleResetSearch}
                                className="w-full rounded-lg bg-gray-200 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-300 dark:bg-gray-600 dark:text-gray-200 dark:hover:bg-gray-500"
                            >
                                重置搜索
                            </button>
                        )}
                    </div>
                </div>
            </div>

            {/* 搜索结果区 */}
            <div className="flex-1 overflow-y-auto bg-gray-50 p-6 dark:bg-gray-900">
                {!hasSearched ? (
                    <div className="flex h-full items-center justify-center">
                        <div className="text-center">
                            <Search className="mx-auto mb-4 h-12 w-12 text-gray-400" />
                            <p className="text-gray-500 dark:text-gray-400">
                                {mode === 'select' ? '请输入搜索条件查找邮件' : '请输入搜索条件开始搜索'}
                            </p>
                        </div>
                    </div>
                ) : searchResults.length === 0 ? (
                    <div className="flex h-full items-center justify-center">
                        <div className="text-center">
                            <Search className="mx-auto mb-4 h-12 w-12 text-gray-400" />
                            <p className="text-gray-500 dark:text-gray-400">
                                {loading ? '正在搜索...' : '没有找到匹配的邮件'}
                            </p>
                        </div>
                    </div>
                ) : (
                    <div>
                        <div className="mb-4 flex items-center justify-between">
                            <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                                搜索结果 ({totalCount} 封邮件)
                            </h3>
                            {mode === 'select' && selectedEmail && (
                                <div className="flex items-center text-sm text-green-600 dark:text-green-400">
                                    <Check className="mr-1 h-4 w-4" />
                                    已选择邮件
                                </div>
                            )}
                        </div>
                        <div className="space-y-2">
                            {searchResults.map((email) => (
                                <div
                                    key={email.ID}
                                    onClick={() => handleEmailClick(email)}
                                    className={`rounded-lg border p-4 transition-all ${mode === 'select'
                                            ? 'cursor-pointer hover:border-primary-500'
                                            : 'hover:bg-gray-50'
                                        } ${mode === 'select' && selectedEmail?.ID === email.ID
                                            ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                                            : 'border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800'
                                        } dark:hover:bg-gray-700`}
                                >
                                    <div className="flex items-start justify-between">
                                        <div className="flex-1">
                                            <div className="flex items-center space-x-2">
                                                <span className="text-sm font-medium text-gray-900 dark:text-white">
                                                    {Array.isArray(email.From) ? email.From[0] : email.From}
                                                </span>
                                                <span className="text-xs text-gray-500 dark:text-gray-400">
                                                    {formatDate(email.Date)}
                                                </span>
                                            </div>
                                            <h4 className="mt-1 text-sm font-medium text-gray-700 dark:text-gray-300">
                                                {email.Subject || '(无主题)'}
                                            </h4>
                                            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400 line-clamp-2">
                                                {email.Body}
                                            </p>
                                        </div>
                                        <div className="flex items-center space-x-2">
                                            {email.Attachments && email.Attachments.length > 0 && (
                                                <Paperclip className="h-4 w-4 text-gray-400" />
                                            )}
                                            {mode === 'select' && selectedEmail?.ID === email.ID && (
                                                <Check className="h-4 w-4 text-primary-600" />
                                            )}
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                )}
            </div>
        </div>
    )
}