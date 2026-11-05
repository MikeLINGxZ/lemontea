import { useRef, useState } from 'react'
import type { CSSProperties, ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { Settings, Sun, Moon, Monitor, ChevronRight, Info, Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Window } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window'
import { useAppStore } from '@/store/appStore'
import { cn } from '@/lib/utils'
import type { Theme } from '@/types'

/** 渲染侧栏页尾界面，返回组件内容 */
export function SidebarFooter() {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const { theme, setTheme } = useAppStore() // 从状态仓库读取主题、集合主题
  const [open, setOpen] = useState(false) // 展开及其更新方法
  const [menuStyle, setMenuStyle] = useState<CSSProperties | null>(null) // 菜单样式及其更新方法
  const [themeMenuStyle, setThemeMenuStyle] = useState<CSSProperties | null>(null) // 主题菜单样式及其更新方法
  const settingsButtonRef = useRef<HTMLButtonElement>(null) // 设置按钮引用，供后续读取当前值
  const themeItemRef = useRef<HTMLDivElement>(null) // 主题条目引用，供后续读取当前值

  /** 主题选项列表 */
  const themeOptions: { /* 值 */ value: Theme; /* 标签 */ label: string; /* 图标 */ icon: ReactNode }[] = [
    { value: 'auto', label: t('settings.themeAuto'), icon: <Monitor size={14} /> },
    { value: 'light', label: t('settings.themeLight'), icon: <Sun size={14} /> },
    { value: 'dark', label: t('settings.themeDark'), icon: <Moon size={14} /> },
  ]

  /** 关闭处理，无返回值 */
  const close = () => {
    setOpen(false)
    setMenuStyle(null)
    setThemeMenuStyle(null)
  }

  /** 切换菜单，无返回值 */
  const toggleMenu = () => {
    if (open) {
      close()
      return
    }

    const rect = settingsButtonRef.current?.getBoundingClientRect() // 矩形区域
    if (!rect) return

    setMenuStyle({
      left: rect.left,
      bottom: window.innerHeight - rect.top + 4,
    })
    setOpen(true)
  }

  /** 打开主题菜单，无返回值 */
  const openThemeMenu = () => {
    if (typeof document === 'undefined') return

    const rect = themeItemRef.current?.getBoundingClientRect() // 矩形区域
    if (!rect) return

    setThemeMenuStyle({
      left: rect.right,
      bottom: window.innerHeight - rect.bottom,
    })
  }

  return (
    <div className="relative z-50 px-2 py-2 border-t border-border">
      <button
        ref={settingsButtonRef}
        onClick={toggleMenu}
        className="flex items-center gap-2 w-full px-3 py-2 rounded-lg hover:bg-accent transition-colors text-sm text-foreground"
      >
        <Settings size={16} className="text-muted-foreground" />
        {t('settings.settings')}
      </button>

      {open && menuStyle && typeof document !== 'undefined' && createPortal(
        <>
          <div className="fixed inset-0 z-[90]" onClick={close} />
          <div
            className="fixed z-[100] w-56 rounded-lg border border-border bg-popover py-1 shadow-lg"
            style={menuStyle}
          >
            <div
              ref={themeItemRef}
              className="relative"
              onMouseEnter={openThemeMenu}
              onMouseLeave={() => setThemeMenuStyle(null)}
            >
              <button className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-accent text-left">
                <Sun size={14} />
                <span className="flex-1">{t('settings.theme')}</span>
                <ChevronRight size={12} className="text-muted-foreground" />
              </button>

              {themeMenuStyle && createPortal(
                <div
                  className="fixed z-[100] w-36 rounded-lg border border-border bg-popover py-1 shadow-lg"
                  style={themeMenuStyle}
                >
                  {themeOptions.map(opt => (
                    <button
                      key={opt.value}
                      onClick={() => {
                        setTheme(opt.value)
                        close()
                      }}
                      className={cn(
                        'w-full flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-accent text-left',
                        theme === opt.value && 'text-primary font-medium'
                      )}
                    >
                      {opt.icon}
                      <span className="flex-1">{opt.label}</span>
                      {theme === opt.value && <Check size={12} />}
                    </button>
                  ))}
                </div>,
                document.body
              )}
            </div>

            <div className="h-px bg-border mx-2 my-1" />

            <button
              className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-accent text-left"
              onClick={() => {
                void Window.OpenSettings({tab:"about"})
                close()
              }}
            >
              <Info size={14} />
              {t('settings.about')}
            </button>

            <button
              className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-accent text-left"
              onClick={() => {
                void Window.OpenSettings({tab:""})
                close()
              }}
            >
              <Settings size={14} />
              {t('settings.settings')}
            </button>
          </div>
        </>,
        document.body
      )}
    </div>
  )
}
