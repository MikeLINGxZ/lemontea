import { AlertBannerRegion } from '@/components/alert/AlertBannerRegion'
import { AlertToastStack } from '@/components/alert/AlertToastStack'

/** 集中挂载横幅提醒区和浮动提醒区，返回组件内容 */
export function AlertViewport() {
  return (
    <>
      <AlertBannerRegion />
      <AlertToastStack />
    </>
  )
}
