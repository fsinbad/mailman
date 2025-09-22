'use client'

import { useState, useEffect, useRef } from 'react'
import { X, Plus, FileText, AlertCircle, Check } from 'lucide-react'
import { emailAccountService } from '@/services/email-account.service'
import { oauth2Service } from '@/services/oauth2.service'
import { cn } from '@/lib/utils'
import { motion, AnimatePresence } from 'framer-motion'
import { createPortal } from 'react-dom'
import OAuth2PopupAuth from '@/components/oauth2/oauth2-popup-auth'

interface AddAccountModalProps {
    isOpen: boolean
    onClose: () => void
    onSuccess?: () => void
    onError?: (error: string) => void
    presetProvider?: string
    presetAuthType?: string
    autoTriggerOAuth2?: boolean
}

interface MailProvider {
    id: number
    name: string
    type: string
    imapServer: string
    imapPort: number
    smtpServer: string
    smtpPort: number
}

interface SingleAccountForm {
    email: string
    authType: 'password' | 'oauth2'
    password: string
    clientId: string
    accessToken: string
    refreshToken: string
    useProxy: boolean
    proxyUrl: string
    proxyUsername: string
    proxyPassword: string
    isDomainMail: boolean
    domain: string
    oauth2ProviderConfigId?: number
}

interface BatchAccountData {
    email: string
    password: string
    clientId: string
    refreshToken: string
    isValid: boolean
    error?: string
}

export default function AddAccountModal({
    isOpen,
    onClose,
    onSuccess,
    onError,
    presetProvider,
    presetAuthType,
    autoTriggerOAuth2
}: AddAccountModalProps) {
    const [isVisible, setIsVisible] = useState(false)
    const [isAnimating, setIsAnimating] = useState(false)
    const [activeTab, setActiveTab] = useState<'single' | 'batch'>('single')
    const [providers, setProviders] = useState<MailProvider[]>([])
    const [selectedProvider, setSelectedProvider] = useState<number | null>(null)
    const [loading, setLoading] = useState(false)
    const [loadingProviders, setLoadingProviders] = useState(true)
    const [isTabTransitioning, setIsTabTransitioning] = useState(false)
    const [gmailOAuth2Available, setGmailOAuth2Available] = useState(false)
    const [outlookOAuth2Available, setOutlookOAuth2Available] = useState(false)
    const [showOAuth2Popup, setShowOAuth2Popup] = useState(false)
    const [oauth2Configs, setOauth2Configs] = useState<any[]>([])
    const [loadingOAuth2Configs, setLoadingOAuth2Configs] = useState(false)
    const modalRoot = typeof document !== 'undefined' ? document.body : null

    // 用于动画高度过渡的ref
    const contentRef = useRef<HTMLDivElement>(null)
    const [contentHeight, setContentHeight] = useState<number | 'auto'>('auto')

    // 单独添加表单
    const [singleForm, setSingleForm] = useState<SingleAccountForm>({
        email: '',
        authType: 'password',
        password: '',
        clientId: '',
        accessToken: '',
        refreshToken: '',
        useProxy: false,
        proxyUrl: '',
        proxyUsername: '',
        proxyPassword: '',
        isDomainMail: false,
        domain: '',
        oauth2ProviderConfigId: undefined
    })

    // 单独添加的一键解析
    const [singleParseText, setSingleParseText] = useState('')
    const [showSingleParse, setShowSingleParse] = useState(false)
    const [gettingOAuth2Auth, setGettingOAuth2Auth] = useState(false)

    // 批量添加
    const [batchAuthType] = useState<'token'>('token')
    const [batchSeparator, setBatchSeparator] = useState('----')
    const [batchText, setBatchText] = useState('')
    const [batchAccounts, setBatchAccounts] = useState<BatchAccountData[]>([])
    const [showBatchPreview, setShowBatchPreview] = useState(false)

    // 处理模态框动画和body类
    useEffect(() => {
        if (isOpen) {
            setIsVisible(true)
            setTimeout(() => setIsAnimating(true), 10)
            loadProviders()
            // 添加类以锁定body滚动
            document.body.classList.add('modal-open')
        } else {
            setIsAnimating(false)
            setTimeout(() => setIsVisible(false), 300)
            // 移除类以恢复body滚动
            document.body.classList.remove('modal-open')
        }

        // 清理函数
        return () => {
            document.body.classList.remove('modal-open')
        }
    }, [isOpen])

    // 处理预设参数
    useEffect(() => {
        if (isOpen && providers.length > 0) {
            applyPresetParams()
        }
    }, [isOpen, providers, presetProvider, presetAuthType])

    // 应用预设参数
    const applyPresetParams = async () => {
        if (presetProvider) {
            const provider = providers.find(p => p.type === presetProvider)
            if (provider) {
                setSelectedProvider(provider.id)
                await loadOAuth2Configs(provider.type)

                // 设置认证类型
                if (presetAuthType) {
                    setSingleForm(prev => ({
                        ...prev,
                        authType: presetAuthType as 'password' | 'oauth2'
                    }))
                }

                // 如果需要自动触发OAuth2
                if (autoTriggerOAuth2 && presetAuthType === 'oauth2') {
                    // 延迟0.2秒后自动触发OAuth2授权
                    setTimeout(() => {
                        handleOAuth2Auth()
                    }, 200)
                }
            }
        }
    }

    // 监听内容高度变化
    useEffect(() => {
        if (contentRef.current) {
            const resizeObserver = new ResizeObserver((entries) => {
                for (let entry of entries) {
                    const { height } = entry.contentRect
                    setContentHeight(height)
                }
            })

            resizeObserver.observe(contentRef.current)

            return () => {
                resizeObserver.disconnect()
            }
        }
    }, [activeTab, showSingleParse, singleForm.authType, singleForm.useProxy, singleForm.isDomainMail, showBatchPreview, loadingProviders])

    const loadProviders = async () => {
        try {
            setLoadingProviders(true)
            const data = await emailAccountService.getProviders()
            setProviders(data)
            // 默认选择第一个提供商
            if (data.length > 0) {
                setSelectedProvider(data[0].id)
                // 加载第一个提供商的OAuth2配置
                await loadOAuth2Configs(data[0].type)
            }

            // 检查OAuth2配置是否已配置
            try {
                const configs = await oauth2Service.getGlobalConfigs()
                const gmailConfig = configs.find(config => config.provider_type === 'gmail')
                const outlookConfig = configs.find(config => config.provider_type === 'outlook')
                setGmailOAuth2Available(!!gmailConfig && gmailConfig.is_enabled)
                setOutlookOAuth2Available(!!outlookConfig && outlookConfig.is_enabled)
            } catch (error) {
                console.error('Failed to check OAuth2 configuration:', error)
                setGmailOAuth2Available(false)
                setOutlookOAuth2Available(false)
            }
        } catch (error) {
            console.error('Failed to load providers:', error)
            onError?.('加载邮件提供商失败')
        } finally {
            setLoadingProviders(false)
        }
    }

    // 加载OAuth2配置
    const loadOAuth2Configs = async (providerType: string) => {
        if (providerType !== 'gmail' && providerType !== 'outlook') {
            setOauth2Configs([])
            return
        }

        try {
            setLoadingOAuth2Configs(true)
            const configs = await oauth2Service.getGlobalConfigsByProvider(providerType as any)
            setOauth2Configs(configs)

            // 如果有配置，默认选择第一个
            if (configs.length > 0) {
                setSingleForm(prev => ({
                    ...prev,
                    oauth2ProviderConfigId: configs[0].id
                }))
            }
        } catch (error) {
            console.error('Failed to load OAuth2 configs:', error)
            setOauth2Configs([])
        } finally {
            setLoadingOAuth2Configs(false)
        }
    }

    // 获取选中的提供商
    const getSelectedProvider = () => {
        return providers.find(p => p.id === selectedProvider)
    }

    // 判断是否支持OAuth2
    const supportsOAuth2 = () => {
        const provider = getSelectedProvider()
        if (provider?.type === 'outlook') {
            return outlookOAuth2Available // Outlook 需要检查系统是否已配置OAuth2
        }
        if (provider?.type === 'gmail') {
            // Gmail 需要检查系统是否已配置OAuth2
            return gmailOAuth2Available
        }
        return false
    }

    // 获取OAuth2授权 - 使用popup方式，支持Gmail和Outlook
    const handleOAuth2Auth = async () => {
        if (!selectedProvider) return

        try {
            setGettingOAuth2Auth(true)

            // 检查是否为支持的提供商
            const provider = getSelectedProvider()
            if (provider?.type !== 'gmail' && provider?.type !== 'outlook') {
                setGettingOAuth2Auth(false)
                return
            }

            // 显示OAuth2 popup授权
            setShowOAuth2Popup(true)
        } catch (err) {
            console.error('OAuth2 authorization error:', err)
            onError?.('启动OAuth2授权失败')
            setGettingOAuth2Auth(false)
        }
    }

    // OAuth2授权成功回调
    const handleOAuth2Success = async (result: { emailAddress: string; customSettings: any }) => {
        try {
            setShowOAuth2Popup(false)
            setGettingOAuth2Auth(false)

            // 将OAuth2授权结果回填到表单
            const newFormData = {
                ...singleForm,
                email: result.emailAddress,
                authType: 'oauth2' as const,
                accessToken: result.customSettings.access_token || '',
                refreshToken: result.customSettings.refresh_token || '',
                clientId: result.customSettings.client_id || ''
            }

            console.log('OAuth2授权结果:', result)
            console.log('准备回填的表单数据:', newFormData)

            setSingleForm(newFormData)

            // 显示成功提示
            console.log('OAuth2授权成功，数据已回填到表单')
        } catch (error) {
            console.error('Failed to fill OAuth2 data:', error)
            onError?.('OAuth2数据回填失败')
        }
    }

    // OAuth2授权取消回调
    const handleOAuth2Cancel = () => {
        setShowOAuth2Popup(false)
        setGettingOAuth2Auth(false)
    }

    // OAuth2授权失败回调
    const handleOAuth2Error = (error: string) => {
        setShowOAuth2Popup(false)
        setGettingOAuth2Auth(false)
        onError?.(error)
    }

    // 解析单个账户文本
    const parseSingleAccountText = () => {
        const parts = singleParseText.trim().split(batchSeparator)
        if (parts.length >= 4) {
            setSingleForm(prev => ({
                ...prev,
                email: parts[0].trim(),
                authType: 'oauth2',
                accessToken: parts[1].trim(),
                clientId: parts[2].trim(),
                refreshToken: parts[3].trim()
            }))
            setShowSingleParse(false)
            setSingleParseText('')
        } else {
            onError?.(`格式错误：需要4个字段，但只有 ${parts.length} 个。格式应为：邮箱${batchSeparator}Access Token${batchSeparator}Client ID${batchSeparator}Refresh Token`)
        }
    }

    // 解析批量文本
    const parseBatchText = () => {
        const lines = batchText.trim().split('\n').filter(line => line.trim())
        const accounts: BatchAccountData[] = []

        lines.forEach((line, index) => {
            const parts = line.split(batchSeparator)
            if (parts.length >= 4) {
                accounts.push({
                    email: parts[0].trim(),
                    password: parts[1].trim(),
                    clientId: parts[2].trim(),
                    refreshToken: parts[3].trim(),
                    isValid: true
                })
            } else {
                accounts.push({
                    email: line,
                    password: '',
                    clientId: '',
                    refreshToken: '',
                    isValid: false,
                    error: `第 ${index + 1} 行格式错误：需要4个字段，但只有 ${parts.length} 个`
                })
            }
        })

        setBatchAccounts(accounts)
        setShowBatchPreview(true)
    }

    // 提交单个账户
    const handleSingleSubmit = async () => {
        if (!selectedProvider) {
            onError?.('请选择邮件提供商')
            return
        }

        setLoading(true)
        try {
            const provider = getSelectedProvider()
            const payload: any = {
                email_address: singleForm.email,
                auth_type: singleForm.authType,
                mail_provider_id: selectedProvider
            }

            // 如果使用OAuth2且有指定的配置ID，则添加到payload中
            if (singleForm.authType === 'oauth2' && singleForm.oauth2ProviderConfigId) {
                payload.oauth2_provider_id = singleForm.oauth2ProviderConfigId
            }

            if (singleForm.authType === 'password') {
                payload.password = singleForm.password
            } else if (singleForm.authType === 'oauth2') {
                // OAuth2 使用 customSettings
                payload.custom_settings = {
                    client_id: singleForm.clientId,
                    access_token: singleForm.accessToken,
                    refresh_token: singleForm.refreshToken
                }
            }

            if (singleForm.useProxy) {
                payload.proxy = singleForm.proxyUrl
                // 如果需要代理认证，可以构建完整的代理URL
                if (singleForm.proxyUsername && singleForm.proxyPassword) {
                    try {
                        const url = new URL(singleForm.proxyUrl)
                        url.username = singleForm.proxyUsername
                        url.password = singleForm.proxyPassword
                        payload.proxy = url.toString()
                    } catch (e) {
                        // 如果URL解析失败，使用原始值
                        payload.proxy = singleForm.proxyUrl
                    }
                }
            }

            if (singleForm.isDomainMail) {
                payload.is_domain_mail = true
                payload.domain = singleForm.domain
            }

            await emailAccountService.createAccount(payload)
            onSuccess?.()
            handleClose()
        } catch (error: any) {
            onError?.(error.message || '添加账户失败')
        } finally {
            setLoading(false)
        }
    }

    // 提交批量账户
    const handleBatchSubmit = async () => {
        if (!selectedProvider) {
            onError?.('请选择邮件提供商')
            return
        }

        const validAccounts = batchAccounts.filter(acc => acc.isValid)
        if (validAccounts.length === 0) {
            onError?.('没有有效的账户数据')
            return
        }

        setLoading(true)
        let successCount = 0
        let failCount = 0

        try {
            for (const account of validAccounts) {
                try {
                    const payload: any = {
                        email_address: account.email,
                        auth_type: 'oauth2',
                        mail_provider_id: selectedProvider,
                        custom_settings: {
                            access_token: account.password, // 这里password字段实际是access_token
                            client_id: account.clientId,
                            refresh_token: account.refreshToken
                        }
                    }

                    await emailAccountService.createAccount(payload)
                    successCount++
                } catch (error) {
                    failCount++
                    console.error(`Failed to add account ${account.email}:`, error)
                }
            }

            if (successCount > 0) {
                onSuccess?.()
            }

            if (failCount > 0) {
                onError?.(`成功添加 ${successCount} 个账户，失败 ${failCount} 个`)
            } else {
                handleClose()
            }
        } catch (error: any) {
            onError?.(error.message || '批量添加失败')
        } finally {
            setLoading(false)
        }
    }

    const handleClose = () => {
        setIsAnimating(false)
        setTimeout(() => {
            // 重置表单
            setSingleForm({
                email: '',
                authType: 'password',
                password: '',
                clientId: '',
                accessToken: '',
                refreshToken: '',
                useProxy: false,
                proxyUrl: '',
                proxyUsername: '',
                proxyPassword: '',
                isDomainMail: false,
                domain: '',
                oauth2ProviderConfigId: undefined
            })
            setBatchText('')
            setBatchAccounts([])
            setShowBatchPreview(false)
            setSingleParseText('')
            setShowSingleParse(false)
            setActiveTab('single')
            onClose()
        }, 300)
    }

    if (!isVisible || !modalRoot) return null

    // 使用Portal渲染模态框到body
    return createPortal(
        <div className="modal-backdrop">
            <style jsx global>{`
                body.modal-open {
                    overflow: hidden;
                }
                .modal-backdrop {
                    position: fixed;
                    top: 0;
                    left: 0;
                    right: 0;
                    bottom: 0;
                    z-index: 50;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    padding: 1rem;
                    background-color: ${isAnimating ? 'rgba(0, 0, 0, 0.5)' : 'rgba(0, 0, 0, 0)'};
                    transition: background-color 0.3s ease;
                }
                .modal-content {
                    max-height: 90vh;
                    border-radius: 0.75rem;
                    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
                    overflow: hidden;
                    transform: ${isAnimating ? 'scale(1)' : 'scale(0.95)'};
                    opacity: ${isAnimating ? '1' : '0'};
                    transition: all 0.3s ease;
                }
                .modal-body {
                    max-height: calc(90vh - 120px);
                    overflow-y: auto;
                    overscroll-behavior: contain;
                }
                .modal-body::-webkit-scrollbar {
                    width: 6px;
                }
                .modal-body::-webkit-scrollbar-track {
                    background: transparent;
                }
                .modal-body::-webkit-scrollbar-thumb {
                    background-color: rgba(156, 163, 175, 0.5);
                    border-radius: 3px;
                }
            `}</style>
            <div
                className="modal-backdrop"
                onClick={(e) => {
                    if (e.target === e.currentTarget) {
                        handleClose();
                    }
                }}
            >
                <div className="modal-content w-full max-w-3xl bg-white dark:bg-gray-800 flex flex-col">
                    {/* 标题栏 */}
                    <div className="flex items-center justify-between border-b border-gray-200 p-6 dark:border-gray-700">
                        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                            添加邮箱账户
                        </h2>
                        <button
                            onClick={handleClose}
                            className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"
                        >
                            <X className="h-5 w-5" />
                        </button>
                    </div>

                    {/* 主内容容器 - 优化滚动行为 */}
                    <div className="modal-body flex-1">
                        <div
                            className="transition-all duration-300 ease-in-out"
                            style={{
                                height: contentHeight === 'auto' ? 'auto' : `${contentHeight}px`
                            }}
                        >
                            <div ref={contentRef} className="flex flex-col">
                                {/* Provider 选择器 */}
                                <div className="border-b border-gray-200 p-6 dark:border-gray-700">
                                    <label className="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                        邮件提供商
                                    </label>
                                    {loadingProviders ? (
                                        <div className="flex items-center space-x-2 text-gray-500">
                                            <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary-600 border-t-transparent"></div>
                                            <span>加载中...</span>
                                        </div>
                                    ) : (
                                        <select
                                            value={selectedProvider || ''}
                                            onChange={async (e) => {
                                                const id = parseInt(e.target.value)
                                                setSelectedProvider(id)
                                                // 根据提供商类型设置默认认证方式
                                                const provider = providers.find(p => p.id === id)
                                                if (provider?.type === 'outlook') {
                                                    setSingleForm(prev => ({ ...prev, authType: 'oauth2', oauth2ProviderConfigId: undefined }))
                                                } else {
                                                    setSingleForm(prev => ({ ...prev, authType: 'password', oauth2ProviderConfigId: undefined }))
                                                }

                                                // 加载对应的OAuth2配置
                                                if (provider) {
                                                    await loadOAuth2Configs(provider.type)
                                                }
                                            }}
                                            className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                            required
                                        >
                                            <option value="">请选择提供商</option>
                                            {providers.map(provider => (
                                                <option key={provider.id} value={provider.id}>
                                                    {provider.name} ({provider.type})
                                                </option>
                                            ))}
                                        </select>
                                    )}
                                </div>

                                {/* Tab 切换 */}
                                <div className="flex border-b border-gray-200 dark:border-gray-700">
                                    <button
                                        onClick={() => {
                                            if (activeTab !== 'single') {
                                                setIsTabTransitioning(true)
                                                setTimeout(() => {
                                                    setActiveTab('single')
                                                    setIsTabTransitioning(false)
                                                }, 150)
                                            }
                                        }}
                                        className={cn(
                                            "flex-1 px-6 py-3 text-sm font-medium transition-colors",
                                            activeTab === 'single'
                                                ? "border-b-2 border-primary-600 text-primary-600"
                                                : "text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
                                        )}
                                    >
                                        <div className="flex items-center justify-center space-x-2">
                                            <Plus className="h-4 w-4" />
                                            <span>单独添加</span>
                                        </div>
                                    </button>
                                    <button
                                        onClick={() => {
                                            if (activeTab !== 'batch' && getSelectedProvider()?.type === 'outlook') {
                                                setIsTabTransitioning(true)
                                                setTimeout(() => {
                                                    setActiveTab('batch')
                                                    setIsTabTransitioning(false)
                                                }, 150)
                                            }
                                        }}
                                        className={cn(
                                            "flex-1 px-6 py-3 text-sm font-medium transition-colors",
                                            activeTab === 'batch'
                                                ? "border-b-2 border-primary-600 text-primary-600"
                                                : getSelectedProvider()?.type !== 'outlook'
                                                    ? "text-gray-400 cursor-not-allowed dark:text-gray-600"
                                                    : "text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
                                        )}
                                        disabled={getSelectedProvider()?.type !== 'outlook'}
                                    >
                                        <div className="flex items-center justify-center space-x-2">
                                            <FileText className={cn(
                                                "h-4 w-4",
                                                getSelectedProvider()?.type !== 'outlook' && "opacity-50"
                                            )} />
                                            <span>批量添加</span>
                                            {getSelectedProvider()?.type !== 'outlook' && (
                                                <span className="text-xs">(仅Outlook)</span>
                                            )}
                                        </div>
                                    </button>
                                </div>

                                {/* Tab 内容区域 - 可滚动，带动画 */}
                                <div className="flex-1 overflow-y-auto">
                                    <AnimatePresence mode="wait">
                                        {activeTab === 'single' ? (
                                            <motion.div
                                                key="single-tab"
                                                initial={{ opacity: 0, y: 10 }}
                                                animate={{ opacity: 1, y: 0 }}
                                                exit={{ opacity: 0, y: -10 }}
                                                transition={{ duration: 0.3 }}
                                                className="p-6"
                                            >
                                                {/* 单独添加表单 */}
                                                <div className="space-y-4">
                                                    {/* 邮箱地址 */}
                                                    <div>
                                                        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                            邮箱地址
                                                        </label>
                                                        <input
                                                            type="email"
                                                            value={singleForm.email}
                                                            onChange={(e) => setSingleForm({ ...singleForm, email: e.target.value })}
                                                            className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                                            placeholder="example@outlook.com"
                                                            required
                                                        />
                                                    </div>

                                                    {/* 验证方式 */}
                                                    <div>
                                                        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                            验证方式
                                                        </label>
                                                        <div className="flex space-x-4">
                                                            <label className="flex items-center">
                                                                <input
                                                                    type="radio"
                                                                    value="password"
                                                                    checked={singleForm.authType === 'password'}
                                                                    onChange={(e) => setSingleForm({ ...singleForm, authType: 'password' })}
                                                                    className="mr-2"
                                                                    disabled={!supportsOAuth2() && singleForm.authType === 'oauth2'}
                                                                />
                                                                <span className="text-sm">密码</span>
                                                            </label>
                                                            <label className="flex items-center">
                                                                <input
                                                                    type="radio"
                                                                    value="oauth2"
                                                                    checked={singleForm.authType === 'oauth2'}
                                                                    onChange={(e) => setSingleForm({ ...singleForm, authType: 'oauth2' })}
                                                                    className="mr-2"
                                                                    disabled={!supportsOAuth2()}
                                                                />
                                                                <span className={cn(
                                                                    "text-sm",
                                                                    !supportsOAuth2() && "text-gray-400"
                                                                )}>
                                                                    OAuth2 {!supportsOAuth2() && getSelectedProvider()?.type === 'gmail' && "(需要先配置Gmail OAuth2)"}
                                                                    {!supportsOAuth2() && getSelectedProvider()?.type !== 'gmail' && getSelectedProvider()?.type !== 'outlook' && "(不支持OAuth2)"}
                                                                </span>
                                                            </label>
                                                        </div>
                                                    </div>

                                                    {/* 密码输入 */}
                                                    {singleForm.authType === 'password' && (
                                                        <div>
                                                            <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                密码
                                                            </label>
                                                            <input
                                                                type="text"
                                                                value={singleForm.password}
                                                                onChange={(e) => setSingleForm({ ...singleForm, password: e.target.value })}
                                                                className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                                                placeholder="输入密码"
                                                                required
                                                            />
                                                        </div>
                                                    )}

                                                    {/* OAuth2 输入 */}
                                                    {singleForm.authType === 'oauth2' && (
                                                        <>
                                                            {/* OAuth2 配置选择器 */}
                                                            {(getSelectedProvider()?.type === 'gmail' || getSelectedProvider()?.type === 'outlook') && oauth2Configs.length > 0 && (
                                                                <div className="mb-4">
                                                                    <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                        OAuth2 配置
                                                                    </label>
                                                                    {loadingOAuth2Configs ? (
                                                                        <div className="flex items-center space-x-2 text-gray-500 p-3 border border-gray-300 rounded-lg">
                                                                            <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary-600 border-t-transparent"></div>
                                                                            <span>加载配置中...</span>
                                                                        </div>
                                                                    ) : (
                                                                        <select
                                                                            value={singleForm.oauth2ProviderConfigId || ''}
                                                                            onChange={(e) => setSingleForm({ ...singleForm, oauth2ProviderConfigId: e.target.value ? parseInt(e.target.value) : undefined })}
                                                                            className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                                                            required
                                                                        >
                                                                            <option value="">请选择OAuth2配置</option>
                                                                            {oauth2Configs.map(config => (
                                                                                <option key={config.id} value={config.id}>
                                                                                    {config.name} ({config.client_id ? `${config.client_id.substring(0, 8)}...` : 'N/A'})
                                                                                </option>
                                                                            ))}
                                                                        </select>
                                                                    )}
                                                                    {oauth2Configs.length === 0 && !loadingOAuth2Configs && (
                                                                        <div className="mt-2 text-sm text-yellow-600 dark:text-yellow-400">
                                                                            没有找到可用的OAuth2配置，请先在OAuth2配置页面添加配置
                                                                        </div>
                                                                    )}
                                                                </div>
                                                            )}

                                                            {/* Gmail OAuth2 获取授权 */}
                                                            {getSelectedProvider()?.type === 'gmail' && (
                                                                <div className="mb-4 rounded-lg bg-green-50 dark:bg-green-900/20 p-4">
                                                                    <div className="flex items-center justify-between mb-2">
                                                                        <h4 className="text-sm font-medium text-green-900 dark:text-green-200">
                                                                            Gmail OAuth2 授权
                                                                        </h4>
                                                                    </div>
                                                                    <p className="text-xs text-green-700 dark:text-green-300 mb-3">
                                                                        点击下方按钮获取Gmail OAuth2授权，完成授权后系统会自动跳转并填充Token信息
                                                                    </p>
                                                                    <button
                                                                        type="button"
                                                                        onClick={handleOAuth2Auth}
                                                                        disabled={gettingOAuth2Auth}
                                                                        className="w-full rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-green-700 disabled:opacity-50"
                                                                    >
                                                                        {gettingOAuth2Auth ? '获取中...' : (getSelectedProvider()?.type === 'outlook' ? '获取Outlook授权' : '获取Gmail授权')}
                                                                    </button>
                                                                </div>
                                                            )}

                                                            {/* 一键解析功能 - 仅对 Outlook OAuth2 显示 */}
                                                            {getSelectedProvider()?.type === 'outlook' && (
                                                                <div className="mb-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 overflow-hidden">
                                                                    <div className="p-4">
                                                                        <div className="flex items-center justify-between mb-2">
                                                                            <h4 className="text-sm font-medium text-blue-900 dark:text-blue-200">
                                                                                快速导入（可选）
                                                                            </h4>
                                                                            <button
                                                                                type="button"
                                                                                onClick={() => setShowSingleParse(!showSingleParse)}
                                                                                className="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
                                                                            >
                                                                                {showSingleParse ? '收起' : '展开'}
                                                                            </button>
                                                                        </div>
                                                                        <AnimatePresence>
                                                                            {showSingleParse && (
                                                                                <motion.div
                                                                                    initial={{ height: 0, opacity: 0 }}
                                                                                    animate={{ height: "auto", opacity: 1 }}
                                                                                    exit={{ height: 0, opacity: 0 }}
                                                                                    transition={{ duration: 0.3 }}
                                                                                    className="space-y-3 overflow-hidden"
                                                                                >
                                                                                    <p className="text-xs text-blue-700 dark:text-blue-300">
                                                                                        粘贴格式：邮箱{batchSeparator}Access Token{batchSeparator}Client ID{batchSeparator}Refresh Token
                                                                                    </p>
                                                                                    <div className="flex space-x-2">
                                                                                        <input
                                                                                            type="text"
                                                                                            value={singleParseText}
                                                                                            onChange={(e) => setSingleParseText(e.target.value)}
                                                                                            className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                                                                            placeholder={`example@outlook.com${batchSeparator}token${batchSeparator}client-id${batchSeparator}refresh-token`}
                                                                                        />
                                                                                        <button
                                                                                            type="button"
                                                                                            onClick={parseSingleAccountText}
                                                                                            disabled={!singleParseText.trim()}
                                                                                            className="rounded-lg bg-blue-600 px-4 py-2 text-sm text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
                                                                                        >
                                                                                            解析
                                                                                        </button>
                                                                                    </div>
                                                                                </motion.div>
                                                                            )}
                                                                        </AnimatePresence>
                                                                    </div>
                                                                </div>
                                                            )}

                                                            <div>
                                                                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                    Client ID
                                                                </label>
                                                                <input
                                                                    type="text"
                                                                    value={singleForm.clientId}
                                                                    onChange={(e) => setSingleForm({ ...singleForm, clientId: e.target.value })}
                                                                    className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                                                    placeholder="9e5f94bc-e8a4-4e73-b8be-63364c29d753"
                                                                    required
                                                                />
                                                            </div>
                                                            <div>
                                                                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                    Access Token
                                                                </label>
                                                                <textarea
                                                                    value={singleForm.accessToken}
                                                                    onChange={(e) => setSingleForm({ ...singleForm, accessToken: e.target.value })}
                                                                    className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                                                    placeholder="输入 Access Token"
                                                                    rows={3}
                                                                    required
                                                                />
                                                            </div>
                                                            <div>
                                                                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                    Refresh Token
                                                                </label>
                                                                <textarea
                                                                    value={singleForm.refreshToken}
                                                                    onChange={(e) => setSingleForm({ ...singleForm, refreshToken: e.target.value })}
                                                                    className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                                                    placeholder="输入 Refresh Token"
                                                                    rows={3}
                                                                    required
                                                                />
                                                            </div>
                                                        </>
                                                    )}

                                                    {/* 代理设置 */}
                                                    <div className="space-y-3">
                                                        <label className="flex items-center space-x-2">
                                                            <input
                                                                type="checkbox"
                                                                checked={singleForm.useProxy}
                                                                onChange={(e) => setSingleForm({ ...singleForm, useProxy: e.target.checked })}
                                                                className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                                                            />
                                                            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                使用代理
                                                            </span>
                                                        </label>

                                                        <AnimatePresence>
                                                            {singleForm.useProxy && (
                                                                <motion.div
                                                                    initial={{ height: 0, opacity: 0 }}
                                                                    animate={{ height: "auto", opacity: 1 }}
                                                                    exit={{ height: 0, opacity: 0 }}
                                                                    transition={{ duration: 0.3 }}
                                                                    className="space-y-3 overflow-hidden"
                                                                >
                                                                    <div>
                                                                        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                            代理地址
                                                                        </label>
                                                                        <input
                                                                            type="text"
                                                                            value={singleForm.proxyUrl}
                                                                            onChange={(e) => setSingleForm({ ...singleForm, proxyUrl: e.target.value })}
                                                                            className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                                                            placeholder="socks5://127.0.0.1:1080"
                                                                            required
                                                                        />
                                                                    </div>
                                                                    <div className="grid grid-cols-2 gap-3">
                                                                        <div>
                                                                            <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                                代理用户名（可选）
                                                                            </label>
                                                                            <input
                                                                                type="text"
                                                                                value={singleForm.proxyUsername}
                                                                                onChange={(e) => setSingleForm({ ...singleForm, proxyUsername: e.target.value })}
                                                                                className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                                                            />
                                                                        </div>
                                                                        <div>
                                                                            <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                                代理密码（可选）
                                                                            </label>
                                                                            <input
                                                                                type="password"
                                                                                value={singleForm.proxyPassword}
                                                                                onChange={(e) => setSingleForm({ ...singleForm, proxyPassword: e.target.value })}
                                                                                className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                                                            />
                                                                        </div>
                                                                    </div>
                                                                </motion.div>
                                                            )}
                                                        </AnimatePresence>
                                                    </div>

                                                    {/* 域名邮箱设置 */}
                                                    <div className="space-y-3">
                                                        <label className="flex items-center space-x-2">
                                                            <input
                                                                type="checkbox"
                                                                checked={singleForm.isDomainMail}
                                                                onChange={(e) => setSingleForm({ ...singleForm, isDomainMail: e.target.checked })}
                                                                className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                                                            />
                                                            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                启用域名邮箱
                                                            </span>
                                                        </label>

                                                        <AnimatePresence>
                                                            {singleForm.isDomainMail && (
                                                                <motion.div
                                                                    initial={{ height: 0, opacity: 0 }}
                                                                    animate={{ height: "auto", opacity: 1 }}
                                                                    exit={{ height: 0, opacity: 0 }}
                                                                    transition={{ duration: 0.3 }}
                                                                    className="overflow-hidden"
                                                                >
                                                                    <div className="pt-3">
                                                                        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                            域名
                                                                        </label>
                                                                        <input
                                                                            type="text"
                                                                            value={singleForm.domain}
                                                                            onChange={(e) => setSingleForm({ ...singleForm, domain: e.target.value })}
                                                                            className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                                                            placeholder="example.com"
                                                                            required
                                                                        />
                                                                    </div>
                                                                </motion.div>
                                                            )}
                                                        </AnimatePresence>
                                                    </div>
                                                </div>
                                            </motion.div>
                                        ) : (
                                            <motion.div
                                                key="batch-tab"
                                                initial={{ opacity: 0, y: 10 }}
                                                animate={{ opacity: 1, y: 0 }}
                                                exit={{ opacity: 0, y: -10 }}
                                                transition={{ duration: 0.3 }}
                                                className="p-6"
                                            >
                                                {/* 批量添加表单 */}
                                                <div className="space-y-4">
                                                    {getSelectedProvider()?.type !== 'outlook' ? (
                                                        <div className="rounded-lg bg-yellow-50 p-4 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-200">
                                                            <div className="flex items-start space-x-2">
                                                                <AlertCircle className="h-5 w-5 flex-shrink-0 mt-0.5" />
                                                                <div>
                                                                    <p className="font-medium">批量添加仅支持 Outlook</p>
                                                                    <p className="mt-1 text-sm">请选择 Outlook 提供商以使用批量添加功能</p>
                                                                </div>
                                                            </div>
                                                        </div>
                                                    ) : (
                                                        <>
                                                            <div className="flex items-center space-x-4">
                                                                <div className="flex-1">
                                                                    <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                        授权方式
                                                                    </label>
                                                                    <select
                                                                        value={batchAuthType}
                                                                        disabled
                                                                        className="w-full rounded-lg border border-gray-300 px-3 py-2 bg-gray-50 dark:border-gray-600 dark:bg-gray-700"
                                                                    >
                                                                        <option value="token">Token</option>
                                                                    </select>
                                                                </div>
                                                                <div className="flex-1">
                                                                    <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                        分隔符
                                                                    </label>
                                                                    <input
                                                                        type="text"
                                                                        value={batchSeparator}
                                                                        onChange={(e) => setBatchSeparator(e.target.value)}
                                                                        className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
                                                                    />
                                                                </div>
                                                            </div>

                                                            <div>
                                                                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                    账户数据（每行一个账户）
                                                                </label>
                                                                <textarea
                                                                    value={batchText}
                                                                    onChange={(e) => setBatchText(e.target.value)}
                                                                    className="w-full rounded-lg border border-gray-300 px-3 py-2 focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 font-mono text-sm"
                                                                    placeholder={`邮箱${batchSeparator}Access Token${batchSeparator}Client ID${batchSeparator}Refresh Token`}
                                                                    rows={8}
                                                                />
                                                                <p className="mt-1 text-xs text-gray-500">
                                                                    格式：邮箱{batchSeparator}Access Token{batchSeparator}Client ID{batchSeparator}Refresh Token
                                                                </p>
                                                            </div>

                                                            <button
                                                                onClick={parseBatchText}
                                                                disabled={!batchText.trim()}
                                                                className="w-full rounded-lg bg-primary-600 px-4 py-2 text-white transition-colors hover:bg-primary-700 disabled:opacity-50"
                                                            >
                                                                一键解析
                                                            </button>

                                                            {/* 解析预览 */}
                                                            <AnimatePresence>
                                                                {showBatchPreview && batchAccounts.length > 0 && (
                                                                    <motion.div
                                                                        initial={{ height: 0, opacity: 0 }}
                                                                        animate={{ height: "auto", opacity: 1 }}
                                                                        exit={{ height: 0, opacity: 0 }}
                                                                        transition={{ duration: 0.3 }}
                                                                        className="space-y-2 pt-4 overflow-hidden"
                                                                    >
                                                                        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">
                                                                            解析结果（{batchAccounts.filter(a => a.isValid).length} 个有效，{batchAccounts.filter(a => !a.isValid).length} 个无效）
                                                                        </h3>
                                                                        <div className="max-h-48 overflow-y-auto rounded-lg border border-gray-200 dark:border-gray-700">
                                                                            {batchAccounts.map((account, index) => (
                                                                                <div
                                                                                    key={index}
                                                                                    className={cn(
                                                                                        "flex items-center justify-between p-3 text-sm",
                                                                                        index % 2 === 0 ? "bg-gray-50 dark:bg-gray-800" : "bg-white dark:bg-gray-900",
                                                                                        !account.isValid && "opacity-60"
                                                                                    )}
                                                                                >
                                                                                    <div className="flex items-center space-x-2">
                                                                                        {account.isValid ? (
                                                                                            <Check className="h-4 w-4 text-green-500" />
                                                                                        ) : (
                                                                                            <AlertCircle className="h-4 w-4 text-red-500" />
                                                                                        )}
                                                                                        <span className="font-mono">{account.email}</span>
                                                                                    </div>
                                                                                    {!account.isValid && (
                                                                                        <span className="text-xs text-red-600 dark:text-red-400">
                                                                                            {account.error}
                                                                                        </span>
                                                                                    )}
                                                                                </div>
                                                                            ))}
                                                                        </div>
                                                                    </motion.div>
                                                                )}
                                                            </AnimatePresence>
                                                        </>
                                                    )}
                                                </div>
                                            </motion.div>
                                        )}
                                    </AnimatePresence>
                                </div>
                            </div>
                        </div>

                        {/* 底部操作栏 */}
                        <div className="flex items-center justify-end space-x-3 border-t border-gray-200 p-6 dark:border-gray-700">
                            <button
                                onClick={handleClose}
                                className="rounded-lg border border-gray-300 px-4 py-2 text-gray-700 transition-colors hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                                disabled={loading}
                            >
                                取消
                            </button>
                            <button
                                onClick={activeTab === 'single' ? handleSingleSubmit : handleBatchSubmit}
                                disabled={
                                    loading ||
                                    !selectedProvider ||
                                    (activeTab === 'single' && !singleForm.email) ||
                                    (activeTab === 'batch' && (!showBatchPreview || batchAccounts.filter(a => a.isValid).length === 0))
                                }
                                className="flex items-center space-x-2 rounded-lg bg-primary-600 px-4 py-2 text-white transition-colors hover:bg-primary-700 disabled:opacity-50"
                            >
                                {loading ? (
                                    <>
                                        <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></div>
                                        <span>处理中...</span>
                                    </>
                                ) : (
                                    <>
                                        <Plus className="h-4 w-4" />
                                        <span>{activeTab === 'single' ? '添加账户' : `批量添加 (${batchAccounts.filter(a => a.isValid).length})`}</span>
                                    </>
                                )}
                            </button>
                        </div>
                    </div>
                </div>
            </div>

            {/* OAuth2 Popup 授权组件 */}
            {showOAuth2Popup && selectedProvider && (
                <OAuth2PopupAuth
                    provider={getSelectedProvider()?.type.toLowerCase() as any}
                    configId={singleForm.oauth2ProviderConfigId}
                    onSuccess={handleOAuth2Success}
                    onCancel={handleOAuth2Cancel}
                    onError={handleOAuth2Error}
                />
            )}
        </div>
        , modalRoot)
}
