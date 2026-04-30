import { useEffect, useState } from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { RotateCcw, FileText } from 'lucide-react';
import { useGatewayStore } from '../../store/gatewayStore';
import type { ManagedInstanceData } from '../../api/gateway.api';
import ContainerLogDialog from './ContainerLogDialog';

type InstanceStatus = 'PROVISIONING' | 'RUNNING' | 'STOPPED' | 'ERROR' | 'REMOVING';

const EMPTY_INSTANCES: ManagedInstanceData[] = [];

const statusBadgeClass: Record<InstanceStatus, string> = {
  PROVISIONING: 'bg-yellow-500/15 text-yellow-400 border-yellow-500/30',
  RUNNING: 'bg-green-500/15 text-green-400 border-green-500/30',
  STOPPED: '',
  ERROR: 'bg-red-500/15 text-red-400 border-red-500/30',
  REMOVING: 'bg-yellow-500/15 text-yellow-400 border-yellow-500/30',
};

interface GatewayInstanceListProps {
  gatewayId: string;
}

export default function GatewayInstanceList({ gatewayId }: GatewayInstanceListProps) {
  const instances = useGatewayStore((s) => s.instances[gatewayId] ?? EMPTY_INSTANCES);
  const fetchInstances = useGatewayStore((s) => s.fetchInstances);
  const watchInstances = useGatewayStore((s) => s.watchInstances);
  const unwatchInstances = useGatewayStore((s) => s.unwatchInstances);
  const restartInstance = useGatewayStore((s) => s.restartInstance);
  const [logsOpen, setLogsOpen] = useState(false);
  const [logsInstance, setLogsInstance] = useState<ManagedInstanceData | null>(null);

  useEffect(() => {
    watchInstances(gatewayId);
    void fetchInstances(gatewayId);
    return () => {
      unwatchInstances(gatewayId);
    };
  }, [gatewayId, fetchInstances, watchInstances, unwatchInstances]);

  if (instances.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4 text-center">
        No instances deployed
      </p>
    );
  }

  return (
    <>
    <div className="mt-2 rounded-lg border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b">
            <th className="text-left py-2 px-3 font-medium">Container ID</th>
            <th className="text-left py-2 px-3 font-medium">Name</th>
            <th className="text-left py-2 px-3 font-medium">Status</th>
            <th className="text-left py-2 px-3 font-medium">Health</th>
            <th className="text-left py-2 px-3 font-medium">Host:Port</th>
            <th className="text-left py-2 px-3 font-medium">Created</th>
            <th className="text-right py-2 px-3 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          {instances.map((inst) => (
            <tr key={inst.id} className="border-b border-border/50">
              <td className="py-2 px-3">
                <span className="font-mono text-sm" title={inst.containerId}>
                  {inst.containerId.slice(0, 12)}
                </span>
              </td>
              <td className="py-2 px-3">{inst.containerName}</td>
              <td className="py-2 px-3">
                <Badge className={statusBadgeClass[inst.status as InstanceStatus] ?? ''}>
                  {inst.status}
                </Badge>
              </td>
              <td className="py-2 px-3">
                <span
                  className={inst.healthStatus === 'healthy' ? 'text-green-400' : 'text-muted-foreground'}
                  title={inst.errorMessage || ''}
                >
                  {inst.healthStatus || 'N/A'}
                </span>
              </td>
              <td className="py-2 px-3">
                <span className="font-mono text-sm">
                  {inst.host}:{inst.port}
                </span>
              </td>
              <td className="py-2 px-3">
                <span className="text-xs">
                  {new Date(inst.createdAt).toLocaleString()}
                </span>
              </td>
              <td className="py-2 px-3 text-right">
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  onClick={() => { setLogsInstance(inst); setLogsOpen(true); }}
                  disabled={inst.status === 'PROVISIONING'}
                  title="View logs"
                >
                  <FileText className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  onClick={() => restartInstance(gatewayId, inst.id)}
                  disabled={inst.status !== 'RUNNING' && inst.status !== 'ERROR'}
                  title="Restart instance"
                >
                  <RotateCcw className="h-4 w-4" />
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
    <ContainerLogDialog
      open={logsOpen}
      onClose={() => setLogsOpen(false)}
      gatewayId={gatewayId}
      instance={logsInstance}
    />
    </>
  );
}
