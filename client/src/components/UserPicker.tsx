import { useEffect, useMemo, useRef, useState } from 'react';
import { Loader2, Search, X } from 'lucide-react';
import { searchUsers, type UserSearchResult } from '../api/user.api';
import { Avatar, AvatarFallback, AvatarImage } from './ui/avatar';
import { Button } from './ui/button';
import { Card } from './ui/card';
import { Input } from './ui/input';
import { ScrollArea } from './ui/scroll-area';
import { cn } from '../lib/utils';

interface UserPickerProps {
  onSelect: (user: UserSearchResult | null) => void;
  scope: 'tenant' | 'team';
  teamId?: string;
  placeholder?: string;
  excludeUserIds?: string[];
  size?: 'small' | 'medium';
  value?: UserSearchResult | null;
  clearAfterSelect?: boolean;
  disabled?: boolean;
  className?: string;
}

const EMPTY_EXCLUDED_USER_IDS: string[] = [];
const USER_SEARCH_DEBOUNCE_MS = import.meta.env.MODE === 'test' ? 0 : 250;

function getUserLabel(user: UserSearchResult) {
  return user.username || user.email;
}

function getUserInitial(user: UserSearchResult) {
  return getUserLabel(user).charAt(0).toUpperCase();
}

export default function UserPicker({
  onSelect,
  scope,
  teamId,
  placeholder = 'Search users...',
  excludeUserIds = EMPTY_EXCLUDED_USER_IDS,
  size = 'small',
  value,
  clearAfterSelect = false,
  disabled = false,
  className,
}: UserPickerProps) {
  const [query, setQuery] = useState('');
  const [options, setOptions] = useState<UserSearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchError, setSearchError] = useState('');
  const [internalValue, setInternalValue] = useState<UserSearchResult | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const requestIdRef = useRef(0);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const scopeKeyRef = useRef(`${scope}:${teamId || ''}`);

  const isControlled = value !== undefined;
  const selectedUser = isControlled ? value : internalValue;
  const excludedIds = useMemo(() => new Set(excludeUserIds), [excludeUserIds]);
  const inputHeightClassName = size === 'medium' ? 'h-10' : 'h-9';

  const clearSelection = () => {
    if (!isControlled) {
      setInternalValue(null);
    }
    setQuery('');
    setOptions([]);
    setSearchError('');
    onSelect(null);
    requestAnimationFrame(() => inputRef.current?.focus());
  };

  const commitSelection = (user: UserSearchResult) => {
    if (!clearAfterSelect && !isControlled) {
      setInternalValue(user);
    }

    if (clearAfterSelect) {
      setQuery('');
      setOptions([]);
      setSearchError('');
    } else {
      setQuery(getUserLabel(user));
      setOptions([]);
      setSearchError('');
    }

    onSelect(user);
  };

  useEffect(() => {
    if (selectedUser && !clearAfterSelect) {
      setQuery(getUserLabel(selectedUser));
      return;
    }

    if (!selectedUser) {
      setQuery('');
    }
  }, [clearAfterSelect, selectedUser]);

  useEffect(() => {
    const nextScopeKey = `${scope}:${teamId || ''}`;
    if (scopeKeyRef.current === nextScopeKey) {
      return;
    }

    scopeKeyRef.current = nextScopeKey;

    setOptions([]);
    setSearchError('');
    if (selectedUser) {
      clearSelection();
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- reset selection when the searchable scope changes
  }, [scope, teamId]);

  useEffect(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }

    const trimmedQuery = query.trim();
    if (disabled || selectedUser || trimmedQuery.length < 1) {
      setOptions([]);
      setLoading(false);
      setSearchError('');
      return undefined;
    }

    setLoading(true);
    setSearchError('');
    const requestId = ++requestIdRef.current;

    debounceRef.current = setTimeout(async () => {
      try {
        const results = await searchUsers(trimmedQuery, scope, teamId);
        if (requestId !== requestIdRef.current) {
          return;
        }

        setOptions(results.filter((user) => !excludedIds.has(user.id)));
      } catch {
        if (requestId !== requestIdRef.current) {
          return;
        }

        setOptions([]);
        setSearchError('Unable to load users right now.');
      } finally {
        if (requestId === requestIdRef.current) {
          setLoading(false);
        }
      }
    }, USER_SEARCH_DEBOUNCE_MS);

    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
  }, [disabled, excludedIds, query, scope, selectedUser, teamId]);

  const shouldRenderResults = !selectedUser && !disabled && (query.trim().length > 0 || loading || Boolean(searchError));

  return (
    <div className={cn('w-full space-y-2', className)}>
      {selectedUser && !clearAfterSelect ? (
        <Card className="border-border/70 bg-background/70 p-3">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex min-w-0 items-center gap-3">
              <Avatar className="size-9">
                <AvatarImage src={selectedUser.avatarData || undefined} alt={getUserLabel(selectedUser)} />
                <AvatarFallback>{getUserInitial(selectedUser)}</AvatarFallback>
              </Avatar>
              <div className="min-w-0">
                <div className="truncate text-sm font-medium text-foreground">
                  {getUserLabel(selectedUser)}
                </div>
                {selectedUser.username && (
                  <div className="truncate text-xs text-muted-foreground">
                    {selectedUser.email}
                  </div>
                )}
              </div>
            </div>
            <Button type="button" variant="outline" size="sm" onClick={clearSelection}>
              <X className="size-4" />
              Change
            </Button>
          </div>
        </Card>
      ) : (
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            ref={inputRef}
            value={query}
            disabled={disabled}
            placeholder={placeholder}
            aria-label={placeholder}
            className={cn(inputHeightClassName, 'pl-9 pr-9')}
            onChange={(event) => setQuery(event.target.value)}
          />
          <div className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground">
            {loading ? (
              <Loader2 className="size-4 animate-spin" />
            ) : query ? (
              <button
                type="button"
                aria-label="Clear search"
                className="rounded p-1 transition-colors hover:bg-accent hover:text-accent-foreground"
                onClick={() => {
                  setQuery('');
                  setOptions([]);
                  setSearchError('');
                  inputRef.current?.focus();
                }}
              >
                <X className="size-4" />
              </button>
            ) : null}
          </div>
        </div>
      )}

      {selectedUser && !clearAfterSelect && (
        <p className="text-xs text-muted-foreground">
          The selected user stays active until you change it.
        </p>
      )}

      {shouldRenderResults && (
        <Card className="overflow-hidden border-border/70">
          <ScrollArea className="max-h-60">
            <div className="p-1">
              {loading && (
                <div className="flex items-center gap-2 px-3 py-3 text-sm text-muted-foreground">
                  <Loader2 className="size-4 animate-spin" />
                  Searching users...
                </div>
              )}

              {!loading && searchError && (
                <div className="px-3 py-3 text-sm text-destructive">
                  {searchError}
                </div>
              )}

              {!loading && !searchError && options.length === 0 && (
                <div className="px-3 py-3 text-sm text-muted-foreground">
                  No users found.
                </div>
              )}

              {!loading && !searchError && options.map((option) => (
                <button
                  key={option.id}
                  type="button"
                  aria-label={`Select ${getUserLabel(option)}`}
                  className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors hover:bg-accent hover:text-accent-foreground"
                  onClick={() => commitSelection(option)}
                >
                  <Avatar className="size-8">
                    <AvatarImage src={option.avatarData || undefined} alt={getUserLabel(option)} />
                    <AvatarFallback>{getUserInitial(option)}</AvatarFallback>
                  </Avatar>
                  <div className="min-w-0">
                    <div className="truncate text-sm font-medium text-foreground">
                      {getUserLabel(option)}
                    </div>
                    {option.username && (
                      <div className="truncate text-xs text-muted-foreground">
                        {option.email}
                      </div>
                    )}
                  </div>
                </button>
              ))}
            </div>
          </ScrollArea>
        </Card>
      )}
    </div>
  );
}
