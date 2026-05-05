import React, { useEffect } from 'react';
import { Routes, Route, Navigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Layout from '@/components/layout';
import { initializeStores } from '@/stores';
import { useViewportHeight } from '@/hooks/useViewportHeight';
import ErrorBoundary from '@/components/ErrorBoundary';

const Chat = React.lazy(() => import('@/pages/home'));
const NotFound = React.lazy(() => import('@/pages/common/NotFound.tsx'));
const Settings = React.lazy(() => import('@/pages/settings'));
const Onboarding = React.lazy(() => import('@/pages/onboarding'));
const AddProviderPage = React.lazy(
  () => import('@/pages/forms/AddProviderPage')
);
const AddAgentPage = React.lazy(() => import('@/pages/forms/AddAgentPage'));
const AddSkillPage = React.lazy(() => import('@/pages/forms/AddSkillPage'));
const EditMemoryPage = React.lazy(() => import('@/pages/forms/EditMemoryPage'));
const EmailPluginSettingsPage = React.lazy(() => import('@/pages/forms/EmailPluginSettingsPage'));

function EntryRedirect() {
  const [searchParams] = useSearchParams();
  const entry = searchParams.get('entry');

  const tab = searchParams.get('tab');
  const id = searchParams.get('id');

  switch (entry) {
    case 'settings':
      return (
        <Navigate to={tab ? `/settings?tab=${tab}` : '/settings'} replace />
      );
    case 'onboarding':
      return <Navigate to="/onboarding" replace />;
    case 'form_provider':
      return <Navigate to="/forms/provider" replace />;
    case 'form_agent':
      return <Navigate to="/forms/agent" replace />;
    case 'form_skill':
      return <Navigate to="/forms/skill" replace />;
    case 'form_memory':
      return (
        <Navigate
          to={id ? `/forms/memory?id=${id}` : '/forms/memory'}
          replace
        />
      );
    case 'form_plugin_email':
      return (
        <Navigate
          to={id ? `/forms/plugin-email?id=${id}` : '/forms/plugin-email'}
          replace
        />
      );
    case 'home':
    case null:
      return <Navigate to="/home" replace />;
    default:
      return <Navigate to="/home" replace />;
  }
}

function App() {
  const { t } = useTranslation();

  // 初始化视口高度检测
  useViewportHeight();

  // 初始化所有stores
  useEffect(() => {
    void initializeStores();
  }, []);

  return (
    <ErrorBoundary>
      <React.Suspense
        fallback={
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'center',
              alignItems: 'center',
              height: '100vh',
              gap: '16px',
            }}
          >
            <div style={{ display: 'flex', gap: '4px', alignItems: 'center' }}>
              {[...Array(5)].map((_, i) => (
                <div
                  key={i}
                  style={{
                    width: '4px',
                    height: '30px',
                    background: 'linear-gradient(45deg, #667eea, #764ba2)',
                    borderRadius: '2px',
                    animation: `loading-wave 1.2s ease-in-out infinite ${i * 0.1}s`,
                  }}
                />
              ))}
            </div>
            <span
              style={{ fontSize: '14px', color: '#666', whiteSpace: 'nowrap' }}
            >
              {t('common.loading')}
            </span>
            <style>{`
            @keyframes loading-wave {
              0%, 40%, 100% {
                transform: scaleY(0.4);
                opacity: 0.6;
              }
              20% {
                transform: scaleY(1);
                opacity: 1;
              }
            }
          `}</style>
          </div>
        }
      >
        <Routes>
          {/* 应用入口页 - 根据窗口入口参数分发到对应页面 */}
          <Route path="/" element={<EntryRedirect />} />

          {/* 聊天页面 */}
          <Route path="/home" element={<Chat />} />
          <Route path="/home/:chatUuid" element={<Chat />} />

          <Route path="/settings" element={<Settings />} />
          <Route path="/onboarding" element={<Onboarding />} />

          {/* 独立表单窗口 */}
          <Route path="/forms/provider" element={<AddProviderPage />} />
          <Route path="/forms/agent" element={<AddAgentPage />} />
          <Route path="/forms/skill" element={<AddSkillPage />} />
          <Route path="/forms/memory" element={<EditMemoryPage />} />
          <Route path="/forms/plugin-email" element={<EmailPluginSettingsPage />} />

          {/* 其他路由 - 使用Layout */}
          <Route path="/app" element={<Layout />}></Route>

          {/* 兼容旧链接 */}
          <Route path="/:chatUuid" element={<Chat />} />

          {/* 404页面 */}
          <Route path="*" element={<NotFound />} />
        </Routes>
      </React.Suspense>
    </ErrorBoundary>
  );
}

export default App;
