import { useState, useEffect } from 'react';
import {
  Dialog, DialogContent, DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Key, KeyRound, Globe, X } from 'lucide-react';
import { useAuthStore } from '../../store/authStore';
import { useTabsStore } from '../../store/tabsStore';
import { ConnectionData } from '../../api/connections.api';

type ConnectMode = 'saved' | 'profile' | 'manual' | 'domain';

interface ConnectAsDialogProps {
  open: boolean;
  onClose: () => void;
  connection: ConnectionData | null;
}

export default function ConnectAsDialog({ open, onClose, connection }: ConnectAsDialogProps) {
  const openTab = useTabsStore((s) => s.openTab);
  const user = useAuthStore((s) => s.user);
  const domainConfigured = Boolean(user?.domainUsername && user?.hasDomainPassword);

  const [mode, setMode] = useState<ConnectMode>('saved');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [domain, setDomain] = useState('');
  const [error, setError] = useState('');

  /* eslint-disable react-hooks/set-state-in-effect -- reset form state when dialog opens */
  useEffect(() => {
    if (open) {
      setMode('saved');
      setUsername('');
      setPassword('');
      setDomain('');
      setError('');
    }
  }, [open]);

  useEffect(() => {
    if (mode === 'profile') {
      setUsername(user?.username || user?.email || '');
      setPassword('');
    } else if (mode === 'manual') {
      setUsername('');
      setPassword('');
      setDomain('');
    }
    setError('');
  }, [mode, user]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const handleConnect = () => {
    if (!connection) return;

    if (mode === 'saved') {
      openTab(connection);
    } else if (mode === 'domain') {
      openTab(connection, { username: '', password: '', credentialMode: 'domain' });
    } else {
      if (!username.trim()) {
        setError('Username is required');
        return;
      }
      if (!password) {
        setError('Password is required');
        return;
      }
      openTab(connection, { username: username.trim(), password, ...(domain.trim() ? { domain: domain.trim() } : {}) });
    }
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) onClose(); }}>
      <DialogContent
        showCloseButton={false}
        className="flex h-[100dvh] w-screen max-w-none flex-col gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
      >
        <DialogTitle className="sr-only">Connect As &mdash; {connection?.name}</DialogTitle>
        <DialogDescription className="sr-only">Choose credentials to connect</DialogDescription>

        {/* Compact header */}
        <div className="flex h-8 shrink-0 items-center gap-2 border-b px-3">
          <span className="text-xs font-medium">Connect As &mdash; {connection?.name}</span>
          <div className="ml-auto">
            <Button variant="ghost" size="icon-xs" onClick={onClose}>
              <X className="size-3.5" />
            </Button>
          </div>
        </div>

        <ScrollArea className="flex-1">
          <div className="mx-auto max-w-md px-6 py-4">
            {error && (
              <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
                {error}
              </div>
            )}

            <div className="flex flex-col gap-1">
              <label className="flex items-center gap-3 rounded-lg px-3 py-2.5 cursor-pointer hover:bg-accent/50 transition-colors">
                <input
                  type="radio"
                  name="connect-mode"
                  value="saved"
                  checked={mode === 'saved'}
                  onChange={() => setMode('saved')}
                  className="accent-primary"
                />
                <div className="flex items-center gap-2">
                  <span className="text-sm">Use saved credentials</span>
                  {connection?.credentialSecretId && (
                    <Badge variant="outline" className="gap-1">
                      {connection.credentialSecretType === 'SSH_KEY'
                        ? <Key className="size-3" />
                        : <KeyRound className="size-3" />}
                      {connection.credentialSecretName ?? 'Keychain secret'}
                    </Badge>
                  )}
                </div>
              </label>

              <label className="flex items-center gap-3 rounded-lg px-3 py-2.5 cursor-pointer hover:bg-accent/50 transition-colors">
                <input
                  type="radio"
                  name="connect-mode"
                  value="profile"
                  checked={mode === 'profile'}
                  onChange={() => setMode('profile')}
                  className="accent-primary"
                />
                <span className="text-sm">Use profile username</span>
              </label>

              <label className="flex items-center gap-3 rounded-lg px-3 py-2.5 cursor-pointer hover:bg-accent/50 transition-colors">
                <input
                  type="radio"
                  name="connect-mode"
                  value="manual"
                  checked={mode === 'manual'}
                  onChange={() => setMode('manual')}
                  className="accent-primary"
                />
                <span className="text-sm">Enter credentials manually</span>
              </label>

              <label className={`flex items-start gap-3 rounded-lg px-3 py-2.5 cursor-pointer hover:bg-accent/50 transition-colors ${!domainConfigured ? 'opacity-50 pointer-events-none' : ''}`}>
                <input
                  type="radio"
                  name="connect-mode"
                  value="domain"
                  checked={mode === 'domain'}
                  onChange={() => setMode('domain')}
                  disabled={!domainConfigured}
                  className="accent-primary mt-0.5"
                />
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm">Use domain credentials</span>
                    {domainConfigured && (
                      <Badge variant="outline" className="gap-1">
                        <Globe className="size-3" />
                        {user?.domainName ? `${user.domainName}\\${user.domainUsername}` : user?.domainUsername}
                      </Badge>
                    )}
                  </div>
                  {!domainConfigured && (
                    <span className="text-xs text-muted-foreground">
                      Not configured — set up in Settings &gt; Domain Profile
                    </span>
                  )}
                </div>
              </label>
            </div>

            {(mode === 'profile' || mode === 'manual') && (
              <div className="flex flex-col gap-4 mt-4">
                <div className="space-y-2">
                  <Label htmlFor="connect-username">Username</Label>
                  <Input
                    id="connect-username"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    readOnly={mode === 'profile'}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="connect-password">Password</Label>
                  <Input
                    id="connect-password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    autoFocus={mode === 'profile'}
                  />
                </div>
                {connection?.type === 'RDP' && (
                  <div className="space-y-2">
                    <Label htmlFor="connect-domain">Domain (optional)</Label>
                    <Input
                      id="connect-domain"
                      value={domain}
                      onChange={(e) => setDomain(e.target.value)}
                      placeholder="e.g. CONTOSO"
                    />
                  </div>
                )}
              </div>
            )}
          </div>
        </ScrollArea>

        <div className="flex shrink-0 items-center justify-end gap-2 border-t px-4 py-2">
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleConnect}>Connect</Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
