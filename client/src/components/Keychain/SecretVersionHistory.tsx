import { useState, useEffect, useCallback } from 'react';
import {
  Eye, EyeOff, RotateCcw, Circle, ChevronDown, Loader2,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from '@/components/ui/dialog';
import { cn } from '@/lib/utils';
import { listVersions, restoreVersion, getSecretVersionData } from '../../api/secrets.api';
import type { SecretVersion, SecretPayload } from '../../api/secrets.api';

interface SecretVersionHistoryProps {
  secretId: string;
  currentVersion: number;
  currentData?: SecretPayload;
  onRestore: () => void;
}

/** Extract flat key-value pairs from a SecretPayload for diffing */
function flattenPayload(data: SecretPayload): Record<string, string> {
  const entries: Record<string, string> = {};
  for (const [key, value] of Object.entries(data)) {
    if (key === 'type') continue;
    if (value === undefined || value === null) continue;
    if (typeof value === 'object') {
      entries[key] = JSON.stringify(value);
    } else {
      entries[key] = String(value);
    }
  }
  return entries;
}

const SENSITIVE_KEYS = ['password', 'privateKey', 'passphrase', 'apiKey', 'certificate', 'chain', 'content'];

function DiffView({ versionData, currentData }: { versionData: SecretPayload; currentData?: SecretPayload }) {
  const [revealedKeys, setRevealedKeys] = useState<Set<string>>(new Set());
  const versionFields = flattenPayload(versionData);
  const currentFields = currentData ? flattenPayload(currentData) : null;

  const allKeys = new Set([
    ...Object.keys(versionFields),
    ...(currentFields ? Object.keys(currentFields) : []),
  ]);

  const toggleReveal = (key: string) => {
    setRevealedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  return (
    <div className="mt-2 space-y-1">
      {[...allKeys].map((key) => {
        const vVal = versionFields[key];
        const cVal = currentFields?.[key];
        const changed = currentFields !== null && vVal !== cVal;
        const isSensitive = SENSITIVE_KEYS.includes(key);
        const isRevealed = revealedKeys.has(key);

        return (
          <div
            key={key}
            className={cn(
              'p-2 rounded-md border text-sm',
              changed ? 'bg-yellow-500/10 border-yellow-500/30' : 'bg-accent/50 border-border',
            )}
          >
            <div className="flex items-center justify-between">
              <span className="text-xs text-muted-foreground font-semibold">
                {key}
                {changed && (
                  <Badge variant="secondary" className="ml-1 text-[0.6rem] px-1 py-0">changed</Badge>
                )}
              </span>
              {isSensitive && vVal && (
                <Button
                  variant="ghost" size="icon" className="h-5 w-5"
                  onClick={() => toggleReveal(key)}
                  title={isRevealed ? 'Hide' : 'Reveal'}
                >
                  {isRevealed ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
                </Button>
              )}
            </div>
            <p className="font-mono text-xs break-all whitespace-pre-wrap">
              {isSensitive && vVal && !isRevealed ? '\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022' : (vVal ?? '(empty)')}
            </p>
          </div>
        );
      })}
    </div>
  );
}

export default function SecretVersionHistory({
  secretId,
  currentVersion,
  currentData,
  onRestore,
}: SecretVersionHistoryProps) {
  const [versions, setVersions] = useState<SecretVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [restoreTarget, setRestoreTarget] = useState<number | null>(null);
  const [restoring, setRestoring] = useState(false);

  // Expanded version data
  const [expandedVersion, setExpandedVersion] = useState<number | null>(null);
  const [versionData, setVersionData] = useState<Record<number, SecretPayload>>({});
  const [loadingVersion, setLoadingVersion] = useState<number | null>(null);

  const loadVersions = useCallback(async () => {
    setLoading(true);
    try {
      const data = await listVersions(secretId);
      setVersions(data);
    } catch {
      // silently fail
    } finally {
      setLoading(false);
    }
  }, [secretId]);

  useEffect(() => {
    loadVersions();
    setExpandedVersion(null);
    setVersionData({});
  }, [loadVersions]);

  const handleRestore = async (version: number) => {
    setRestoring(true);
    try {
      await restoreVersion(secretId, version);
      setRestoreTarget(null);
      setExpandedVersion(null);
      setVersionData({});
      await loadVersions();
      onRestore();
    } catch {
      // silently fail
    } finally {
      setRestoring(false);
    }
  };

  const handleToggleView = async (version: number) => {
    if (expandedVersion === version) {
      setExpandedVersion(null);
      return;
    }

    setExpandedVersion(version);

    if (!versionData[version]) {
      setLoadingVersion(version);
      try {
        const data = await getSecretVersionData(secretId, version);
        setVersionData((prev) => ({ ...prev, [version]: data }));
      } catch {
        // silently fail -- shared secrets can't view version data
        setExpandedVersion(null);
      } finally {
        setLoadingVersion(null);
      }
    }
  };

  const formatDate = (iso: string) => {
    const d = new Date(iso);
    return d.toLocaleDateString(undefined, {
      month: 'short', day: 'numeric', year: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });
  };

  if (loading) {
    return (
      <div className="flex justify-center py-4">
        <Loader2 className="h-5 w-5 animate-spin" />
      </div>
    );
  }

  if (versions.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-2">
        No version history available.
      </p>
    );
  }

  return (
    <div>
      {versions.map((v, idx) => {
        const isCurrent = v.version === currentVersion;
        const isLast = idx === versions.length - 1;
        const isExpanded = expandedVersion === v.version;
        const isLoadingData = loadingVersion === v.version;

        return (
          <div key={v.id} className="flex gap-3">
            {/* Timeline line + dot (clickable) */}
            <div
              className="flex flex-col items-center w-5 shrink-0 cursor-pointer"
              onClick={() => handleToggleView(v.version)}
            >
              <Circle
                className={cn(
                  'h-3 w-3 mt-1',
                  isCurrent ? 'text-primary fill-primary' : 'text-muted-foreground',
                )}
              />
              {!isLast && (
                <div className="flex-1 w-0.5 bg-border min-h-[24px]" />
              )}
            </div>

            {/* Content */}
            <div className="flex-1 pb-3 min-w-0">
              <div className="flex items-center gap-1 flex-wrap">
                <div
                  onClick={() => handleToggleView(v.version)}
                  className="flex items-center gap-1 cursor-pointer hover:opacity-80"
                >
                  <ChevronDown
                    className={cn(
                      'h-4 w-4 text-muted-foreground transition-transform duration-200',
                      isExpanded && 'rotate-180',
                    )}
                  />
                  <span className={cn('text-sm', isCurrent && 'font-semibold')}>
                    Version {v.version}
                  </span>
                  {isLoadingData && <Loader2 className="h-3 w-3 animate-spin ml-1" />}
                </div>
                {isCurrent && (
                  <Badge className="text-[0.65rem] px-1.5 py-0">current</Badge>
                )}

                {!isCurrent && (
                  <div className="ml-auto">
                    <Button
                      variant="ghost" size="icon" className="h-7 w-7"
                      onClick={() => setRestoreTarget(v.version)}
                      title="Restore this version"
                    >
                      <RotateCcw className="h-4 w-4" />
                    </Button>
                  </div>
                )}
              </div>

              <span className="text-xs text-muted-foreground ml-5">
                {v.changer?.username || v.changer?.email || 'Unknown'} &mdash; {formatDate(v.createdAt)}
              </span>
              {v.changeNote && (
                <span className="text-xs text-muted-foreground italic block ml-5">
                  {v.changeNote}
                </span>
              )}

              {/* Expanded version data with diff */}
              {isExpanded && versionData[v.version] && (
                <DiffView
                  versionData={versionData[v.version]}
                  currentData={isCurrent ? undefined : currentData}
                />
              )}
            </div>
          </div>
        );
      })}

      <Dialog open={restoreTarget !== null} onOpenChange={(v) => { if (!v) setRestoreTarget(null); }}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>Restore Version</DialogTitle>
            <DialogDescription>
              Restore to version {restoreTarget}? This will create a new version with the restored data.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRestoreTarget(null)}>Cancel</Button>
            <Button
              onClick={() => { if (restoreTarget !== null) handleRestore(restoreTarget); }}
              disabled={restoring}
            >
              {restoring ? 'Restoring...' : 'Restore'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
