import { FileText, Code2, Languages, PenLine } from 'lucide-react'
import { useTranslation } from 'react-i18next'

/** 快捷起始 */
const QUICK_STARTS = [
  { icon: FileText, tKey: 'summarize' },
  { icon: Code2, tKey: 'code' },
  { icon: Languages, tKey: 'translate' },
  { icon: PenLine, tKey: 'write' },
] as const

/** 渲染欢迎 页面界面，返回组件内容 */
export function WelcomeScreen() {
  const { t } = useTranslation() // 获取当前语言的翻译方法

  return (
    <div className="flex w-full flex-col items-center justify-center px-8 gap-8">
      {/* 欢迎页图标 */}
      <div className="flex flex-col items-center gap-3">
        <svg width="52" height="52" viewBox="0 0 52 52" fill="none">
          <circle cx="26" cy="26" r="22" fill="hsl(var(--primary))" />
          <circle cx="26" cy="26" r="11" fill="hsl(var(--primary-foreground))" opacity="0.9" />
          <circle cx="26" cy="15" r="3.5" fill="hsl(var(--primary-foreground))" />
        </svg>
        <div className="text-center">
          <h2 className="text-xl font-semibold text-foreground">{t('chat.welcome')}</h2>
          <p className="text-sm text-muted-foreground mt-1">{t('chat.welcomeSubtitle')}</p>
        </div>
      </div>

      {/* 快速开始选项 */}
      <div className="grid grid-cols-2 gap-3 w-full max-w-2xl">
        {QUICK_STARTS.map(({ icon: Icon, tKey }) => (
          <button
            key={tKey}
            className="w-full flex items-start gap-3 p-4 rounded-xl border border-border hover:bg-accent hover:border-primary/30 transition-colors text-left"
          >
            <div className="shrink-0 w-9 h-9 rounded-lg bg-primary/10 flex items-center justify-center">
              <Icon size={18} className="text-primary" />
            </div>
            <div className="flex-1 min-w-0">
              <div className="text-sm font-medium text-foreground">
                {t(`quickStart.${tKey}`)}
              </div>
              <div className="text-xs text-muted-foreground mt-0.5 line-clamp-2 whitespace-normal break-words">
                {t(`quickStart.${tKey}Desc`)}
              </div>
            </div>
          </button>
        ))}
      </div>
    </div>
  )
}
