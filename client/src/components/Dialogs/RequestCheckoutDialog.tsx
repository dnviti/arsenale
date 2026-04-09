import { useState } from 'react';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
  DialogDescription, DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select, SelectTrigger, SelectValue, SelectContent, SelectItem,
} from '@/components/ui/select';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import { requestCheckout } from '../../api/checkout.api';

interface RequestCheckoutDialogProps {
  open: boolean;
  onClose: () => void;
  /** Pre-filled target. Exactly one should be provided. */
  secretId?: string;
  connectionId?: string;
  resourceName: string;
}

const DURATION_OPTIONS = [
  { value: 15, label: '15 minutes' },
  { value: 30, label: '30 minutes' },
  { value: 60, label: '1 hour' },
  { value: 120, label: '2 hours' },
  { value: 240, label: '4 hours' },
  { value: 480, label: '8 hours' },
  { value: 1440, label: '24 hours' },
];

export default function RequestCheckoutDialog({
  open,
  onClose,
  secretId,
  connectionId,
  resourceName,
}: RequestCheckoutDialogProps) {
  const [durationMinutes, setDurationMinutes] = useState(60);
  const [reason, setReason] = useState('');
  const { loading, error, setError, run } = useAsyncAction();

  const handleSubmit = async () => {
    const success = await run(async () => {
      await requestCheckout({
        secretId,
        connectionId,
        durationMinutes,
        reason: reason.trim() || undefined,
      });
    }, 'Failed to submit checkout request');
    if (success) {
      setReason('');
      setDurationMinutes(60);
      onClose();
    }
  };

  const handleClose = () => {
    setError('');
    setReason('');
    setDurationMinutes(60);
    onClose();
  };

  const resourceType = secretId ? 'secret' : 'connection';

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) handleClose(); }}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Request Temporary Access</DialogTitle>
          <DialogDescription>
            Request temporary access to {resourceType} <strong>{resourceName}</strong>.
            The owner or an administrator will be notified to approve your request.
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
            {error}
          </div>
        )}

        <div className="space-y-2">
          <Label>Duration</Label>
          <Select value={String(durationMinutes)} onValueChange={(v) => setDurationMinutes(Number(v))}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {DURATION_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={String(opt.value)}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="checkout-reason">Reason (optional)</Label>
          <Textarea
            id="checkout-reason"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            rows={2}
            maxLength={500}
          />
          <p className="text-xs text-muted-foreground">{reason.length}/500</p>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? 'Submitting...' : 'Request Access'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
