import React, { useState, useEffect, useMemo } from 'react';
import { Layout, Menu, Button } from 'antd';
import {
  SettingOutlined,
  ApiOutlined,
  InfoCircleOutlined,
  ArrowLeftOutlined,
  FileTextOutlined,
  RobotOutlined,
  ThunderboltOutlined,
  BulbOutlined,
  AppstoreAddOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import ProviderSettingPage from './provider';
import AboutPage from './about';
import GeneralSettingsPage from './general';
import PromptSettingsPage from './prompt';
import AgentSettingsPage from './agents';
import SkillSettingsPage from './skills';
import MemorySettingsPage from './memory';
import PluginSettingsPage from './plugins';
import { useViewportHeight } from '@/hooks/useViewportHeight';
import { initializeFontSize } from '@/stores/fontSizeStore';
import { useLabStore } from '@/stores/labStore';
import styles from './index.module.scss';

const { Sider, Content } = Layout;

interface SettingsPageProps {
  className?: string;
}

const SettingsPage: React.FC<SettingsPageProps> = ({ className }) => {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const validKeys = useMemo(() => ['general', 'memory', 'provider', 'agents', 'skills', 'plugins', 'prompt', 'about'], []);
  const initialTab = useMemo(() => {
    const tab = searchParams.get('tab');
    return tab && validKeys.includes(tab) ? tab : 'general';
  }, [searchParams, validKeys]);
  const [selectedKey, setSelectedKey] = useState(initialTab);
  const [showContent, setShowContent] = useState(false); // 控制移动端内容显示
  const { isMobile } = useViewportHeight(); // 使用移动端检测
  const memorySystemEnabled = useLabStore(state => state.memorySystemEnabled);

  // 监听设备切换，处理从桌面端切换到移动端的情况
  useEffect(() => {
    if (isMobile) {
      // 切换到移动端时，如果当前是从桌面端切换过来，显示当前选中的内容
      // 这样用户不需要重新点击菜单
      setShowContent(true);
    } else {
      // 切换到桌面端时，重置移动端状态
      setShowContent(false);
    }
  }, [isMobile]);

  // 初始化字体大小设置
  useEffect(() => {
    // 初始化字体大小设置
    initializeFontSize();
  }, []);

  useEffect(() => {
    document.title = t('app.settingsTitle');
  }, [t]);

  const menuItems = useMemo(() => {
    const items = [
      {
        key: 'general',
        icon: <SettingOutlined />,
        label: t('settings.menu.general'),
      },
      {
        key: 'provider',
        icon: <ApiOutlined />,
        label: t('settings.menu.provider'),
      },
      {
        key: 'agents',
        icon: <RobotOutlined />,
        label: t('settings.menu.agents'),
      },
      {
        key: 'skills',
        icon: <ThunderboltOutlined />,
        label: t('settings.menu.skills'),
      },
      {
        key: 'plugins',
        icon: <AppstoreAddOutlined />,
        label: t('settings.menu.plugins'),
      },
      {
        key: 'prompt',
        icon: <FileTextOutlined />,
        label: t('settings.menu.prompt'),
      },
      {
        key: 'about',
        icon: <InfoCircleOutlined />,
        label: t('settings.menu.about'),
      },
    ];
    if (memorySystemEnabled) {
      // Insert memory after general
      const generalIdx = items.findIndex(i => i.key === 'general');
      items.splice(generalIdx + 1, 0, {
        key: 'memory',
        icon: <BulbOutlined />,
        label: <span style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flex: 1 }}>
          {t('settings.menu.memory')}
          <span style={{ fontSize: 10, padding: '0 4px', lineHeight: '16px', borderRadius: 4, background: 'rgba(250, 173, 20, 0.15)', color: '#faad14' }}>Lab</span>
        </span>,
      });
    }
    return items;
  }, [t, memorySystemEnabled]);

  const handleMenuClick = ({ key }: { key: string }) => {
    setSelectedKey(key);
    // 移动端点击菜单后显示内容
    if (isMobile) {
      setShowContent(true);
    }
  };

  const handleBackToMenu = () => {
    // 移动端返回菜单
    setShowContent(false);
  };

  // 决定是否显示加载状态 - 已移除，使用App.tsx的统一loading

  const renderContent = () => {
    const content = (() => {
      switch (selectedKey) {
        case 'provider':
          return <ProviderSettingPage />;
        case 'general':
          return <GeneralSettingsPage />;
        case 'memory':
          return <MemorySettingsPage />;
        case 'prompt':
          return <PromptSettingsPage />;
        case 'agents':
          return <AgentSettingsPage />;
        case 'skills':
          return <SkillSettingsPage />;
        case 'plugins':
          return <PluginSettingsPage />;
        case 'account':
          return <div className={styles.placeholder}>{t('settings.placeholders.account')}</div>;
        case 'security':
          return <div className={styles.placeholder}>{t('settings.placeholders.security')}</div>;
        case 'notifications':
          return <div className={styles.placeholder}>{t('settings.placeholders.notifications')}</div>;
        case 'about':
          return <AboutPage />;
        default:
          return <GeneralSettingsPage />;
      }
    })();

    // 移动端在内容顶部添加返回按钮
    if (isMobile) {
      return (
        <div className={styles.mobileContent}>
          <div className={styles.mobileHeader}>
            <Button 
              type="text" 
              icon={<ArrowLeftOutlined />}
              onClick={handleBackToMenu}
              className={styles.backButton}
            >
              {t('settings.back')}
            </Button>
            <span className={styles.mobileTitle}>
              {menuItems.find(item => item.key === selectedKey)?.label}
            </span>
          </div>
          <div className={styles.mobileContentBody}>
            {content}
          </div>
        </div>
      );
    }

    return content;
  };

  return (
    <Layout className={`${styles.settingsLayout} ${className || ''}`}>
      {/* 移动端：根据状态显示菜单或内容 */}
      {isMobile ? (
        <>
          {/* 移动端菜单 */}
          <div className={`${styles.mobileMenu} ${showContent ? styles.hidden : ''}`}>
            <div className={styles.siderHeader}>
              <h3>{t('settings.title')}</h3>
            </div>
            <Menu
              mode="inline"
              selectedKeys={[]} // 移动端不显示选中状态
              items={menuItems}
              onClick={handleMenuClick}
              className={styles.settingsMenu}
            />
          </div>
          
          {/* 移动端内容 */}
          <div className={`${styles.mobileContentContainer} ${showContent ? styles.visible : ''}`}>
            {renderContent()}
          </div>
        </>
      ) : (
        /* 桌面端：正常的侧边栏布局 */
        <>
          <Sider
            width={240}
            className={styles.settingsSider}
            theme="light"
          >
            <div className={styles.siderHeader}>
              <h3>{t('settings.title')}</h3>
            </div>
            <Menu
              mode="inline"
              selectedKeys={[selectedKey]}
              items={menuItems}
              onClick={handleMenuClick}
              className={styles.settingsMenu}
            />
          </Sider>
          <Layout className={styles.settingsContent}>
            <Content className={`${styles.contentArea} ${(selectedKey === 'prompt' || selectedKey === 'provider' || selectedKey === 'agents' || selectedKey === 'skills' || selectedKey === 'general' || selectedKey === 'about') ? styles.contentAreaLocked : ''}`}>
              {renderContent()}
            </Content>
          </Layout>
        </>
      )}
    </Layout>
  );
};

export default SettingsPage;
