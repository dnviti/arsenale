import { useState, useCallback, useEffect, useRef } from 'react';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { cn } from '@/lib/utils';
import type { DbSessionConfig } from '../../api/database.api';
import { updateDbSessionConfig } from '../../api/database.api';
import { extractApiError } from '../../utils/apiError';

interface DbSessionConfigPopoverProps {
  open: boolean;
  anchorEl: HTMLElement | null;
  onClose: () => void;
  protocol: string;
  sessionId: string | null;
  currentConfig: DbSessionConfig;
  onConfigApplied: (config: DbSessionConfig, activeDatabase?: string) => void;
}

// Per-protocol field visibility
const FIELD_SUPPORT: Record<string, Record<string, boolean>> = {
  postgresql: { activeDatabase: true, timezone: true, searchPath: true, encoding: true, initCommands: true },
  mysql:      { activeDatabase: true, timezone: true, searchPath: false, encoding: true, initCommands: true },
  mssql:      { activeDatabase: true, timezone: false, searchPath: true, encoding: false, initCommands: true },
  oracle:     { activeDatabase: false, timezone: true, searchPath: true, encoding: true, initCommands: true },
  db2:        { activeDatabase: true, timezone: true, searchPath: true, encoding: false, initCommands: true },
  mongodb:    {},
};

const FIELD_LABELS: Record<string, { label: string; placeholder: string; helperText: string }> = {
  activeDatabase: {
    label: 'Active Database',
    placeholder: 'e.g. mydb',
    helperText: 'Switch the active database for queries',
  },
  timezone: {
    label: 'Timezone',
    placeholder: 'e.g. UTC, America/New_York',
    helperText: 'Session timezone for date/time operations',
  },
  searchPath: {
    label: 'Schema / Search Path',
    placeholder: 'e.g. public, myschema',
    helperText: 'Default schema or search path for unqualified names',
  },
  encoding: {
    label: 'Encoding',
    placeholder: 'e.g. UTF8, latin1',
    helperText: 'Client character encoding',
  },
};

export default function DbSessionConfigPopover({
  open,
  anchorEl,
  onClose,
  protocol,
  sessionId,
  currentConfig,
  onConfigApplied,
}: DbSessionConfigPopoverProps) {
  const [config, setConfig] = useState<DbSessionConfig>({});
  const [initCommandsText, setInitCommandsText] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const popoverRef = useRef<HTMLDivElement>(null);

  // Sync local state when popover opens or currentConfig changes
  useEffect(() => {
    if (open) {
      setConfig({ ...currentConfig });
      setInitCommandsText(currentConfig.initCommands?.join('\n') ?? '');
      setError('');
    }
  }, [open, currentConfig]);

  // Close on outside click
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (
        popoverRef.current &&
        !popoverRef.current.contains(e.target as Node) &&
        anchorEl &&
        !anchorEl.contains(e.target as Node)
      ) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open, anchorEl, onClose]);

  const fields = FIELD_SUPPORT[protocol] ?? {};

  const handleFieldChange = useCallback((field: keyof DbSessionConfig, value: string) => {
    setConfig((prev) => ({
      ...prev,
      [field]: value || undefined,
    }));
  }, []);

  const handleApply = useCallback(async () => {
    if (!sessionId) return;
    setLoading(true);
    setError('');

    try {
      const finalConfig: DbSessionConfig = { ...config };
      // Parse initCommands from multiline text
      if (initCommandsText.trim()) {
        finalConfig.initCommands = initCommandsText
          .split('\n')
          .map((l) => l.trim())
          .filter(Boolean);
      } else {
        finalConfig.initCommands = undefined;
      }

      // Remove empty/undefined values
      const cleanConfig: DbSessionConfig = {};
      if (finalConfig.activeDatabase) cleanConfig.activeDatabase = finalConfig.activeDatabase;
      if (finalConfig.timezone) cleanConfig.timezone = finalConfig.timezone;
      if (finalConfig.searchPath) cleanConfig.searchPath = finalConfig.searchPath;
      if (finalConfig.encoding) cleanConfig.encoding = finalConfig.encoding;
      if (finalConfig.initCommands?.length) cleanConfig.initCommands = finalConfig.initCommands;

      const result = await updateDbSessionConfig(sessionId, cleanConfig);
      onConfigApplied(cleanConfig, result.activeDatabase);
    } catch (err) {
      setError(extractApiError(err, 'Failed to apply session config'));
    } finally {
      setLoading(false);
    }
  }, [sessionId, config, initCommandsText, onConfigApplied]);

  const handleReset = useCallback(async () => {
    if (!sessionId) return;
    setLoading(true);
    setError('');

    try {
      const result = await updateDbSessionConfig(sessionId, {});
      setConfig({});
      setInitCommandsText('');
      onConfigApplied({}, result.activeDatabase);
    } catch (err) {
      setError(extractApiError(err, 'Failed to reset session config'));
    } finally {
      setLoading(false);
    }
  }, [sessionId, onConfigApplied]);

  const hasAnyField = Object.values(fields).some(Boolean);
  const hasChanges = Object.values(config).some((v) => v !== undefined && v !== '') || initCommandsText.trim() !== '';

  if (!open) return null;

  // Position relative to anchor
  const anchorRect = anchorEl?.getBoundingClientRect();
  const style: React.CSSProperties = anchorRect
    ? {
        position: 'fixed',
        top: anchorRect.bottom + 4,
        left: anchorRect.left,
        zIndex: 50,
      }
    : { position: 'fixed', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', zIndex: 50 };

  return (
    <div
      ref={popoverRef}
      style={style}
      className="w-[340px] max-h-[480px] overflow-auto rounded-xl border border-border bg-popover shadow-lg p-4"
    >
      <h4 className="text-sm font-semibold mb-1">
        Session Configuration
      </h4>
      <p className="text-xs text-muted-foreground mb-3">
        {protocol.toUpperCase()} session parameters
      </p>

      {!hasAnyField && (
        <p className="text-sm text-muted-foreground">
          Session configuration is not available for {protocol.toUpperCase()}.
        </p>
      )}

      {error && (
        <div className="mb-3 rounded border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400 flex items-center justify-between">
          <span>{error}</span>
          <button onClick={() => setError('')} className="text-red-400 hover:text-red-300 ml-2 text-xs">dismiss</button>
        </div>
      )}

      <div className="flex flex-col gap-3">
        {fields.activeDatabase && (
          <div
            title={protocol === 'postgresql' ? 'Changing database will recreate the connection pool' : ''}
          >
            <Label className="text-xs">{FIELD_LABELS.activeDatabase.label}</Label>
            <Input
              className="h-8 text-sm mt-1"
              placeholder={FIELD_LABELS.activeDatabase.placeholder}
              value={config.activeDatabase ?? ''}
              onChange={(e) => handleFieldChange('activeDatabase', e.target.value)}
              disabled={loading}
            />
            <p className="text-xs text-muted-foreground mt-0.5">
              {protocol === 'postgresql'
                ? 'Warning: changes require pool recreation'
                : FIELD_LABELS.activeDatabase.helperText}
            </p>
          </div>
        )}

        {fields.timezone && (
          <div>
            <Label className="text-xs">{FIELD_LABELS.timezone.label}</Label>
            <Input
              className="h-8 text-sm mt-1"
              placeholder={FIELD_LABELS.timezone.placeholder}
              value={config.timezone ?? ''}
              onChange={(e) => handleFieldChange('timezone', e.target.value)}
              disabled={loading}
            />
            <p className="text-xs text-muted-foreground mt-0.5">{FIELD_LABELS.timezone.helperText}</p>
          </div>
        )}

        {fields.searchPath && (
          <div>
            <Label className="text-xs">{FIELD_LABELS.searchPath.label}</Label>
            <Input
              className="h-8 text-sm mt-1"
              placeholder={FIELD_LABELS.searchPath.placeholder}
              value={config.searchPath ?? ''}
              onChange={(e) => handleFieldChange('searchPath', e.target.value)}
              disabled={loading}
            />
            <p className="text-xs text-muted-foreground mt-0.5">{FIELD_LABELS.searchPath.helperText}</p>
          </div>
        )}

        {fields.encoding && (
          <div>
            <Label className="text-xs">{FIELD_LABELS.encoding.label}</Label>
            <Input
              className="h-8 text-sm mt-1"
              placeholder={FIELD_LABELS.encoding.placeholder}
              value={config.encoding ?? ''}
              onChange={(e) => handleFieldChange('encoding', e.target.value)}
              disabled={loading}
            />
            <p className="text-xs text-muted-foreground mt-0.5">{FIELD_LABELS.encoding.helperText}</p>
          </div>
        )}

        {fields.initCommands && (
          <div>
            <Label className="text-xs">Init Commands</Label>
            <textarea
              className={cn(
                'mt-1 w-full min-h-[60px] max-h-[120px] rounded-md border border-input bg-transparent px-3 py-2',
                'text-sm font-mono placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50',
                'disabled:cursor-not-allowed disabled:opacity-50',
              )}
              placeholder={'SET ...\nALTER SESSION SET ...'}
              value={initCommandsText}
              onChange={(e) => setInitCommandsText(e.target.value)}
              disabled={loading}
              rows={3}
            />
            <p className="text-xs text-muted-foreground mt-0.5">
              One SET/ALTER SESSION command per line (OPERATOR+ only)
            </p>
          </div>
        )}
      </div>

      {hasAnyField && (
        <>
          <Separator className="my-3" />
          <div className="flex gap-2 justify-end">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleReset}
              disabled={loading || !Object.values(currentConfig).some((v) => v !== undefined)}
            >
              Reset
            </Button>
            <Button
              size="sm"
              onClick={handleApply}
              disabled={loading || !hasChanges}
            >
              {loading && <Loader2 className="size-3.5 animate-spin" />}
              Apply
            </Button>
          </div>
        </>
      )}
    </div>
  );
}
