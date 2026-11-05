import { useEffect, useRef, useState, type ReactElement } from 'react'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { File as FileBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file'
import { isVideoAttachment } from '@/lib/attachments'
import type { Attachment } from '@/types'

/** 媒体预览弹窗的属性 */
interface Props {
  /** 正在预览的附件 */
  attachment: Attachment
  /** 消息中已加载的图片地址 */
  imageSource?: string
  /** 关闭弹窗的回调，无参数和返回值 */
  onClose: () => void
}

/** 在应用内放大图片或播放视频，接收附件、已有图片地址和关闭回调，返回模态预览 */
export function AttachmentPreview({ attachment, imageSource, onClose }: Props): ReactElement {
  const { t } = useTranslation() // 当前语言的翻译方法
  const dialogRef = useRef<HTMLDialogElement>(null) // 原生模态弹窗引用
  const video = isVideoAttachment(attachment) // 附件是否为视频
  const [source, setSource] = useState<string | null>(imageSource ?? null) // 媒体地址，空字符串表示无法预览

  // 使用原生模态层锁定背景焦点，并支持 Escape 关闭
  useEffect((): void => {
    if (dialogRef.current && !dialogRef.current.open) dialogRef.current.showModal()
  }, [])

  // 视频在打开弹窗时读取，关闭后忽略过期结果并释放播放地址
  useEffect((): (() => void) | undefined => {
    if (imageSource) {
      setSource(imageSource)
      return
    }
    let active = true // 当前媒体读取是否仍有效
    let videoUrl = '' // 需在关闭时释放的本地播放地址

    // 读取本地媒体并准备浏览器可显示的地址，无参数和返回值
    void (async (): Promise<void> => {
      try {
        const data = await FileBinding.ReadMediaPreview(attachment.path) // 后端返回的媒体数据
        if (!active) return
        if (video && data.startsWith('data:video/')) {
          const media = await (await fetch(data)).blob() // 由本地数据构造的可定位播放内容
          if (!active) return
          videoUrl = URL.createObjectURL(media)
          setSource(videoUrl)
        } else {
          setSource(!video && data.startsWith('data:image/') ? data : '')
        }
      } catch {
        if (active) setSource('')
      }
    })()

    return (): void => {
      active = false
      if (videoUrl) URL.revokeObjectURL(videoUrl)
    }
  }, [attachment.path, imageSource, video])

  return (
    <dialog
      ref={dialogRef}
      aria-label={attachment.name}
      className="m-auto w-max min-w-56 max-w-[calc(100vw-2rem)] overflow-auto rounded-2xl border border-border bg-background p-0 text-foreground shadow-2xl backdrop:bg-black/70"
      onClose={onClose}
      onClick={(event): void => {
        // 点击遮罩时关闭，媒体区域内的点击保留给播放器控件
        if (event.target === event.currentTarget) event.currentTarget.close()
      }}
    >
      <div className="flex flex-col">
        <div className="flex items-center justify-between gap-4 px-4 py-2">
          <span className="min-w-0 truncate text-sm" title={attachment.name}>{attachment.name}</span>
          <button
            type="button"
            aria-label={t('chat.closePreview')}
            className="shrink-0 rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
            onClick={() => dialogRef.current?.close()}
          >
            <X size={18} />
          </button>
        </div>
        {source ? (
          video ? (
            <video
              src={source}
              aria-label={attachment.name}
              controls
              playsInline
              className="block max-h-[75dvh] max-w-full bg-black"
              onError={() => setSource('')}
            />
          ) : (
            <img
              src={source}
              alt={attachment.name}
              className="block max-h-[75dvh] max-w-full object-contain"
              onError={() => setSource('')}
            />
          )
        ) : (
          <p role="status" className="px-4 pb-4 text-sm text-muted-foreground">
            {t(source === null ? 'chat.loadingPreview' : 'chat.previewUnavailable')}
          </p>
        )}
      </div>
    </dialog>
  )
}
