import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

/** 标签页的可选值 */
type Tab = 'smart' | 'npm'

/** CLI安装模态框组件接收的属性 */
type Props = {
  /** 展开 */
  open: boolean
  /** 处理关闭事件，无返回值 */
  onClose: () => void
  /** 处理 npm 提交，接收包名和显示名称，返回后台安装任务 */
  onNpmSubmit: (npmPackage: string, name: string) => Promise<void>
  /** 处理智能提交事件，接收内容，无返回值 */
  onSmartSubmit: (content: string) => void
}

/** 展示 CLI 安装入口和进度，并按安装方式提交表单，接收展开、on关闭、on npm提交、on智能提交，返回组件内容 */
export function CliInstallModal({ open, onClose, onNpmSubmit, onSmartSubmit }: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [tab, setTab] = useState<Tab>('smart') // 标签页及其更新方法
  const [content, setContent] = useState('') // 内容及其更新方法
  const [npmPkg, setNpmPkg] = useState('') // npm包名及其更新方法
  const [name, setName] = useState('') // 名称及其更新方法

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (!open) {
      setContent('')
      setNpmPkg('')
      setName('')
      setTab('smart')
    }
  }, [open])

  if (!open) return null

  /** 处理智能提交，无返回值 */
  const handleSmartSubmit = () => {
    if (!content.trim()) return
    onSmartSubmit(content.trim())
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button type="button" className="absolute inset-0 bg-black/35" onClick={onClose} />
      <div role="dialog" className="relative z-10 w-full max-w-lg rounded-3xl border bg-background p-5 shadow-2xl">
        <h3 className="mb-4 text-base font-semibold">{t('settingsPage.plugins.cliInstallTitle')}</h3>

        {/* 安装方式切换区 */}
        <div className="mb-4 flex rounded-lg bg-muted p-0.5">
          <button
            type="button"
            onClick={() => setTab('smart')}
            className={cn(
              'flex-1 rounded-md py-1.5 text-sm font-medium transition-colors',
              tab === 'smart'
                ? 'bg-background text-foreground shadow-xs'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            {t('settingsPage.plugins.cliSmartInstall')}
          </button>
          <button
            type="button"
            onClick={() => setTab('npm')}
            className={cn(
              'flex-1 rounded-md py-1.5 text-sm font-medium transition-colors',
              tab === 'npm'
                ? 'bg-background text-foreground shadow-xs'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            {t('settingsPage.plugins.cliFromNpm')}
          </button>
        </div>

        {tab === 'smart' && (
          <>
            <p className="mb-3 text-xs text-muted-foreground">{t('settingsPage.plugins.cliFromDocsHint')}</p>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder={t('settingsPage.plugins.cliDocsPlaceholder')}
              rows={7}
              className="mb-4 w-full resize-none rounded-lg border bg-background px-3 py-2 text-sm"
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <button type="button" onClick={onClose} className="rounded-xl border px-4 py-2 text-sm">
                {t('settingsPage.plugins.cliCancel')}
              </button>
              <button
                type="button"
                onClick={handleSmartSubmit}
                disabled={!content.trim()}
                className="rounded-xl bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-60"
              >
                {t('settingsPage.plugins.cliInstall')}
              </button>
            </div>
          </>
        )}

        {tab === 'npm' && (
          <form onSubmit={/** 提交表单并在后台安装，接收提交事件，无返回值 */ (event) => {
            event.preventDefault()
            const npmPackage = npmPkg.trim() // 去除首尾空格的包名
            if (!npmPackage) return
            void onNpmSubmit(npmPackage, name.trim())
            onClose()
          }}>
            <input
              value={npmPkg}
              onChange={(e) => setNpmPkg(e.target.value)}
              placeholder={t('settingsPage.plugins.cliNpmPlaceholder')}
              className="mb-3 w-full rounded-lg border bg-background px-3 py-2 text-sm"
              autoFocus
            />
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('settingsPage.plugins.cliNamePlaceholder')}
              className="mb-4 w-full rounded-lg border bg-background px-3 py-2 text-sm"
            />
            <div className="flex justify-end gap-2">
              <button type="button" onClick={onClose} className="rounded-xl border px-4 py-2 text-sm">
                {t('settingsPage.plugins.cliCancel')}
              </button>
              <button
                type="submit"
                disabled={!npmPkg.trim()}
                className="rounded-xl bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-60"
              >
                {t('settingsPage.plugins.cliInstall')}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
