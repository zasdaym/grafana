import React, { useCallback, useMemo, useState } from 'react';

import { Drawer, Tab, TabsBar } from '@grafana/ui';

import { GRAFANA_RULES_SOURCE_NAME } from '../../utils/datasource';

import AlertmanagerConfig from './AlertmanagerConfig';
import { useSettings } from './SettingsContext';
import { AlertmanagerConfigurationVersionManager } from './VersionManager';

type ActiveTab = 'configuration' | 'versions';

export function useEditConfigurationDrawer(): [React.ReactNode, (dataSourceName: string) => void, () => void] {
  const [activeTab, setActiveTab] = useState<ActiveTab>('configuration');
  const [dataSourceName, setDataSourceName] = useState<string | undefined>();
  const [open, setOpen] = useState(false);
  const { updateAlertmanagerSettings, resetAlertmanagerSettings } = useSettings();

  const showConfiguration = useCallback((dataSourceName: string) => {
    setDataSourceName(dataSourceName);
    setOpen(true);
  }, []);

  const handleDismiss = useCallback(() => {
    setActiveTab('configuration');
    setOpen(false);
  }, []);

  const drawer = useMemo(() => {
    if (!open) {
      return null;
    }

    const handleReset = (uid: string) => {
      resetAlertmanagerSettings(uid);
    };

    const isGrafanaAlertmanager = dataSourceName === GRAFANA_RULES_SOURCE_NAME;
    const title = isGrafanaAlertmanager ? 'Internal Grafana Alertmanager' : dataSourceName;

    // @todo check copy
    return (
      <Drawer
        onClose={handleDismiss}
        title={title}
        subtitle="Edit the Alertmanager configuration"
        size="lg"
        tabs={
          <TabsBar>
            <Tab
              label="JSON Model"
              key="configuration"
              icon="arrow"
              active={activeTab === 'configuration'}
              onChangeTab={() => setActiveTab('configuration')}
            />
            <Tab
              label="Versions"
              key="versions"
              icon="history"
              active={activeTab === 'versions'}
              onChangeTab={() => setActiveTab('versions')}
              hidden={!isGrafanaAlertmanager}
            />
          </TabsBar>
        }
      >
        {activeTab === 'configuration' && dataSourceName && (
          <AlertmanagerConfig
            alertmanagerName={dataSourceName}
            onDismiss={handleDismiss}
            onSave={updateAlertmanagerSettings}
            onReset={handleReset}
          />
        )}
        {activeTab === 'versions' && dataSourceName && (
          <AlertmanagerConfigurationVersionManager alertmanagerName={dataSourceName} />
        )}
      </Drawer>
    );
  }, [open, dataSourceName, handleDismiss, activeTab, updateAlertmanagerSettings, resetAlertmanagerSettings]);

  return [drawer, showConfiguration, handleDismiss];
}
