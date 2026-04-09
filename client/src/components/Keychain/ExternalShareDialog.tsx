import { useState, useEffect } from 'react';
import { Trash2, Copy, Link } from 'lucide-react';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Alert } from '@/components/ui/alert';
import { Switch } from '@/components/ui/switch';
import { Separator } from '@/components/ui/separator';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { cn } from '@/lib/utils';
import {
  createExternalShare, listExternalShares, revokeExternalShare,
} from '../../api/secrets.api';
import { useNotificationStore } from '../../store/notificationStore';
import type { ExternalShareResult, ExternalShareListItem } from '../../api/secrets.api';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import { useCopyToClipboard } from '../../hooks/useCopyToClipboard';

interface ExternalShareDialogProps {
  open: boolean;
  onClose: () => void;
  secretId: string;
  secretName: string;
}

const EXPIRY_OPTIONS = [
  { label: '1 hour', value: 60 },
  { label: '24 hours', value: 1440 },
  { label: '7 days', value: 10080 },
  { label: '30 days', value: 43200 },
];

export default function ExternalShareDialog({
  open,
  onClose,
  secretId,
  secretName,
}: ExternalShareDialogProps) {
  const [expiresInMinutes, setExpiresInMinutes] = useState(1440);
  const [maxAccessCount, setMaxAccessCount] = useState('');
  const [usePin, setUsePin] = useState(false);
  const [pin, setPin] = useState('');
  const { loading, error, setError, run } = useAsyncAction();
  const [result, setResult] = useState<ExternalShareResult | null>(null);
  const { copied, copy: copyToClipboard } = useCopyToClipboard();
  const [shares, setShares] = useState<ExternalShareListItem[]>([]);
  const notify = useNotificationStore((s) => s.notify);

  useEffect(() => {
    if (open && secretId) {
      loadShares();
      setResult(null);
      setError('');
      setPin('');
      setUsePin(false);
      setMaxAccessCount('');
      setExpiresInMinutes(1440);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, secretId]);

  const loadShares = async () => {
    try {
      const data = await listExternalShares(secretId);
      setShares(data);
    } catch {
      // silently fail
    }
  };

  const handleCreate = async () => {
    if (usePin && !/^\d{4,8}$/.test(pin)) {
      setError('PIN must be 4-8 digits');
      return;
    }
    const input: { expiresInMinutes: number; maxAccessCount?: number; pin?: string } = {
      expiresInMinutes,
    };
    if (maxAccessCount) {
      const count = parseInt(maxAccessCount, 10);
      if (isNaN(count) || count < 1) {
        setError('Max access count must be a positive number');
        return;
      }
      input.maxAccessCount = count;
    }
    if (usePin && pin) {
      input.pin = pin;
    }
    await run(async () => {
      const res = await createExternalShare(secretId, input);
      setResult(res);
      notify('Share link created successfully!', 'success');
      await loadShares();
    }, 'Failed to create external share');
  };

  const handleCopy = () => {
    if (result?.shareUrl) {
      copyToClipboard(result.shareUrl);
    }
  };

  const handleRevoke = async (shareId: string) => {
    try {
      await revokeExternalShare(shareId);
      await loadShares();
    } catch {
      // silently fail
    }
  };

  const formatExpiry = (iso: string) => {
    const d = new Date(iso);
    const now = new Date();
    const diffMs = d.getTime() - now.getTime();
    if (diffMs <= 0) return 'Expired';
    const hours = Math.floor(diffMs / (1000 * 60 * 60));
    if (hours < 24) return `${hours}h left`;
    const days = Math.floor(hours / 24);
    return `${days}d left`;
  };

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose(); }}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>External Share: {secretName}</DialogTitle>
          <DialogDescription className="sr-only">Create an external share link for this secret</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          {error && <Alert variant="destructive">{error}</Alert>}

          {result ? (
            <div>
              <div className="relative">
                <Input
                  readOnly
                  value={result.shareUrl}
                  className="pr-10"
                />
                <Button
                  variant="ghost"
                  size="icon"
                  className="absolute right-1 top-1/2 -translate-y-1/2 h-7 w-7"
                  onClick={handleCopy}
                  title={copied ? 'Copied!' : 'Copy link'}
                >
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
              {result.hasPin && (
                <p className="text-xs text-muted-foreground mt-1">
                  The recipient will need the PIN to access this secret.
                </p>
              )}
              <Button
                variant="outline"
                className="w-full mt-4"
                onClick={() => setResult(null)}
              >
                Create Another Link
              </Button>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="space-y-1.5">
                <Label>Expires in</Label>
                <Select value={String(expiresInMinutes)} onValueChange={(v) => setExpiresInMinutes(Number(v))}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {EXPIRY_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={String(opt.value)}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-1.5">
                <Label>Max access count (optional)</Label>
                <Input
                  value={maxAccessCount}
                  onChange={(e) => setMaxAccessCount(e.target.value)}
                  type="number"
                  placeholder="Unlimited"
                />
              </div>

              <div className="flex items-center gap-2">
                <Switch
                  checked={usePin}
                  onCheckedChange={(checked) => {
                    setUsePin(checked);
                    if (!checked) setPin('');
                  }}
                />
                <Label>Require PIN</Label>
              </div>

              {usePin && (
                <div className="space-y-1.5">
                  <Label>PIN (4-8 digits)</Label>
                  <Input
                    value={pin}
                    onChange={(e) => setPin(e.target.value.replace(/\D/g, '').slice(0, 8))}
                    placeholder="e.g. 1234"
                  />
                </div>
              )}

              <Button
                className="w-full"
                onClick={handleCreate}
                disabled={loading}
              >
                <Link className="h-4 w-4 mr-2" />
                {loading ? 'Creating...' : 'Create Share Link'}
              </Button>
            </div>
          )}

          {shares.length > 0 && (
            <>
              <Separator />
              <h4 className="text-sm font-medium">Existing Links</h4>
              <div className="space-y-1">
                {shares.map((share) => {
                  const isActive = !share.isRevoked &&
                    new Date(share.expiresAt) > new Date() &&
                    (share.maxAccessCount === null || share.accessCount < share.maxAccessCount);
                  return (
                    <div key={share.id} className="flex items-center justify-between py-2 px-1">
                      <div>
                        <div className="flex items-center gap-1.5">
                          <span className="text-sm">
                            {share.accessCount} access{share.accessCount !== 1 ? 'es' : ''}
                            {share.maxAccessCount !== null && ` / ${share.maxAccessCount}`}
                          </span>
                          {share.hasPin && <Badge variant="outline" className="text-[0.65rem] px-1.5 py-0">PIN</Badge>}
                          <Badge
                            variant={share.isRevoked ? 'destructive' : isActive ? 'default' : 'secondary'}
                            className="text-[0.65rem] px-1.5 py-0"
                          >
                            {share.isRevoked ? 'Revoked' : isActive ? formatExpiry(share.expiresAt) : 'Expired'}
                          </Badge>
                        </div>
                        <p className="text-xs text-muted-foreground mt-0.5">
                          Created {new Date(share.createdAt).toLocaleDateString()}
                        </p>
                      </div>
                      {isActive && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          onClick={() => handleRevoke(share.id)}
                          title="Revoke"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      )}
                    </div>
                  );
                })}
              </div>
            </>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
