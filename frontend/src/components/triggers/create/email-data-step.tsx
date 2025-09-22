'use client'

import { useState, useCallback, useEffect } from 'react'
import { Search, Mail, Edit, Check, Paperclip, RefreshCw } from 'lucide-react'
import { Email } from '@/types'
import { emailService } from '@/services/email.service'
import { formatDate } from '@/lib/utils'

interface EmailDataStepProps {
    data: any
    onDataChange: (key: string, value: any) => void
    onNext: () => void
    onPrevious: () => void
    stepStatus?: boolean[]
}

export default function EmailDataStep({ data, onDataChange }: EmailDataStepProps) {
    console.log('EmailDataStep rendering with data:', data)
    const emailData = data.emailData || { source: 'api', selectedEmail: null, isManualInput: false, manualEmailData: '' }
    console.log('EmailDataStep emailData:', emailData)

    const [mode, setMode] = useState<'select' | 'manual'>(
        emailData.isManualInput ? 'manual' : 'select'
    )
    console.log('EmailDataStep initial mode:', mode)
    const [searchKeyword, setSearchKeyword] = useState('')
    const [searchResults, setSearchResults] = useState<Email[]>([])
    const [loading, setLoading] = useState(false)
    const [hasSearched, setHasSearched] = useState(false)

    // 确保mode状态与data.emailData.isManualInput保持同步
    useEffect(() => {
        const currentEmailData = data.emailData || { source: 'api', selectedEmail: null, isManualInput: false, manualEmailData: '' }
        const newMode = currentEmailData.isManualInput ? 'manual' : 'select'
        console.log('useEffect triggered - isManualInput:', currentEmailData.isManualInput, 'setting mode to:', newMode)
        setMode(newMode)
    }, [data.emailData?.isManualInput])

    const handleEmailSelect = useCallback((email: Email) => {
        const currentEmailData = data.emailData || { source: 'api', selectedEmail: null, isManualInput: false, manualEmailData: '' }
        const newEmailData = {
            ...currentEmailData,
            selectedEmail: email,
            isManualInput: false,
            sampleData: email // 设置样本数据用于后续步骤
        }
        onDataChange('emailData', newEmailData)
    }, [onDataChange])

    const handleManualDataChange = useCallback((value: string) => {
        try {
            const parsedData = JSON.parse(value)
            const currentEmailData = data.emailData || { source: 'api', selectedEmail: null, isManualInput: false, manualEmailData: '' }
            const newEmailData = {
                ...currentEmailData,
                selectedEmail: null,
                isManualInput: true,
                manualEmailData: value,
                sampleData: parsedData // 设置样本数据用于后续步骤
            }
            onDataChange('emailData', newEmailData)
        } catch (e) {
            // 如果JSON解析失败，仍然保存原始数据
            const currentEmailData = data.emailData || { source: 'api', selectedEmail: null, isManualInput: false, manualEmailData: '' }
            const newEmailData = {
                ...currentEmailData,
                selectedEmail: null,
                isManualInput: true,
                manualEmailData: value
            }
            onDataChange('emailData', newEmailData)
        }
    }, [onDataChange])

    const handleModeChange = useCallback((newMode: 'select' | 'manual') => {
        console.log('handleModeChange called with:', newMode)
        alert(`模式切换到: ${newMode}`)
        setMode(newMode)
        const currentEmailData = data.emailData || { source: 'api', selectedEmail: null, isManualInput: false, manualEmailData: '' }
        const newEmailData = {
            ...currentEmailData,
            selectedEmail: newMode === 'manual' ? null : currentEmailData.selectedEmail,
            isManualInput: newMode === 'manual',
            manualEmailData: currentEmailData.manualEmailData
        }
        console.log('Updating emailData with:', newEmailData)
        onDataChange('emailData', newEmailData)
    }, [onDataChange])

    const handleSearch = useCallback(async () => {
        if (!searchKeyword.trim()) return

        setLoading(true)
        try {
            const response = await emailService.searchEmails({
                keyword: searchKeyword,
                limit: 10
            })
            setSearchResults(response.emails || [])
            setHasSearched(true)
        } catch (error) {
            console.error('Search failed:', error)
            setSearchResults([])
            setHasSearched(true)
        } finally {
            setLoading(false)
        }
    }, [searchKeyword])

    const handleRefresh = useCallback(async () => {
        setLoading(true)
        try {
            const response = await emailService.searchEmails({
                limit: 10
            })
            setSearchResults(response.emails || [])
            setHasSearched(true)
        } catch (error) {
            console.error('Refresh failed:', error)
            setSearchResults([])
            setHasSearched(true)
        } finally {
            setLoading(false)
        }
    }, [])

    return (
        <div className="space-y-6">
            <div>
                <h2 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                    第一步：选择邮件数据
                </h2>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                    选择现有邮件或手动输入示例数据用于配置触发器
                </p>
            </div>

            {/* 模式选择 */}
            <div className="flex space-x-4">
                <button
                    onClick={() => handleModeChange('select')}
                    className={`flex items-center space-x-2 px-4 py-2 rounded-lg border transition-colors ${mode === 'select'
                        ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300'
                        : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700'
                        }`}
                >
                    <Mail className="h-4 w-4" />
                    <span>选择邮件</span>
                </button>
                <button
                    onClick={() => {
                        alert('点击了手动输入按钮!')
                        handleModeChange('manual')
                    }}
                    className={`flex items-center space-x-2 px-4 py-2 rounded-lg border transition-colors ${mode === 'manual'
                        ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300'
                        : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700'
                        }`}
                >
                    <Edit className="h-4 w-4" />
                    <span>手动输入</span>
                </button>
            </div>

            {/* 内容区域 */}
            <div className="border border-gray-200 rounded-lg dark:border-gray-700">
                {mode === 'select' ? (
                    <div className="p-6">
                        {/* 搜索框 */}
                        <div className="mb-4 flex space-x-2">
                            <div className="flex-1 relative">
                                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                                <input
                                    type="text"
                                    value={searchKeyword}
                                    onChange={(e) => setSearchKeyword(e.target.value)}
                                    onKeyDown={(e) => {
                                        if (e.key === 'Enter') {
                                            handleSearch()
                                        }
                                    }}
                                    placeholder="搜索邮件内容..."
                                    className="w-full rounded-lg border border-gray-300 bg-white py-2 pl-9 pr-3 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
                                />
                            </div>
                            <button
                                onClick={handleSearch}
                                disabled={loading || !searchKeyword.trim()}
                                className="px-4 py-2 bg-primary-600 text-white rounded-lg text-sm font-medium hover:bg-primary-700 disabled:bg-gray-400 disabled:cursor-not-allowed"
                            >
                                {loading ? '搜索中...' : '搜索'}
                            </button>
                            <button
                                onClick={handleRefresh}
                                disabled={loading}
                                className="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg text-sm font-medium hover:bg-gray-300 disabled:bg-gray-400 disabled:cursor-not-allowed dark:bg-gray-600 dark:text-gray-200 dark:hover:bg-gray-500"
                            >
                                <RefreshCw className="h-4 w-4" />
                            </button>
                        </div>

                        {/* 邮件列表 */}
                        <div className="space-y-2 max-h-80 overflow-y-auto">
                            {!hasSearched && !loading ? (
                                <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                                    <Mail className="mx-auto h-12 w-12 mb-2 text-gray-400" />
                                    <p>请搜索邮件或点击刷新按钮加载最新邮件</p>
                                </div>
                            ) : loading ? (
                                <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                                    <RefreshCw className="mx-auto h-8 w-8 mb-2 animate-spin text-gray-400" />
                                    <p>正在加载...</p>
                                </div>
                            ) : searchResults.length === 0 ? (
                                <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                                    <Search className="mx-auto h-12 w-12 mb-2 text-gray-400" />
                                    <p>没有找到匹配的邮件</p>
                                </div>
                            ) : (
                                searchResults.map((email) => (
                                    <div
                                        key={email.ID}
                                        onClick={() => handleEmailSelect(email)}
                                        className={`p-3 rounded-lg border cursor-pointer transition-all ${emailData.selectedEmail?.ID === email.ID
                                            ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                                            : 'border-gray-200 bg-white hover:border-primary-300 dark:border-gray-600 dark:bg-gray-800 dark:hover:border-primary-500'
                                            }`}
                                    >
                                        <div className="flex items-start justify-between">
                                            <div className="flex-1 min-w-0">
                                                <div className="flex items-center space-x-2 mb-1">
                                                    <span className="text-sm font-medium text-gray-900 dark:text-white truncate">
                                                        {Array.isArray(email.From) ? email.From[0] : email.From}
                                                    </span>
                                                    <span className="text-xs text-gray-500 dark:text-gray-400">
                                                        {formatDate(email.Date)}
                                                    </span>
                                                </div>
                                                <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1 truncate">
                                                    {email.Subject || '(无主题)'}
                                                </h4>
                                                <p className="text-xs text-gray-500 dark:text-gray-400 line-clamp-2">
                                                    {email.Body}
                                                </p>
                                            </div>
                                            <div className="flex items-center space-x-2 ml-2">
                                                {email.Attachments && email.Attachments.length > 0 && (
                                                    <Paperclip className="h-4 w-4 text-gray-400" />
                                                )}
                                                {emailData.selectedEmail?.ID === email.ID && (
                                                    <Check className="h-4 w-4 text-primary-600" />
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                ))
                            )}
                        </div>
                    </div>
                ) : (
                    <div className="p-6">
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                    邮件数据 (JSON 格式)
                                </label>
                                <textarea
                                    value={emailData.manualEmailData}
                                    onChange={(e) => handleManualDataChange(e.target.value)}
                                    placeholder={`请输入邮件数据的JSON格式，例如：
{
  "ID": "example-id",
  "From": ["sender@example.com"],
  "To": ["recipient@example.com"],
  "Subject": "邮件主题",
  "Body": "邮件内容",
  "Date": "2024-01-01T00:00:00Z",
  "Attachments": []
}`}
                                    className="w-full h-64 px-3 py-2 border border-gray-300 rounded-lg font-mono text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                                />
                            </div>
                            <div className="text-xs text-gray-500 dark:text-gray-400">
                                <p>提示：您可以输入标准的邮件JSON数据格式。这个数据将用于后续的表达式测试和动作配置。</p>
                            </div>
                        </div>
                    </div>
                )}
            </div>

            {/* 当前选择状态 */}
            {mode === 'select' && emailData.selectedEmail && (
                <div className="bg-green-50 border border-green-200 rounded-lg p-4 dark:bg-green-900/20 dark:border-green-800">
                    <h3 className="text-sm font-medium text-green-800 dark:text-green-200 mb-2">
                        已选择邮件
                    </h3>
                    <div className="text-sm text-green-700 dark:text-green-300">
                        <p><strong>发件人:</strong> {Array.isArray(emailData.selectedEmail.From) ? emailData.selectedEmail.From[0] : emailData.selectedEmail.From}</p>
                        <p><strong>主题:</strong> {emailData.selectedEmail.Subject || '(无主题)'}</p>
                        <p><strong>时间:</strong> {new Date(emailData.selectedEmail.Date).toLocaleString()}</p>
                    </div>
                </div>
            )}

            {mode === 'manual' && emailData.manualEmailData && (
                <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 dark:bg-blue-900/20 dark:border-blue-800">
                    <h3 className="text-sm font-medium text-blue-800 dark:text-blue-200 mb-2">
                        手动输入数据
                    </h3>
                    <div className="text-sm text-blue-700 dark:text-blue-300">
                        <p>已输入 {emailData.manualEmailData.length} 个字符的邮件数据</p>
                        <p>数据将用于后续的表达式测试和动作配置</p>
                    </div>
                </div>
            )}
        </div>
    )
}