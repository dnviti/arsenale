import { KeyboardEvent, useState } from 'react';
import { Plus, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { SettingsFieldCard } from './settings-ui';
import { isValidNetworkEntry } from './networkAccessUtils';

interface NetworkEntryEditorProps {
  label: string;
  description?: string;
  inputLabel: string;
  placeholder: string;
  emptyState: string;
  entries: string[];
  disabled?: boolean;
  helperText?: string;
  onChange: (entries: string[]) => void;
}

export default function NetworkEntryEditor({
  label,
  description,
  inputLabel,
  placeholder,
  emptyState,
  entries,
  disabled,
  helperText,
  onChange,
}: NetworkEntryEditorProps) {
  const [draftValue, setDraftValue] = useState('');
  const [error, setError] = useState('');

  const handleAdd = () => {
    const trimmedValue = draftValue.trim();
    if (!trimmedValue) {
      return;
    }

    if (!isValidNetworkEntry(trimmedValue)) {
      setError('Use a valid IPv4 or IPv6 address, with an optional CIDR prefix.');
      return;
    }

    if (entries.includes(trimmedValue)) {
      setError('This entry is already present.');
      return;
    }

    onChange([...entries, trimmedValue]);
    setDraftValue('');
    setError('');
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key !== 'Enter') {
      return;
    }

    event.preventDefault();
    handleAdd();
  };

  const handleRemove = (entry: string) => {
    onChange(entries.filter((currentEntry) => currentEntry !== entry));
  };

  return (
    <SettingsFieldCard label={label} description={description}>
      <div className="space-y-3">
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input
            aria-label={inputLabel}
            value={draftValue}
            placeholder={placeholder}
            disabled={disabled}
            onChange={(event) => {
              setDraftValue(event.target.value);
              if (error) {
                setError('');
              }
            }}
            onKeyDown={handleKeyDown}
          />
          <Button
            type="button"
            variant="outline"
            className="sm:self-start"
            disabled={disabled || !draftValue.trim()}
            onClick={handleAdd}
          >
            <Plus className="size-4" />
            Add
          </Button>
        </div>

        <p className={cn('text-sm leading-6', error ? 'text-destructive' : 'text-muted-foreground')}>
          {error || helperText || emptyState}
        </p>

        {entries.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {entries.map((entry) => (
              <div
                key={entry}
                className="inline-flex items-center gap-2 rounded-full border border-border/70 bg-background px-3 py-1.5 text-sm text-foreground"
              >
                <span>{entry}</span>
                <button
                  type="button"
                  className="rounded-full text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50"
                  onClick={() => handleRemove(entry)}
                  disabled={disabled}
                  aria-label={`Remove ${entry}`}
                >
                  <X className="size-3.5" />
                </button>
              </div>
            ))}
          </div>
        ) : (
          <div className="rounded-xl border border-dashed border-border/70 bg-background/40 px-4 py-3 text-sm text-muted-foreground">
            {emptyState}
          </div>
        )}
      </div>
    </SettingsFieldCard>
  );
}
