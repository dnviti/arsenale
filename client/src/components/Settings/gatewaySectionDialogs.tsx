import { Loader2, Network } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { GatewayData } from '../../api/gateway.api';

export function GatewayDeleteDialog({
  deleting,
  gateway,
  onConfirm,
  onOpenChange,
}: {
  deleting: boolean;
  gateway: GatewayData | null;
  onConfirm: () => void;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={Boolean(gateway)} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Delete Gateway</DialogTitle>
          <DialogDescription>
            Delete <strong>{gateway?.name}</strong>? Connections using this gateway will fall back
            to direct routing.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" variant="destructive" disabled={deleting} onClick={onConfirm}>
            {deleting ? <Loader2 className="size-4 animate-spin" /> : null}
            {deleting ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function GatewayForceDeleteDialog({
  deleting,
  gateway,
  connectionCount,
  onConfirm,
  onOpenChange,
}: {
  connectionCount: number;
  deleting: boolean;
  gateway: GatewayData | null;
  onConfirm: () => void;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={Boolean(gateway)} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Gateway In Use</DialogTitle>
          <DialogDescription>
            <strong>{gateway?.name}</strong> is still assigned to <strong>{connectionCount}</strong>{' '}
            connection(s). Deleting it will remove the gateway reference from those connections.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" variant="destructive" disabled={deleting} onClick={onConfirm}>
            {deleting ? <Loader2 className="size-4 animate-spin" /> : null}
            {deleting ? 'Deleting...' : 'Delete Anyway'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function GatewayRotateKeyDialog({
  loading,
  onConfirm,
  onOpenChange,
  open,
}: {
  loading: boolean;
  onConfirm: () => void;
  onOpenChange: (open: boolean) => void;
  open: boolean;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Rotate SSH Key Pair</DialogTitle>
          <DialogDescription>
            This generates a new tenant key pair and automatically pushes the public key to every
            managed SSH gateway. Existing connections can fail briefly during the changeover.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" disabled={loading} onClick={onConfirm}>
            {loading ? <Loader2 className="size-4 animate-spin" /> : <Network className="size-4" />}
            {loading ? 'Rotating...' : 'Rotate Key Pair'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
