import { useState, useEffect } from 'react';
import {
  Dialog, DialogContent, DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select, SelectTrigger, SelectValue, SelectContent, SelectItem,
} from '@/components/ui/select';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Download, Loader2, X } from 'lucide-react';
import { downloadExport } from '../../api/importExport.api';
import { getVaultStatus } from '../../api/vault.api';
import { useAsyncAction } from '../../hooks/useAsyncAction';

interface ExportDialogProps {
  open: boolean;
  onClose: () => void;
  folderId?: string;
  connectionIds?: string[];
}

export default function ExportDialog({ open, onClose, folderId, connectionIds }: ExportDialogProps) {
  const [format, setFormat] = useState<'CSV' | 'JSON'>('JSON');
  const [includeCredentials, setIncludeCredentials] = useState(false);
  const { loading, error, clearError, run } = useAsyncAction();
  const [vaultUnlocked, setVaultUnlocked] = useState(false);

  useEffect(() => {
    if (open) {
      checkVaultStatus();
      setIncludeCredentials(false);
      clearError();
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- clearError is stable (useCallback with [])
  }, [open]);

  const checkVaultStatus = async () => {
    try {
      const status = await getVaultStatus();
      setVaultUnlocked(status.unlocked);
    } catch {
      setVaultUnlocked(false);
    }
  };

  const handleExport = async () => {
    await run(async () => {
      const today = new Date().toISOString().split('T')[0];
      const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
      const filename = format === 'JSON'
        ? `arsenale-connections-${today}.json`
        : `connections-export-${timestamp}.csv`;

      await downloadExport({
        format,
        includeCredentials: includeCredentials && vaultUnlocked,
        folderId,
        connectionIds,
      }, filename);
    }, 'Export failed');
  };

  const handleClose = () => {
    clearError();
    onClose();
  };

  const scopeText = connectionIds
    ? `${connectionIds.length} selected connection(s)`
    : folderId
    ? 'this folder and subfolders'
    : 'all connections';

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) handleClose(); }}>
      <DialogContent
        showCloseButton={false}
        className="flex h-[100dvh] w-screen max-w-none flex-col gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
      >
        <DialogTitle className="sr-only">Export Connections</DialogTitle>
        <DialogDescription className="sr-only">Export connections to a file</DialogDescription>

        {/* Compact header */}
        <div className="flex h-8 shrink-0 items-center gap-2 border-b px-3">
          <span className="text-xs font-medium">Export Connections</span>
          <div className="ml-auto">
            <Button variant="ghost" size="icon-xs" onClick={handleClose}>
              <X className="size-3.5" />
            </Button>
          </div>
        </div>

        <ScrollArea className="flex-1">
          <div className="mx-auto max-w-2xl px-6 py-4">
            {error && (
              <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
                {error}
              </div>
            )}

            <div className="space-y-2">
              <Label>Format</Label>
              <Select value={format} onValueChange={(v) => setFormat(v as 'CSV' | 'JSON')}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="JSON">JSON (Recommended)</SelectItem>
                  <SelectItem value="CSV">CSV (Spreadsheet)</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <p className="mt-4 text-sm text-muted-foreground">
              Exporting {scopeText}
            </p>

            <div className="mt-4 flex items-center gap-2">
              <Checkbox
                id="include-credentials"
                checked={includeCredentials}
                onCheckedChange={(v) => setIncludeCredentials(v === true)}
                disabled={!vaultUnlocked}
              />
              <Label htmlFor="include-credentials" className="font-normal">
                Include credentials in export (requires vault unlocked)
              </Label>
            </div>

            {includeCredentials && !vaultUnlocked && (
              <div className="mt-4 rounded-md border border-yellow-600/50 bg-yellow-600/10 px-4 py-3 text-sm text-yellow-500">
                Vault is locked. Please unlock your vault to export credentials.
              </div>
            )}

            {includeCredentials && vaultUnlocked && (
              <div className="mt-4 rounded-md border border-yellow-600/50 bg-yellow-600/10 px-4 py-3 text-sm text-yellow-500">
                Credentials will be decrypted and included in plain text. Store this file securely.
              </div>
            )}
          </div>
        </ScrollArea>

        <div className="flex shrink-0 items-center justify-end gap-2 border-t px-4 py-2">
          <Button variant="outline" onClick={handleClose} disabled={loading}>
            Cancel
          </Button>
          <Button
            onClick={handleExport}
            disabled={loading || (includeCredentials && !vaultUnlocked)}
            className="gap-2"
          >
            {loading ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
            {loading ? 'Exporting...' : 'Export'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
