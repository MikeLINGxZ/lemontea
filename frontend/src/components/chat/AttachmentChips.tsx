import { useEffect, useState, type ReactElement } from 'react'
import { useTranslation } from 'react-i18next'
import { FileText, Image as ImageIcon, Paperclip, Video, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { isImageMime, isPdfMime, isVideoAttachment } from '@/lib/attachments'
import type { Attachment } from '@/types'
import { File as FileBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file'
import { AttachmentPreview } from './AttachmentPreview'

/** 附件标签条组件接收的属性 */
interface Props {
  /** 条目列表 */
  items: Attachment[]
  /** 展示变体 */
  variant: 'input' | 'message'
  /** 处理移除事件，接收索引，无返回值 */
  onRemove?: (index: number) => void
  /** 处理展开事件，接收当前条目，无返回值 */
  onOpen?: (item: Attachment) => void
}

/** 标签条图标处理，接收当前附件，返回对应的处理结果 */
function chipIcon(att: Attachment) {
  if (isImageMime(att.mime)) return <ImageIcon size={12} />
  if (isVideoAttachment(att)) return <Video size={12} />
  if (isPdfMime(att.mime)) return <FileText size={12} />
  return <Paperclip size={12} />
}

/** 输入框附件卡片接收的属性 */
interface InputAttachmentCardProps {
  /** 当前卡片对应的附件 */
  attachment: Attachment
  /** 当前附件在列表中的位置 */
  index: number
  /** 移除附件的回调，接收附件索引，无返回值 */
  onRemove?: (index: number) => void
}

/** 显示待发送附件的卡片，接收附件、索引和移除回调，返回卡片及其预览弹窗 */
function InputAttachmentCard({ attachment, index, onRemove }: InputAttachmentCardProps): ReactElement {
  const { t } = useTranslation() // 当前语言的翻译方法
  const isImage = attachment.kind === 'image' || isImageMime(attachment.mime) // 是否显示图片缩略图
  const [source, setSource] = useState('') // 缩略图的本地数据地址
  const [previewOpen, setPreviewOpen] = useState(false) // 是否打开应用内预览

  // 仅为当前图片读取缩略图，移除卡片时忽略未完成的读取
  useEffect((): (() => void) | undefined => {
    if (!isImage) return
    let active = true // 当前卡片是否仍接收读取结果

    // 读取本地图片并更新缩略图，无参数和返回值
    void (async (): Promise<void> => {
      try {
        const preview = await FileBinding.ReadMediaPreview(attachment.path) // 后端返回的图片数据地址
        if (active && preview.startsWith('data:image/')) setSource(preview)
      } catch {
        // 图片读取失败时保留文件名，仍可点击查看预览状态
      }
    })()

    // 停止接收已移除卡片的读取结果，无参数和返回值
    return (): void => { active = false }
  }, [attachment.path, isImage])

  return (
    <li className={cn(
      'relative h-16 w-20 shrink-0 rounded-lg border border-border bg-muted/60 text-foreground',
      !isImage && 'flex flex-col items-center justify-center gap-1 px-2',
    )}>
      {isImage ? (
        <button
          type="button"
          title={attachment.name}
          aria-label={attachment.name}
          className="flex h-full w-full cursor-pointer items-center justify-center overflow-hidden rounded-[inherit] hover:bg-accent focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
          onClick={() => setPreviewOpen(true)}
        >
          {source ? (
            <img
              src={source}
              alt=""
              className="h-full w-full object-cover"
              decoding="async"
              onError={() => setSource('')}
            />
          ) : (
            <span className="flex min-w-0 flex-col items-center gap-1 px-2 text-xs">
              {chipIcon(attachment)}
              <span className="w-full truncate">{attachment.name}</span>
            </span>
          )}
        </button>
      ) : (
        <>
          <span className="shrink-0 text-muted-foreground">{chipIcon(attachment)}</span>
          <span className="w-full truncate text-center text-xs" title={attachment.name}>{attachment.name}</span>
        </>
      )}
      {onRemove && (
        <button
          type="button"
          aria-label={t('chat.attachmentRemove')}
          className="absolute right-1 top-1 rounded-md bg-background/90 p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
          onClick={() => onRemove(index)}
        >
          <X size={14} />
        </button>
      )}
      {previewOpen && (
        <AttachmentPreview
          attachment={attachment}
          imageSource={source || undefined}
          onClose={() => setPreviewOpen(false)}
        />
      )}
    </li>
  )
}

/** 渲染附件标签条界面，接收条目列表、展示变体、on移除、on展开，返回组件内容 */
export function AttachmentChips({ items, variant, onRemove, onOpen }: Props) {
  if (items.length === 0) return null

  if (variant === 'input') {
    return (
      <ul className="flex gap-2 overflow-x-auto px-3 py-2">
        {items.map((attachment, index) => (
          <InputAttachmentCard
            key={`${attachment.path}-${index}`}
            attachment={attachment}
            index={index}
            onRemove={onRemove}
          />
        ))}
      </ul>
    )
  }

  return (
    <div className="flex flex-wrap gap-1.5 px-3 py-2">
      {items.map((att, index) => (
        <button
          key={`${att.path}-${index}`}
          type="button"
          onClick={() => onOpen?.(att)}
          className={cn(
            'inline-flex items-center gap-1 max-w-[18rem] rounded-md border border-border bg-muted/60 px-2 py-1 text-xs text-foreground',
            'bg-background/60 hover:bg-background/80 transition-colors'
          )}
          title={att.name}
        >
          {chipIcon(att)}
          <span className="truncate">{att.name}</span>
        </button>
      ))}
    </div>
  )
}
