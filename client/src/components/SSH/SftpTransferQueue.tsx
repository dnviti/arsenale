import {
  Upload,
  Download,
  XCircle,
  CheckCircle,
  AlertCircle,
  ChevronDown,
  ChevronUp,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import type { TransferItem } from '../../hooks/useSftpTransfers';

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

interface SftpTransferQueueProps {
  transfers: TransferItem[];
  onCancel: (transferId: string) => void;
  onClearCompleted: () => void;
}

export default function SftpTransferQueue({ transfers, onCancel, onClearCompleted }: SftpTransferQueueProps) {
  const open = useUiPreferencesStore((s) => s.sshSftpTransferQueueOpen);
  const toggle = useUiPreferencesStore((s) => s.toggle);

  const hasCompleted = transfers.some((t) => t.status !== 'active');

  if (transfers.length === 0) return null;

  return (
    <div className="border-t border-border">
      <div
        className="flex items-center px-3 py-1 cursor-pointer"
        onClick={() => toggle('sshSftpTransferQueueOpen')}
      >
        <span className="text-xs font-semibold flex-1">
          Transfers ({transfers.length})
        </span>
        {hasCompleted && (
          <Button
            variant="ghost"
            size="sm"
            className="text-xs h-6 px-1"
            onClick={(e) => { e.stopPropagation(); onClearCompleted(); }}
          >
            Clear
          </Button>
        )}
        <Button variant="ghost" size="icon" className="h-6 w-6">
          {open ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
        </Button>
      </div>

      {open && (
        <div className="max-h-[200px] overflow-auto">
          {transfers.map((t) => {
            const progress = t.totalBytes > 0 ? (t.bytesTransferred / t.totalBytes) * 100 : 0;

            return (
              <div key={t.transferId} className="flex items-center gap-2 px-3 py-1">
                <div className="w-5 shrink-0">
                  {t.status === 'complete' ? (
                    <CheckCircle className="h-4 w-4 text-green-500" />
                  ) : t.status === 'error' ? (
                    <AlertCircle className="h-4 w-4 text-red-500" />
                  ) : t.status === 'cancelled' ? (
                    <XCircle className="h-4 w-4 text-muted-foreground" />
                  ) : t.direction === 'upload' ? (
                    <Upload className="h-4 w-4 text-primary" />
                  ) : (
                    <Download className="h-4 w-4 text-primary" />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-xs truncate">{t.filename}</p>
                  <p className={`text-[0.7rem] truncate ${t.status === 'error' ? 'text-red-400' : 'text-muted-foreground'}`}>
                    {t.status === 'error'
                      ? t.errorMessage
                      : t.status === 'active'
                        ? `${formatSize(t.bytesTransferred)} / ${formatSize(t.totalBytes)}`
                        : t.status === 'complete'
                          ? formatSize(t.totalBytes)
                          : t.status}
                  </p>
                </div>
                {t.status === 'active' && (
                  <>
                    <div className="w-14">
                      <Progress value={progress} className="h-1.5" />
                    </div>
                    <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => onCancel(t.transferId)}>
                      <XCircle className="h-3.5 w-3.5" />
                    </Button>
                  </>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
