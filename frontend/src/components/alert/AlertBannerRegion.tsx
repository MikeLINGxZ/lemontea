import { useAlertStore } from '@/alert/store'
import { AlertCard } from '@/components/alert/AlertCard'

/** 按队列展示页面顶部的横幅提醒，返回组件内容 */
export function AlertBannerRegion() {
  const banners = useAlertStore((state) => state.banners) // 从状态仓库读取横幅提醒列表

  return (
    <section
      aria-label="Banner alerts"
      className="pointer-events-none fixed inset-x-0 top-16 z-50 flex justify-center px-6"
    >
      {banners.map((alert) => (
        <div key={alert.id} className="pointer-events-auto w-full max-w-2xl">
          <AlertCard alert={alert} />
        </div>
      ))}
    </section>
  )
}
