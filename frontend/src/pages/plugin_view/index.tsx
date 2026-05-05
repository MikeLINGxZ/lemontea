import React from 'react';
import { Alert } from 'antd';
import { useSearchParams } from 'react-router-dom';
import PluginViewFrame from '@/components/plugin/PluginViewFrame';

const PluginViewPage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const pluginId = searchParams.get('id') || '';
  const viewId = searchParams.get('view') || 'settings';

  if (!pluginId) {
    return (
      <div style={{ padding: 20 }}>
        <Alert
          type="error"
          message="Plugin id is required"
          description="Open this page with both id and view query parameters."
        />
      </div>
    );
  }

  return (
    <div style={{ width: '100vw', height: '100vh', background: '#fff' }}>
      <PluginViewFrame
        pluginId={pluginId}
        viewId={viewId}
        location="settings_window"
      />
    </div>
  );
};

export default PluginViewPage;
