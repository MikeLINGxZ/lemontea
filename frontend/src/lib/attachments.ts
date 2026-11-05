import type { Attachment, AttachmentKind } from '@/types'

/** 附件上限数量 */
export const ATTACHMENT_MAX_COUNT = 10
/** 附件上限字节 */
export const ATTACHMENT_MAX_BYTES = 20 * 1024 * 1024

/** 扩展名到 MIME 类型的对应关系 */
const EXT_MIME: Record<string, string> = {
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.webp': 'image/webp',
  '.gif': 'image/gif',
  '.mp4': 'video/mp4', // MP4 视频
  '.m4v': 'video/mp4', // M4V 视频
  '.mov': 'video/quicktime', // QuickTime 视频
  '.webm': 'video/webm', // WebM 视频
  '.pdf': 'application/pdf',
  '.txt': 'text/plain',
  '.log': 'text/plain',
  '.md': 'text/markdown',
  '.markdown': 'text/markdown',
  '.json': 'application/json',
  '.html': 'text/html',
  '.css': 'text/css',
  '.js': 'text/javascript',
  '.ts': 'application/typescript',
  '.tsx': 'application/typescript',
  '.go': 'text/x-go',
  '.py': 'text/x-python',
  '.rs': 'text/x-rust',
  '.yaml': 'application/yaml',
  '.yml': 'application/yaml',
  '.toml': 'application/toml',
}

/** 提取文件名处理，接收文件路径，返回处理后的文本 */
function basename(path: string): string {
  // 同时识别 Unix 和 Windows 的路径分隔符
  const idx = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'))
  return idx >= 0 ? path.slice(idx + 1) : path
}

/** 扩展名of处理，接收文件路径，返回处理后的文本 */
function extOf(path: string): string {
  const name = basename(path) // 名称
  const dot = name.lastIndexOf('.') // 扩展名分隔符的位置
  return dot >= 0 ? name.slice(dot).toLowerCase() : ''
}

/** 推断附件元信息处理，接收文件路径，返回对应的处理结果 */
export function inferAttachmentMeta(path: string): Attachment {
  const name = basename(path) // 名称
  const mime = EXT_MIME[extOf(path)] ?? 'application/octet-stream' // 文件的 MIME 类型
  const kind: AttachmentKind = mime.startsWith('image/') ? 'image' : 'file' // 类别标识
  return { path, name, mime, kind }
}

/** 判断是否为图片的 MIME 类型类型，接收文件的 MIME 类型，返回是否满足条件 */
export function isImageMime(mime: string): boolean {
  return mime.startsWith('image/')
}

/** 判断附件是否为视频，兼容历史附件未记录媒体类型的情况，接收附件，返回判断结果 */
export function isVideoAttachment(attachment: Attachment): boolean {
  return attachment.mime.startsWith('video/') ||
    ((!attachment.mime || attachment.mime === 'application/octet-stream') &&
      (EXT_MIME[extOf(attachment.path)]?.startsWith('video/') ?? false))
}

/** 判断是否为PDF 的 MIME 类型类型，接收文件的 MIME 类型，返回是否满足条件 */
export function isPdfMime(mime: string): boolean {
  return mime === 'application/pdf'
}
