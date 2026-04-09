import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select';
import { useGatewayStore } from '../../store/gatewayStore';
import type { GatewayDeploymentMode, GatewayTemplateData } from '../../api/gateway.api';
import SessionTimeoutConfig from '../orchestration/SessionTimeoutConfig';
import { extractApiError } from '../../utils/apiError';

interface GatewayTemplateDialogProps {
  open: boolean;
  onClose: () => void;
  template?: GatewayTemplateData | null;
}

export default function GatewayTemplateDialog({ open, onClose, template }: GatewayTemplateDialogProps) {
  const [name, setName] = useState('');
  const [type, setType] = useState<'GUACD' | 'SSH_BASTION' | 'MANAGED_SSH' | 'DB_PROXY'>('MANAGED_SSH');
  const [deploymentMode, setDeploymentMode] = useState<GatewayDeploymentMode>('MANAGED_GROUP');
  const [host, setHost] = useState('');
  const [port, setPort] = useState('');
  const [description, setDescription] = useState('');
  const [apiPort, setApiPort] = useState('');
  const [monitoringEnabled, setMonitoringEnabled] = useState(true);
  const [monitorIntervalMs, setMonitorIntervalMs] = useState('5000');
  const [inactivityTimeout, setInactivityTimeout] = useState('60');
  const [autoScaleEnabled, setAutoScaleEnabled] = useState(false);
  const [minReplicasVal, setMinReplicasVal] = useState('1');
  const [maxReplicasVal, setMaxReplicasVal] = useState('5');
  const [sessPerInstance, setSessPerInstance] = useState('10');
  const [cooldownVal, setCooldownVal] = useState('300');
  const [publishPorts, setPublishPorts] = useState(false);
  const [lbStrategy, setLbStrategy] = useState<'ROUND_ROBIN' | 'LEAST_CONNECTIONS'>('ROUND_ROBIN');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const createTemplate = useGatewayStore((s) => s.createTemplate);
  const updateTemplate = useGatewayStore((s) => s.updateTemplate);

  const isEditMode = Boolean(template);

  useEffect(() => {
    if (open && template) {
      setName(template.name);
      setType(template.type);
      setDeploymentMode(template.deploymentMode ?? ((template.type === 'SSH_BASTION' || template.host) ? 'SINGLE_INSTANCE' : 'MANAGED_GROUP'));
      setHost(template.host);
      setPort(String(template.port));
      setDescription(template.description || '');
      setApiPort(template.apiPort ? String(template.apiPort) : '');
      setMonitoringEnabled(template.monitoringEnabled);
      setMonitorIntervalMs(String(template.monitorIntervalMs));
      setInactivityTimeout(String(Math.floor(template.inactivityTimeoutSeconds / 60)));
      setAutoScaleEnabled(template.autoScale);
      setMinReplicasVal(String(template.minReplicas));
      setMaxReplicasVal(String(template.maxReplicas));
      setSessPerInstance(String(template.sessionsPerInstance));
      setCooldownVal(String(template.scaleDownCooldownSeconds));
      setPublishPorts(template.publishPorts ?? false);
      setLbStrategy(template.lbStrategy ?? 'ROUND_ROBIN');
    } else if (open) {
      setName(''); setType('MANAGED_SSH'); setDeploymentMode('MANAGED_GROUP');
      setHost(''); setPort('2222'); setDescription(''); setApiPort('9022');
      setMonitoringEnabled(true); setMonitorIntervalMs('5000'); setInactivityTimeout('60');
      setAutoScaleEnabled(false); setMinReplicasVal('1'); setMaxReplicasVal('5');
      setSessPerInstance('10'); setCooldownVal('300'); setPublishPorts(false); setLbStrategy('ROUND_ROBIN');
    }
    setError('');
  }, [open, template]);

  const handleTypeChange = (newType: 'GUACD' | 'SSH_BASTION' | 'MANAGED_SSH' | 'DB_PROXY') => {
    setType(newType);
    if (newType === 'SSH_BASTION') setDeploymentMode('SINGLE_INSTANCE');
    const defaultPort = newType === 'GUACD' ? '4822' : newType === 'MANAGED_SSH' ? '2222' : newType === 'DB_PROXY' ? '5432' : '22';
    if (!port || port === '4822' || port === '2222' || port === '5432' || port === '22') setPort(defaultPort);
    if (newType === 'MANAGED_SSH' && !apiPort) setApiPort('9022');
    else if (newType !== 'MANAGED_SSH') setApiPort('');
  };

  const isManagedType = type === 'MANAGED_SSH' || type === 'GUACD' || type === 'DB_PROXY';
  const isGroupMode = deploymentMode === 'MANAGED_GROUP';

  const handleSave = async () => {
    if (!name.trim()) { setError('Name is required'); return; }
    if (!isGroupMode && !host.trim()) { setError('Host is required for SSH Bastion gateways'); return; }
    if (!port.trim()) { setError('Port is required'); return; }
    setLoading(true); setError('');
    try {
      const data = {
        name: name.trim(), type, deploymentMode,
        host: isGroupMode ? '' : host.trim(), port: parseInt(port, 10),
        description: description.trim() || undefined,
        apiPort: apiPort ? parseInt(apiPort, 10) : undefined,
        monitoringEnabled, monitorIntervalMs: parseInt(monitorIntervalMs, 10),
        inactivityTimeoutSeconds: parseInt(inactivityTimeout, 10) * 60,
        autoScale: autoScaleEnabled, minReplicas: parseInt(minReplicasVal, 10),
        maxReplicas: parseInt(maxReplicasVal, 10), sessionsPerInstance: parseInt(sessPerInstance, 10),
        scaleDownCooldownSeconds: parseInt(cooldownVal, 10), publishPorts, lbStrategy,
      };
      if (isEditMode && template) { await updateTemplate(template.id, data); }
      else { await createTemplate(data); }
      onClose();
    } catch (err: unknown) {
      setError(extractApiError(err, `Failed to ${isEditMode ? 'update' : 'create'} template`));
    } finally { setLoading(false); }
  };

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose(); }}>
      <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEditMode ? 'Edit Gateway Template' : 'New Gateway Template'}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          {error && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">{error}</div>
          )}

          <div className="space-y-1.5">
            <Label>Template Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} required />
          </div>

          <div className="space-y-1.5">
            <Label>Gateway Type</Label>
            <Select value={type} onValueChange={(v) => handleTypeChange(v as typeof type)}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="GUACD">GUACD (RDP/VNC proxy)</SelectItem>
                <SelectItem value="SSH_BASTION">SSH Bastion</SelectItem>
                <SelectItem value="MANAGED_SSH">Managed SSH</SelectItem>
                <SelectItem value="DB_PROXY">DB Proxy (Database Gateway)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {isManagedType ? (
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
          ) : (
            <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-3 py-2 text-sm text-blue-400">
              SSH bastion templates are always single-instance.
            </div>
          )}

          {isGroupMode ? (
            <div className="space-y-2">
              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 px-3 py-2 text-sm text-blue-400">
                This template creates a logical gateway group. Instances get their own runtime addresses when deployed.
              </div>
              <div className="space-y-1.5">
                <Label>Service Port</Label>
                <Input value={port} onChange={(e) => setPort(e.target.value)} type="number" required />
                {publishPorts && <p className="text-xs text-muted-foreground">External host ports are assigned per instance at deploy time.</p>}
              </div>
            </div>
          ) : (
            <div className="flex gap-3">
              <div className="flex-1 space-y-1.5">
                <Label>Host</Label>
                <Input value={host} onChange={(e) => setHost(e.target.value)} required />
              </div>
              <div className="w-[120px] space-y-1.5">
                <Label>Port</Label>
                <Input value={port} onChange={(e) => setPort(e.target.value)} type="number" required />
              </div>
            </div>
          )}

          {isManagedType && isGroupMode && (
            <div className="flex items-center gap-3">
              <Switch checked={publishPorts} onCheckedChange={setPublishPorts} />
              <Label>Publish Ports (external access)</Label>
            </div>
          )}

          {isManagedType && isGroupMode && (
            <div className="space-y-1.5">
              <Label>Load Balancing Strategy</Label>
              <Select value={lbStrategy} onValueChange={(v) => setLbStrategy(v as 'ROUND_ROBIN' | 'LEAST_CONNECTIONS')}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="ROUND_ROBIN">Round Robin</SelectItem>
                  <SelectItem value="LEAST_CONNECTIONS">Least Connections</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}

          <div className="space-y-1.5">
            <Label>Description</Label>
            <Textarea value={description} onChange={(e) => setDescription(e.target.value)} rows={2} />
          </div>

          {type === 'MANAGED_SSH' && (
            <div className="space-y-1.5">
              <Label>gRPC Port (key management)</Label>
              <Input value={apiPort} onChange={(e) => setApiPort(e.target.value)} type="number" disabled={publishPorts} />
              {publishPorts && <p className="text-xs text-muted-foreground">Auto-assigned at deploy</p>}
            </div>
          )}

          <div className="space-y-2">
            <div className="flex items-center gap-3">
              <Switch checked={monitoringEnabled} onCheckedChange={setMonitoringEnabled} />
              <Label>Enable health monitoring</Label>
            </div>
            {monitoringEnabled && (
              <div className="space-y-1.5">
                <Label>Monitor Interval (ms)</Label>
                <Input value={monitorIntervalMs} onChange={(e) => setMonitorIntervalMs(e.target.value)} type="number" />
              </div>
            )}
          </div>

          <SessionTimeoutConfig value={inactivityTimeout} onChange={setInactivityTimeout} />

          {isManagedType && isGroupMode && (
            <div className="rounded-lg border p-4 space-y-3">
              <p className="text-sm font-medium">Auto-Scaling Configuration</p>
              <div className="flex items-center gap-3">
                <Switch checked={autoScaleEnabled} onCheckedChange={setAutoScaleEnabled} />
                <Label>Enable auto-scaling</Label>
              </div>
              {autoScaleEnabled && (
                <div className="space-y-3">
                  <div className="flex gap-3">
                    <div className="flex-1 space-y-1"><Label className="text-xs">Min Replicas</Label><Input value={minReplicasVal} onChange={(e) => setMinReplicasVal(e.target.value)} type="number" className="h-8" /></div>
                    <div className="flex-1 space-y-1"><Label className="text-xs">Max Replicas</Label><Input value={maxReplicasVal} onChange={(e) => setMaxReplicasVal(e.target.value)} type="number" className="h-8" /></div>
                  </div>
                  <div className="space-y-1"><Label className="text-xs">Sessions per Instance</Label><Input value={sessPerInstance} onChange={(e) => setSessPerInstance(e.target.value)} type="number" className="h-8" /></div>
                  <div className="space-y-1"><Label className="text-xs">Scale-Down Cooldown (seconds)</Label><Input value={cooldownVal} onChange={(e) => setCooldownVal(e.target.value)} type="number" className="h-8" /></div>
                </div>
              )}
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleSave} disabled={loading}>
            {loading ? 'Saving...' : isEditMode ? 'Update Template' : 'Create Template'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
