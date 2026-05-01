import { useState, useEffect, useCallback } from 'react';
import {
  RefreshCw, CheckCircle, AlertCircle, Clock, Loader2,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Alert } from '@/components/ui/alert';
import { Switch } from '@/components/ui/switch';
import {
  Accordion, AccordionContent, AccordionItem, AccordionTrigger,
} from '@/components/ui/accordion';
import {
  getRotationStatus, enableRotation, disableRotation,
  triggerRotation, getRotationHistory,
} from '../../api/secrets.api';
import type { RotationStatusResult, RotationHistoryEntry } from '../../api/secrets.api';
import { extractApiError } from '../../utils/apiError';

interface PasswordRotationPanelProps {
  secretId: string;
  isReadOnly?: boolean;
}

const STATUS_ICONS: Record<string, React.ReactNode> = {
  SUCCESS: <CheckCircle className="h-4 w-4 text-green-500" />,
  FAILED: <AlertCircle className="h-4 w-4 text-destructive" />,
  PENDING: <Clock className="h-4 w-4 text-yellow-500" />,
};

export default function PasswordRotationPanel({ secretId, isReadOnly }: PasswordRotationPanelProps) {
  const [status, setStatus] = useState<RotationStatusResult | null>(null);
  const [history, setHistory] = useState<RotationHistoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [rotating, setRotating] = useState(false);
  const [toggling, setToggling] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [intervalDays, setIntervalDays] = useState(30);

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      const [s, h] = await Promise.all([
        getRotationStatus(secretId),
        getRotationHistory(secretId, 10),
      ]);
      setStatus(s);
      setHistory(h);
      setIntervalDays(s.intervalDays);
    } catch (err) {
      setError(extractApiError(err, 'Failed to load rotation status'));
    } finally {
      setLoading(false);
    }
  }, [secretId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleToggle = async (enabled: boolean) => {
    setToggling(true);
    setError(null);
    setSuccessMsg(null);
    try {
      if (enabled) {
        await enableRotation(secretId, intervalDays);
        setSuccessMsg('Password rotation enabled');
      } else {
        await disableRotation(secretId);
        setSuccessMsg('Password rotation disabled');
      }
      await fetchData();
    } catch (err) {
      setError(extractApiError(err, 'Failed to update rotation settings'));
    } finally {
      setToggling(false);
    }
  };

  const handleTrigger = async () => {
    setRotating(true);
    setError(null);
    setSuccessMsg(null);
    try {
      const result = await triggerRotation(secretId);
      if (result.success) {
        setSuccessMsg('Password rotated successfully');
      } else {
        setError(`Rotation failed: ${result.error || 'Unknown error'}`);
      }
      await fetchData();
    } catch (err) {
      setError(extractApiError(err, 'Failed to trigger rotation'));
    } finally {
      setRotating(false);
    }
  };

  const formatDate = (iso: string) =>
    new Date(iso).toLocaleDateString(undefined, {
      month: 'short', day: 'numeric', year: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });

  if (loading) {
    return (
      <div className="flex justify-center p-4">
        <Loader2 className="h-5 w-5 animate-spin" />
      </div>
    );
  }

  return (
    <div>
      {error && (
        <Alert variant="destructive" className="mb-2">
          <div className="flex items-center justify-between">
            <span>{error}</span>
            <button onClick={() => setError(null)} className="text-xs underline">Dismiss</button>
          </div>
        </Alert>
      )}
      {successMsg && (
        <Alert variant="success" className="mb-2">
          <div className="flex items-center justify-between">
            <span>{successMsg}</span>
            <button onClick={() => setSuccessMsg(null)} className="text-xs underline">Dismiss</button>
          </div>
        </Alert>
      )}

      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <Switch
            checked={status?.enabled ?? false}
            onCheckedChange={(checked) => handleToggle(checked)}
            disabled={isReadOnly || toggling}
          />
          <span className="text-sm">Auto-rotate password</span>
        </div>
        {status?.enabled && !isReadOnly && (
          <Button
            size="sm"
            variant="outline"
            onClick={handleTrigger}
            disabled={rotating}
          >
            {rotating ? <Loader2 className="h-4 w-4 mr-1 animate-spin" /> : <RefreshCw className="h-4 w-4 mr-1" />}
            Rotate Now
          </Button>
        )}
      </div>

      {status?.enabled && (
        <div className="mb-3">
          <div className="space-y-1.5 mb-2">
            <Label>Interval (days)</Label>
            <Input
              type="number"
              value={intervalDays}
              onChange={(e) => setIntervalDays(Math.max(1, parseInt(e.target.value, 10) || 1))}
              onBlur={() => {
                if (intervalDays !== status.intervalDays) {
                  handleToggle(true);
                }
              }}
              disabled={isReadOnly || toggling}
              min={1}
              max={365}
              className="w-36"
            />
          </div>
          <div className="flex gap-1 flex-wrap">
            {status.lastRotatedAt && (
              <Badge variant="outline">Last rotated: {formatDate(status.lastRotatedAt)}</Badge>
            )}
            {status.nextRotationAt && (
              <Badge variant="outline">Next: {formatDate(status.nextRotationAt)}</Badge>
            )}
          </div>
        </div>
      )}

      {/* Rotation history */}
      {history.length > 0 && (
        <Accordion type="single" collapsible className="mt-2">
          <AccordionItem value="history">
            <AccordionTrigger>
              <span className="text-sm">Rotation History ({history.length})</span>
            </AccordionTrigger>
            <AccordionContent>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b">
                      <th className="text-left py-1.5 px-2 text-xs font-medium text-muted-foreground">Status</th>
                      <th className="text-left py-1.5 px-2 text-xs font-medium text-muted-foreground">Trigger</th>
                      <th className="text-left py-1.5 px-2 text-xs font-medium text-muted-foreground">Target</th>
                      <th className="text-left py-1.5 px-2 text-xs font-medium text-muted-foreground">Date</th>
                    </tr>
                  </thead>
                  <tbody>
                    {history.map((entry) => (
                      <tr key={entry.id} className="border-b last:border-0">
                        <td className="py-1.5 px-2">
                          <div className="flex items-center gap-1" title={entry.errorMessage ?? entry.status}>
                            {STATUS_ICONS[entry.status]}
                            <span className="text-xs">{entry.status}</span>
                          </div>
                        </td>
                        <td className="py-1.5 px-2">
                          <span className="text-xs">{entry.trigger}</span>
                        </td>
                        <td className="py-1.5 px-2">
                          <span className="text-xs font-mono">
                            {entry.targetUser}@{entry.targetHost}
                          </span>
                        </td>
                        <td className="py-1.5 px-2">
                          <span className="text-xs">{formatDate(entry.createdAt)}</span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </AccordionContent>
          </AccordionItem>
        </Accordion>
      )}
    </div>
  );
}
