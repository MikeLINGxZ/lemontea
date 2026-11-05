import { useState } from 'react'
import { AlertViewport } from '@/components/alert/AlertViewport'
import { AlertEventProvider } from '@/components/providers/AlertEventProvider'
import { AppSettingsSyncProvider } from '@/components/providers/AppSettingsSyncProvider'
import { FontSizeProvider } from '@/components/providers/FontSizeProvider'
import { ThemeProvider } from '@/components/providers/ThemeProvider'
import type { SupportedProvider } from '@/types/settings'
import { OnboardingStepForm } from './OnboardingStepForm'
import { OnboardingStepIntro } from './OnboardingStepIntro'
import { OnboardingStepRuntime } from './OnboardingStepRuntime'
import { OnboardingStepSelect } from './OnboardingStepSelect'
import { Stepper } from './Stepper'

/** 首次引导步骤的可选值 */
export type OnboardingStep = 'intro' | 'select' | 'config' | 'runtime'

/** 驱动首次引导的步骤跳转、配置保存和退出流程，返回组件内容 */
export function OnboardingApp() {
	return (
		<AppSettingsSyncProvider>
			<AlertEventProvider>
				<ThemeProvider>
					<FontSizeProvider>
						<OnboardingWizard />
						<AlertViewport />
					</FontSizeProvider>
				</ThemeProvider>
			</AlertEventProvider>
		</AppSettingsSyncProvider>
	)
}

/** 组织首次引导的运行环境和模型服务商配置步骤，返回组件内容 */
function OnboardingWizard() {
	const [step, setStep] = useState<OnboardingStep>('intro') // 步骤及其更新方法
	const [provider, setProvider] = useState<SupportedProvider | null>(null) // 模型服务商及其更新方法

	return (
		<div className="flex h-screen flex-col overflow-hidden bg-background text-foreground">
			<div className="shrink-0 px-6 pb-6 pt-12">
				<Stepper current={step} />
			</div>
			<div className="flex min-h-0 flex-1 flex-col overflow-hidden px-6 pb-6">
				{step === 'intro' && (
					<OnboardingStepIntro onStart={() => setStep('select')} />
				)}
				{step === 'select' && (
					<OnboardingStepSelect
						selected={provider}
						onSelect={setProvider}
						onBack={() => setStep('intro')}
						onNext={() => setStep('config')}
					/>
				)}
				{step === 'config' && provider && (
					<OnboardingStepForm
						provider={provider}
						onBack={() => setStep('select')}
						onDone={() => setStep('runtime')}
					/>
				)}
				{step === 'runtime' && (
					<OnboardingStepRuntime onBack={() => setStep('config')} />
				)}
			</div>
		</div>
	)
}
