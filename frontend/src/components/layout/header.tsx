'use client'

import { Bell, Search, User, LogOut, ChevronDown, Upload, Camera, UserCircle } from 'lucide-react'
import { useState, useRef, useEffect } from 'react'
import { cn } from '@/lib/utils'
import { useAuth } from '@/contexts/auth-context'
import { authService } from '@/services/auth.service'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'

// 添加全局样式
if (typeof document !== 'undefined') {
    const style = document.createElement('style')
    style.innerHTML = `
    @keyframes fadeIn {
      from { opacity: 0; }
      to { opacity: 1; }
    }
    
    @keyframes slideUp {
      from { transform: translateY(30px); opacity: 0; }
      to { transform: translateY(0); opacity: 1; }
    }
  `
    document.head.appendChild(style)
}

export function Header() {
    const [searchQuery, setSearchQuery] = useState('')
    const [showUserProfileModal, setShowUserProfileModal] = useState(false)
    const [avatarPreview, setAvatarPreview] = useState<string | null>(null)
    const [isSubmitting, setIsSubmitting] = useState(false)
    const [formData, setFormData] = useState({
        username: '',
        email: '',
        old_password: '',
        new_password: '',
    })
    const fileInputRef = useRef<HTMLInputElement>(null)
    const { user, logout, refreshUser } = useAuth()

    // 处理头像变更
    const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0]
        if (!file) return

        // 检查文件类型是否为图片
        if (!file.type.startsWith('image/')) {
            alert('请上传图片文件')
            return
        }

        // 检查文件大小，限制为2MB
        if (file.size > 2 * 1024 * 1024) {
            alert('图片大小不能超过2MB')
            return
        }

        const reader = new FileReader()
        reader.onload = (event) => {
            if (event.target?.result) {
                setAvatarPreview(event.target.result as string)
            }
        }
        reader.readAsDataURL(file)
    }

    // 处理表单字段变更
    const handleFormChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const { name, value } = e.target
        setFormData(prev => ({ ...prev, [name]: value }))
    }

    // 处理表单提交
    const handleSubmit = async () => {
        try {
            setIsSubmitting(true)

            // 构建请求数据
            const updateData: any = {}

            // 仅包含已修改的字段
            if (formData.username && formData.username !== user?.username) {
                updateData.username = formData.username
            }

            if (formData.email && formData.email !== user?.email) {
                updateData.email = formData.email
            }

            // 仅在同时提供旧密码和新密码时更新密码
            if (formData.old_password && formData.new_password) {
                updateData.old_password = formData.old_password
                updateData.new_password = formData.new_password
            }

            // 如果有新头像预览，添加到请求数据中
            if (avatarPreview) {
                updateData.avatar = avatarPreview
            }

            // 如果没有任何字段被修改，直接关闭模态框
            if (Object.keys(updateData).length === 0) {
                setShowUserProfileModal(false)
                return
            }

            // 发送更新请求
            await authService.updateUser(updateData)

            // 刷新用户信息
            await refreshUser()

            // 关闭模态框
            setShowUserProfileModal(false)

            // 重置表单数据
            setFormData({
                username: '',
                email: '',
                old_password: '',
                new_password: '',
            })
            setAvatarPreview(null)

        } catch (error) {
            console.error('更新用户信息失败:', error)
            alert('更新用户信息失败，请重试')
        } finally {
            setIsSubmitting(false)
        }
    }

    return (
        <header className="flex h-16 items-center border-b border-gray-200 bg-white px-6 dark:border-gray-700 dark:bg-card">
            <div className="flex flex-1 items-center justify-between">
                {/* 搜索栏 */}
                <div className="flex-1 max-w-lg">
                    <div className="relative">
                        <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-400" />
                        <input
                            type="text"
                            placeholder="搜索邮件..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className={cn(
                                "w-full rounded-lg border border-input bg-background py-2 pl-10 pr-4 text-sm",
                                "placeholder-muted-foreground transition-colors",
                                "focus:border-primary-500 focus:bg-background focus:outline-none focus:ring-2 focus:ring-primary-500/20",
                                "dark:text-gray-200"
                            )}
                        />
                    </div>
                </div>

                {/* 右侧操作区 */}
                <div className="flex items-center space-x-4 ml-6">
                    {/* 通知按钮 */}
                    <button className="relative rounded-lg p-2 text-gray-600 transition-colors hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800">
                        <Bell className="h-5 w-5" />
                        <span className="absolute right-1 top-1 h-2 w-2 rounded-full bg-red-500"></span>
                    </button>

                    {/* 用户菜单 */}
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <button className="flex items-center space-x-1">
                                {user?.avatar ? (
                                    <div className="h-8 w-8 rounded-full overflow-hidden">
                                        <img
                                            src={user.avatar}
                                            alt={user.username}
                                            className="h-full w-full object-cover"
                                        />
                                    </div>
                                ) : (
                                    <div className="h-8 w-8 rounded-full bg-gradient-to-br from-primary-400 to-primary-600 flex items-center justify-center">
                                        <User className="h-5 w-5 text-white" />
                                    </div>
                                )}
                                <ChevronDown className="h-3 w-3 text-gray-500" />
                            </button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent
                            align="end"
                            className="w-60 p-0 rounded-md bg-white shadow-md border border-gray-100 dark:bg-gray-800 dark:border-gray-700 dark:shadow-gray-900/30"
                            sideOffset={8}
                        >
                            <div className="p-3 border-b border-gray-100 dark:border-gray-700">
                                <div className="font-medium text-gray-800 dark:text-white">{user?.username}</div>
                                <div className="text-sm text-gray-500 dark:text-gray-400">{user?.email}</div>
                                {/* 添加账号按钮已隐藏 */}
                            </div>
                            <div className="py-1">
                                <DropdownMenuItem
                                    onClick={() => setShowUserProfileModal(true)}
                                    className="flex items-center px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
                                >
                                    <UserCircle className="mr-2 h-4 w-4 text-gray-500 dark:text-gray-400" />
                                    <span>个人资料</span>
                                </DropdownMenuItem>
                                <DropdownMenuItem
                                    onClick={() => logout()}
                                    className="flex items-center px-3 py-2 text-sm text-red-500 dark:text-red-400 hover:bg-gray-50 dark:hover:bg-gray-700"
                                >
                                    <LogOut className="mr-2 h-4 w-4" />
                                    <span>退出登录</span>
                                </DropdownMenuItem>
                            </div>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </div>

            {/* 用户资料编辑模态框 */}
            {showUserProfileModal && (
                <div
                    className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 transition-opacity duration-300 ease-in-out"
                    style={{ animation: "fadeIn 0.3s ease-in-out" }}
                >
                    <div
                        className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl dark:bg-gray-800"
                        style={{ animation: "slideUp 0.3s ease-out" }}
                    >
                        <h2 className="mb-4 text-xl font-bold text-center">编辑个人资料</h2>

                        {/* 头像上传 */}
                        <div className="mb-6 flex flex-col items-center">
                            <div className="relative mb-2 h-24 w-24">
                                <div className="h-full w-full overflow-hidden rounded-full">
                                    {avatarPreview || user?.avatar ? (
                                        <img
                                            src={avatarPreview || user?.avatar}
                                            alt={user?.username}
                                            className="h-full w-full object-cover"
                                        />
                                    ) : (
                                        <div className="flex h-full w-full items-center justify-center bg-gradient-to-br from-primary-400 to-primary-600">
                                            <User className="h-12 w-12 text-white" />
                                        </div>
                                    )}
                                </div>
                                <div className="absolute -right-2 -bottom-2">
                                    <button
                                        type="button"
                                        onClick={() => fileInputRef.current?.click()}
                                        className="flex h-8 w-8 items-center justify-center rounded-full bg-blue-500 text-white hover:bg-blue-600 shadow-md z-10"
                                    >
                                        <Camera className="h-4 w-4" />
                                    </button>
                                </div>
                            </div>
                            <input
                                ref={fileInputRef}
                                type="file"
                                accept="image/*"
                                className="hidden"
                                onChange={handleAvatarChange}
                            />
                            <span className="text-sm text-gray-500 dark:text-gray-400">
                                点击更换头像
                            </span>
                        </div>

                        {/* 表单字段 */}
                        <div className="mb-4">
                            <label className="mb-1 block text-sm font-medium">用户名</label>
                            <input
                                type="text"
                                name="username"
                                defaultValue={user?.username || ''}
                                onChange={handleFormChange}
                                className="w-full rounded-md border border-gray-300 p-2 dark:border-gray-600 dark:bg-gray-700"
                            />
                        </div>

                        <div className="mb-4">
                            <label className="mb-1 block text-sm font-medium">邮箱</label>
                            <input
                                type="email"
                                name="email"
                                defaultValue={user?.email || ''}
                                onChange={handleFormChange}
                                className="w-full rounded-md border border-gray-300 p-2 dark:border-gray-600 dark:bg-gray-700"
                            />
                        </div>

                        <div className="mb-4">
                            <label className="mb-1 block text-sm font-medium">旧密码</label>
                            <input
                                type="password"
                                name="old_password"
                                onChange={handleFormChange}
                                className="w-full rounded-md border border-gray-300 p-2 dark:border-gray-600 dark:bg-gray-700"
                            />
                        </div>

                        <div className="mb-6">
                            <label className="mb-1 block text-sm font-medium">新密码</label>
                            <input
                                type="password"
                                name="new_password"
                                onChange={handleFormChange}
                                className="w-full rounded-md border border-gray-300 p-2 dark:border-gray-600 dark:bg-gray-700"
                            />
                        </div>

                        {/* 操作按钮 */}
                        <div className="flex justify-end space-x-3">
                            <Button
                                onClick={() => setShowUserProfileModal(false)}
                                variant="outline"
                                disabled={isSubmitting}
                                className="rounded-full px-6 py-2 transition-all duration-200 hover:bg-gray-100"
                            >
                                取消
                            </Button>
                            <Button
                                onClick={handleSubmit}
                                disabled={isSubmitting}
                                className="rounded-full px-6 py-2 bg-primary-500 hover:bg-primary-600 text-white transition-all duration-200"
                            >
                                {isSubmitting ? '保存中...' : '保存'}
                            </Button>
                        </div>
                    </div>
                </div>
            )}
        </header>
    )
}