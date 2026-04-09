import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Slider } from '@/components/ui/slider';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Rocket, Trash2, Save } from 'lucide-react';
import { useGatewayStore } from '../../store/gatewayStore';
import type { GatewayData } from '../../api/gateway.api';
import { isGatewayGroup } from '../../utils/gatewayMode';

interface ScalingControlsProps {
  gatewayId: string;
  gateway: GatewayData;
}

const recommendationBadgeClass: Record<string, string> = {
  stable: 'bg-green-500/15 text-green-400 border-green-500/30',
  'scale-up': 'bg-blue-500/15 text-blue-400 border-blue-500/30',
  'scale-down': 'bg-yellow-500/15 text-yellow-400 border-yellow-500/30',
};

export default function ScalingControls({ gatewayId, gateway }: ScalingControlsProps) {
  const scalingStatus = useGatewayStore((s) => s.scalingStatus[gatewayId]);
  const fetchScalingStatus = useGatewayStore((s) => s.fetchScalingStatus);
  const watchScalingStatus = useGatewayStore((s) => s.watchScalingStatus);
  const unwatchScalingStatus = useGatewayStore((s) => s.unwatchScalingStatus);
  const deployGatewayAction = useGatewayStore((s) => s.deployGateway);
  const undeployGatewayAction = useGatewayStore((s) => s.undeployGateway);
  const scaleGatewayAction = useGatewayStore((s) => s.scaleGateway);
  const updateScalingConfigAction = useGatewayStore((s) => s.updateScalingConfig);

  const [replicas, setReplicas] = useState(gateway.desiredReplicas);
  const [autoScale, setAutoScale] = useState(gateway.autoScale);
  const [minReplicas, setMinReplicas] = useState(String(gateway.minReplicas));
  const [maxReplicas, setMaxReplicas] = useState(String(gateway.maxReplicas));
  const [sessionsPerInstance, setSessionsPerInstance] = useState(String(gateway.sessionsPerInstance));
  const [cooldown, setCooldown] = useState(String(gateway.scaleDownCooldownSeconds));
  const [undeployOpen, setUndeployOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const isGroup = isGatewayGroup(gateway);

  useEffect(() => {
    if (!isGroup) return undefined;

    watchScalingStatus(gatewayId);
    void fetchScalingStatus(gatewayId);

    return () => {
      unwatchScalingStatus(gatewayId);
    };
  }, [gatewayId, isGroup, fetchScalingStatus, watchScalingStatus, unwatchScalingStatus]);

  useEffect(() => {
    setReplicas(gateway.desiredReplicas);
    setAutoScale(gateway.autoScale);
    setMinReplicas(String(gateway.minReplicas));
    setMaxReplicas(String(gateway.maxReplicas));
    setSessionsPerInstance(String(gateway.sessionsPerInstance));
    setCooldown(String(gateway.scaleDownCooldownSeconds));
  }, [
    gateway.desiredReplicas,
    gateway.autoScale,
    gateway.minReplicas,
    gateway.maxReplicas,
    gateway.sessionsPerInstance,
    gateway.scaleDownCooldownSeconds,
  ]);

  useEffect(() => {
    if (scalingStatus && gateway.autoScale) {
      setReplicas(scalingStatus.targetReplicas);
    }
  }, [scalingStatus, gateway.autoScale]);

  const handleDeploy = async () => {
    setLoading(true);
    setError(null);
    try {
      await deployGatewayAction(gatewayId);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const handleUndeploy = async () => {
    setLoading(true);
    setError(null);
    setUndeployOpen(false);
    try {
      await undeployGatewayAction(gatewayId);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const handleScale = async () => {
    setLoading(true);
    setError(null);
    try {
      await scaleGatewayAction(gatewayId, replicas);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const handleSaveConfig = async () => {
    setLoading(true);
    setError(null);
    try {
      await updateScalingConfigAction(gatewayId, {
        autoScale,
        minReplicas: Number(minReplicas),
        maxReplicas: Number(maxReplicas),
        sessionsPerInstance: Number(sessionsPerInstance),
        scaleDownCooldownSeconds: Number(cooldown),
      });
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mt-2">
      {error && (
        <div className="mb-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400 flex items-center justify-between">
          <span>{error}</span>
          <button onClick={() => setError(null)} className="text-red-400 hover:text-red-300 text-xs">dismiss</button>
        </div>
      )}

      {/* Deploy / Undeploy */}
      <div className="flex items-center gap-3 mb-3">
        {!isGroup ? (
          <Button size="sm" onClick={handleDeploy} disabled={loading}>
            <Rocket className="h-4 w-4 mr-1" />
            Deploy
          </Button>
        ) : (
          <Button variant="outline" size="sm" className="text-red-400 border-red-500/30 hover:bg-red-500/10" onClick={() => setUndeployOpen(true)} disabled={loading}>
            <Trash2 className="h-4 w-4 mr-1" />
            Undeploy All
          </Button>
        )}
      </div>

      {/* Replicas slider */}
      {isGroup && !gateway.autoScale && (
        <div className="rounded-lg border p-4 mb-3">
          <p className="text-sm font-medium mb-2">Manual Scaling</p>
          <div className="flex items-center gap-3">
            <span className="text-sm min-w-[60px]">
              Replicas: {replicas}
            </span>
            <Slider
              value={[replicas]}
              onValueChange={(v) => setReplicas(v[0])}
              min={0}
              max={10}
              step={1}
              className="flex-1 max-w-[300px]"
            />
            <Button
              variant="outline"
              size="sm"
              onClick={handleScale}
              disabled={loading || replicas === gateway.desiredReplicas}
            >
              Apply
            </Button>
          </div>
        </div>
      )}

      {/* Auto-Scale config */}
      {isGroup && (
        <div className="rounded-lg border p-4 mb-3">
          <div className="flex items-center gap-3 mb-2">
            <Switch checked={autoScale} onCheckedChange={setAutoScale} />
            <span className="text-sm font-medium">Auto-Scale</span>
          </div>

          {autoScale && (
            <div className="space-y-3 mt-3">
              <div className="flex flex-wrap gap-3">
                <div className="w-[130px]">
                  <Label className="text-xs">Min Replicas</Label>
                  <Input type="number" value={minReplicas} onChange={(e) => setMinReplicas(e.target.value)} min={0} max={20} className="h-8" />
                </div>
                <div className="w-[130px]">
                  <Label className="text-xs">Max Replicas</Label>
                  <Input type="number" value={maxReplicas} onChange={(e) => setMaxReplicas(e.target.value)} min={1} max={20} className="h-8" />
                </div>
                <div className="w-[160px]">
                  <Label className="text-xs">Sessions/Instance</Label>
                  <Input type="number" value={sessionsPerInstance} onChange={(e) => setSessionsPerInstance(e.target.value)} min={1} max={100} className="h-8" />
                </div>
                <div className="w-[130px]">
                  <Label className="text-xs">Cooldown (s)</Label>
                  <Input type="number" value={cooldown} onChange={(e) => setCooldown(e.target.value)} min={60} max={3600} className="h-8" />
                </div>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleSaveConfig}
                disabled={loading}
              >
                <Save className="h-4 w-4 mr-1" />
                Save Scaling Config
              </Button>
            </div>
          )}
        </div>
      )}

      {/* Scaling status */}
      {scalingStatus && isGroup && (
        <div className="rounded-lg border p-4">
          <p className="text-sm font-medium mb-2">Scaling Status</p>
          <div className="flex items-center gap-2 flex-wrap">
            <Badge className={recommendationBadgeClass[scalingStatus.recommendation] ?? ''}>
              {scalingStatus.recommendation === 'stable' ? 'Stable'
                : scalingStatus.recommendation === 'scale-up' ? 'Scaling Up'
                : 'Scaling Down'}
            </Badge>
            <span className="text-sm">
              {scalingStatus.activeSessions} sessions across {scalingStatus.currentReplicas} instances
              (target: {scalingStatus.targetReplicas})
            </span>
            {scalingStatus.cooldownRemaining > 0 && (
              <span className="text-xs text-muted-foreground">
                Cooldown: {scalingStatus.cooldownRemaining}s remaining
              </span>
            )}
          </div>
          {scalingStatus.instanceSessions && scalingStatus.instanceSessions.length > 0 && (
            <div className="mt-2">
              <span className="text-xs text-muted-foreground">
                Per-instance distribution:
              </span>
              <div className="flex flex-wrap gap-1 mt-1">
                {scalingStatus.instanceSessions.map((is) => (
                  <Badge
                    key={is.instanceId}
                    variant="outline"
                    className={is.count === 0 ? '' : 'border-primary/50 text-primary'}
                  >
                    {is.containerName.split('-').pop()}: {is.count}
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Undeploy confirmation */}
      <Dialog open={undeployOpen} onOpenChange={(v) => { if (!v) setUndeployOpen(false); }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Undeploy Gateway</DialogTitle>
            <DialogDescription>
              This will remove all managed instances for <strong>{gateway.name}</strong>.
              Active sessions through this gateway will be terminated.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setUndeployOpen(false)}>Cancel</Button>
            <Button variant="destructive" onClick={handleUndeploy}>
              Undeploy
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
