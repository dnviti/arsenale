import { AlertCircle, CheckCircle2, Pencil, Plus, SkipForward } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import type { DiscoveredDeviceData, SyncPlanData } from '../../api/sync.api';

interface SyncPreviewDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  plan: SyncPlanData | null;
  confirming: boolean;
}

type PreviewEntry =
  | DiscoveredDeviceData
  | { device: DiscoveredDeviceData; changes: string[] }
  | { device: DiscoveredDeviceData; reason?: string; error?: string };

function getEntryDevice(entry: PreviewEntry): DiscoveredDeviceData {
  return 'device' in entry ? entry.device : entry;
}

function getEntryDetails(entry: PreviewEntry): string {
  if ('changes' in entry) {
    return entry.changes.join(', ');
  }
  if ('reason' in entry && entry.reason) {
    return entry.reason;
  }
  if ('error' in entry && entry.error) {
    return entry.error;
  }

  const device = getEntryDevice(entry);
  const location = [device.siteName, device.rackName].filter(Boolean).join(' / ');
  return location || device.description || 'No additional details';
}

function PreviewSection({
  title,
  tone,
  icon,
  entries,
}: {
  title: string;
  tone: 'success' | 'info' | 'warning' | 'destructive';
  icon: React.ReactNode;
  entries: PreviewEntry[];
}) {
  if (entries.length === 0) return null;

  return (
    <section className="space-y-3">
      <div className="flex items-center gap-2">
        <Badge variant="outline" className={cn(
          tone === 'success' && 'border-primary/30 bg-primary/10 text-foreground',
          tone === 'info' && 'border-border bg-background text-foreground',
          tone === 'warning' && 'border-chart-5/30 bg-chart-5/10 text-foreground',
          tone === 'destructive' && 'border-destructive/30 bg-destructive/10 text-destructive',
        )}>
          {icon}
          {title}
        </Badge>
        <span className="text-sm text-muted-foreground">{entries.length}</span>
      </div>

      <div className="space-y-2">
        {entries.map((entry) => {
          const device = getEntryDevice(entry);

          return (
            <div
              key={device.externalId}
              className="rounded-xl border border-border/70 bg-background/70 p-4"
            >
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="text-sm font-medium text-foreground">{device.name}</div>
                <Badge variant="outline">{device.protocol}</Badge>
              </div>
              <div className="mt-2 text-sm text-muted-foreground">
                {device.host}:{device.port}
              </div>
              <div className="mt-2 text-sm leading-6 text-muted-foreground">
                {getEntryDetails(entry)}
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}

export default function SyncPreviewDialog({
  open,
  onClose,
  onConfirm,
  plan,
  confirming,
}: SyncPreviewDialogProps) {
  if (!plan) return null;

  const totalItems =
    plan.toCreate.length +
    plan.toUpdate.length +
    plan.toSkip.length +
    plan.errors.length;
  const canConfirm = plan.toCreate.length > 0 || plan.toUpdate.length > 0;

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) onClose(); }}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>Sync Preview</DialogTitle>
          <DialogDescription>
            Review the import plan before applying changes to your connection inventory.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-wrap gap-2">
          <Badge variant="outline"><Plus />Create {plan.toCreate.length}</Badge>
          <Badge variant="outline"><Pencil />Update {plan.toUpdate.length}</Badge>
          <Badge variant="outline"><SkipForward />Skip {plan.toSkip.length}</Badge>
          {plan.errors.length > 0 && (
            <Badge variant="outline" className="border-destructive/30 bg-destructive/10 text-destructive">
              <AlertCircle />
              Errors {plan.errors.length}
            </Badge>
          )}
        </div>

        {totalItems === 0 ? (
          <Alert variant="info">
            <AlertDescription>No changes to apply.</AlertDescription>
          </Alert>
        ) : (
          <ScrollArea className="max-h-[60vh] pr-4">
            <div className="space-y-5">
              <PreviewSection
                title="Create"
                tone="success"
                icon={<Plus className="mr-1 size-3.5" />}
                entries={plan.toCreate}
              />
              <PreviewSection
                title="Update"
                tone="info"
                icon={<Pencil className="mr-1 size-3.5" />}
                entries={plan.toUpdate}
              />
              <PreviewSection
                title="Skip"
                tone="warning"
                icon={<SkipForward className="mr-1 size-3.5" />}
                entries={plan.toSkip}
              />
              <PreviewSection
                title="Errors"
                tone="destructive"
                icon={<AlertCircle className="mr-1 size-3.5" />}
                entries={plan.errors}
              />
            </div>
          </ScrollArea>
        )}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={confirming}>
            Cancel
          </Button>
          <Button
            type="button"
            onClick={onConfirm}
            disabled={confirming || !canConfirm}
          >
            {confirming ? <CheckCircle2 className="animate-pulse" /> : null}
            {confirming ? 'Importing...' : 'Confirm Import'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
