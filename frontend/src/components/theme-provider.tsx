'use client'

import * as React from 'react'

type Theme = 'dark' | 'light' | 'system'

type ThemeProviderProps = {
    children: React.ReactNode
    attribute?: string
    defaultTheme?: Theme
    enableSystem?: boolean
    storageKey?: string
}

type ThemeProviderState = {
    theme: Theme
    setTheme: (theme: Theme) => void
}

const ThemeProviderContext = React.createContext<ThemeProviderState | undefined>(
    undefined
)

export function ThemeProvider({
    children,
    attribute = 'class',
    defaultTheme = 'system',
    enableSystem = true,
    storageKey = 'theme',
    ...props
}: ThemeProviderProps) {
    const [theme, setTheme] = React.useState<Theme>(defaultTheme)
    const [mounted, setMounted] = React.useState(false)

    React.useEffect(() => {
        if (typeof window === 'undefined') return

        setMounted(true)
        const storedTheme = localStorage.getItem(storageKey) as Theme
        console.log('[ThemeProvider] 初始化，存储的主题:', storedTheme)

        if (storedTheme) {
            setTheme(storedTheme)
        } else {
            // 如果没有存储的主题，检查系统偏好
            const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches
                ? 'dark'
                : 'light'
            console.log('[ThemeProvider] 使用系统主题:', systemTheme)
            setTheme(defaultTheme === 'system' ? 'system' : systemTheme)
        }
    }, [storageKey, defaultTheme])

    React.useEffect(() => {
        if (!mounted || typeof window === 'undefined') return

        const root = window.document.documentElement
        console.log('[ThemeProvider] 应用主题变化:', theme)

        root.classList.remove('light', 'dark')
        console.log('[ThemeProvider] 移除类后的html类:', root.className)

        let appliedTheme: string
        if (theme === 'system' && enableSystem) {
            const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches
                ? 'dark'
                : 'light'
            appliedTheme = systemTheme
            console.log('[ThemeProvider] 系统主题为:', systemTheme)
        } else {
            appliedTheme = theme
            console.log('[ThemeProvider] 直接应用主题:', theme)
        }

        root.classList.add(appliedTheme)
        console.log('[ThemeProvider] 应用主题后的html类:', root.className)
    }, [theme, enableSystem, mounted])

    // Listen for system theme changes
    React.useEffect(() => {
        if (!mounted || !enableSystem || theme !== 'system') return

        const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

        const handleChange = () => {
            const root = window.document.documentElement
            root.classList.remove('light', 'dark')
            const systemTheme = mediaQuery.matches ? 'dark' : 'light'
            root.classList.add(systemTheme)
        }

        mediaQuery.addEventListener('change', handleChange)
        return () => mediaQuery.removeEventListener('change', handleChange)
    }, [mounted, enableSystem, theme])

    const value = React.useMemo(
        () => ({
            theme,
            setTheme: (newTheme: Theme) => {
                console.log('[ThemeProvider] setTheme 被调用:', theme, '->', newTheme)
                if (typeof window !== 'undefined') {
                    localStorage.setItem(storageKey, newTheme)
                    console.log('[ThemeProvider] 主题已保存到 localStorage:', newTheme)
                }
                setTheme(newTheme)
            },
        }),
        [theme, storageKey]
    )

    if (!mounted) {
        return <>{children}</>
    }

    return (
        <ThemeProviderContext.Provider {...props} value={value}>
            {children}
        </ThemeProviderContext.Provider>
    )
}

export const useTheme = () => {
    const context = React.useContext(ThemeProviderContext)

    if (context === undefined) {
        // During SSR or before ThemeProvider is ready, return a working fallback
        const fallbackTheme = typeof window !== 'undefined'
            ? (localStorage.getItem('theme') as Theme) || 'system'
            : 'system'

        return {
            theme: fallbackTheme,
            setTheme: (newTheme: Theme) => {
                if (typeof window !== 'undefined') {
                    localStorage.setItem('theme', newTheme)
                    const root = window.document.documentElement
                    root.classList.remove('light', 'dark')

                    if (newTheme === 'system') {
                        const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches
                            ? 'dark'
                            : 'light'
                        root.classList.add(systemTheme)
                    } else {
                        root.classList.add(newTheme)
                    }
                }
            }
        }
    }

    return context
}