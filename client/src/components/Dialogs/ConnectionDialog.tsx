import { useState, useEffect } from 'react';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
  DialogDescription, DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Switch } from '@/components/ui/switch';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Select, SelectTrigger, SelectValue, SelectContent, SelectItem,
} from '@/components/ui/select';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { cn } from '@/lib/utils';
import { Keyboard, KeyRound, Cloud, X } from 'lucide-react';
import { createConnection, updateConnection, ConnectionInput, ConnectionUpdate, ConnectionData, DlpPolicy, DbSettings, DbProtocol, OracleConnectionType, OracleRole } from '../../api/connections.api';
import { useConnectionsStore } from '../../store/connectionsStore';
import type { SshTerminalConfig } from '../../constants/terminalThemes';
import { mergeTerminalConfig } from '../../constants/terminalThemes';
import { useTerminalSettingsStore } from '../../store/terminalSettingsStore';
import TerminalSettingsSection from '../Settings/TerminalSettingsSection';
import type { RdpSettings } from '../../constants/rdpDefaults';
import { mergeRdpConfig } from '../../constants/rdpDefaults';
import { useRdpSettingsStore } from '../../store/rdpSettingsStore';
import RdpSettingsSection from '../Settings/RdpSettingsSection';
import type { VncSettings } from '../../constants/vncDefaults';
import { mergeVncConfig } from '../../constants/vncDefaults';
import VncSettingsSection from '../Settings/VncSettingsSection';
import { useGatewayStore } from '../../store/gatewayStore';
import { useAuthStore } from '../../store/authStore';
import { useTenantStore } from '../../store/tenantStore';
import SecretPicker from '../Keychain/SecretPicker';
import { useVaultStore } from '../../store/vaultStore';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import { useFeatureFlagsStore } from '../../store/featureFlagsStore';
import { listVaultProviders, VaultProviderData } from '../../api/externalVault.api';
import { gatewayEndpointLabel } from '../../utils/gatewayMode';
import {
  cloudProviderHint,
  nextSSLModeForCloudProvider,
  normalizeCloudProviderSelection,
  remapSSLModeOnProtocolChange,
  supportsCloudProviderPresets,
  tlsModeOptions,
} from '../../utils/dbConnectionSecurity';

interface ConnectionDialogProps {
  open: boolean;
  onClose: () => void;
  connection?: ConnectionData | null;
  folderId?: string | null;
  teamId?: string | null;
}

function supportsPersistedExecutionPlans(protocol?: DbProtocol): boolean {
  return protocol === 'postgresql' || protocol === 'mysql' || protocol === 'oracle' || protocol === 'mssql';
}

function normalizeDbSettings(settings: Partial<DbSettings>): DbSettings | null {
  if (!settings.protocol) {
    return null;
  }

  const supportsCloudPresets = supportsCloudProviderPresets(settings.protocol);

  return {
    ...settings,
    protocol: settings.protocol,
    cloudProvider: supportsCloudPresets ? settings.cloudProvider : undefined,
    sslMode: settings.sslMode,
    persistExecutionPlan: supportsPersistedExecutionPlans(settings.protocol)
      ? settings.persistExecutionPlan
      : undefined,
  };
}

export default function ConnectionDialog({ open, onClose, connection, folderId, teamId }: ConnectionDialogProps) {
  const databaseProxyEnabled = useFeatureFlagsStore((s) => s.databaseProxyEnabled);
  const connectionsEnabled = useFeatureFlagsStore((s) => s.connectionsEnabled);
  const [name, setName] = useState('');
  const [type, setType] = useState<'SSH' | 'RDP' | 'VNC' | 'DATABASE' | 'DB_TUNNEL'>('SSH');
  const [host, setHost] = useState('');
  const [port, setPort] = useState('22');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [domain, setDomain] = useState('');
  const [description, setDescription] = useState('');
  const [enableDrive, setEnableDrive] = useState(false);
  const [sshTerminalConfig, setSshTerminalConfig] = useState<Partial<SshTerminalConfig>>({});
  const [rdpSettings, setRdpSettings] = useState<Partial<RdpSettings>>({});
  const [vncSettings, setVncSettings] = useState<Partial<VncSettings>>({});
  const [dbSettings, setDbSettings] = useState<Partial<DbSettings>>({});
  const [gatewayId, setGatewayId] = useState('');
  const [credentialMode, setCredentialMode] = useState<'manual' | 'keychain' | 'external-vault'>('manual');
  const [selectedSecretId, setSelectedSecretId] = useState<string | null>(null);
  const [selectedVaultProviderId, setSelectedVaultProviderId] = useState<string | null>(null);
  const [vaultSecretPath, setVaultSecretPath] = useState('');
  const [vaultProviders, setVaultProviders] = useState<VaultProviderData[]>([]);
  const [defaultConnectMode, setDefaultConnectMode] = useState<string>('');
  const [dlpPolicy, setDlpPolicy] = useState<DlpPolicy>({});
  const [targetDbHost, setTargetDbHost] = useState('');
  const [targetDbPort, setTargetDbPort] = useState('');
  const [dbType, setDbType] = useState('');
  const [activeSection, setActiveSection] = useState<string>('general');
  const { loading, error, setError, clearError, run } = useAsyncAction();
  const fetchConnections = useConnectionsStore((s) => s.fetchConnections);
  const userDefaults = useTerminalSettingsStore((s) => s.userDefaults);
  const rdpUserDefaults = useRdpSettingsStore((s) => s.userDefaults);
  const gateways = useGatewayStore((s) => s.gateways);
  const fetchGateways = useGatewayStore((s) => s.fetchGateways);
  const hasTenant = Boolean(useAuthStore((s) => s.user)?.tenantId);
  const tenantEnforced = useTenantStore((s) => s.tenant?.enforcedConnectionSettings);
  const vaultUnlocked = useVaultStore((s) => s.unlocked);

  const isEditMode = Boolean(connection);

  useEffect(() => {
    if (open && hasTenant) {
      fetchGateways();
      listVaultProviders().then(setVaultProviders).catch(() => setVaultProviders([]));
    }
    if (open && connection) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- resetting form state when dialog opens is intentional
      setName(connection.name);
      setType(connection.type);
      setHost(connection.host);
      setPort(String(connection.port));
      setUsername('');
      setPassword('');
      setDomain('');
      setDescription(connection.description || '');
      setEnableDrive(connection.enableDrive ?? false);
      setGatewayId(connection.gatewayId || '');
      setSshTerminalConfig(
        (connection.sshTerminalConfig as Partial<SshTerminalConfig>) ?? {}
      );
      setRdpSettings(
        (connection.rdpSettings as Partial<RdpSettings>) ?? {}
      );
      setVncSettings(
        (connection.vncSettings as Partial<VncSettings>) ?? {}
      );
      setDbSettings(
        (connection.dbSettings as Partial<DbSettings>) ?? {}
      );
      if (connection.externalVaultProviderId) {
        setCredentialMode('external-vault');
        setSelectedVaultProviderId(connection.externalVaultProviderId);
        setVaultSecretPath(connection.externalVaultPath ?? '');
        setSelectedSecretId(null);
      } else if (connection.credentialSecretId) {
        setCredentialMode('keychain');
        setSelectedSecretId(connection.credentialSecretId);
        setSelectedVaultProviderId(null);
        setVaultSecretPath('');
      } else {
        setCredentialMode('manual');
        setSelectedSecretId(null);
        setSelectedVaultProviderId(null);
        setVaultSecretPath('');
      }
      setDefaultConnectMode(connection.defaultCredentialMode ?? '');
      setDlpPolicy((connection.dlpPolicy as DlpPolicy) ?? {});
      setTargetDbHost((connection as ConnectionData & { targetDbHost?: string }).targetDbHost ?? '');
      setTargetDbPort((connection as ConnectionData & { targetDbPort?: number }).targetDbPort?.toString() ?? '');
      setDbType((connection as ConnectionData & { dbType?: string }).dbType ?? '');
    } else if (open && !connection) {
      setName('');
      setType('SSH');
      setHost('');
      setPort('22');
      setUsername('');
      setPassword('');
      setDomain('');
      setDescription('');
      setEnableDrive(false);
      setGatewayId('');
      setSshTerminalConfig({});
      setRdpSettings({});
      setVncSettings({});
      setDbSettings({});
      setCredentialMode('manual');
      setSelectedSecretId(null);
      setSelectedVaultProviderId(null);
      setVaultSecretPath('');
      setDefaultConnectMode('');
      setDlpPolicy({});
      setTargetDbHost('');
      setTargetDbPort('');
      setDbType('');
    }
  }, [open, connection, fetchGateways, hasTenant]);

  const handleTypeChange = (newType: 'SSH' | 'RDP' | 'VNC' | 'DATABASE' | 'DB_TUNNEL') => {
    setType(newType);
    const knownPorts = ['22', '3389', '5900', '5432', '3306', '27017', '1521', '1433', '50000'];
    if (newType === 'SSH' && knownPorts.includes(port)) setPort('22');
    if (newType === 'RDP' && knownPorts.includes(port)) setPort('3389');
    if (newType === 'VNC' && knownPorts.includes(port)) setPort('5900');
    if (newType === 'DATABASE' && knownPorts.includes(port)) setPort('5432');
    if (newType === 'DB_TUNNEL' && knownPorts.includes(port)) setPort('22');
    setGatewayId('');
    setActiveSection('general');
    if (newType === 'DATABASE') {
      setDbSettings((prev) => ({ protocol: 'postgresql', ...prev }));
    }
  };

  const availableGateways = gateways.filter((g) => {
    if (type === 'SSH' || type === 'DB_TUNNEL') return g.type === 'SSH_BASTION' || g.type === 'MANAGED_SSH';
    if (type === 'RDP' || type === 'VNC') return g.type === 'GUACD';
    if (type === 'DATABASE') return g.type === 'DB_PROXY';
    return false;
  });
  const dbTLSOptions = tlsModeOptions(dbSettings.protocol);
  const currentDbTLSOption = dbTLSOptions.find((option) => option.value === (dbSettings.sslMode || '__default__'))
    ?? dbTLSOptions[0];
  const dbCloudHint = cloudProviderHint(dbSettings.protocol, dbSettings.cloudProvider);

  const sections: { key: string; label: string }[] = [
    { key: 'general', label: 'General' },
    { key: 'credentials', label: 'Credentials' },
    { key: 'options', label: 'Options' },
    ...(type === 'SSH' ? [{ key: 'terminal', label: 'Terminal' }] : []),
    ...(type === 'RDP' ? [{ key: 'rdp', label: 'RDP Settings' }] : []),
    ...(type === 'VNC' ? [{ key: 'vnc', label: 'VNC Settings' }] : []),
    ...(type === 'DATABASE' ? [{ key: 'database', label: 'Database' }] : []),
    ...((type === 'RDP' || type === 'VNC') ? [{ key: 'dlp', label: 'DLP Policy' }] : []),
  ];

  const handleSubmit = async () => {
    if (!name || !host) {
      setError('Name and host are required');
      return;
    }
    if (type === 'DB_TUNNEL' && (!targetDbHost || !targetDbPort)) {
      setError('Target database host and port are required for DB Tunnel connections');
      return;
    }
    if (credentialMode === 'keychain' && !selectedSecretId) {
      setError('Please select a secret from the keychain');
      return;
    }
    if (credentialMode === 'external-vault' && (!selectedVaultProviderId || !vaultSecretPath)) {
      setError('Please select a vault provider and enter a secret path');
      return;
    }
    if (credentialMode === 'manual' && !isEditMode && !username) {
      setError('Username is required for new connections');
      return;
    }

    const ok = await run(async () => {
      if (isEditMode && connection) {
        const data: ConnectionUpdate = {
          name,
          type,
          host,
          port: parseInt(port, 10),
          description: description || null,
          enableDrive,
          gatewayId: gatewayId || null,
          credentialSecretId: credentialMode === 'keychain' ? selectedSecretId : null,
          externalVaultProviderId: credentialMode === 'external-vault' ? selectedVaultProviderId : null,
          externalVaultPath: credentialMode === 'external-vault' ? vaultSecretPath : null,
          ...(type === 'SSH' && {
            sshTerminalConfig: Object.keys(sshTerminalConfig).length > 0 ? sshTerminalConfig : null,
          }),
          ...(type === 'RDP' && {
            rdpSettings: Object.keys(rdpSettings).length > 0 ? rdpSettings : null,
          }),
          ...(type === 'VNC' && {
            vncSettings: Object.keys(vncSettings).length > 0 ? vncSettings : null,
          }),
          ...(type === 'DATABASE' && {
            dbSettings: normalizeDbSettings(dbSettings),
          }),
          defaultCredentialMode: (defaultConnectMode as 'saved' | 'domain' | 'prompt') || null,
          ...((type === 'RDP' || type === 'VNC') && {
            dlpPolicy: Object.values(dlpPolicy).some(Boolean) ? dlpPolicy : null,
          }),
          ...(type === 'DB_TUNNEL' && {
            targetDbHost: targetDbHost || null,
            targetDbPort: targetDbPort ? parseInt(targetDbPort, 10) : null,
            dbType: dbType || null,
          }),
        };
        if (credentialMode === 'manual') {
          if (username) data.username = username;
          if (password) data.password = password;
          if (domain) data.domain = domain;
        }
        await updateConnection(connection.id, data);
      } else {
        const data: ConnectionInput = {
          name,
          type,
          host,
          port: parseInt(port, 10),
          description: description || undefined,
          enableDrive,
          gatewayId: gatewayId || null,
          ...(credentialMode === 'keychain' && selectedSecretId
            ? { credentialSecretId: selectedSecretId }
            : credentialMode === 'external-vault' && selectedVaultProviderId
              ? { externalVaultProviderId: selectedVaultProviderId, externalVaultPath: vaultSecretPath }
              : credentialMode === 'manual' ? { username, password, ...(domain ? { domain } : {}) } : {}),
          ...(folderId ? { folderId } : {}),
          ...(teamId ? { teamId } : {}),
          ...(type === 'SSH' && Object.keys(sshTerminalConfig).length > 0 && {
            sshTerminalConfig,
          }),
          ...(type === 'RDP' && Object.keys(rdpSettings).length > 0 && {
            rdpSettings,
          }),
          ...(type === 'VNC' && Object.keys(vncSettings).length > 0 && {
            vncSettings,
          }),
          ...(type === 'DATABASE' && dbSettings.protocol && {
            dbSettings: normalizeDbSettings(dbSettings) as DbSettings,
          }),
          ...(defaultConnectMode ? { defaultCredentialMode: defaultConnectMode as 'saved' | 'domain' | 'prompt' } : {}),
          ...((type === 'RDP' || type === 'VNC') && Object.values(dlpPolicy).some(Boolean) && {
            dlpPolicy,
          }),
          ...(type === 'DB_TUNNEL' && {
            targetDbHost,
            targetDbPort: parseInt(targetDbPort, 10),
            ...(dbType ? { dbType } : {}),
          }),
        };
        await createConnection(data);
      }
      await fetchConnections();
    }, isEditMode ? 'Failed to update connection' : 'Failed to create connection');
    if (ok) handleClose();
  };

  const handleClose = () => {
    setName('');
    setType('SSH');
    setHost('');
    setPort('22');
    setUsername('');
    setPassword('');
    setDomain('');
    setDescription('');
    setEnableDrive(false);
    setGatewayId('');
    setSshTerminalConfig({});
    setRdpSettings({});
    setVncSettings({});
    setDbSettings({});
    setCredentialMode('manual');
    setSelectedSecretId(null);
    setSelectedVaultProviderId(null);
    setVaultSecretPath('');
    setDefaultConnectMode('');
    setDlpPolicy({});
    setTargetDbHost('');
    setTargetDbPort('');
    setDbType('');
    clearError();
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) handleClose(); }}>
      <DialogContent
        showCloseButton={false}
        className="flex h-[100dvh] w-screen max-w-none flex-col gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
      >
        {/* Compact header */}
        <div className="flex h-8 shrink-0 items-center gap-2 border-b px-3">
          <span className="text-xs font-medium">{isEditMode ? 'Edit Connection' : 'New Connection'}</span>
          <div className="ml-auto">
            <Button variant="ghost" size="icon-xs" onClick={handleClose}>
              <X className="size-3.5" />
            </Button>
          </div>
        </div>

        {error && (
          <div className="mx-4 mt-3 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-xs text-destructive">
            {error}
          </div>
        )}

        <div className="flex min-h-0 flex-1">
          {/* Left nav */}
          <nav className="flex w-[180px] shrink-0 flex-col gap-1 border-r p-2">
            {sections.map((s) => (
              <button
                key={s.key}
                type="button"
                onClick={() => setActiveSection(s.key)}
                className={cn(
                  'flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-left text-sm transition-colors',
                  activeSection === s.key
                    ? 'bg-accent text-accent-foreground font-medium'
                    : 'text-muted-foreground hover:bg-accent/50',
                )}
              >
                {s.label}
              </button>
            ))}
          </nav>

          {/* Right content */}
          <ScrollArea className="flex-1 px-6 py-4">
            <div className="mx-auto max-w-3xl flex flex-col gap-4">

              {/* General */}
              {activeSection === 'general' && (
                <>
                  <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div className="space-y-2">
                      <Label htmlFor="conn-name">Name</Label>
                      <Input id="conn-name" value={name} onChange={(e) => setName(e.target.value)} required />
                    </div>
                    <div className="space-y-2">
                      <Label>Type</Label>
                      <Select value={type} onValueChange={(v) => handleTypeChange(v as typeof type)} disabled={isEditMode}>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {connectionsEnabled && <SelectItem value="SSH">SSH</SelectItem>}
                          {connectionsEnabled && <SelectItem value="RDP">RDP</SelectItem>}
                          {connectionsEnabled && <SelectItem value="VNC">VNC</SelectItem>}
                          {databaseProxyEnabled && <SelectItem value="DATABASE">Database</SelectItem>}
                          {databaseProxyEnabled && <SelectItem value="DB_TUNNEL">Database (SSH Tunnel)</SelectItem>}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  {hasTenant && availableGateways.length > 0 && (
                    <div className="space-y-2">
                      <Label>Gateway (optional)</Label>
                      <Select value={gatewayId || '__none__'} onValueChange={(v) => setGatewayId(v === '__none__' ? '' : v)}>
                        <SelectTrigger>
                          <SelectValue placeholder="None (Direct connection)" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="__none__">None (Direct connection)</SelectItem>
                          {availableGateways.map((gw) => (
                            <SelectItem key={gw.id} value={gw.id}>
                              {gw.name} — {gatewayEndpointLabel(gw)}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}

                  <div className="flex gap-3">
                    <div className="flex-1 space-y-2">
                      <Label htmlFor="conn-host">{type === 'DB_TUNNEL' ? 'Bastion Host' : 'Host'}</Label>
                      <Input id="conn-host" value={host} onChange={(e) => setHost(e.target.value)} required />
                    </div>
                    <div className="w-[120px] space-y-2">
                      <Label htmlFor="conn-port">{type === 'DB_TUNNEL' ? 'Bastion Port' : 'Port'}</Label>
                      <Input id="conn-port" value={port} onChange={(e) => setPort(e.target.value)} type="number" />
                    </div>
                  </div>

                  {type === 'DB_TUNNEL' && (
                    <>
                      <div className="flex gap-3">
                        <div className="flex-1 space-y-2">
                          <Label htmlFor="conn-target-host">Target DB Host</Label>
                          <Input id="conn-target-host" value={targetDbHost} onChange={(e) => setTargetDbHost(e.target.value)} required placeholder="e.g. db.internal.example.com" />
                        </div>
                        <div className="w-[120px] space-y-2">
                          <Label htmlFor="conn-target-port">Target DB Port</Label>
                          <Input id="conn-target-port" value={targetDbPort} onChange={(e) => setTargetDbPort(e.target.value)} type="number" required placeholder="5432" />
                        </div>
                      </div>
                      <div className="space-y-2">
                        <Label>Database Type</Label>
                        <Select value={dbType || '__generic__'} onValueChange={(v) => setDbType(v === '__generic__' ? '' : v)}>
                          <SelectTrigger>
                            <SelectValue placeholder="Generic" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="__generic__">Generic</SelectItem>
                            <SelectItem value="postgresql">PostgreSQL</SelectItem>
                            <SelectItem value="mysql">MySQL</SelectItem>
                            <SelectItem value="mariadb">MariaDB</SelectItem>
                            <SelectItem value="mongodb">MongoDB</SelectItem>
                            <SelectItem value="redis">Redis</SelectItem>
                            <SelectItem value="mssql">SQL Server</SelectItem>
                            <SelectItem value="oracle">Oracle</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </>
                  )}
                </>
              )}

              {/* Credentials */}
              {activeSection === 'credentials' && (
                <div className="flex flex-col gap-4">
                  <ToggleGroup
                    type="single"
                    value={credentialMode}
                    onValueChange={(val) => { if (val) setCredentialMode(val as typeof credentialMode); }}
                    className="w-full"
                  >
                    <ToggleGroupItem value="manual" className="flex-1 gap-1.5">
                      <Keyboard className="size-4" /> Manual
                    </ToggleGroupItem>
                    <ToggleGroupItem value="keychain" className="flex-1 gap-1.5" disabled={!vaultUnlocked}>
                      <KeyRound className="size-4" /> From Keychain
                    </ToggleGroupItem>
                    {hasTenant && vaultProviders.length > 0 && (
                      <ToggleGroupItem value="external-vault" className="flex-1 gap-1.5">
                        <Cloud className="size-4" /> External Vault
                      </ToggleGroupItem>
                    )}
                  </ToggleGroup>

                  {credentialMode === 'external-vault' ? (
                    <>
                      <div className="space-y-2">
                        <Label>Vault Provider</Label>
                        <Select value={selectedVaultProviderId ?? ''} onValueChange={(v) => setSelectedVaultProviderId(v || null)}>
                          <SelectTrigger>
                            <SelectValue placeholder="Select provider" />
                          </SelectTrigger>
                          <SelectContent>
                            {vaultProviders.filter((p) => p.enabled).map((p) => (
                              <SelectItem key={p.id} value={p.id}>{p.name} — {p.serverUrl}</SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="vault-path">Secret Path</Label>
                        <Input
                          id="vault-path"
                          value={vaultSecretPath}
                          onChange={(e) => setVaultSecretPath(e.target.value)}
                          required
                          placeholder="e.g. servers/web1"
                        />
                        <p className="text-xs text-muted-foreground">Path within the KV v2 mount (must contain username/password fields)</p>
                      </div>
                    </>
                  ) : credentialMode === 'keychain' ? (
                    <SecretPicker
                      value={selectedSecretId}
                      onChange={(id) => setSelectedSecretId(id)}
                      connectionType={type === 'DB_TUNNEL' ? 'SSH' : type}
                      error={!selectedSecretId && !!error}
                      initialName={connection?.credentialSecretName}
                      initialType={connection?.credentialSecretType as 'LOGIN' | 'SSH_KEY' | undefined}
                    />
                  ) : (
                    <>
                      <div className="space-y-2">
                        <Label htmlFor="conn-username">Username</Label>
                        <Input
                          id="conn-username"
                          value={username}
                          onChange={(e) => setUsername(e.target.value)}
                          required={!isEditMode}
                          placeholder={isEditMode ? 'Leave blank to keep unchanged' : undefined}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="conn-password">Password</Label>
                        <Input
                          id="conn-password"
                          value={password}
                          onChange={(e) => setPassword(e.target.value)}
                          type="password"
                          placeholder={isEditMode ? 'Leave blank to keep unchanged' : undefined}
                        />
                      </div>
                      {type === 'RDP' && (
                        <div className="space-y-2">
                          <Label htmlFor="conn-domain">Domain (optional)</Label>
                          <Input
                            id="conn-domain"
                            value={domain}
                            onChange={(e) => setDomain(e.target.value)}
                            placeholder={isEditMode ? 'Leave blank to keep unchanged' : 'e.g. CONTOSO'}
                          />
                        </div>
                      )}
                    </>
                  )}
                </div>
              )}

              {/* Options */}
              {activeSection === 'options' && (
                <div className="flex flex-col gap-4">
                  <div className="space-y-2">
                    <Label>Default connect behavior</Label>
                    <Select value={defaultConnectMode || '__saved__'} onValueChange={(v) => setDefaultConnectMode(v === '__saved__' ? '' : v)}>
                      <SelectTrigger>
                        <SelectValue placeholder="Use saved credentials (default)" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="__saved__">Use saved credentials (default)</SelectItem>
                        <SelectItem value="domain">Use domain profile credentials</SelectItem>
                        <SelectItem value="prompt">Always ask (show Connect As dialog)</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="conn-desc">Description (optional)</Label>
                    <Textarea
                      id="conn-desc"
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                      rows={2}
                    />
                  </div>
                  {type === 'RDP' && (
                    <div className="flex items-center gap-2">
                      <Checkbox
                        id="enable-drive"
                        checked={enableDrive}
                        onCheckedChange={(v) => setEnableDrive(v === true)}
                      />
                      <Label htmlFor="enable-drive" className="font-normal">
                        Enable file sharing (drive redirection)
                      </Label>
                    </div>
                  )}
                </div>
              )}

              {/* SSH Terminal Appearance */}
              {activeSection === 'terminal' && type === 'SSH' && (
                <TerminalSettingsSection
                  value={sshTerminalConfig}
                  onChange={setSshTerminalConfig}
                  mode="connection"
                  resolvedDefaults={mergeTerminalConfig(userDefaults)}
                  enforcedFields={tenantEnforced?.ssh}
                />
              )}

              {/* RDP Settings */}
              {activeSection === 'rdp' && type === 'RDP' && (
                <RdpSettingsSection
                  value={rdpSettings}
                  onChange={setRdpSettings}
                  mode="connection"
                  resolvedDefaults={mergeRdpConfig(rdpUserDefaults)}
                  enforcedFields={tenantEnforced?.rdp}
                />
              )}

              {/* VNC Settings */}
              {activeSection === 'vnc' && type === 'VNC' && (
                <VncSettingsSection
                  value={vncSettings}
                  onChange={setVncSettings}
                  mode="connection"
                  resolvedDefaults={mergeVncConfig()}
                  enforcedFields={tenantEnforced?.vnc}
                />
              )}

              {/* Database Settings */}
              {activeSection === 'database' && type === 'DATABASE' && (
                <div className="flex flex-col gap-4">
                  <div className="space-y-2">
                    <Label>Database Protocol</Label>
                    <Select
                      value={dbSettings.protocol ?? 'postgresql'}
                      onValueChange={(v) => {
                        const proto = v as DbProtocol;
                        setDbSettings((prev) => ({
                          protocol: proto,
                          databaseName: prev.databaseName,
                          cloudProvider: supportsCloudProviderPresets(proto) ? prev.cloudProvider : undefined,
                          sslMode: remapSSLModeOnProtocolChange(
                            prev.protocol,
                            proto,
                            prev.sslMode,
                            supportsCloudProviderPresets(proto) ? prev.cloudProvider : undefined,
                          ),
                          persistExecutionPlan: supportsPersistedExecutionPlans(proto)
                            ? prev.persistExecutionPlan
                            : undefined,
                          ...(proto === 'oracle' ? { oracleConnectionType: 'basic' as OracleConnectionType } : {}),
                        }));
                        const protoPorts: Record<string, string> = { postgresql: '5432', mysql: '3306', mongodb: '27017', oracle: '1521', mssql: '1433', db2: '50000' };
                        setPort(protoPorts[proto] ?? '5432');
                      }}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="postgresql">PostgreSQL</SelectItem>
                        <SelectItem value="mysql">MySQL / MariaDB</SelectItem>
                        <SelectItem value="mongodb">MongoDB</SelectItem>
                        <SelectItem value="oracle">Oracle (TNS)</SelectItem>
                        <SelectItem value="mssql">Microsoft SQL Server (TDS)</SelectItem>
                        <SelectItem value="db2">IBM DB2 (DRDA)</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="db-name">Database Name (optional)</Label>
                    <Input
                      id="db-name"
                      value={dbSettings.databaseName ?? ''}
                      onChange={(e) => setDbSettings((prev) => ({ ...prev, databaseName: e.target.value || undefined }))}
                      placeholder="e.g. mydb"
                    />
                  </div>
                  {supportsCloudProviderPresets(dbSettings.protocol) && (
                    <>
                      <div className="space-y-2">
                        <Label>Cloud Provider Preset</Label>
                        <Select
                          value={dbSettings.cloudProvider ?? 'generic'}
                          onValueChange={(v) => {
                            const nextProvider = normalizeCloudProviderSelection(v);
                            setDbSettings((prev) => ({
                              ...prev,
                              cloudProvider: nextProvider,
                              sslMode: nextSSLModeForCloudProvider(
                                prev.protocol,
                                prev.sslMode,
                                prev.cloudProvider,
                                nextProvider,
                              ),
                            }));
                          }}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="generic">Generic / self-hosted</SelectItem>
                            <SelectItem value="azure">Azure Database</SelectItem>
                            <SelectItem value="aws">AWS RDS / Aurora</SelectItem>
                            <SelectItem value="gcp">GCP Cloud SQL</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="space-y-2">
                        <Label>TLS Mode</Label>
                        <Select
                          value={dbSettings.sslMode || '__default__'}
                          onValueChange={(v) => setDbSettings((prev) => ({ ...prev, sslMode: v === '__default__' ? undefined : v }))}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {dbTLSOptions.map((option) => (
                              <SelectItem key={option.value || 'default'} value={option.value}>
                                {option.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      {currentDbTLSOption && (
                        <p className="text-xs text-muted-foreground">{currentDbTLSOption.helperText}</p>
                      )}
                      {dbCloudHint && (
                        <div className="rounded-md border border-blue-600/50 bg-blue-600/10 px-4 py-3 text-sm text-blue-400">
                          {dbCloudHint}
                        </div>
                      )}
                      {dbSettings.sslMode === 'skip-verify' && (
                        <div className="rounded-md border border-yellow-600/50 bg-yellow-600/10 px-4 py-3 text-sm text-yellow-500">
                          Skip verification accepts any server certificate. Use it only when you control the network and cannot trust the certificate chain yet.
                        </div>
                      )}
                    </>
                  )}
                  {supportsPersistedExecutionPlans(dbSettings.protocol ?? 'postgresql') && (
                    <div>
                      <div className="flex items-center gap-3">
                        <Switch
                          id="persist-plan"
                          checked={Boolean(dbSettings.persistExecutionPlan)}
                          onCheckedChange={(v) => setDbSettings((prev) => ({
                            ...prev,
                            persistExecutionPlan: v || undefined,
                          }))}
                        />
                        <Label htmlFor="persist-plan" className="font-normal">Persist execution plans in audit logs</Label>
                      </div>
                      <p className="text-xs text-muted-foreground mt-1">
                        Store the DB proxy execution plan with each audited query so it remains visible after the session closes.
                      </p>
                    </div>
                  )}
                  {dbSettings.protocol === 'oracle' && (
                    <>
                      {/* Connection type toggle: Basic | TNS | Custom */}
                      <ToggleGroup
                        type="single"
                        value={dbSettings.oracleConnectionType ?? 'basic'}
                        onValueChange={(val) => {
                          if (!val) return;
                          setDbSettings((prev) => ({
                            protocol: 'oracle' as DbProtocol,
                            databaseName: prev.databaseName,
                            persistExecutionPlan: prev.persistExecutionPlan,
                            oracleConnectionType: val as OracleConnectionType,
                            oracleRole: prev.oracleRole,
                            ...(val === 'basic' ? { oracleSid: prev.oracleSid, oracleServiceName: prev.oracleServiceName } : {}),
                            ...(val === 'tns' ? { oracleTnsAlias: prev.oracleTnsAlias, oracleTnsDescriptor: prev.oracleTnsDescriptor } : {}),
                            ...(val === 'custom' ? { oracleConnectString: prev.oracleConnectString } : {}),
                          }));
                        }}
                        className="w-full"
                      >
                        <ToggleGroupItem value="basic" className="flex-1">Basic</ToggleGroupItem>
                        <ToggleGroupItem value="tns" className="flex-1">TNS</ToggleGroupItem>
                        <ToggleGroupItem value="custom" className="flex-1">Custom</ToggleGroupItem>
                      </ToggleGroup>

                      {/* Basic mode */}
                      {(dbSettings.oracleConnectionType ?? 'basic') === 'basic' && (
                        <div className="flex gap-3">
                          <div className="w-[160px] space-y-2">
                            <Label>Identifier Type</Label>
                            <Select
                              value={dbSettings.oracleSid ? 'sid' : 'service'}
                              onValueChange={(v) => {
                                if (v === 'sid') {
                                  setDbSettings((prev) => ({ ...prev, oracleSid: prev.oracleServiceName || prev.oracleSid || '', oracleServiceName: undefined }));
                                } else {
                                  setDbSettings((prev) => ({ ...prev, oracleServiceName: prev.oracleSid || prev.oracleServiceName || '', oracleSid: undefined }));
                                }
                              }}
                            >
                              <SelectTrigger>
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="service">Service Name</SelectItem>
                                <SelectItem value="sid">SID</SelectItem>
                              </SelectContent>
                            </Select>
                          </div>
                          <div className="flex-1 space-y-2">
                            <Label>{dbSettings.oracleSid !== undefined ? 'SID' : 'Service Name'}</Label>
                            <Input
                              value={dbSettings.oracleSid ?? dbSettings.oracleServiceName ?? ''}
                              onChange={(e) => {
                                const val = e.target.value || undefined;
                                if (dbSettings.oracleSid !== undefined) {
                                  setDbSettings((prev) => ({ ...prev, oracleSid: val }));
                                } else {
                                  setDbSettings((prev) => ({ ...prev, oracleServiceName: val }));
                                }
                              }}
                              placeholder={dbSettings.oracleSid !== undefined ? 'e.g. ORCL' : 'e.g. FREEPDB1'}
                            />
                          </div>
                        </div>
                      )}

                      {/* TNS mode */}
                      {dbSettings.oracleConnectionType === 'tns' && (
                        <>
                          <div className="space-y-2">
                            <Label>TNS Alias</Label>
                            <Input
                              value={dbSettings.oracleTnsAlias ?? ''}
                              onChange={(e) => setDbSettings((prev) => ({ ...prev, oracleTnsAlias: e.target.value || undefined }))}
                              placeholder="e.g. MYDB"
                            />
                            <p className="text-xs text-muted-foreground">Alias from tnsnames.ora (resolved via TNS_ADMIN)</p>
                          </div>
                          <div className="space-y-2">
                            <Label>TNS Descriptor</Label>
                            <Textarea
                              value={dbSettings.oracleTnsDescriptor ?? ''}
                              onChange={(e) => setDbSettings((prev) => ({ ...prev, oracleTnsDescriptor: e.target.value || undefined }))}
                              rows={3}
                              placeholder="(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=...)(PORT=...))(CONNECT_DATA=(SERVICE_NAME=...)))"
                            />
                            <p className="text-xs text-muted-foreground">Full TNS descriptor (overrides alias if both provided)</p>
                          </div>
                        </>
                      )}

                      {/* Custom mode */}
                      {dbSettings.oracleConnectionType === 'custom' && (
                        <div className="space-y-2">
                          <Label>Connect String</Label>
                          <Textarea
                            value={dbSettings.oracleConnectString ?? ''}
                            onChange={(e) => setDbSettings((prev) => ({ ...prev, oracleConnectString: e.target.value || undefined }))}
                            rows={3}
                            placeholder="host:port/service_name or full TNS descriptor"
                          />
                          <p className="text-xs text-muted-foreground">Raw connect string passed directly to the Oracle driver</p>
                        </div>
                      )}

                      {/* Oracle Role (all modes) */}
                      <div className="space-y-2">
                        <Label>Role</Label>
                        <Select
                          value={dbSettings.oracleRole ?? 'normal'}
                          onValueChange={(v) => setDbSettings((prev) => ({ ...prev, oracleRole: (v === 'normal' ? undefined : v) as OracleRole | undefined }))}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="normal">Normal</SelectItem>
                            <SelectItem value="sysdba">SYSDBA</SelectItem>
                            <SelectItem value="sysoper">SYSOPER</SelectItem>
                            <SelectItem value="sysasm">SYSASM</SelectItem>
                            <SelectItem value="sysbackup">SYSBACKUP</SelectItem>
                            <SelectItem value="sysdg">SYSDG</SelectItem>
                            <SelectItem value="syskm">SYSKM</SelectItem>
                            <SelectItem value="sysrac">SYSRAC</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </>
                  )}
                  {dbSettings.protocol === 'mssql' && (
                    <>
                      <div className="space-y-2">
                        <Label>Instance Name (optional)</Label>
                        <Input
                          value={dbSettings.mssqlInstanceName ?? ''}
                          onChange={(e) => setDbSettings((prev) => ({ ...prev, mssqlInstanceName: e.target.value || undefined }))}
                          placeholder="e.g. SQLEXPRESS"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label>Authentication Mode</Label>
                        <Select
                          value={dbSettings.mssqlAuthMode ?? 'sql'}
                          onValueChange={(v) => setDbSettings((prev) => ({ ...prev, mssqlAuthMode: v as 'sql' | 'windows' }))}
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="sql">SQL Server Authentication</SelectItem>
                            <SelectItem value="windows">Windows Authentication (NTLM/Kerberos)</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </>
                  )}
                  {dbSettings.protocol === 'db2' && (
                    <div className="space-y-2">
                      <Label>Database Alias (optional)</Label>
                      <Input
                        value={dbSettings.db2DatabaseAlias ?? ''}
                        onChange={(e) => setDbSettings((prev) => ({ ...prev, db2DatabaseAlias: e.target.value || undefined }))}
                        placeholder="e.g. SAMPLE"
                      />
                      <p className="text-xs text-muted-foreground">Alias as cataloged on the DB2 Connect gateway</p>
                    </div>
                  )}
                </div>
              )}

              {/* DLP */}
              {activeSection === 'dlp' && (type === 'RDP' || type === 'VNC') && (
                <>
                  <p className="text-xs text-muted-foreground mb-3">
                    These restrictions are additive to the organization&apos;s DLP policy.
                  </p>
                  <div className="flex flex-col gap-2">
                    <div className="flex items-center gap-2">
                      <Checkbox id="dlp-copy" checked={dlpPolicy.disableCopy ?? false} onCheckedChange={(v) => setDlpPolicy((p) => ({ ...p, disableCopy: (v === true) || undefined }))} />
                      <Label htmlFor="dlp-copy" className="font-normal">Disable clipboard copy (remote to local)</Label>
                    </div>
                    <div className="flex items-center gap-2">
                      <Checkbox id="dlp-paste" checked={dlpPolicy.disablePaste ?? false} onCheckedChange={(v) => setDlpPolicy((p) => ({ ...p, disablePaste: (v === true) || undefined }))} />
                      <Label htmlFor="dlp-paste" className="font-normal">Disable clipboard paste (local to remote)</Label>
                    </div>
                    {type === 'RDP' && (
                      <>
                        <div className="flex items-center gap-2">
                          <Checkbox id="dlp-download" checked={dlpPolicy.disableDownload ?? false} onCheckedChange={(v) => setDlpPolicy((p) => ({ ...p, disableDownload: (v === true) || undefined }))} />
                          <Label htmlFor="dlp-download" className="font-normal">Disable file download from shared drive</Label>
                        </div>
                        <div className="flex items-center gap-2">
                          <Checkbox id="dlp-upload" checked={dlpPolicy.disableUpload ?? false} onCheckedChange={(v) => setDlpPolicy((p) => ({ ...p, disableUpload: (v === true) || undefined }))} />
                          <Label htmlFor="dlp-upload" className="font-normal">Disable file upload to shared drive</Label>
                        </div>
                      </>
                    )}
                  </div>
                </>
              )}

            </div>
          </ScrollArea>
        </div>

        <div className="flex shrink-0 items-center justify-end gap-2 border-t px-4 py-2">
          <Button variant="outline" size="sm" onClick={handleClose}>Cancel</Button>
          <Button size="sm" onClick={handleSubmit} disabled={loading}>
            {loading
              ? (isEditMode ? 'Saving...' : 'Creating...')
              : (isEditMode ? 'Save' : 'Create')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
