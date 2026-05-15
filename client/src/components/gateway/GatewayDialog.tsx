import { useState, useEffect, useCallback, useRef } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select';
import {
  Accordion,
  AccordionItem,
  AccordionTrigger,
  AccordionContent,
} from '@/components/ui/accordion';
import {
  Save,
  Lock,
  RefreshCw,
  Trash2,
  Power,
  History,
  Gauge,
  Loader2,
} from 'lucide-react';
import { useGatewayStore } from '../../store/gatewayStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import type {
  GatewayData,
  GatewayDeploymentMode,
  TunnelEventData,
  TunnelMetricsData,
  TunnelTokenResponse,
} from '../../api/gateway.api';
import {
  forceDisconnectTunnel as forceDisconnectApi,
  getTunnelEvents as getTunnelEventsApi,
  getTunnelMetrics as getTunnelMetricsApi,
} from '../../api/gateway.api';
import SessionTimeoutConfig from '../orchestration/SessionTimeoutConfig';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import { extractApiError } from '../../utils/apiError';
import { useFeatureFlagsStore } from '../../store/featureFlagsStore';
import GatewayEgressPolicyEditor from './GatewayEgressPolicyEditor';
import GatewayTunnelInstallPanel from './GatewayTunnelInstallPanel';

interface GatewayDialogProps {
  open: boolean;
  onClose: () => void;
  gateway?: GatewayData | null;
}

export default function GatewayDialog({ open, onClose, gateway }: GatewayDialogProps) {
  const [name, setName] = useState('');
  const [type, setType] = useState<'GUACD' | 'SSH_BASTION' | 'MANAGED_SSH' | 'DB_PROXY'>('GUACD');
  const [deploymentMode, setDeploymentMode] = useState<GatewayDeploymentMode>('SINGLE_INSTANCE');
  const [host, setHost] = useState('');
  const [port, setPort] = useState('');
  const [description, setDescription] = useState('');
  const [isDefault, setIsDefault] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [sshPrivateKey, setSshPrivateKey] = useState('');
  const [apiPort, setApiPort] = useState('');
  const [monitoringEnabled, setMonitoringEnabled] = useState(true);
  const [monitorIntervalMs, setMonitorIntervalMs] = useState('5000');
  const [inactivityTimeout, setInactivityTimeout] = useState('60');
  const [autoScaleEnabled, setAutoScaleEnabled] = useState(false);
  const [minReplicasVal, setMinReplicasVal] = useState('0');
  const [maxReplicasVal, setMaxReplicasVal] = useState('5');
  const [sessPerInstance, setSessPerInstance] = useState('10');
  const [cooldownVal, setCooldownVal] = useState('300');
  const [publishPorts, setPublishPorts] = useState(false);
  const [lbStrategy, setLbStrategy] = useState<'ROUND_ROBIN' | 'LEAST_CONNECTIONS'>('ROUND_ROBIN');

  const [tunnelBundle, setTunnelBundle] = useState<TunnelTokenResponse | null>(null);
  const [enableTunnelOnCreate, setEnableTunnelOnCreate] = useState(false);
  const [createdGateway, setCreatedGateway] = useState<GatewayData | null>(null);
  const [tunnelDeploying, setTunnelDeploying] = useState(false);
  const [tunnelError, setTunnelError] = useState('');
  const [rotateConfirmOpen, setRotateConfirmOpen] = useState(false);
  const [revokeConfirmOpen, setRevokeConfirmOpen] = useState(false);
  const [disconnectConfirmOpen, setDisconnectConfirmOpen] = useState(false);
  const [tunnelEvents, setTunnelEvents] = useState<TunnelEventData[]>([]);
  const [tunnelEventsLoading, setTunnelEventsLoading] = useState(false);
  const [tunnelMetrics, setTunnelMetrics] = useState<TunnelMetricsData | null>(null);
  const [tunnelMetricsLoading, setTunnelMetricsLoading] = useState(false);
  const preCreateTunnelEndpointRef = useRef<{ type: GatewayData['type']; host: string; port: string } | null>(null);

  const { loading, error, setError, run } = useAsyncAction();
  const { loading: scalingSaving, run: runScaling } = useAsyncAction();
  const { loading: tunnelActionLoading, run: runTunnelAction } = useAsyncAction();

  const createGateway = useGatewayStore((s) => s.createGateway);
  const updateGateway = useGatewayStore((s) => s.updateGateway);
  const updateScalingConfig = useGatewayStore((s) => s.updateScalingConfig);
  const generateTunnelTokenAction = useGatewayStore((s) => s.generateTunnelToken);
  const revokeTunnelTokenAction = useGatewayStore((s) => s.revokeTunnelToken);
  const zeroTrustEnabled = useFeatureFlagsStore((s) => s.zeroTrustEnabled);

  const tunnelSectionOpen = useUiPreferencesStore((s) => s.tunnelSectionOpen);
  const tunnelEventLogOpen = useUiPreferencesStore((s) => s.tunnelEventLogOpen);
  const tunnelMetricsOpen = useUiPreferencesStore((s) => s.tunnelMetricsOpen);
  const setUiPref = useUiPreferencesStore((s) => s.set);

  const isEditMode = Boolean(gateway);
  const activeGateway = gateway ?? createdGateway;
  const gatewayCreatedWithTunnel = !gateway && createdGateway != null;
  const isTunnelEnabled = activeGateway?.tunnelEnabled ?? enableTunnelOnCreate;
  const isTunnelConnected = activeGateway?.tunnelConnected ?? false;
  const supportsGroupMode = type === 'MANAGED_SSH' || type === 'GUACD' || type === 'DB_PROXY';
  const isGroupMode = deploymentMode === 'MANAGED_GROUP';

  useEffect(() => {
    if (open && gateway) {
      setName(gateway.name); setType(gateway.type);
      setDeploymentMode(gateway.deploymentMode ?? (gateway.isManaged ? 'MANAGED_GROUP' : 'SINGLE_INSTANCE'));
      setHost(gateway.host); setPort(String(gateway.port));
      setDescription(gateway.description || ''); setIsDefault(gateway.isDefault);
      setUsername(''); setPassword(''); setSshPrivateKey('');
      setApiPort(gateway.apiPort ? String(gateway.apiPort) : '');
      setMonitoringEnabled(gateway.monitoringEnabled);
      setMonitorIntervalMs(String(gateway.monitorIntervalMs));
      setInactivityTimeout(String(Math.floor(gateway.inactivityTimeoutSeconds / 60)));
      setAutoScaleEnabled(gateway.autoScale); setMinReplicasVal(String(gateway.minReplicas));
      setMaxReplicasVal(String(gateway.maxReplicas)); setSessPerInstance(String(gateway.sessionsPerInstance));
      setCooldownVal(String(gateway.scaleDownCooldownSeconds));
      setPublishPorts(gateway.publishPorts ?? false); setLbStrategy(gateway.lbStrategy ?? 'ROUND_ROBIN');
    } else if (open) {
      setName(''); setType('GUACD'); setDeploymentMode('SINGLE_INSTANCE');
      setHost(''); setPort(''); setDescription(''); setIsDefault(false);
      setUsername(''); setPassword(''); setSshPrivateKey(''); setApiPort('');
      setMonitoringEnabled(true); setMonitorIntervalMs('5000'); setInactivityTimeout('60');
      setAutoScaleEnabled(false); setMinReplicasVal('0'); setMaxReplicasVal('5');
      setSessPerInstance('10'); setCooldownVal('300'); setPublishPorts(false); setLbStrategy('ROUND_ROBIN');
    }
    setError(''); setTunnelBundle(null); setTunnelError(''); setTunnelDeploying(false);
    setEnableTunnelOnCreate(false); setCreatedGateway(null);
    preCreateTunnelEndpointRef.current = null;
    setRotateConfirmOpen(false); setRevokeConfirmOpen(false); setDisconnectConfirmOpen(false);
    setTunnelEvents([]); setTunnelMetrics(null);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, gateway]);

  const handleTypeChange = (newType: 'GUACD' | 'SSH_BASTION' | 'MANAGED_SSH' | 'DB_PROXY') => {
    setType(newType);
    const defaultPort = enableTunnelOnCreate ? gatewayRuntimeDefaultPort(newType) : gatewayDirectDefaultPort(newType);
    if (!port || port === '4822' || port === '22' || port === '2222' || port === '5432') setPort(defaultPort);
    if (newType === 'MANAGED_SSH' && !apiPort) setApiPort('9022');
    else if (newType !== 'MANAGED_SSH') setApiPort('');
    if (newType === 'SSH_BASTION') setDeploymentMode('SINGLE_INSTANCE');
    if (enableTunnelOnCreate) setHost('127.0.0.1');
  };

  const handleSubmit = async () => {
    setError('');
    if (!name.trim()) { setError('Gateway name is required'); return; }
    const tunnelManagedCreate = !isEditMode && enableTunnelOnCreate && !isGroupMode;
    const effectiveHost = tunnelManagedCreate ? '127.0.0.1' : host.trim();
    const effectivePort = tunnelManagedCreate ? gatewayRuntimeDefaultPort(type) : port;
    if (!isGroupMode && !tunnelManagedCreate && !host.trim()) { setError('Host is required'); return; }
    const portNum = parseInt(effectivePort, 10);
    if (!effectivePort || isNaN(portNum) || portNum < 1 || portNum > 65535) { setError('Port must be between 1 and 65535'); return; }

    const ok = await run(async () => {
      if (isEditMode && gateway) {
        const data: Record<string, unknown> = {};
        const normalizedHost = isGroupMode ? '' : effectiveHost;
        const existingDeploymentMode = gateway.deploymentMode ?? (gateway.isManaged ? 'MANAGED_GROUP' : 'SINGLE_INSTANCE');
        if (name.trim() !== gateway.name) data.name = name.trim();
        if (deploymentMode !== existingDeploymentMode) data.deploymentMode = deploymentMode;
        if (normalizedHost !== gateway.host) data.host = normalizedHost;
        if (portNum !== gateway.port) data.port = portNum;
        if ((description.trim() || null) !== gateway.description) data.description = description.trim() || null;
        if (isDefault !== gateway.isDefault) data.isDefault = isDefault;
        if (gateway.type === 'MANAGED_SSH') {
          const newApiPort = apiPort ? parseInt(apiPort, 10) : null;
          if (newApiPort !== gateway.apiPort) data.apiPort = newApiPort;
        }
        if (type === 'SSH_BASTION') {
          if (username) data.username = username;
          if (password) data.password = password;
          if (sshPrivateKey) data.sshPrivateKey = sshPrivateKey;
        }
        if (supportsGroupMode && publishPorts !== (gateway.publishPorts ?? false)) data.publishPorts = publishPorts;
        if (supportsGroupMode && lbStrategy !== (gateway.lbStrategy ?? 'ROUND_ROBIN')) data.lbStrategy = lbStrategy;
        if (monitoringEnabled !== gateway.monitoringEnabled) data.monitoringEnabled = monitoringEnabled;
        const intervalNum = parseInt(monitorIntervalMs, 10);
        if (intervalNum && intervalNum !== gateway.monitorIntervalMs) data.monitorIntervalMs = intervalNum;
        const timeoutSec = parseInt(inactivityTimeout, 10) * 60;
        if (timeoutSec && timeoutSec !== gateway.inactivityTimeoutSeconds) data.inactivityTimeoutSeconds = timeoutSec;
        await updateGateway(gateway.id, data);
      } else {
        const apiPortNum = apiPort ? parseInt(apiPort, 10) : undefined;
        const created = await createGateway({
          name: name.trim(), type, deploymentMode,
          host: isGroupMode ? '' : effectiveHost, port: portNum,
          description: description.trim() || undefined, isDefault: isDefault || undefined,
          monitoringEnabled, monitorIntervalMs: parseInt(monitorIntervalMs, 10) || 5000,
          inactivityTimeoutSeconds: (parseInt(inactivityTimeout, 10) || 60) * 60,
          ...(type === 'SSH_BASTION' && username ? { username } : {}),
          ...(type === 'SSH_BASTION' && password ? { password } : {}),
          ...(type === 'SSH_BASTION' && sshPrivateKey ? { sshPrivateKey } : {}),
          ...(type === 'MANAGED_SSH' && apiPortNum ? { apiPort: apiPortNum } : {}),
          ...(supportsGroupMode && publishPorts ? { publishPorts } : {}),
          ...(supportsGroupMode ? { lbStrategy } : {}),
        });
        if (enableTunnelOnCreate) {
          setCreatedGateway(created);
          setUiPref('tunnelSectionOpen', true);
          try {
            const bundle = await generateTunnelTokenAction(created.id);
            setTunnelBundle(bundle);
            setCreatedGateway({
              ...created,
              tunnelEnabled: bundle.tunnelEnabled,
              tunnelConnected: bundle.tunnelConnected,
              tunnelClientCertExp: bundle.tunnelClientCertExp ?? created.tunnelClientCertExp,
            });
          } catch (err) {
            setEnableTunnelOnCreate(false);
            setTunnelError(extractApiError(err, 'Gateway was created, but tunnel activation failed'));
          }
        }
      }
    }, isEditMode ? 'Failed to update gateway' : 'Failed to create gateway');
    if (ok && (isEditMode || !enableTunnelOnCreate)) handleClose();
  };

  const updateCreatedGatewayTunnelState = useCallback((id: string, updates: Partial<GatewayData>) => {
    setCreatedGateway((current) => (current?.id === id ? { ...current, ...updates } : current));
  }, []);

  const handleEnableTunnel = async () => {
    if (!activeGateway) {
      setTunnelError('');
      preCreateTunnelEndpointRef.current = { type, host, port };
      setEnableTunnelOnCreate(true);
      setHost('127.0.0.1');
      setPort(gatewayRuntimeDefaultPort(type));
      setUiPref('tunnelSectionOpen', true);
      return;
    }
    setTunnelError(''); setTunnelDeploying(true);
    const ok = await runTunnelAction(async () => {
      const result = await generateTunnelTokenAction(activeGateway.id);
      setTunnelBundle(result);
      updateCreatedGatewayTunnelState(activeGateway.id, {
        tunnelEnabled: result.tunnelEnabled,
        tunnelConnected: result.tunnelConnected,
        tunnelClientCertExp: result.tunnelClientCertExp ?? activeGateway.tunnelClientCertExp,
      });
    }, 'Failed to enable tunnel');
    setTunnelDeploying(false);
    if (!ok) setTunnelError('Failed to generate tunnel token');
  };

  const handleDisableTunnelOnCreate = () => {
    setTunnelError('');
    setEnableTunnelOnCreate(false);
    const previousEndpoint = preCreateTunnelEndpointRef.current;
    preCreateTunnelEndpointRef.current = null;
    if (previousEndpoint?.type === type) {
      setHost(previousEndpoint.host);
      setPort(previousEndpoint.port || gatewayDirectDefaultPort(type));
      return;
    }
    setHost('');
    setPort(gatewayDirectDefaultPort(type));
  };

  const handleRotateTunnel = async () => {
    if (!activeGateway) return;
    setRotateConfirmOpen(false); setTunnelError('');
    const ok = await runTunnelAction(async () => {
      const result = await generateTunnelTokenAction(activeGateway.id);
      setTunnelBundle(result);
      updateCreatedGatewayTunnelState(activeGateway.id, {
        tunnelEnabled: result.tunnelEnabled,
        tunnelConnected: result.tunnelConnected,
        tunnelClientCertExp: result.tunnelClientCertExp ?? activeGateway.tunnelClientCertExp,
      });
    }, 'Failed to rotate tunnel token');
    if (!ok) setTunnelError('Failed to rotate tunnel token');
  };

  const handleRevokeTunnel = async () => {
    if (!activeGateway) return;
    setRevokeConfirmOpen(false); setTunnelError('');
    const ok = await runTunnelAction(async () => {
      await revokeTunnelTokenAction(activeGateway.id);
      setTunnelBundle(null);
      setTunnelMetrics(null);
      updateCreatedGatewayTunnelState(activeGateway.id, {
        tunnelEnabled: false,
        tunnelConnected: false,
        tunnelConnectedAt: null,
        tunnelClientCertExp: null,
      });
    }, 'Failed to revoke tunnel token');
    if (!ok) setTunnelError('Failed to revoke tunnel token');
  };

  const gatewayId = activeGateway?.id;

  const fetchTunnelEvents = useCallback(async () => {
    if (!gatewayId) return;
    setTunnelEventsLoading(true);
    try { const { events } = await getTunnelEventsApi(gatewayId); setTunnelEvents(events); }
    catch (err) { setTunnelError(extractApiError(err, 'Failed to load tunnel events')); }
    finally { setTunnelEventsLoading(false); }
  }, [gatewayId]);

  const fetchTunnelMetrics = useCallback(async () => {
    if (!gatewayId) return;
    setTunnelMetricsLoading(true);
    try { const metrics = await getTunnelMetricsApi(gatewayId); setTunnelMetrics(metrics); }
    catch { setTunnelMetrics(null); }
    finally { setTunnelMetricsLoading(false); }
  }, [gatewayId]);

  useEffect(() => {
    if (open && gatewayId && isTunnelEnabled) {
      fetchTunnelEvents();
      if (isTunnelConnected) fetchTunnelMetrics();
    }
  }, [open, gatewayId, isTunnelEnabled, isTunnelConnected, fetchTunnelEvents, fetchTunnelMetrics]);

  const handleForceDisconnect = useCallback(async () => {
    if (!gatewayId) return;
    setDisconnectConfirmOpen(false); setTunnelError('');
    const ok = await runTunnelAction(async () => { await forceDisconnectApi(gatewayId); }, 'Failed to disconnect tunnel');
    if (ok) await useGatewayStore.getState().fetchGateways();
  }, [gatewayId, runTunnelAction]);

  const serverUrl = window.location.origin;

  const formatUptime = (connectedAt: string): string => {
    const diff = Date.now() - new Date(connectedAt).getTime();
    const hours = Math.floor(diff / 3600000); const minutes = Math.floor((diff % 3600000) / 60000);
    if (hours > 0) return `${hours}h ${minutes}m`;
    return `${minutes}m`;
  };

  const certExpDisplay = (): string | null => {
    if (!activeGateway?.tunnelClientCertExp) return null;
    const exp = new Date(activeGateway.tunnelClientCertExp); const now = new Date();
    const diffDays = Math.ceil((exp.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
    const expStr = exp.toLocaleDateString();
    if (diffDays <= 0) return `Expired on ${expStr}`;
    if (diffDays <= 7) return `Expires ${expStr} — renewal imminent`;
    return `Expires ${expStr} (next renewal in ${diffDays} days)`;
  };

  const handleClose = () => {
    setName(''); setType('GUACD'); setDeploymentMode('SINGLE_INSTANCE');
    setHost(''); setPort(''); setDescription(''); setIsDefault(false);
    setUsername(''); setPassword(''); setSshPrivateKey(''); setApiPort('');
    setMonitoringEnabled(true); setMonitorIntervalMs('5000'); setInactivityTimeout('60');
    setAutoScaleEnabled(false); setMinReplicasVal('0'); setMaxReplicasVal('5');
    setSessPerInstance('10'); setCooldownVal('300'); setPublishPorts(false); setLbStrategy('ROUND_ROBIN');
    setError(''); setTunnelBundle(null); setTunnelError(''); setTunnelDeploying(false);
    setEnableTunnelOnCreate(false); setCreatedGateway(null);
    setRotateConfirmOpen(false); setRevokeConfirmOpen(false); setDisconnectConfirmOpen(false);
    setTunnelEvents([]); setTunnelMetrics(null);
    onClose();
  };

  const certInfo = certExpDisplay();

  const renderTunnelStatusChip = () => {
    if (!isTunnelEnabled) return null;
    if (!activeGateway && enableTunnelOnCreate) {
      return <Badge className="bg-blue-500/15 text-blue-400 border-blue-500/30">Enabled on create</Badge>;
    }
    if (tunnelDeploying || tunnelActionLoading) {
      return <span className="flex items-center gap-1 text-xs text-muted-foreground"><Loader2 className="h-3.5 w-3.5 animate-spin" />Deploying...</span>;
    }
    return isTunnelConnected
      ? <Badge className="bg-green-500/15 text-green-400 border-green-500/30">Connected</Badge>
      : <Badge className="bg-red-500/15 text-red-400 border-red-500/30">Disconnected</Badge>;
  };

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) handleClose(); }}>
      <DialogContent className="flex max-h-[min(88vh,calc(100vh-2rem))] w-[calc(100vw-1rem)] max-w-[calc(100vw-1rem)] flex-col overflow-hidden sm:w-[90vw] sm:max-w-[90vw]">
        <DialogHeader>
          <DialogTitle>{isEditMode ? 'Edit Gateway' : gatewayCreatedWithTunnel ? 'Gateway Created' : 'New Gateway'}</DialogTitle>
          <DialogDescription>
            {gatewayCreatedWithTunnel
              ? 'Use the enrollment bundle below to install the remote gateway container.'
              : 'Configure gateway routing, health checks, and optional zero-trust tunnel enrollment.'}
          </DialogDescription>
        </DialogHeader>
        <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden pr-1">
          {error && <div className="mb-4 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>}
          <div className="space-y-4">
          <div className="space-y-1.5"><Label htmlFor="gateway-name">Name</Label><Input id="gateway-name" value={name} onChange={(e) => setName(e.target.value)} required autoFocus maxLength={100} /></div>
          <div className="space-y-1.5">
            <Label>Type</Label>
            <Select value={type} onValueChange={(v) => handleTypeChange(v as typeof type)} disabled={isEditMode}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="GUACD">GUACD (RDP Gateway)</SelectItem>
                <SelectItem value="SSH_BASTION">SSH Bastion (Jump Host)</SelectItem>
                <SelectItem value="MANAGED_SSH">Managed SSH Gateway</SelectItem>
                <SelectItem value="DB_PROXY">DB Proxy (Database Gateway)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {supportsGroupMode && (
            <div className="space-y-1.5">
              <Label>Deployment Mode</Label>
              <Select value={deploymentMode} onValueChange={(v) => setDeploymentMode(v as GatewayDeploymentMode)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="SINGLE_INSTANCE">Single Instance</SelectItem>
                  <SelectItem value="MANAGED_GROUP">Managed Group</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}
          {!supportsGroupMode && <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-3 py-2 text-sm text-blue-400">SSH bastions are always single-instance gateways.</div>}
          {type === 'MANAGED_SSH' && <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-3 py-2 text-sm text-blue-400">This gateway uses the server&apos;s SSH key pair for authentication. No credentials needed.</div>}
          {type === 'DB_PROXY' && <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-3 py-2 text-sm text-blue-400">Database proxy gateway. Credentials are injected per-session from the vault.</div>}
          {type === 'MANAGED_SSH' && (
            <div className="space-y-1.5"><Label>gRPC Port (key management)</Label><Input value={apiPort} onChange={(e) => setApiPort(e.target.value)} type="number" disabled={publishPorts} />{publishPorts ? <p className="text-xs text-muted-foreground">Auto-assigned at deploy</p> : <p className="text-xs text-muted-foreground">gRPC port for key management mTLS (default: 9022)</p>}</div>
          )}
          {supportsGroupMode && isGroupMode && (
            <div className="flex items-center gap-3"><Switch checked={publishPorts} onCheckedChange={(v) => { setPublishPorts(v); if (v) { const dp = type === 'GUACD' ? '4822' : type === 'DB_PROXY' ? '5432' : '2222'; setPort(dp); } }} /><Label>Publish Ports (external access)</Label></div>
          )}
          {publishPorts && supportsGroupMode && isGroupMode && <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-3 py-1 text-sm text-blue-400">Each deployed instance will get a unique randomly-assigned host port for external access.</div>}
          {supportsGroupMode && isGroupMode && (
            <div className="space-y-1.5">
              <Label>Load Balancing Strategy</Label>
              <Select value={lbStrategy} onValueChange={(v) => setLbStrategy(v as 'ROUND_ROBIN' | 'LEAST_CONNECTIONS')}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent><SelectItem value="ROUND_ROBIN">Round Robin</SelectItem><SelectItem value="LEAST_CONNECTIONS">Least Connections</SelectItem></SelectContent>
              </Select>
            </div>
          )}
          {isGroupMode ? (
            <>
              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-3 py-2 text-sm text-blue-400">This gateway is a logical group. The port below is the service port used by deployed instances.</div>
              <div className="space-y-1.5"><Label htmlFor="gateway-service-port">Service Port</Label><Input id="gateway-service-port" value={port} onChange={(e) => setPort(e.target.value)} type="number" />{publishPorts && <p className="text-xs text-muted-foreground">External host ports are assigned per instance at deploy time.</p>}</div>
            </>
          ) : (
            <div className="flex gap-3">
              <div className="flex-1 space-y-1.5"><Label htmlFor="gateway-host">Host</Label><Input id="gateway-host" value={host} onChange={(e) => setHost(e.target.value)} required readOnly={isTunnelEnabled} />{isTunnelEnabled && <p className="text-xs text-muted-foreground">Managed by tunnel</p>}</div>
              <div className="w-[120px] space-y-1.5"><Label htmlFor="gateway-port">Port</Label><Input id="gateway-port" value={port} onChange={(e) => setPort(e.target.value)} type="number" disabled={isTunnelEnabled} /></div>
            </div>
          )}
          {type === 'SSH_BASTION' && (
            <>
              <div className="space-y-1.5"><Label>Username</Label><Input value={username} onChange={(e) => setUsername(e.target.value)} placeholder={isEditMode ? 'Leave blank to keep unchanged' : undefined} /></div>
              <div className="space-y-1.5"><Label>Password</Label><Input value={password} onChange={(e) => setPassword(e.target.value)} type="password" placeholder={isEditMode ? 'Leave blank to keep unchanged' : undefined} /></div>
              <div className="space-y-1.5"><Label>SSH Private Key (PEM)</Label><Textarea value={sshPrivateKey} onChange={(e) => setSshPrivateKey(e.target.value)} rows={4} className="font-mono text-xs" placeholder={isEditMode ? (gateway?.hasSshKey ? 'Key configured — leave blank to keep unchanged' : 'Paste PEM-encoded private key') : 'Paste PEM-encoded private key (optional)'} /></div>
            </>
          )}
          <div className="space-y-1.5"><Label>Description (optional)</Label><Textarea value={description} onChange={(e) => setDescription(e.target.value)} rows={2} maxLength={500} /></div>
          <div className="flex items-center gap-3"><Checkbox checked={isDefault} onCheckedChange={(v) => setIsDefault(v === true)} id="gw-default" /><Label htmlFor="gw-default">Set as default {type === 'GUACD' ? 'GUACD' : type === 'MANAGED_SSH' ? 'Managed SSH' : type === 'DB_PROXY' ? 'DB Proxy' : 'SSH Bastion'} gateway</Label></div>
          <div className="flex items-center gap-3"><Checkbox checked={monitoringEnabled} onCheckedChange={(v) => setMonitoringEnabled(v === true)} id="gw-monitor" /><Label htmlFor="gw-monitor">Enable health monitoring</Label></div>
          {monitoringEnabled && <div className="space-y-1.5"><Label>Monitor interval (ms)</Label><Input value={monitorIntervalMs} onChange={(e) => setMonitorIntervalMs(e.target.value)} type="number" /><p className="text-xs text-muted-foreground">How often to check connectivity (1000-3600000ms)</p></div>}
          <SessionTimeoutConfig value={inactivityTimeout} onChange={setInactivityTimeout} />

          {/* Auto-Scaling */}
          {isEditMode && isGroupMode && supportsGroupMode && (
            <Accordion type="single" collapsible>
              <AccordionItem value="scaling">
                <AccordionTrigger><span className="text-sm font-medium">Auto-Scaling Configuration</span></AccordionTrigger>
                <AccordionContent>
                  <div className="space-y-3">
                    <div className="flex items-center gap-3"><Switch checked={autoScaleEnabled} onCheckedChange={setAutoScaleEnabled} /><Label>Enable Auto-Scale</Label></div>
                    {autoScaleEnabled && (
                      <div className="flex flex-wrap gap-3">
                        <div className="w-[120px] space-y-1"><Label className="text-xs">Min Replicas</Label><Input value={minReplicasVal} onChange={(e) => setMinReplicasVal(e.target.value)} type="number" className="h-8" /></div>
                        <div className="w-[120px] space-y-1"><Label className="text-xs">Max Replicas</Label><Input value={maxReplicasVal} onChange={(e) => setMaxReplicasVal(e.target.value)} type="number" className="h-8" /></div>
                        <div className="w-[150px] space-y-1"><Label className="text-xs">Sessions/Instance</Label><Input value={sessPerInstance} onChange={(e) => setSessPerInstance(e.target.value)} type="number" className="h-8" /></div>
                        <div className="w-[120px] space-y-1"><Label className="text-xs">Cooldown (s)</Label><Input value={cooldownVal} onChange={(e) => setCooldownVal(e.target.value)} type="number" className="h-8" /></div>
                      </div>
                    )}
                    <Button variant="outline" size="sm" disabled={scalingSaving} onClick={() => runScaling(async () => { await updateScalingConfig(gateway?.id ?? '', { autoScale: autoScaleEnabled, minReplicas: Number(minReplicasVal), maxReplicas: Number(maxReplicasVal), sessionsPerInstance: Number(sessPerInstance), scaleDownCooldownSeconds: Number(cooldownVal) }); }, 'Failed to save scaling config')}>
                      <Save className="h-4 w-4 mr-1" />{scalingSaving ? 'Saving...' : 'Save Scaling Config'}
                    </Button>
                  </div>
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          )}

          {zeroTrustEnabled && (
            <Accordion
              type="single"
              collapsible
              value={tunnelSectionOpen || !isEditMode ? 'tunnel' : ''}
              onValueChange={(v) => setUiPref('tunnelSectionOpen', v === 'tunnel')}
            >
              <AccordionItem value="tunnel">
                <AccordionTrigger>
                  <div className="flex items-center gap-2 w-full">
                    <Lock className={`h-4 w-4 ${isTunnelEnabled ? 'text-primary' : 'text-muted-foreground'}`} />
                    <span className="text-sm font-medium flex-1">Zero-Trust Tunnel</span>
                    {renderTunnelStatusChip()}
                  </div>
                </AccordionTrigger>
                <AccordionContent>
                  <div className="space-y-3">
                    {tunnelError && <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400 flex justify-between"><span>{tunnelError}</span><button onClick={() => setTunnelError('')} className="text-xs">dismiss</button></div>}

                    {activeGateway && (
                      <>
                        <GatewayEgressPolicyEditor
                          gatewayId={activeGateway.id}
                          policy={activeGateway.egressPolicy}
                        />
                        <Separator />
                      </>
                    )}

                    {!isTunnelEnabled ? (
                      <>
                        <p className="text-sm text-muted-foreground">Enable a zero-trust tunnel so the gateway agent connects outbound to this server. No inbound ports required.</p>
                        <Button variant="outline" size="sm" disabled={tunnelDeploying || tunnelActionLoading} onClick={handleEnableTunnel}>
                          {tunnelDeploying || tunnelActionLoading ? <Loader2 className="h-4 w-4 mr-1 animate-spin" /> : <Lock className="h-4 w-4 mr-1" />}
                          {tunnelDeploying || tunnelActionLoading ? 'Enabling...' : 'Enable Zero-Trust Tunnel'}
                        </Button>
                      </>
                    ) : !activeGateway ? (
                      <div className="space-y-3">
                        <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-3 py-2 text-sm text-blue-400">
                          Tunnel will be enabled when the gateway is created. The one-time token and remote install commands will appear after creation.
                        </div>
                        <Button variant="outline" size="sm" onClick={handleDisableTunnelOnCreate}>
                          Disable Before Create
                        </Button>
                      </div>
                    ) : (
                      <>
                        <div className="flex items-center gap-2">
                          <span className="text-sm text-muted-foreground">Status:</span>
                          {renderTunnelStatusChip()}
                          {activeGateway.tunnelConnectedAt && isTunnelConnected && <span className="text-xs text-muted-foreground">since {new Date(activeGateway.tunnelConnectedAt).toLocaleString()}</span>}
                        </div>
                        {certInfo && <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-3 py-1 text-sm text-blue-400">{certInfo}</div>}
                        <Separator />

                        {tunnelBundle ? (
                          <GatewayTunnelInstallPanel
                            gateway={activeGateway}
                            tokenBundle={tunnelBundle}
                            serverUrl={serverUrl}
                          />
                        ) : (
                          <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-3 py-1 text-sm text-blue-400">
                            Tunnel is enabled. Rotate the token to get a fresh remote install bundle.
                          </div>
                        )}

                        <div className="flex gap-2 flex-wrap">
                          {isTunnelConnected && (
                            !disconnectConfirmOpen ? (
                              <Button size="sm" variant="outline" className="text-red-400 border-red-500/30" disabled={tunnelActionLoading} onClick={() => setDisconnectConfirmOpen(true)}><Power className="h-3.5 w-3.5 mr-1" />Force Disconnect</Button>
                            ) : (
                              <>
                                <p className="text-xs text-red-400 self-center">This will forcefully disconnect the tunnel agent.</p>
                                <Button size="sm" variant="destructive" onClick={handleForceDisconnect} disabled={tunnelActionLoading}>Yes, Disconnect</Button>
                                <Button size="sm" variant="outline" onClick={() => setDisconnectConfirmOpen(false)}>Cancel</Button>
                              </>
                            )
                          )}
                          {!rotateConfirmOpen ? (
                            <Button size="sm" variant="outline" className="text-yellow-400 border-yellow-500/30" disabled={tunnelActionLoading} onClick={() => setRotateConfirmOpen(true)}><RefreshCw className="h-3.5 w-3.5 mr-1" />Rotate Token</Button>
                          ) : (
                            <><p className="text-xs text-yellow-400 self-center">Confirm rotate?</p><Button size="sm" className="bg-yellow-600 hover:bg-yellow-700" onClick={handleRotateTunnel} disabled={tunnelActionLoading}>Yes, Rotate</Button><Button size="sm" variant="outline" onClick={() => setRotateConfirmOpen(false)}>Cancel</Button></>
                          )}
                          {!revokeConfirmOpen ? (
                            <Button size="sm" variant="outline" className="text-red-400 border-red-500/30" disabled={tunnelActionLoading} onClick={() => setRevokeConfirmOpen(true)}><Trash2 className="h-3.5 w-3.5 mr-1" />Revoke Token</Button>
                          ) : (
                            <><p className="text-xs text-red-400 self-center">Confirm revoke?</p><Button size="sm" variant="destructive" onClick={handleRevokeTunnel} disabled={tunnelActionLoading}>Yes, Revoke</Button><Button size="sm" variant="outline" onClick={() => setRevokeConfirmOpen(false)}>Cancel</Button></>
                          )}
                        </div>

                        {isTunnelConnected && (
                          <Accordion type="single" collapsible value={tunnelMetricsOpen ? 'metrics' : ''} onValueChange={(v) => setUiPref('tunnelMetricsOpen', v === 'metrics')}>
                            <AccordionItem value="metrics">
                              <AccordionTrigger><div className="flex items-center gap-2"><Gauge className="h-4 w-4 text-primary" /><span className="text-sm font-medium">Live Metrics</span></div></AccordionTrigger>
                              <AccordionContent>
                                {tunnelMetricsLoading ? <div className="flex justify-center py-2"><Loader2 className="h-5 w-5 animate-spin" /></div>
                                : tunnelMetrics?.connectedAt ? (
                                  <div className="flex gap-1.5 flex-wrap">
                                    <Badge variant="outline">Uptime: {formatUptime(tunnelMetrics.connectedAt)}</Badge>
                                    <Badge variant="outline" className={tunnelMetrics.pingPongLatency != null && tunnelMetrics.pingPongLatency < 100 ? 'border-green-500/30 text-green-400' : ''}>RTT: {tunnelMetrics.pingPongLatency != null ? `${tunnelMetrics.pingPongLatency}ms` : 'N/A'}</Badge>
                                    <Badge variant="outline">Streams: {tunnelMetrics.activeStreams ?? 0}</Badge>
                                    <Badge variant="outline">Agent: {tunnelMetrics.clientVersion ?? 'unknown'}</Badge>
                                  </div>
                                ) : <p className="text-xs text-muted-foreground">No metrics available</p>}
                                <Button size="sm" variant="ghost" onClick={fetchTunnelMetrics} className="mt-2">Refresh</Button>
                              </AccordionContent>
                            </AccordionItem>
                          </Accordion>
                        )}

                        <Accordion type="single" collapsible value={tunnelEventLogOpen ? 'events' : ''} onValueChange={(v) => setUiPref('tunnelEventLogOpen', v === 'events')}>
                          <AccordionItem value="events">
                            <AccordionTrigger><div className="flex items-center gap-2"><History className="h-4 w-4" /><span className="text-sm font-medium">Connection Event Log</span></div></AccordionTrigger>
                            <AccordionContent>
                              {tunnelEventsLoading ? <div className="flex justify-center py-2"><Loader2 className="h-5 w-5 animate-spin" /></div>
                              : tunnelEvents.length === 0 ? <p className="text-xs text-muted-foreground">No tunnel events recorded yet.</p>
                              : (
                                <div className="max-h-[200px] overflow-auto space-y-1">
                                  {tunnelEvents.map((evt, idx) => (
                                    <div key={idx} className="flex items-center gap-2 py-0.5">
                                      <Badge className={evt.action === 'TUNNEL_CONNECT' ? 'bg-green-500/15 text-green-400 border-green-500/30 min-w-[85px] justify-center' : 'bg-red-500/15 text-red-400 border-red-500/30 min-w-[85px] justify-center'}>
                                        {evt.action === 'TUNNEL_CONNECT' ? 'Connect' : 'Disconnect'}
                                      </Badge>
                                      <span className="text-xs text-muted-foreground whitespace-nowrap">{new Date(evt.timestamp).toLocaleString()}</span>
                                      {evt.ipAddress && <span className="text-xs text-muted-foreground">{evt.ipAddress}</span>}
                                      {evt.details && typeof evt.details === 'object' && 'clientVersion' in evt.details && <span className="text-xs text-muted-foreground">v{String(evt.details.clientVersion)}</span>}
                                      {evt.details && typeof evt.details === 'object' && 'forced' in evt.details && <Badge variant="outline" className="text-yellow-400 border-yellow-500/30">Forced</Badge>}
                                    </div>
                                  ))}
                                </div>
                              )}
                              <Button size="sm" variant="ghost" onClick={fetchTunnelEvents} className="mt-2">Refresh</Button>
                            </AccordionContent>
                          </AccordionItem>
                        </Accordion>
                      </>
                    )}
                  </div>
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          )}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>{gatewayCreatedWithTunnel ? 'Close' : 'Cancel'}</Button>
          {!gatewayCreatedWithTunnel && (
            <Button onClick={handleSubmit} disabled={loading}>
              {loading
                ? (isEditMode ? 'Saving...' : 'Creating...')
                : (isEditMode ? 'Save' : enableTunnelOnCreate ? 'Create and Enable Tunnel' : 'Create')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function gatewayRuntimeDefaultPort(type: GatewayData['type']): string {
  switch (type) {
    case 'GUACD':
      return '4822';
    case 'MANAGED_SSH':
    case 'SSH_BASTION':
      return '2222';
    case 'DB_PROXY':
      return '5432';
    default:
      return '4822';
  }
}

function gatewayDirectDefaultPort(type: GatewayData['type']): string {
  switch (type) {
    case 'GUACD':
      return '4822';
    case 'MANAGED_SSH':
      return '2222';
    case 'DB_PROXY':
      return '5432';
    case 'SSH_BASTION':
    default:
      return '22';
  }
}
