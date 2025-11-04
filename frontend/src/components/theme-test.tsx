'use client'

import { useTheme } from '@/components/theme-provider'
import { Sun, Moon } from 'lucide-react'

export function ThemeTest() {
    const { theme, setTheme } = useTheme()

    const toggleTheme = () => {
        const newTheme = theme === 'dark' ? 'light' : 'dark'
        console.log('[ThemeTest] 切换主题:', theme, '->', newTheme)
        setTheme(newTheme)
    }

    console.log('[ThemeTest] 当前主题:', theme)
    console.log('[ThemeTest] 当前html类:', typeof document !== 'undefined' ? document.documentElement.className : 'N/A')

    return (
        <div className="p-4 bg-white dark:bg-gray-800 rounded-lg shadow-md">
            <h3 className="text-lg font-semibold mb-2 text-gray-900 dark:text-white">
                主题测试组件
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
                当前主题: <span className="font-mono bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">{theme}</span>
            </p>
            <button
                onClick={toggleTheme}
                className="flex items-center gap-2 px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600 dark:bg-blue-600 dark:hover:bg-blue-700 transition-colors"
            >
                {theme === 'dark' ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
                切换主题
            </button>
        </div>
    )
}