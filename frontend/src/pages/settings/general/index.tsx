import React, { useState } from 'react';
import { Button, Card, Input, message, Select, Slider, Switch, Typography } from 'antd';
import {
  CheckOutlined,
  ExperimentOutlined,
  FontSizeOutlined,
  GlobalOutlined,
  EnvironmentOutlined,
  ReloadOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { isMobileDevice } from '@/hooks/useViewportHeight';
import { useTranslation } from 'react-i18next';
import { useFontSizeStore, FONT_SIZE_OPTIONS, FONT_SIZE_OFFSETS } from '@/stores/fontSizeStore';
import { useLanguageStore } from '@/stores/languageStore';
import { useLabStore } from '@/stores/labStore';
import { Service } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/service';
import { AppPreferences } from '@bindings/gitlab.linhf.cn/project/lemontea/lemon_tea_desktop/backend/models/view_models';
import { translateError } from '@/utils/errorHandler';
import { LANGUAGE_OPTIONS, REGION_OPTIONS } from '@/i18n/types';
import type { AppLanguage, AppRegion } from '@/i18n/types';
import styles from './index.module.scss';

const { Title, Text } = Typography;

type SettingSection = 'display' | 'language-region' | 'lab';

const GeneralSettingsPage: React.FC = () => {
  const { t } = useTranslation();
  const [activeSection, setActiveSection] = useState<SettingSection>('display');
  const [isMobile, setIsMobile] = useState(() => isMobileDevice());
  const [showDetailOnMobile, setShowDetailOnMobile] = useState(false);

  React.useEffect(() => {
    const handleResize = () => setIsMobile(isMobileDevice());
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const sections: { key: SettingSection; title: string; icon: React.ReactNode }[] = [
    { key: 'display', title: t('settings.general.menuDisplay'), icon: <FontSizeOutlined /> },
    { key: 'language-region', title: t('settings.general.menuLanguageRegion'), icon: <GlobalOutlined /> },
    { key: 'lab', title: t('settings.general.menuLab'), icon: <ExperimentOutlined /> },
  ];

  const handleSelectSection = (key: SettingSection) => {
    setActiveSection(key);
    if (isMobile) setShowDetailOnMobile(true);
  };

  const renderSectionList = () => (
    <Card className={styles.listCard} title={t('settings.general.listTitle')}>
      <div className={styles.sectionList}>
        {sections.map(section => (
          <button
            key={section.key}
            type="button"
            className={`${styles.sectionItem} ${activeSection === section.key ? styles.selected : ''}`}
            onClick={() => handleSelectSection(section.key)}
          >
            <span className={styles.sectionIcon}>{section.icon}</span>
            <span className={styles.sectionTitle}>{section.title}</span>
          </button>
        ))}
      </div>
    </Card>
  );

  const renderDetail = () => (
    <Card className={styles.detailCard}>
      {activeSection === 'display' && <DisplaySettings />}
      {activeSection === 'language-region' && <LanguageRegionSettings />}
      {activeSection === 'lab' && <LabSettings />}
    </Card>
  );

  return (
    <div className={styles.generalSettings}>
      {isMobile ? (
        <>
          {!showDetailOnMobile && renderSectionList()}
          {showDetailOnMobile && (
            <div className={styles.mobileDetail}>
              <Button
                type="text"
                className={styles.mobileBackButton}
                onClick={() => setShowDetailOnMobile(false)}
              >
                {t('settings.back')}
              </Button>
              {renderDetail()}
            </div>
          )}
        </>
      ) : (
        <div className={styles.desktopLayout}>
          <div className={styles.listColumn}>{renderSectionList()}</div>
          <div className={styles.detailColumn}>{renderDetail()}</div>
        </div>
      )}
    </div>
  );
};

// ---- Display Settings ----

const DisplaySettings: React.FC = () => {
  const { t } = useTranslation();
  const { fontSizeOffset, setFontSizeOffset } = useFontSizeStore();
  const [previewOffset, setPreviewOffset] = useState(fontSizeOffset);
  const [hasChanges, setHasChanges] = useState(false);

  const handlePreviewChange = (value: number) => {
    setPreviewOffset(value as any);
    setHasChanges(value !== fontSizeOffset);
  };

  const handleApply = () => {
    setFontSizeOffset(previewOffset as any);
    setHasChanges(false);
    message.success(t('settings.general.applied'));
  };

  const handleReset = () => {
    setPreviewOffset(FONT_SIZE_OFFSETS.NORMAL);
    setHasChanges(FONT_SIZE_OFFSETS.NORMAL !== fontSizeOffset);
  };

  const getFontSizeLabel = (offset: number) => t(`settings.general.fontSizes.${offset}`);

  const sliderMarks = FONT_SIZE_OPTIONS.reduce((marks, option) => {
    marks[option.value] = {
      style: { fontSize: '11px', color: 'var(--text-color-secondary)' },
      label: getFontSizeLabel(option.value),
    };
    return marks;
  }, {} as any);

  return (
    <div className={styles.settingContent}>
      <div className={styles.settingHeader}>
        <Title level={4}>{t('settings.general.displayTitle')}</Title>
        <Text type="secondary">{t('settings.general.fontSizeDescription')}</Text>
      </div>

      <div className={styles.sliderSection}>
        <div className={styles.sliderHeader}>
          <Text strong>{t('settings.general.currentSize')}</Text>
          <div className={styles.currentSize}>
            <Text strong>{getFontSizeLabel(previewOffset)}</Text>
            <Text type="secondary">{14 + previewOffset}px</Text>
          </div>
        </div>
        <Slider
          min={FONT_SIZE_OFFSETS.VERY_SMALL}
          max={FONT_SIZE_OFFSETS.EXTRA_LARGE}
          step={2}
          value={previewOffset}
          onChange={handlePreviewChange}
          marks={sliderMarks}
        />
      </div>

      <div className={styles.presetSection}>
        <Text strong>{t('settings.general.quickSelect')}</Text>
        <div className={styles.presetButtons}>
          {FONT_SIZE_OPTIONS.map(option => (
            <button
              key={option.value}
              className={`${styles.presetButton} ${previewOffset === option.value ? styles.active : ''}`}
              onClick={() => handlePreviewChange(option.value)}
            >
              <span className={styles.buttonLabel}>{getFontSizeLabel(option.value)}</span>
              <span className={styles.buttonSize}>{option.description}</span>
            </button>
          ))}
        </div>
      </div>

      <div className={styles.previewArea}>
        <div className={styles.previewHeader}>
          <FontSizeOutlined />
          <Text strong>{t('settings.general.previewTitle')}</Text>
        </div>
        <div
          className={styles.previewContent}
          style={{
            fontSize: `${14 + previewOffset}px`,
            lineHeight: 1.5715 + (previewOffset > 0 ? -0.05 : previewOffset < 0 ? 0.05 : 0),
          }}
        >
          <div>{t('settings.general.previewText')}</div>
          <div style={{ fontSize: `${12 + previewOffset}px` }}>{t('settings.general.previewSmall')}</div>
          <div style={{ fontWeight: 600 }}>{t('settings.general.previewBold')}</div>
          <div style={{ opacity: 0.65 }}>{t('settings.general.previewSecondary')}</div>
        </div>
      </div>

      <div className={styles.actions}>
        <Button icon={<ReloadOutlined />} onClick={handleReset} disabled={previewOffset === FONT_SIZE_OFFSETS.NORMAL}>
          {t('settings.general.reset')}
        </Button>
        <Button type="primary" icon={<CheckOutlined />} onClick={handleApply} disabled={!hasChanges}>
          {t('settings.general.apply')}
        </Button>
      </div>
    </div>
  );
};

// ---- Language & Region Settings ----

const LanguageRegionSettings: React.FC = () => {
  const { t } = useTranslation();
  const { language, region, setLanguage, setRegion } = useLanguageStore();

  const handleLanguageChange = async (nextLanguage: AppLanguage) => {
    try {
      await setLanguage(nextLanguage);
      void message.success(t('settings.languageRegion.languageChanged'));
    } catch (error) {
      void message.error(translateError(error));
    }
  };

  const handleRegionChange = async (nextRegion: AppRegion) => {
    try {
      await setRegion(nextRegion);
      void message.success(t('settings.languageRegion.regionChanged'));
    } catch (error) {
      void message.error(translateError(error));
    }
  };

  return (
    <div className={styles.settingContent}>
      <div className={styles.settingHeader}>
        <Title level={4}>{t('settings.languageRegion.title')}</Title>
        <Text type="secondary">{t('settings.languageRegion.description')}</Text>
      </div>

      <div className={styles.formSection}>
        <div className={styles.formItem}>
          <div className={styles.formLabel}>
            <div className={styles.formLabelIcon}>
              <GlobalOutlined />
              <Text strong>{t('settings.languageRegion.languageLabel')}</Text>
            </div>
            <Text type="secondary">{t('settings.languageRegion.languageDescription')}</Text>
          </div>
          <Select
            value={language}
            onChange={handleLanguageChange}
            options={LANGUAGE_OPTIONS.map(option => ({
              value: option.value,
              label: option.nativeLabel,
            }))}
            style={{ width: '100%', maxWidth: 320 }}
          />
          <Text type="secondary" className={styles.formHint}>
            {t('settings.languageRegion.languageHint')}
          </Text>
        </div>

        <div className={styles.formItem}>
          <div className={styles.formLabel}>
            <div className={styles.formLabelIcon}>
              <EnvironmentOutlined />
              <Text strong>{t('settings.languageRegion.regionLabel')}</Text>
            </div>
            <Text type="secondary">{t('settings.languageRegion.regionDescription')}</Text>
          </div>
          <Select
            value={region}
            onChange={handleRegionChange}
            options={REGION_OPTIONS.map(option => ({
              value: option.value,
              label: t(option.labelKey),
            }))}
            style={{ width: '100%', maxWidth: 320 }}
          />
          <Text type="secondary" className={styles.formHint}>
            {t('settings.languageRegion.regionHint')}
          </Text>
        </div>
      </div>
    </div>
  );
};

// ---- Lab Settings ----

const LabSettings: React.FC = () => {
  const { t } = useTranslation();
  const {
    memorySystemEnabled, setMemorySystemEnabled,
    vectorSearchEnabled, setVectorSearchEnabled,
    embeddingConfig, setEmbeddingConfig,
  } = useLabStore();
  const [expandedSettings, setExpandedSettings] = useState<Record<string, boolean>>({});

  const toggleSettings = (key: string) => {
    setExpandedSettings(prev => ({ ...prev, [key]: !prev[key] }));
  };

  const saveLabPreferences = async (nextState: {
    memorySystemEnabled: boolean;
    vectorSearchEnabled: boolean;
    embeddingConfig: typeof embeddingConfig;
  }) => {
    const current = await Service.GetAppPreferences();
    await Service.UpdateAppPreferences(new AppPreferences({
      ...current,
      memory_system_enabled: nextState.memorySystemEnabled,
      vector_search_enabled: nextState.vectorSearchEnabled,
      embedding_provider: nextState.embeddingConfig.provider,
      embedding_base_url: nextState.embeddingConfig.baseUrl,
      embedding_api_key: nextState.embeddingConfig.apiKey,
      embedding_model: nextState.embeddingConfig.model,
    }));
  };

  const handleMemorySystemToggle = async (enabled: boolean) => {
    const previousMemory = memorySystemEnabled;
    const previousVector = vectorSearchEnabled;
    const nextVector = enabled ? vectorSearchEnabled : false;

    setMemorySystemEnabled(enabled);
    if (!enabled && vectorSearchEnabled) {
      setVectorSearchEnabled(false);
    }

    try {
      await saveLabPreferences({
        memorySystemEnabled: enabled,
        vectorSearchEnabled: nextVector,
        embeddingConfig,
      });
      if (!enabled && previousVector) {
        await Service.DisableEmbedding();
      }
    } catch (error) {
      setMemorySystemEnabled(previousMemory);
      setVectorSearchEnabled(previousVector);
      message.error(translateError(error, t('settings.saveFailed')));
    }
  };

  const handleVectorSearchToggle = async (enabled: boolean) => {
    const previousEnabled = vectorSearchEnabled;
    setVectorSearchEnabled(enabled);
    if (enabled) {
      setExpandedSettings(prev => ({ ...prev, vectorSearch: true }));
      try {
        await saveLabPreferences({
          memorySystemEnabled,
          vectorSearchEnabled: true,
          embeddingConfig,
        });
        await Service.ConfigureEmbedding(
          embeddingConfig.provider,
          embeddingConfig.baseUrl,
          embeddingConfig.apiKey,
          embeddingConfig.model,
        );
        message.success(t('settings.saved'));
      } catch (e) {
        console.error('ConfigureEmbedding failed:', e);
        setVectorSearchEnabled(previousEnabled);
        await saveLabPreferences({
          memorySystemEnabled,
          vectorSearchEnabled: previousEnabled,
          embeddingConfig,
        }).catch((persistError) => console.error('Failed to rollback vector search preference:', persistError));
        message.error(translateError(e, t('settings.saveFailed')));
      }
    } else {
      try {
        await saveLabPreferences({
          memorySystemEnabled,
          vectorSearchEnabled: false,
          embeddingConfig,
        });
        await Service.DisableEmbedding();
        message.success(t('settings.saved'));
      } catch (e) {
        setVectorSearchEnabled(previousEnabled);
        message.error(translateError(e, t('settings.saveFailed')));
      }
    }
  };

  const handleEmbeddingConfigSave = async () => {
    if (!vectorSearchEnabled) return;
    try {
      await saveLabPreferences({
        memorySystemEnabled,
        vectorSearchEnabled,
        embeddingConfig,
      });
      await Service.ConfigureEmbedding(
        embeddingConfig.provider,
        embeddingConfig.baseUrl,
        embeddingConfig.apiKey,
        embeddingConfig.model,
      );
      message.success(t('settings.saved'));
    } catch (e) {
      console.error('ConfigureEmbedding failed:', e);
    }
  };

  return (
    <div className={styles.settingContent}>
      <div className={styles.settingHeader}>
        <Title level={4}>{t('settings.general.labTitle')}</Title>
        <Text type="secondary">{t('settings.general.labDescription')}</Text>
      </div>

      <div className={styles.formSection}>
        {/* 记忆系统 */}
        <div className={styles.labCard}>
          <div className={styles.labCardHeader}>
            <div className={styles.labItemInfo}>
              <div className={styles.formLabelIcon}>
                <ExperimentOutlined />
                <Text strong>{t('settings.general.labMemorySystem')}</Text>
              </div>
              <Text type="secondary" className={styles.labItemDesc}>
                {t('settings.general.labMemorySystemDesc')}
              </Text>
            </div>
            <Switch
              checked={memorySystemEnabled}
              onChange={handleMemorySystemToggle}
            />
          </div>
        </div>

        {/* 向量搜索（记忆系统启用时显示） */}
        {memorySystemEnabled && (
          <div className={styles.labCard}>
            <div className={styles.labCardHeader}>
              <div className={styles.labItemInfo}>
                <div className={styles.formLabelIcon}>
                  <ExperimentOutlined />
                  <Text strong>{t('settings.general.labVectorSearch')}</Text>
                </div>
                <Text type="secondary" className={styles.labItemDesc}>
                  {t('settings.general.labVectorSearchDesc')}
                </Text>
              </div>
              <div className={styles.labCardActions}>
                {vectorSearchEnabled && (
                  <Button
                    type="text"
                    size="small"
                    icon={<SettingOutlined />}
                    className={styles.labSettingsBtn}
                    onClick={() => toggleSettings('vectorSearch')}
                  />
                )}
                <Switch
                  checked={vectorSearchEnabled}
                  onChange={handleVectorSearchToggle}
                />
              </div>
            </div>
            {vectorSearchEnabled && expandedSettings['vectorSearch'] && (
              <div className={styles.labCardBody}>
                <div className={styles.embeddingGrid}>
                  <div className={styles.embeddingField}>
                    <Text type="secondary" className={styles.embeddingFieldLabel}>{t('settings.general.embeddingProvider')}</Text>
                    <Select
                      value={embeddingConfig.provider}
                      onChange={(v) => setEmbeddingConfig({ provider: v })}
                      size="small"
                      options={[
                        { value: 'ollama', label: 'Ollama' },
                        { value: 'openai_compat', label: t('settings.general.embeddingOpenAICompat') },
                      ]}
                    />
                  </div>
                  <div className={styles.embeddingField}>
                    <Text type="secondary" className={styles.embeddingFieldLabel}>{t('settings.general.embeddingModel')}</Text>
                    <Input
                      value={embeddingConfig.model}
                      onChange={(e) => setEmbeddingConfig({ model: e.target.value })}
                      placeholder={embeddingConfig.provider === 'ollama' ? 'bge-m3' : 'text-embedding-3-small'}
                      size="small"
                    />
                  </div>
                </div>
                <div className={styles.embeddingField}>
                  <Text type="secondary" className={styles.embeddingFieldLabel}>{t('settings.general.embeddingBaseURL')}</Text>
                  <Input
                    value={embeddingConfig.baseUrl}
                    onChange={(e) => setEmbeddingConfig({ baseUrl: e.target.value })}
                    placeholder={embeddingConfig.provider === 'ollama' ? 'http://localhost:11434' : 'https://api.example.com/v1'}
                    size="small"
                  />
                </div>
                {embeddingConfig.provider === 'openai_compat' && (
                  <div className={styles.embeddingField}>
                    <Text type="secondary" className={styles.embeddingFieldLabel}>API Key</Text>
                    <Input.Password
                      value={embeddingConfig.apiKey}
                      onChange={(e) => setEmbeddingConfig({ apiKey: e.target.value })}
                      placeholder="sk-..."
                      size="small"
                    />
                  </div>
                )}
                <div className={styles.embeddingActions}>
                  <Button type="primary" size="small" onClick={handleEmbeddingConfigSave}>
                    {t('settings.general.embeddingSave')}
                  </Button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default GeneralSettingsPage;
