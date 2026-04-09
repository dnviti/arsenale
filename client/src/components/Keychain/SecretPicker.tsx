import { useState, useEffect, useCallback, useRef } from 'react';
import { KeyRound, Key, Lock, Loader2, ChevronsUpDown, Check, Search } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';
import { listSecrets, type SecretListItem, type SecretType } from '../../api/secrets.api';
import { useVaultStore } from '../../store/vaultStore';

const TYPE_ICONS: Partial<Record<SecretType, React.ReactNode>> = {
  LOGIN: <KeyRound className="h-4 w-4" />,
  SSH_KEY: <Key className="h-4 w-4" />,
};

const TYPE_LABELS: Partial<Record<SecretType, string>> = {
  LOGIN: 'Login',
  SSH_KEY: 'SSH Key',
};

const SCOPE_LABELS: Record<string, string> = {
  PERSONAL: 'Me',
  TEAM: 'Team',
  TENANT: 'Org',
};

interface SecretPickerProps {
  value: string | null;
  onChange: (secretId: string | null, secret: SecretListItem | null) => void;
  connectionType: 'SSH' | 'RDP' | 'VNC' | 'DATABASE';
  disabled?: boolean;
  error?: boolean;
  helperText?: string;
  /** Pre-populated name/type from connection data so the picker shows
   *  the secret immediately without waiting for an API fetch. */
  initialName?: string | null;
  initialType?: SecretType | null;
}

export default function SecretPicker({
  value,
  onChange,
  connectionType,
  disabled,
  error,
  helperText,
  initialName,
  initialType,
}: SecretPickerProps) {
  const vaultUnlocked = useVaultStore((s) => s.unlocked);
  const [options, setOptions] = useState<SecretListItem[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<SecretListItem | null>(null);
  const [isOpen, setIsOpen] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const compatibleTypes: SecretType[] =
    connectionType === 'SSH' ? ['LOGIN', 'SSH_KEY'] : ['LOGIN'];

  // Synchronously set stub from initialName, then upgrade with full API data
  useEffect(() => {
    if (!value) {
      setSelected(null);
      return;
    }

    // Synchronously show stub from initialName (no async wait needed)
    if (initialName && selected?.id !== value) {
      const stub: SecretListItem = {
        id: value,
        name: initialName,
        description: null,
        type: initialType ?? 'LOGIN',
        scope: 'PERSONAL',
        teamId: null,
        tenantId: null,
        folderId: null,
        metadata: null,
        tags: [],
        isFavorite: false,
        pwnedCount: 0,
        expiresAt: null,
        currentVersion: 1,
        createdAt: '',
        updatedAt: '',
      };
      setSelected(stub);
      setOptions((prev) =>
        prev.some((o) => o.id === value) ? prev : [stub, ...prev],
      );
    }

    // Upgrade stub with full metadata from API (async)
    if (!vaultUnlocked) return;

    let cancelled = false;
    (async () => {
      try {
        const results = await listSecrets({});
        if (cancelled) return;
        const match = results.find((s) => s.id === value);
        if (match) {
          setSelected(match);
          setOptions((prev) =>
            prev.some((o) => o.id === match.id) ? prev : [match, ...prev],
          );
        }
      } catch {
        // silent -- secret may not be accessible
      }
    })();
    return () => { cancelled = true; };
  }, [value, vaultUnlocked, initialName, initialType]); // eslint-disable-line react-hooks/exhaustive-deps

  const fetchOptions = useCallback(
    async (search: string) => {
      if (!vaultUnlocked) return;
      setLoading(true);
      try {
        // Fetch for each compatible type and merge
        const promises = compatibleTypes.map((t) =>
          listSecrets({ search: search || undefined, type: t }),
        );
        const results = (await Promise.all(promises)).flat();
        // Deduplicate by id
        const seen = new Set<string>();
        const unique = results.filter((s) => {
          if (seen.has(s.id)) return false;
          seen.add(s.id);
          return true;
        });
        setOptions(unique);
      } catch {
        setOptions([]);
      } finally {
        setLoading(false);
      }
    },
    [vaultUnlocked, connectionType], // eslint-disable-line react-hooks/exhaustive-deps
  );

  // Debounced search
  useEffect(() => {
    if (!vaultUnlocked) return;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      fetchOptions(inputValue);
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [inputValue, fetchOptions, vaultUnlocked]);

  // Re-fetch when connection type changes (compatible types change)
  useEffect(() => {
    if (vaultUnlocked) fetchOptions(inputValue);
  }, [connectionType]); // eslint-disable-line react-hooks/exhaustive-deps

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const isDisabled = disabled || !vaultUnlocked;

  const handleSelect = (option: SecretListItem) => {
    setSelected(option);
    onChange(option.id, option);
    setIsOpen(false);
    setInputValue('');
  };

  const handleClear = () => {
    setSelected(null);
    onChange(null, null);
    setInputValue('');
  };

  return (
    <div ref={containerRef} className="relative space-y-1.5">
      <Label className={cn(error && 'text-destructive')}>Select Secret</Label>
      <div
        className={cn(
          'flex h-10 w-full items-center rounded-lg border bg-background px-3 py-2 text-sm shadow-xs transition-[color,box-shadow,border-color] cursor-pointer',
          error && 'border-destructive',
          isDisabled && 'opacity-50 cursor-not-allowed',
          !isDisabled && 'hover:border-ring',
        )}
        onClick={() => { if (!isDisabled) setIsOpen(!isOpen); }}
      >
        {!vaultUnlocked && <Lock className="h-4 w-4 mr-1.5 text-muted-foreground" />}
        <span className={cn('flex-1 truncate', !selected && 'text-muted-foreground')}>
          {selected ? selected.name : (vaultUnlocked ? 'Search keychain...' : 'Unlock vault to use keychain')}
        </span>
        {loading && <Loader2 className="h-4 w-4 animate-spin mr-1" />}
        <ChevronsUpDown className="h-4 w-4 text-muted-foreground shrink-0" />
      </div>

      {helperText && (
        <p className={cn('text-xs', error ? 'text-destructive' : 'text-muted-foreground')}>
          {!vaultUnlocked ? 'Unlock vault to use keychain' : helperText}
        </p>
      )}

      {isOpen && !isDisabled && (
        <div className="absolute z-50 w-full mt-1 rounded-xl border bg-popover text-popover-foreground shadow-lg overflow-hidden">
          <div className="p-2 border-b">
            <div className="relative">
              <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                placeholder="Search..."
                className="pl-8 h-8"
                autoFocus
              />
            </div>
          </div>
          <div className="max-h-60 overflow-auto p-1">
            {loading && options.length === 0 && (
              <div className="text-sm text-muted-foreground text-center py-4">Searching...</div>
            )}
            {!loading && options.length === 0 && (
              <div className="text-sm text-muted-foreground text-center py-4">No secrets found</div>
            )}
            {options.map((option) => (
              <div
                key={option.id}
                onClick={() => handleSelect(option)}
                className={cn(
                  'flex items-center gap-2 px-2 py-1.5 rounded-lg cursor-pointer text-sm transition-colors hover:bg-accent',
                  selected?.id === option.id && 'bg-accent',
                )}
              >
                {TYPE_ICONS[option.type]}
                <span className="flex-1 truncate">{option.name}</span>
                <Badge variant="outline" className="text-[0.65rem] px-1.5 py-0">
                  {SCOPE_LABELS[option.scope] ?? option.scope}
                </Badge>
                <span className="text-xs text-muted-foreground">
                  {TYPE_LABELS[option.type] ?? option.type}
                </span>
                {selected?.id === option.id && <Check className="h-4 w-4 text-primary shrink-0" />}
              </div>
            ))}
          </div>
          {selected && (
            <div className="border-t p-1">
              <button
                onClick={handleClear}
                className="w-full text-left text-xs text-muted-foreground px-2 py-1 hover:text-foreground transition-colors"
              >
                Clear selection
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
