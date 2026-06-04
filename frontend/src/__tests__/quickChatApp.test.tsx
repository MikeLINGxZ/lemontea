import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { QuickChatApp } from '@/components/quick-chat/QuickChatApp'
import i18n from '@/i18n'
import { useChatStore } from '@/store/chatStore'
import type { Conversation } from '@/types'
import { Window } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window'
import { Agent as AgentBinding } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/agent'
import { Events, Window as RuntimeWindow } from '@wailsio/runtime'

const runtimeEventHandlers = vi.hoisted(() => new Map<string, (event?: unknown) => void>())

vi.mock('@wailsio/runtime', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@wailsio/runtime')>()
  return {
    ...actual,
    Events: {
      ...actual.Events,
      On: vi.fn((eventName: string, handler: (event?: unknown) => void) => {
        runtimeEventHandlers.set(eventName, handler)
        return () => runtimeEventHandlers.delete(eventName)
      }),
    },
    Window: {
      ...actual.Window,
      SetSize: vi.fn(),
      Center: vi.fn(),
    },
  }
})

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/window', () => ({
  Window: {
    CloseQuickChat: vi.fn(),
    SelectQuickChat: vi.fn(),
  },
}))

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/agent', () => ({
  Agent: {
    CreateSession: vi.fn(),
    DeleteSession: vi.fn(),
    RenameSession: vi.fn(),
    ToggleStarSession: vi.fn(),
    ListSessions: vi.fn(),
    LoadSessionMessages: vi.fn(),
    MarkSessionRead: vi.fn(),
    SendMessage: vi.fn(),
    StopGeneration: vi.fn(),
    RespondToConfirm: vi.fn(),
    GenerateTitle: vi.fn(),
  },
}))

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/provider', () => ({
  Provider: {
    ProviderAndModelList: vi.fn().mockResolvedValue({
      provider_models: [
        {
          provider: {
            id: 1,
            provider_name: 'P',
            enabled: true,
            is_default: true,
            base_url: 'http://x',
            api_key: 'k',
            provider_type: 'aliyun',
          },
          models: [{ id: 1, model: 'm', alias: 'm', is_default: true, enable: true }],
        },
      ],
    }),
  },
}))

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/plugin', () => ({
  Plugin: {
    ListAvailableTools: vi.fn().mockResolvedValue({
      tools: [
        { id: 'mcp:filesystem:1.0.0', name: 'filesystem', description: 'Filesystem', category: 'mcp' },
      ],
    }),
  },
}))

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/terminal', () => ({
  Terminal: {
    ListTerminals: vi.fn().mockResolvedValue({ items: [] }),
    ReadTerminalOutput: vi.fn().mockResolvedValue({ chunks: [] }),
  },
}))

vi.mock('@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service/file', () => ({
  File: {
    OpenFile: vi.fn(),
  },
}))

const latestConversation: Conversation = {
  id: 42,
  title: 'Latest planning chat',
  kind: 'user',
  createdAt: '2026-06-01T08:00:00Z',
  updatedAt: '2026-06-01T08:30:00Z',
  starred: false,
  status: 'idle',
}

describe('QuickChatApp', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en')
    vi.clearAllMocks()
    useChatStore.setState({
      conversations: [],
      currentConversationId: null,
      messages: {},
      sessionStatuses: {},
      sessionsLoading: false,
      hasMoreSessions: true,
      sessionsCursor: 0,
    })
    vi.mocked(AgentBinding.ListSessions).mockResolvedValue({
      sessions: [],
      next_cursor: 0,
      has_more: false,
    })
    vi.mocked(AgentBinding.CreateSession).mockResolvedValue({
      session_id: 88,
      title: 'New Chat',
      tags: [],
    })
    vi.mocked(AgentBinding.SendMessage).mockResolvedValue({})
    vi.mocked(Window.CloseQuickChat).mockResolvedValue({})
    vi.mocked(Window.SelectQuickChat).mockResolvedValue({})
    vi.mocked(RuntimeWindow.SetSize).mockResolvedValue(undefined)
    vi.mocked(RuntimeWindow.Center).mockResolvedValue(undefined)
    runtimeEventHandlers.clear()
  })

  it('defaults to new chat when there is no history and disables continue', async () => {
    render(<QuickChatApp />)

    const input = await screen.findByRole('textbox', { name: /quick chat message/i })
    const continueButton = screen.getByRole('button', { name: /continue/i })
    const newButton = screen.getByRole('button', { name: /^new$/i })

    expect(input).toHaveFocus()
    expect(continueButton).toBeDisabled()
    expect(newButton).toHaveAttribute('aria-pressed', 'true')
  })

  it('uses the shell as the draggable surface without wrapping the mode switch', async () => {
    render(<QuickChatApp />)

    const shell = await screen.findByTestId('quick-chat-shell')
    const input = screen.getByRole('textbox', { name: /quick chat message/i })
    const newButton = screen.getByRole('button', { name: /^new$/i })
    const continueButton = screen.getByRole('button', { name: /continue/i })

    expect(shell).toHaveStyle({ '--wails-draggable': 'drag' })
    expect(input).toHaveStyle({ '--wails-draggable': 'no-drag' })
    expect(newButton).toHaveStyle({ '--wails-draggable': 'no-drag' })
    expect(continueButton).toHaveStyle({ '--wails-draggable': 'no-drag' })
    expect(newButton.parentElement).toBe(shell)
    expect(continueButton.parentElement).toBe(shell)
  })

  it('stretches the initial input shell to the full quick chat window height', async () => {
    render(<QuickChatApp />)

    const shell = await screen.findByTestId('quick-chat-shell')

    expect(shell).toHaveClass('h-full')
    expect(shell.className).not.toContain('h-[60px]')
  })

  it('requests the compact picker window size on first mount', async () => {
    render(<QuickChatApp />)

    await screen.findByTestId('quick-chat-shell')

    expect(RuntimeWindow.SetSize).toHaveBeenCalledWith(720, 54)
    expect(RuntimeWindow.Center).toHaveBeenCalled()
  })

  it('does nothing when Enter is pressed with an empty input', async () => {
    vi.mocked(AgentBinding.ListSessions).mockResolvedValue({
      sessions: [{
        id: latestConversation.id,
        title: latestConversation.title,
        kind: latestConversation.kind,
        tags: [],
        created: latestConversation.createdAt,
        updated: latestConversation.updatedAt,
        starred: latestConversation.starred,
        status: latestConversation.status,
      }],
      next_cursor: 0,
      has_more: false,
    })

    render(<QuickChatApp />)

    await screen.findByRole('textbox', { name: /quick chat message/i })
    await userEvent.keyboard('{Enter}')

    await waitFor(() => expect(AgentBinding.ListSessions).toHaveBeenCalled())
    expect(Window.SelectQuickChat).not.toHaveBeenCalled()
    expect(AgentBinding.CreateSession).not.toHaveBeenCalled()
    expect(AgentBinding.SendMessage).not.toHaveBeenCalled()
  })

  it('switches with Tab but does not create a fresh chat on empty Enter', async () => {
    vi.mocked(AgentBinding.ListSessions).mockResolvedValue({
      sessions: [{
        id: latestConversation.id,
        title: latestConversation.title,
        kind: latestConversation.kind,
        tags: [],
        created: latestConversation.createdAt,
        updated: latestConversation.updatedAt,
        starred: latestConversation.starred,
        status: latestConversation.status,
      }],
      next_cursor: 0,
      has_more: false,
    })

    render(<QuickChatApp />)

    await screen.findByRole('textbox', { name: /quick chat message/i })
    await userEvent.keyboard('{Tab}{Enter}')

    await waitFor(() => expect(AgentBinding.ListSessions).toHaveBeenCalled())
    expect(AgentBinding.CreateSession).not.toHaveBeenCalled()
    expect(Window.SelectQuickChat).not.toHaveBeenCalled()
  })

  it('sends typed input and then reuses the full chat page content locally', async () => {
    vi.mocked(AgentBinding.ListSessions).mockResolvedValue({
      sessions: [{
        id: latestConversation.id,
        title: latestConversation.title,
        kind: latestConversation.kind,
        tags: [],
        created: latestConversation.createdAt,
        updated: latestConversation.updatedAt,
        starred: latestConversation.starred,
        status: latestConversation.status,
      }],
      next_cursor: 0,
      has_more: false,
    })

    render(<QuickChatApp />)

    const input = await screen.findByRole('textbox', { name: /quick chat message/i })
    await userEvent.type(input, 'Summarize the plan{Enter}')

    await waitFor(() => {
      expect(AgentBinding.SendMessage).toHaveBeenCalledWith(expect.objectContaining({
        session_id: 42,
        content: 'Summarize the plan',
        base_url: 'http://x',
        api_key: 'k',
        model_name: 'm',
        provider_type: 'aliyun',
        enabled_user_tools: [],
        attachments: [],
      }))
      expect(RuntimeWindow.SetSize).toHaveBeenCalledWith(720, 520)
      expect(RuntimeWindow.Center).toHaveBeenCalled()
    })
    expect(Window.SelectQuickChat).not.toHaveBeenCalled()
    expect(screen.getByRole('heading', { name: latestConversation.title })).toBeInTheDocument()
    expect(screen.getByTestId('quick-chat-messages')).toBeInTheDocument()
    expect(screen.getByText('Summarize the plan')).toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: /quick chat message/i })).toBeNull()
    expect(screen.getByLabelText(/attach file/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^stop$/i)).toBeInTheDocument()
  })

  it('resets back to the input shell when the quick chat reset event is emitted', async () => {
    vi.mocked(AgentBinding.ListSessions).mockResolvedValue({
      sessions: [{
        id: latestConversation.id,
        title: latestConversation.title,
        kind: latestConversation.kind,
        tags: [],
        created: latestConversation.createdAt,
        updated: latestConversation.updatedAt,
        starred: latestConversation.starred,
        status: latestConversation.status,
      }],
      next_cursor: 0,
      has_more: false,
    })

    render(<QuickChatApp />)

    const input = await screen.findByRole('textbox', { name: /quick chat message/i })
    await userEvent.type(input, 'Summarize the plan{Enter}')
    await screen.findByTestId('quick-chat-messages')

    expect(Events.On).toHaveBeenCalledWith('quick_chat:reset', expect.any(Function))
    act(() => {
      runtimeEventHandlers.get('quick_chat:reset')?.()
    })

    const resetInput = await screen.findByRole('textbox', { name: /quick chat message/i })
    expect(resetInput).toHaveValue('')
    expect(resetInput).toHaveFocus()
    expect(screen.queryByTestId('quick-chat-messages')).toBeNull()
  })

  it('closes the quick chat window when Escape is pressed', async () => {
    render(<QuickChatApp />)

    await screen.findByRole('textbox', { name: /quick chat message/i })
    await userEvent.keyboard('{Escape}')

    await waitFor(() => {
      expect(Window.CloseQuickChat).toHaveBeenCalledWith({})
    })
    expect(Window.SelectQuickChat).not.toHaveBeenCalled()
  })

  it('closes the quick chat window when Escape is pressed outside the input', async () => {
    render(<QuickChatApp />)

    const input = await screen.findByRole('textbox', { name: /quick chat message/i })
    input.blur()
    fireEvent.keyDown(window, { key: 'Escape' })

    await waitFor(() => {
      expect(Window.CloseQuickChat).toHaveBeenCalledWith({})
    })
    expect(Window.SelectQuickChat).not.toHaveBeenCalled()
  })
})
