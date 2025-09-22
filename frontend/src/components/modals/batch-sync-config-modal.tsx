'use client'

import { useState, useEffect } from 'react'
import { X, Clock, Settings, AlertCircle, CheckCircle, XCircle, Loader2 } from 'lucide-react'
import { syncConfigService } from '@/services/sync-config.service'
import type { BatchSyncConfigRequest, BatchSyncConfigResponse } from '@/services/sync-config.service'
import { EmailAccount } from '@/types'
import { cn } from '@/lib/utils'

interface BatchSyncConfigModalProps {
    isOpen: boolean
    onClose: () => void
    onSuccess: () => void
    selectedAccounts: EmailAccount[]
}

const SYNC_INTERVALS = [
    { value: 30, label: '30秒', unit: 's' },
    { value: 60, label: '1分钟', unit: 's' },
    { value: 300, label: '5分钟', unit: 's' },
    { value: 600, label: '10分钟', unit: 's' },
    { value: 900, label: '15分钟', unit: 's' },
    { value: 1800, label: '30分钟', unit: 's' },
    { value: 3600, label: '1小时', unit: 's' },
    { value: 21600, label: '6小时', unit: 's' },
    { value: 43200, label: '12小时', unit: 's' },
    { value: 86400, label: '24小时', unit: 's' },
]

export default function BatchSyncConfigModal({ isOpen, onClose, onSuccess, selectedAccounts }: BatchSyncConfigModalProps) {
    const [enableAutoSync, setEnableAutoSync] = useState(true)
    const [syncInterval, setSyncInterval] = useState(300) // 默认5分钟
    const [customInterval, setCustomInterval] = useState('')
    const [useCustomInterval, setUseCustomInterval] = useState(false)

    const [loading, setLoading] = useState(false)
    const [result, setResult] = useState<BatchSyncConfigResponse | null>(null)
    const [showResult, setShowResult] = useState(false)

    // 重置表单状态
    const resetForm = () => {
        setEnableAutoSync(true)
        setSyncInterval(300)
        setCustomInterval('')
        setUseCustomInterval(false)
        setResult(null)
        setShowResult(false)
    }

    // 当模态框打开时重置表单
    useEffect(() => {
        if (isOpen) {
            resetForm()
        }
    }, [isOpen])

    const handleClose = () => {
        resetForm()
        onClose()
    }

    const handleSubmit = async () => {
        try {
            setLoading(true)

            // 计算实际同步间隔
            let actualInterval = syncInterval
            if (useCustomInterval && customInterval) {
                const customValue = parseInt(customInterval)
                if (isNaN(customValue) || customValue < 1) {
                    alert('自定义间隔必须大于等于30秒')
                    return
                }
                actualInterval = customValue
            }

            const accountIds = selectedAccounts.map(account => account.id)
            const configData: BatchSyncConfigRequest = {
                enable_auto_sync: enableAutoSync,
                sync_interval: actualInterval
                // sync_folders removed - system automatically handles all important folders
            }

            const response = await syncConfigService.batchCreateOrUpdateAccountSyncConfig(accountIds, configData)
            setResult(response)
            setShowResult(true)

            if (response.error_count === 0) {
                // 如果全部成功，延迟关闭模态框
                setTimeout(() => {
                    onSuccess()
                    handleClose()
                }, 2000)
            }

        } catch (error) {
            console.error('Failed to batch update sync config:', error)
            alert('批量更新同步配置失败')
        } finally {
            setLoading(false)
        }
    }

    if (!isOpen) return null

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg shadow-xl w-full max-w-md mx-4 dark:bg-gray-800">
                {!showResult ? (
                    <>
                        {/* 标题栏 */}
                        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
                            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                                批量同步配置
                            </h3>
                            <button
                                onClick={handleClose}
                                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                            >
                                <X className="h-5 w-5" />
                            </button>
                        </div>

                        {/* 内容区域 */}
                        <div className="p-6 space-y-6">
                            {/* 选中账户信息 */}
                            <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
                                <div className="flex items-center space-x-2">
                                    <Settings className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                                    <span className="text-sm font-medium text-blue-800 dark:text-blue-200">
                                        将为 {selectedAccounts.length} 个账户配置同步设置
                                    </span>
                                </div>
                                <div className="mt-2 text-xs text-blue-700 dark:text-blue-300">
                                    {selectedAccounts.slice(0, 3).map(account => account.emailAddress).join(', ')}
                                    {selectedAccounts.length > 3 && ` 等 ${selectedAccounts.length} 个账户`}
                                </div>
                            </div>

                            {/* 启用自动同步 */}
                            <div className="space-y-3">
                                <label className="flex items-center space-x-3 cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={enableAutoSync}
                                        onChange={(e) => setEnableAutoSync(e.target.checked)}
                                        className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                                    />
                                    <div>
                                        <span className="text-sm font-medium text-gray-900 dark:text-white">
                                            启用自动同步
                                        </span>
                                        <p className="text-xs text-gray-500 dark:text-gray-400">
                                            开启后将按照设定的间隔自动同步邮件
                                        </p>
                                    </div>
                                </label>
                            </div>

                            {/* 同步间隔设置 */}
                            {enableAutoSync && (
                                <div className="space-y-3">
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                                        同步间隔
                                    </label>

                                    {/* 预设间隔 */}
                                    <div className="grid grid-cols-3 gap-2">
                                        {SYNC_INTERVALS.map((interval) => (
                                            <button
                                                key={interval.value}
                                                onClick={() => {
                                                    setSyncInterval(interval.value)
                                                    setUseCustomInterval(false)
                                                }}
                                                className={cn(
                                                    "px-3 py-2 text-xs font-medium rounded-lg border transition-colors",
                                                    !useCustomInterval && syncInterval === interval.value
                                                        ? "bg-primary-600 text-white border-primary-600"
                                                        : "bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-700 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-600"
                                                )}
                                            >
                                                {interval.label}
                                            </button>
                                        ))}
                                    </div>

                                    {/* 自定义间隔 */}
                                    <div className="flex items-center space-x-3">
                                        <label className="flex items-center space-x-2">
                                            <input
                                                type="radio"
                                                checked={useCustomInterval}
                                                onChange={() => setUseCustomInterval(true)}
                                                className="text-primary-600 focus:ring-primary-500"
                                            />
                                            <span className="text-sm text-gray-700 dark:text-gray-300">
                                                自定义:
                                            </span>
                                        </label>
                                        <div className="flex items-center space-x-2">
                                            <input
                                                type="number"
                                                value={customInterval}
                                                onChange={(e) => {
                                                    setCustomInterval(e.target.value)
                                                    setUseCustomInterval(true)
                                                }}
                                                placeholder="300"
                                                min="30"
                                                className="w-20 px-2 py-1 text-sm border border-gray-300 rounded focus:outline-none focus:ring-1 focus:ring-primary-500 dark:bg-gray-700 dark:border-gray-600"
                                            />
                                            <span className="text-sm text-gray-500 dark:text-gray-400">秒</span>
                                        </div>
                                    </div>

                                    <p className="text-xs text-gray-500 dark:text-gray-400">
                                        请输入大于等于30秒的同步间隔
                                    </p>
                                </div>
                            )}

                            {/* 重要提示 */}
                            <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3">
                                <div className="flex items-start space-x-2">
                                    <AlertCircle className="h-4 w-4 text-yellow-600 dark:text-yellow-400 mt-0.5" />
                                    <div className="text-xs text-yellow-700 dark:text-yellow-300">
                                        <p className="font-medium">重要说明:</p>
                                        <ul className="mt-1 space-y-1">
                                            <li>• 将会覆盖所选账户的现有同步配置</li>
                                            <li>• 如果账户状态一致，不会报错，正常更新</li>
                                            <li>• 建议在非高峰期进行批量配置</li>
                                        </ul>
                                    </div>
                                </div>
                            </div>
                        </div>

                        {/* 操作按钮 */}
                        <div className="flex items-center justify-end space-x-3 p-6 bg-gray-50 dark:bg-gray-700">
                            <button
                                onClick={handleClose}
                                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 dark:bg-gray-600 dark:text-gray-300 dark:border-gray-500 dark:hover:bg-gray-500"
                            >
                                取消
                            </button>
                            <button
                                onClick={handleSubmit}
                                disabled={loading}
                                className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-white bg-primary-600 border border-transparent rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
                            >
                                {loading && <Loader2 className="h-4 w-4 animate-spin" />}
                                <span>{loading ? '配置中...' : '批量配置'}</span>
                            </button>
                        </div>
                    </>
                ) : (
                    <>
                        {/* 结果显示 */}
                        <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
                            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                                配置结果
                            </h3>
                            <button
                                onClick={handleClose}
                                className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                            >
                                <X className="h-5 w-5" />
                            </button>
                        </div>

                        <div className="p-6 space-y-4">
                            {/* 成功统计 */}
                            <div className="flex items-center space-x-3 p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
                                <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400" />
                                <div>
                                    <p className="text-sm font-medium text-green-800 dark:text-green-200">
                                        成功配置 {result?.success_count || 0} 个账户
                                    </p>
                                    <p className="text-xs text-green-600 dark:text-green-400">
                                        同步配置已生效
                                    </p>
                                </div>
                            </div>

                            {/* 错误统计 */}
                            {result && result.error_count > 0 && (
                                <div className="space-y-3">
                                    <div className="flex items-center space-x-3 p-4 bg-red-50 dark:bg-red-900/20 rounded-lg">
                                        <XCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
                                        <div>
                                            <p className="text-sm font-medium text-red-800 dark:text-red-200">
                                                {result.error_count} 个账户配置失败
                                            </p>
                                            <p className="text-xs text-red-600 dark:text-red-400">
                                                请查看详细错误信息
                                            </p>
                                        </div>
                                    </div>

                                    {/* 错误详情 */}
                                    <div className="max-h-32 overflow-y-auto space-y-2">
                                        {result.errors.map((error, index) => (
                                            <div key={index} className="p-3 bg-gray-50 dark:bg-gray-700 rounded border-l-4 border-red-400">
                                                <p className="text-sm font-medium text-gray-900 dark:text-white">
                                                    {error.email_address}
                                                </p>
                                                <p className="text-xs text-red-600 dark:text-red-400 mt-1">
                                                    {error.error}
                                                </p>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}

                            {/* 操作按钮 */}
                            <div className="flex justify-end space-x-3 pt-4">
                                <button
                                    onClick={handleClose}
                                    className="px-4 py-2 text-sm font-medium text-white bg-primary-600 border border-transparent rounded-lg hover:bg-primary-700"
                                >
                                    完成
                                </button>
                            </div>
                        </div>
                    </>
                )}
            </div>
        </div>
    )
}