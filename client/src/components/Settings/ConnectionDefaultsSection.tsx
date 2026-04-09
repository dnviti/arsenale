import { useEffect, useState } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useNotificationStore } from '../../store/notificationStore';
import { useRdpSettingsStore } from '../../store/rdpSettingsStore';
import { useTerminalSettingsStore } from '../../store/terminalSettingsStore';
import type { RdpSettings } from '../../constants/rdpDefaults';
import type { SshTerminalConfig } from '../../constants/terminalThemes';
import RdpSettingsSection from './RdpSettingsSection';
import TerminalSettingsSection from './TerminalSettingsSection';
import { SettingsLoadingState, SettingsPanel } from './settings-ui';

export default function ConnectionDefaultsSection() {
  const notify = useNotificationStore((state) => state.notify);
  const updateSshDefaults = useTerminalSettingsStore((state) => state.updateDefaults);
  const sshLoading = useTerminalSettingsStore((state) => state.loading);
  const updateRdpDefaults = useRdpSettingsStore((state) => state.updateDefaults);
  const rdpLoading = useRdpSettingsStore((state) => state.loading);
  const [sshConfig, setSshConfig] = useState<Partial<SshTerminalConfig>>({});
  const [rdpConfig, setRdpConfig] = useState<Partial<RdpSettings>>({});
  const [sshError, setSshError] = useState('');
  const [rdpError, setRdpError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      useTerminalSettingsStore.getState().fetchDefaults().then(() => {
        const defaults = useTerminalSettingsStore.getState().userDefaults;
        if (defaults) {
          setSshConfig(defaults);
        }
      }),
      useRdpSettingsStore.getState().fetchDefaults().then(() => {
        const defaults = useRdpSettingsStore.getState().userDefaults;
        if (defaults) {
          setRdpConfig(defaults);
        }
      }),
    ]).finally(() => setLoading(false));
  }, []);

  const handleSaveSshDefaults = async () => {
    setSshError('');
    try {
      await updateSshDefaults(sshConfig);
      notify('SSH terminal defaults saved', 'success');
    } catch {
      setSshError('Failed to save SSH defaults');
    }
  };

  const handleSaveRdpDefaults = async () => {
    setRdpError('');
    try {
      await updateRdpDefaults(rdpConfig);
      notify('RDP defaults saved', 'success');
    } catch {
      setRdpError('Failed to save RDP defaults');
    }
  };

  if (loading) {
    return (
      <SettingsPanel
        title="Connection Defaults"
        description="Personal defaults for SSH terminal and RDP sessions."
      >
        <SettingsLoadingState message="Loading connection defaults..." />
      </SettingsPanel>
    );
  }

  return (
    <SettingsPanel
      title="Connection Defaults"
      description="These settings apply to new sessions unless a specific connection overrides them."
      contentClassName="space-y-4"
    >
      <Tabs defaultValue="ssh" className="gap-4">
        <TabsList className="w-full justify-start">
          <TabsTrigger value="ssh">SSH Terminal</TabsTrigger>
          <TabsTrigger value="rdp">RDP</TabsTrigger>
        </TabsList>

        <TabsContent value="ssh" className="space-y-4">
          {sshError && (
            <Alert variant="destructive">
              <AlertDescription>{sshError}</AlertDescription>
            </Alert>
          )}
          <TerminalSettingsSection value={sshConfig} onChange={setSshConfig} mode="global" />
          <div className="flex justify-start">
            <Button type="button" onClick={handleSaveSshDefaults} disabled={sshLoading}>
              {sshLoading ? 'Saving...' : 'Save SSH Defaults'}
            </Button>
          </div>
        </TabsContent>

        <TabsContent value="rdp" className="space-y-4">
          {rdpError && (
            <Alert variant="destructive">
              <AlertDescription>{rdpError}</AlertDescription>
            </Alert>
          )}
          <RdpSettingsSection value={rdpConfig} onChange={setRdpConfig} mode="global" />
          <div className="flex justify-start">
            <Button type="button" onClick={handleSaveRdpDefaults} disabled={rdpLoading}>
              {rdpLoading ? 'Saving...' : 'Save RDP Defaults'}
            </Button>
          </div>
        </TabsContent>
      </Tabs>
    </SettingsPanel>
  );
}
