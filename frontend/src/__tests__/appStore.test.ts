import { describe, it, expect, beforeEach } from 'vitest'
import { useAppStore } from '../store/appStore'

beforeEach(() => {
  useAppStore.setState({
    theme: 'auto',
    fontSize: 'md',
    language: 'zh-CN',
  })
})

describe('appStore', () => {
  it('uses medium font size by default', () => {
    expect(useAppStore.getInitialState().fontSize).toBe('md')
  })

  it('setTheme updates theme', () => {
    useAppStore.getState().setTheme('dark')
    expect(useAppStore.getState().theme).toBe('dark')
  })

  it('setFontSize updates fontSize', () => {
    useAppStore.getState().setFontSize('xl')
    expect(useAppStore.getState().fontSize).toBe('xl')
  })

  it('setLanguage updates language', () => {
    useAppStore.getState().setLanguage('en')
    expect(useAppStore.getState().language).toBe('en')
  })
})
