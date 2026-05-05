import React, {
    forwardRef,
    useImperativeHandle,
    useRef,
    useCallback,
    useEffect,
    useLayoutEffect,
    useState,
} from "react";
import styles from "./index.module.scss";
import ChatMessage from "@/components/chat/message";
import type {Message} from "@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_models";

interface MessageListProps {
    // 消息列表
    messages?: Message[];
    // 是否正在生成消息
    isGenerating?: boolean;
    // 初次加载时使用立即滚动（无动画），如历史聊天首次加载
    useInstantScrollOnFirstLoad?: boolean;
    onApprovalDecision?: (approvalId: string, decision: 'allow' | 'reject') => void;
    onSendApprovalComment?: (approvalId: string, comment: string) => Promise<void> | void;
    onReopenPluginView?: (callId: string) => void;
    openPluginViewCallIds?: string[];
    activePluginViewCallId?: string;
}

export interface MessageListRef {
    // 手动滚动到底部（平滑）
    scrollToBottom: () => void;
    // 立即滚动到底部（无动画）
    scrollToBottomInstant: () => void;
    // 检查是否在底部
    isAtBottom: () => boolean;
}

const MessageList: React.ForwardRefRenderFunction<MessageListRef, MessageListProps> = ({
    messages = [],
    isGenerating = false,
    useInstantScrollOnFirstLoad = false,
    onApprovalDecision,
    onSendApprovalComment,
    onReopenPluginView,
    openPluginViewCallIds = [],
    activePluginViewCallId = '',
}, ref) => {
    const containerRef = useRef<HTMLDivElement>(null);
    const contentRef = useRef<HTMLDivElement>(null);
    const buttonRef = useRef<HTMLDivElement>(null);
    const [isAtBottom, setIsAtBottom] = useState(true);
    const [showScrollButton, setShowScrollButton] = useState(false);
    // 是否启用自动滚动
    const [autoScroll, setAutoScroll] = useState(true);
    const autoScrollRef = useRef(true); // 用于在事件处理中获取最新的 autoScroll 状态
    // 用户是否正在手动滚动
    const isUserScrollingRef = useRef(false);
    // 是否初始化滚动
    const isInitialLoadRef = useRef(true);
    // 上次滚动时间
    const lastScrollTimeRef = useRef(0);
    // 上次滚动位置
    const lastScrollTopRef = useRef(0);
    // 滚动到底部定时器
    const scrollTimeoutRef = useRef<NodeJS.Timeout | null>(null);
    // 用户滚动检测定时器
    const userScrollDetectionRef = useRef<NodeJS.Timeout | null>(null);
    // 滚动锁
    const scrollingToBottomLockRef = useRef(false);
    // 上次生成状态
    const lastGeneratingRef = useRef(false);

    // 获取滚动容器
    const getScrollContainer = useCallback(() => {
        if (containerRef.current) {
            return containerRef.current.closest('[class*="chatMessagesContent"]') || 
                   containerRef.current.parentElement;
        }
        return null;
    }, []);

    // 检查是否在底部
    const checkIsAtBottom = useCallback(() => {
        // 如果正在滚动到底部，跳过检查，避免在滚动过程中重新显示按钮
        if (scrollingToBottomLockRef.current) {
            return true;
        }
        const scrollContainer = getScrollContainer();
        
        if (scrollContainer && scrollContainer instanceof HTMLElement) {
            const { scrollTop, scrollHeight, clientHeight } = scrollContainer;
            const threshold = 50; // 距离底部的阈值
            const atBottom = scrollHeight - scrollTop - clientHeight < threshold;
            setIsAtBottom(atBottom);
            setShowScrollButton(!atBottom);
            return atBottom;
        }
        
        // 回退到检查当前容器
        if (containerRef.current) {
            const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
            const threshold = 50;
            const atBottom = scrollHeight - scrollTop - clientHeight < threshold;
            setIsAtBottom(atBottom);
            setShowScrollButton(!atBottom);
            return atBottom;
        }
        return true;
    }, [getScrollContainer]);

    // 滚动到底部（平滑）
    const scrollToBottomSmooth = useCallback(() => {
        const scrollContainer = getScrollContainer();
        
        if (scrollContainer && scrollContainer instanceof HTMLElement) {
            // 设置滚动标记，防止在滚动过程中重新显示按钮
            scrollingToBottomLockRef.current = true;
            
            scrollContainer.scrollTo({
                top: scrollContainer.scrollHeight,
                behavior: 'smooth'
            });
            setIsAtBottom(true);
            setShowScrollButton(false);
            
            // 平滑滚动通常需要 300-500ms，我们等待滚动完成后再清除标记
            setTimeout(() => {
                scrollingToBottomLockRef.current = false;
                // 滚动完成后再次检查底部状态
                checkIsAtBottom();
            }, 600);
            return;
        }
        
        // 回退到当前容器
        if (containerRef.current) {
            scrollingToBottomLockRef.current = true;
            containerRef.current.scrollTo({
                top: containerRef.current.scrollHeight,
                behavior: 'smooth'
            });
            setIsAtBottom(true);
            setShowScrollButton(false);
            
            setTimeout(() => {
                scrollingToBottomLockRef.current = false;
                checkIsAtBottom();
            }, 600);
        }
    }, [checkIsAtBottom, getScrollContainer]);

    // 立即滚动到底部（无动画）
    const scrollToBottomInstant = useCallback(() => {
        const scrollContainer = getScrollContainer();

        if (scrollContainer && scrollContainer instanceof HTMLElement) {
            scrollingToBottomLockRef.current = true;
            scrollContainer.scrollTop = scrollContainer.scrollHeight;
            setIsAtBottom(true);
            setShowScrollButton(false);
            requestAnimationFrame(() => {
                scrollingToBottomLockRef.current = false;
            });
            return;
        }

        if (containerRef.current) {
            scrollingToBottomLockRef.current = true;
            containerRef.current.scrollTop = containerRef.current.scrollHeight;
            setIsAtBottom(true);
            setShowScrollButton(false);
            requestAnimationFrame(() => {
                scrollingToBottomLockRef.current = false;
            });
        }
    }, [getScrollContainer]);

    // 暴露给父组件的方法
    useImperativeHandle(ref, () => ({
        scrollToBottom: scrollToBottomSmooth,
        scrollToBottomInstant,
        isAtBottom: () => {
            const scrollContainer = getScrollContainer();
            
            if (scrollContainer && scrollContainer instanceof HTMLElement) {
                const { scrollTop, scrollHeight, clientHeight } = scrollContainer;
                return scrollHeight - scrollTop - clientHeight < 10;
            }
            
            // 回退到当前容器
            if (containerRef.current) {
                const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
                return scrollHeight - scrollTop - clientHeight < 10;
            }
            return true;
        },
    }), [scrollToBottomSmooth, scrollToBottomInstant, getScrollContainer]);

    // 更新滚动到底部按钮位置，使其相对于内容区域居中，并避免与输入框重叠
    const updateScrollToBottomButtonPosition = useCallback(() => {
        if (!contentRef.current || !buttonRef.current) return;
        
        const button = buttonRef.current;
        
        // 临时禁用过渡动画，避免位置改变时的动画效果
        const originalTransition = button.style.transition;
        button.style.transition = 'none';
        
        const contentRect = contentRef.current.getBoundingClientRect();
        
        // 计算内容区域的中心位置
        const centerX = contentRect.left + contentRect.width / 2;
        button.style.left = `${centerX}px`;
        // 确保 transform 保持，用于居中
        button.style.transform = 'translateX(-50%)';
        
        // 计算输入框高度，确保按钮不重叠
        const chatInput = document.querySelector('[class*="chatInput"]') as HTMLElement;
        if (chatInput) {
            const inputRect = chatInput.getBoundingClientRect();
            const inputHeight = inputRect.height;
            // 按钮距离输入框顶部至少 20px
            const minBottom = inputHeight + 20;
            // 移动端间距较小（只多10px）
            const isMobile = window.innerWidth <= 768;
            const bottom = isMobile ? Math.max(minBottom + 10, 100) : Math.max(minBottom, 120);
            button.style.bottom = `${bottom}px`;
        }
        
        // 使用 flushSync 或强制重排，确保样式立即应用
        void button.offsetHeight; // 强制重排
        
        // 恢复过渡动画（延迟恢复，确保位置更新完成）
        requestAnimationFrame(() => {
            button.style.transition = originalTransition || '';
        });
    }, []);

    // 处理用户滚动开始（通过输入事件检测）
    // 这是最可靠的用户滚动检测方式，因为这些事件只在用户操作时触发
    // 输入事件（wheel, touchstart, touchmove, keydown）是用户操作的直接证据
    const handleUserScrollStart = useCallback(() => {
        // 如果正在生成中，用户开始滚动则立即取消自动滚动
        if (isGenerating && autoScrollRef.current) {
            isUserScrollingRef.current = true;
            autoScrollRef.current = false;
            setAutoScroll(false);
        }
    }, [isGenerating]);

    // 监听滚动事件（监听父容器的滚动）
    useEffect(() => {
        const container = containerRef.current;
        if (!container) return;

        const scrollContainer = getScrollContainer();
        if (!scrollContainer || !(scrollContainer instanceof HTMLElement)) return;

        const handleScroll = () => {
            const currentScrollTop = scrollContainer.scrollTop;
            const lastScrollTop = lastScrollTopRef.current;
            const scrollDiff = currentScrollTop - lastScrollTop;
            
            // 如果正在执行程序化滚动（自动滚动），忽略 scroll 事件中的用户滚动判断
            // 因为程序化滚动也会触发 scroll 事件，导致误判
            if (!scrollingToBottomLockRef.current && autoScrollRef.current) {
                // 只在非程序化滚动期间且自动滚动启用时检测用户滚动
                // 如果滚动方向向上（scrollDiff < 0），说明用户向上滚动，远离底部
                // 或者如果滚动位置变化较大（> 50px），可能是用户快速滚动
                if (isGenerating && (scrollDiff < -10 || Math.abs(scrollDiff) > 50)) {
                    // 用户向上滚动或快速滚动，取消自动滚动
                    isUserScrollingRef.current = true;
                    autoScrollRef.current = false;
                    setAutoScroll(false);
                }
            }
            
            // 检查是否在底部
            const atBottom = checkIsAtBottom();
            
            // 如果用户滚动到底部，恢复自动滚动
            if (atBottom && isUserScrollingRef.current && !scrollingToBottomLockRef.current) {
                // 清除之前的定时器
                if (userScrollDetectionRef.current) {
                    clearTimeout(userScrollDetectionRef.current);
                }
                // 延迟恢复，避免频繁切换
                userScrollDetectionRef.current = setTimeout(() => {
                    isUserScrollingRef.current = false;
                    if (isGenerating) {
                        autoScrollRef.current = true;
                        setAutoScroll(true);
                    }
                }, 150);
            }
            
            lastScrollTopRef.current = currentScrollTop;
        };

        // 监听用户输入事件（用于立即检测用户滚动意图）
        const handleWheel = () => handleUserScrollStart();
        const handleTouchStart = () => handleUserScrollStart();
        const handleTouchMove = () => handleUserScrollStart();
        const handleKeyDown = (e: KeyboardEvent) => {
            // 检测键盘滚动（PageDown, PageUp, ArrowDown, ArrowUp, Space, End, Home）
            if (['PageDown', 'PageUp', 'ArrowDown', 'ArrowUp', ' ', 'End', 'Home'].includes(e.key)) {
                handleUserScrollStart();
            }
        };

        scrollContainer.addEventListener('scroll', handleScroll, { passive: true });
        scrollContainer.addEventListener('wheel', handleWheel, { passive: true });
        scrollContainer.addEventListener('touchstart', handleTouchStart, { passive: true });
        scrollContainer.addEventListener('touchmove', handleTouchMove, { passive: true });
        scrollContainer.addEventListener('keydown', handleKeyDown);
        
        // 初始化检查一次
        checkIsAtBottom();
        lastScrollTopRef.current = scrollContainer.scrollTop;
        
        return () => {
            scrollContainer.removeEventListener('scroll', handleScroll);
            scrollContainer.removeEventListener('wheel', handleWheel);
            scrollContainer.removeEventListener('touchstart', handleTouchStart);
            scrollContainer.removeEventListener('touchmove', handleTouchMove);
            scrollContainer.removeEventListener('keydown', handleKeyDown);
            if (userScrollDetectionRef.current) {
                clearTimeout(userScrollDetectionRef.current);
            }
        };
    }, [checkIsAtBottom, isGenerating, getScrollContainer, handleUserScrollStart]);

    // 使用 useLayoutEffect 确保按钮在显示时立即设置正确位置，避免闪烁
    useLayoutEffect(() => {
        if (!showScrollButton || !buttonRef.current) return;
        // 立即同步设置位置
        updateScrollToBottomButtonPosition();
    }, [showScrollButton, updateScrollToBottomButtonPosition]);

    // 监听窗口大小变化和内容区域变化，更新按钮位置
    useEffect(() => {
        if (!showScrollButton || !buttonRef.current) return;

        const handleResize = () => {
            updateScrollToBottomButtonPosition();
        };

        window.addEventListener('resize', handleResize);
        // 使用 ResizeObserver 监听内容区域大小变化
        let contentResizeObserver: ResizeObserver | null = null;
        if (contentRef.current) {
            contentResizeObserver = new ResizeObserver(() => {
                updateScrollToBottomButtonPosition();
            });
            contentResizeObserver.observe(contentRef.current);
        }

        // 监听输入框大小变化
        const chatInput = document.querySelector('[class*="chatInput"]') as HTMLElement;
        let inputResizeObserver: ResizeObserver | null = null;
        if (chatInput) {
            inputResizeObserver = new ResizeObserver(() => {
                updateScrollToBottomButtonPosition();
            });
            inputResizeObserver.observe(chatInput);
        }

        return () => {
            window.removeEventListener('resize', handleResize);
            if (contentResizeObserver && contentRef.current) {
                contentResizeObserver.unobserve(contentRef.current);
            }
            if (inputResizeObserver && chatInput) {
                inputResizeObserver.unobserve(chatInput);
            }
        };
    }, [showScrollButton, updateScrollToBottomButtonPosition]);

    // 初次加载时自动滚动到底部
    useEffect(() => {
        if (isInitialLoadRef.current && messages.length > 0) {
            isInitialLoadRef.current = false;
            // 延迟一下确保DOM渲染完成
            setTimeout(() => {
                // 历史聊天首次加载使用立即滚动，避免动画延迟
                if (useInstantScrollOnFirstLoad) {
                    scrollToBottomInstant();
                } else {
                    scrollToBottomSmooth();
                }
                autoScrollRef.current = true;
                setAutoScroll(true);
            }, 100);
        }
    }, [messages.length, scrollToBottomSmooth, scrollToBottomInstant, useInstantScrollOnFirstLoad]);

    // 确保 autoScrollRef 与 autoScroll state 保持同步
    useEffect(() => {
        autoScrollRef.current = autoScroll;
    }, [autoScroll]);

    // 检测重新生成消息（生成状态从 false 变为 true）
    useEffect(() => {
        const wasGenerating = lastGeneratingRef.current;
        lastGeneratingRef.current = isGenerating;
        
        // 如果从非生成状态变为生成状态，恢复自动滚动
        if (!wasGenerating && isGenerating) {
            autoScrollRef.current = true;
            setAutoScroll(true);
            isUserScrollingRef.current = false;
        }
    }, [isGenerating]);

    // 监听内容区域高度变化，自动滚动到底部
    // 解决工具调用卡片展开等导致DOM高度变化但messages未变的情况
    useEffect(() => {
        if (!contentRef.current) return;

        const observer = new ResizeObserver(() => {
            if (autoScrollRef.current && isGenerating && !isUserScrollingRef.current) {
                scrollToBottomInstant();
            } else {
                checkIsAtBottom();
            }
        });
        observer.observe(contentRef.current);

        return () => observer.disconnect();
    }, [isGenerating, scrollToBottomInstant]);

    // 消息变化时的自动滚动逻辑
    useLayoutEffect(() => {
        // 跳过初始加载时的滚动（由单独的 effect 处理）
        if (isInitialLoadRef.current) {
            return;
        }

        // 清理之前的滚动定时器
        if (scrollTimeoutRef.current) {
            clearTimeout(scrollTimeoutRef.current);
            scrollTimeoutRef.current = null;
        }

        const currentMessageCount = messages.length;
        
        // 决定是否需要自动滚动
        // 1. 必须启用自动滚动
        // 2. 必须正在生成消息
        // 3. 用户没有手动滚动
        // 4. 有消息存在
        const shouldAutoScroll = autoScrollRef.current && 
                                 isGenerating && 
                                 !isUserScrollingRef.current && 
                                 currentMessageCount > 0;

        if (shouldAutoScroll) {
            // 生成过程中使用即时滚动，否则流式内容增长时 smooth 动画无法跟上最新底部
            const now = Date.now();
            const timeSinceLastScroll = now - lastScrollTimeRef.current;
            const scrollDelay = timeSinceLastScroll < 16 ? 16 - timeSinceLastScroll : 0;

            scrollTimeoutRef.current = setTimeout(() => {
                if (autoScrollRef.current && isGenerating && !isUserScrollingRef.current) {
                    scrollToBottomInstant();
                    lastScrollTimeRef.current = Date.now();
                }
            }, scrollDelay);
        }
        
        // 清理函数
        return () => {
            if (scrollTimeoutRef.current) {
                clearTimeout(scrollTimeoutRef.current);
                scrollTimeoutRef.current = null;
            }
        };
    }, [messages, autoScroll, isGenerating, scrollToBottomInstant]);

    return (
        <>
            <div ref={containerRef} className={`${styles.MessageList}`}>
                {/* 消息 */}
                <div ref={contentRef} className={styles.content}>
                    {
                        messages.map((message: Message, index: number) => (
                            <div key={index}>
                                <ChatMessage
                                    message={message}
                                    isLoading={isGenerating && index === messages.length - 1 && message.role !== 'user'}
                                    onApprovalDecision={onApprovalDecision}
                                    onSendApprovalComment={onSendApprovalComment}
                                    onReopenPluginView={onReopenPluginView}
                                    openPluginViewCallIds={openPluginViewCallIds}
                                    activePluginViewCallId={activePluginViewCallId}
                                />
                            </div>
                        ))
                    }
                </div>
            </div>
            {/* 滚动到底部按钮 - 使用固定定位，不跟随滚动 */}
            {showScrollButton && (
                <div 
                    ref={buttonRef}
                    className={styles.scrollToBottomButton} 
                    onClick={() => {
                        // 生成中用即时滚动，否则 smooth 动画期间新内容会让我们无法跟到底部
                        (isGenerating ? scrollToBottomInstant : scrollToBottomSmooth)();
                        // 点击按钮后恢复自动滚动
                        autoScrollRef.current = true;
                        setAutoScroll(true);
                        isUserScrollingRef.current = false;
                    }}
                >
                    <svg
                        width="20"
                        height="20"
                        viewBox="0 0 24 24"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                    >
                        <path
                            d="M7 13L12 18L17 13"
                            stroke="currentColor"
                            strokeWidth="2"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                        />
                        <path
                            d="M7 6L12 11L17 6"
                            stroke="currentColor"
                            strokeWidth="2"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                        />
                    </svg>
                </div>
            )}
        </>
    )
};

MessageList.displayName = 'MessageList';

export default forwardRef(MessageList);
