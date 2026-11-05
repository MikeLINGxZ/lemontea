import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

/** CLI安装起点文档模态框组件接收的属性 */
type Props = {
  /** 展开 */
  open: boolean
  /** 处理关闭事件，无返回值 */
  onClose: () => void
  /** 处理提交事件，接收内容，无返回值 */
  onSubmit: (content: string) => void
}

/** 渲染CLI安装起点文档模态框界面，接收展开、on关闭、on提交，返回组件内容 */
export function CliInstallFromDocsModal({ open, onClose, onSubmit }: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法
  const [content, setContent] = useState('') // 内容及其更新方法

  /** 同步外部状态并管理本次副作用的生命周期 */
  useEffect(() => {
    if (!open) setContent('')
  }, [open])

  if (!open) return null

  /** 处理提交，无返回值 */
  const handleSubmit = () => {
    if (!content.trim()) return
    onSubmit(content.trim())
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <button type="button" className="absolute inset-0 bg-black/35" onClick={onClose} />
      <div role="dialog" className="relative z-10 w-full max-w-lg rounded-3xl border bg-background p-5 shadow-2xl">
        <h3 className="mb-2 text-base font-semibold">{t('settingsPage.plugins.cliFromDocs')}</h3>
        <p className="mb-3 text-xs text-muted-foreground">{t('settingsPage.plugins.cliFromDocsHint')}</p>
        <textarea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder={t('settingsPage.plugins.cliDocsPlaceholder')}
          rows={8}
          className="mb-4 w-full resize-none rounded-lg border bg-background px-3 py-2 text-sm"
          autoFocus
        />
        <div className="flex justify-end gap-2">
          <button type="button" onClick={onClose} className="rounded-xl border px-4 py-2 text-sm">
            {t('settingsPage.plugins.cliCancel')}
          </button>
          <button
            type="button"
            onClick={handleSubmit}
            disabled={!content.trim()}
            className="rounded-xl bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-60"
          >
            {t('settingsPage.plugins.cliInstall')}
          </button>
        </div>
      </div>
    </div>
  )
}
