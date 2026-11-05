import { useTranslation } from 'react-i18next'
import { Onboarding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/onboarding/index'

/** 首次引导步骤介绍组件接收的属性 */
type Props = {
	/** 处理开始事件，无返回值 */
	onStart: () => void
}

/** 渲染首次引导步骤介绍界面，接收组件输入属性，返回组件内容 */
export function OnboardingStepIntro(props: Props) {
	const { t } = useTranslation() // 获取当前语言的翻译方法

	/** 处理退出，无返回值 */
	const handleExit = () => {
		void Onboarding.ExitApp({})
	}

	return (
		<div className="flex h-full min-h-0 flex-col gap-6">
			<div className="space-y-4">
				<span className="inline-block rounded-full bg-primary/10 px-3 py-1 text-xs font-semibold uppercase tracking-wider text-primary">
					{t('onboarding.intro.badge')}
				</span>
				<h1 className="text-3xl font-semibold leading-tight text-foreground">
					{t('onboarding.intro.heroTitle')}
				</h1>
			</div>

			<div className="flex flex-col gap-3">
				<IntroCard titleKey="onboarding.intro.cards.whatTitle" textKey="onboarding.intro.cards.whatText" />
				<IntroCard titleKey="onboarding.intro.cards.whyTitle" textKey="onboarding.intro.cards.whyText" />
				<IntroCard titleKey="onboarding.intro.cards.laterTitle" textKey="onboarding.intro.cards.laterText" />
			</div>

			<div className="mt-auto flex justify-end gap-2">
				<button
					type="button"
					onClick={handleExit}
					className="rounded-xl border border-border px-5 py-2 text-sm font-medium text-foreground transition-colors hover:bg-accent"
				>
					{t('onboarding.actions.exit')}
				</button>
				<button
					type="button"
					onClick={props.onStart}
					className="rounded-xl bg-primary px-5 py-2 text-sm font-medium text-primary-foreground"
				>
					{t('onboarding.actions.start')}
				</button>
			</div>
		</div>
	)
}

/** 渲染介绍卡片界面，接收组件输入属性，返回组件内容 */
function IntroCard(props: { /* 标题键 */ titleKey: string; /* 文本键 */ textKey: string }) {
	const { t } = useTranslation() // 获取当前语言的翻译方法

	return (
		<div className="rounded-2xl border border-border bg-card px-4 py-4">
			<div className="text-sm font-semibold text-primary">{t(props.titleKey)}</div>
			<div className="mt-1.5 text-sm leading-relaxed text-muted-foreground">{t(props.textKey)}</div>
		</div>
	)
}
