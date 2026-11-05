import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import type { OnboardingStep } from './OnboardingApp'

/** 步骤导航组件接收的属性 */
type Props = {
	/** 当前 */
	current: OnboardingStep
}

/** 顺序 */
const order: OnboardingStep[] = ['intro', 'select', 'config', 'runtime']

/** 标签键集合 */
const labelKeys: Record<OnboardingStep, string> = {
	intro: 'onboarding.steps.intro',
	select: 'onboarding.steps.select',
	config: 'onboarding.steps.config',
	runtime: 'onboarding.steps.runtime',
}

/** 渲染步骤导航界面，接收组件输入属性，返回组件内容 */
export function Stepper(props: Props) {
	const { t } = useTranslation() // 获取当前语言的翻译方法
	const currentIdx = order.indexOf(props.current) // 当前索引

	return (
		<div className="flex items-center gap-3">
			{order.map((step, idx) => {
				const active = idx === currentIdx // 当前
				const done = idx < currentIdx // 完成
				return (
					<div key={step} className="flex items-center gap-3">
						{idx > 0 && <div className="h-px w-10 bg-border" />}
						<div className="flex items-center gap-2">
							<div
								className={cn(
									'flex h-6 w-6 items-center justify-center rounded-full text-[11px] font-semibold',
									active
										? 'bg-primary text-primary-foreground'
										: done
											? 'bg-primary/30 text-primary'
											: 'bg-muted text-muted-foreground',
								)}
							>
								{idx + 1}
							</div>
							<span
								className={cn(
									'text-sm',
									active ? 'font-medium text-foreground' : 'text-muted-foreground',
								)}
							>
								{t(labelKeys[step])}
							</span>
						</div>
					</div>
				)
			})}
		</div>
	)
}
