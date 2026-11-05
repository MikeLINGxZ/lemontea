import { useTranslation } from 'react-i18next'
import { FileText, Image as ImageIcon, Paperclip, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { isImageMime, isPdfMime } from '@/lib/attachments'
import type { Attachment } from '@/types'

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
  if (isPdfMime(att.mime)) return <FileText size={12} />
  return <Paperclip size={12} />
}

/** 渲染附件标签条界面，接收条目列表、展示变体、on移除、on展开，返回组件内容 */
export function AttachmentChips({ items, variant, onRemove, onOpen }: Props) {
  const { t } = useTranslation() // 获取当前语言的翻译方法

  if (items.length === 0) return null

  return (
    <div className="flex flex-wrap gap-1.5 px-3 py-2">
      {items.map((att, index) => (
        variant === 'message' ? (
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
        ) : (
          <span
            key={`${att.path}-${index}`}
            className="inline-flex items-center gap-1 max-w-[18rem] rounded-md border border-border bg-muted/60 px-2 py-1 text-xs text-foreground"
          >
            {chipIcon(att)}
            <span className="truncate" title={att.name}>{att.name}</span>
            {onRemove && (
            <button
              type="button"
              aria-label={t('chat.attachmentRemove')}
              onClick={() => onRemove(index)}
              className="ml-0.5 rounded text-muted-foreground hover:text-foreground hover:bg-accent p-0.5"
            >
              <X size={10} />
            </button>
            )}
          </span>
        )
      ))}
    </div>
  )
}
