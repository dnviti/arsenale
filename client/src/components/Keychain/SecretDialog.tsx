import { useState, useEffect, useRef } from 'react';
import { Eye, EyeOff, Dices, Upload, Plus, Trash2, X } from 'lucide-react';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Alert } from '@/components/ui/alert';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { useSecretStore } from '../../store/secretStore';
import { useAuthStore } from '../../store/authStore';
import { useTeamStore } from '../../store/teamStore';
import type { SecretDetail, SecretType, SecretScope, SecretPayload } from '../../api/secrets.api';
import type { TenantVaultStatus } from '../../api/secrets.api';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import { isAdminOrAbove } from '../../utils/roles';

interface SecretDialogProps {
  open: boolean;
  onClose: () => void;
  secret?: SecretDetail | null;
}

function generatePassword(length = 20): string {
  const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*_-+=';
  const array = new Uint8Array(length);
  crypto.getRandomValues(array);
  return Array.from(array, (b) => chars[b % chars.length]).join('');
}

export default function SecretDialog({ open, onClose, secret }: SecretDialogProps) {
  const createSecret = useSecretStore((s) => s.createSecret);
  const updateSecret = useSecretStore((s) => s.updateSecret);
  const tenantVaultStatus: TenantVaultStatus | null = useSecretStore((s) => s.tenantVaultStatus);
  const user = useAuthStore((s) => s.user);
  const teams = useTeamStore((s) => s.teams);
  const fetchTeams = useTeamStore((s) => s.fetchTeams);

  const isEditMode = !!secret;

  // Form state
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [type, setType] = useState<SecretType>('LOGIN');
  const [scope, setScope] = useState<SecretScope>('PERSONAL');
  const [teamId, setTeamId] = useState('');
  const [tags, setTags] = useState<string[]>([]);
  const [tagInput, setTagInput] = useState('');
  const [expiresAt, setExpiresAt] = useState('');

  // Type-specific data
  const [loginUsername, setLoginUsername] = useState('');
  const [loginPassword, setLoginPassword] = useState('');
  const [loginDomain, setLoginDomain] = useState('');
  const [loginUrl, setLoginUrl] = useState('');
  const [notes, setNotes] = useState('');
  const [showPassword, setShowPassword] = useState(false);

  const [sshPrivateKey, setSshPrivateKey] = useState('');
  const [sshPublicKey, setSshPublicKey] = useState('');
  const [sshPassphrase, setSshPassphrase] = useState('');
  const [sshAlgorithm, setSshAlgorithm] = useState('');
  const [sshUsername, setSshUsername] = useState('');

  const [certCertificate, setCertCertificate] = useState('');
  const [certPrivateKey, setCertPrivateKey] = useState('');
  const [certChain, setCertChain] = useState('');
  const [certPassphrase, setCertPassphrase] = useState('');
  const [certExpiresAt, setCertExpiresAt] = useState('');

  const [apiKeyValue, setApiKeyValue] = useState('');
  const [apiKeyEndpoint, setApiKeyEndpoint] = useState('');
  const [apiKeyHeaders, setApiKeyHeaders] = useState<Array<{ key: string; value: string }>>([]);

  const [noteContent, setNoteContent] = useState('');

  const { loading, error, setError, run } = useAsyncAction();

  const fileInputRef = useRef<HTMLInputElement>(null);
  const [fileTarget, setFileTarget] = useState<string>('');

  useEffect(() => {
    if (open) {
      if (teams.length === 0 && user?.tenantId) {
        fetchTeams();
      }
      if (secret) {
        setName(secret.name);
        setDescription(secret.description || '');
        setType(secret.type);
        setScope(secret.scope);
        setTeamId(secret.teamId || '');
        setTags(secret.tags || []);
        setExpiresAt(secret.expiresAt ? secret.expiresAt.slice(0, 16) : '');
        populateData(secret.data);
      } else {
        resetForm();
      }
      setError('');
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- only trigger on open/close and secret change
  }, [open, secret]);

  const resetForm = () => {
    setName('');
    setDescription('');
    setType('LOGIN');
    setScope('PERSONAL');
    setTeamId('');
    setTags([]);
    setTagInput('');
    setExpiresAt('');
    setLoginUsername('');
    setLoginPassword('');
    setLoginDomain('');
    setLoginUrl('');
    setNotes('');
    setShowPassword(false);
    setSshPrivateKey('');
    setSshPublicKey('');
    setSshPassphrase('');
    setSshAlgorithm('');
    setSshUsername('');
    setCertCertificate('');
    setCertPrivateKey('');
    setCertChain('');
    setCertPassphrase('');
    setCertExpiresAt('');
    setApiKeyValue('');
    setApiKeyEndpoint('');
    setApiKeyHeaders([]);
    setNoteContent('');
  };

  const populateData = (data: SecretPayload) => {
    switch (data.type) {
      case 'LOGIN':
        setLoginUsername(data.username);
        setLoginPassword(data.password);
        setLoginDomain(data.domain || '');
        setLoginUrl(data.url || '');
        setNotes(data.notes || '');
        break;
      case 'SSH_KEY':
        setSshUsername(data.username || '');
        setSshPrivateKey(data.privateKey);
        setSshPublicKey(data.publicKey || '');
        setSshPassphrase(data.passphrase || '');
        setSshAlgorithm(data.algorithm || '');
        setNotes(data.notes || '');
        break;
      case 'CERTIFICATE':
        setCertCertificate(data.certificate);
        setCertPrivateKey(data.privateKey);
        setCertChain(data.chain || '');
        setCertPassphrase(data.passphrase || '');
        setCertExpiresAt(data.expiresAt || '');
        setNotes(data.notes || '');
        break;
      case 'API_KEY':
        setApiKeyValue(data.apiKey);
        setApiKeyEndpoint(data.endpoint || '');
        setApiKeyHeaders(
          data.headers
            ? Object.entries(data.headers).map(([key, value]) => ({ key, value }))
            : [],
        );
        setNotes(data.notes || '');
        break;
      case 'SECURE_NOTE':
        setNoteContent(data.content);
        break;
    }
  };

  const buildPayload = (): SecretPayload | null => {
    switch (type) {
      case 'LOGIN':
        if (!loginUsername || !loginPassword) { setError('Username and password are required'); return null; }
        return { type: 'LOGIN', username: loginUsername, password: loginPassword, domain: loginDomain || undefined, url: loginUrl || undefined, notes: notes || undefined };
      case 'SSH_KEY':
        if (!sshPrivateKey) { setError('Private key is required'); return null; }
        return { type: 'SSH_KEY', username: sshUsername || undefined, privateKey: sshPrivateKey, publicKey: sshPublicKey || undefined, passphrase: sshPassphrase || undefined, algorithm: sshAlgorithm || undefined, notes: notes || undefined };
      case 'CERTIFICATE':
        if (!certCertificate || !certPrivateKey) { setError('Certificate and private key are required'); return null; }
        return { type: 'CERTIFICATE', certificate: certCertificate, privateKey: certPrivateKey, chain: certChain || undefined, passphrase: certPassphrase || undefined, expiresAt: certExpiresAt || undefined, notes: notes || undefined };
      case 'API_KEY':
        if (!apiKeyValue) { setError('API key is required'); return null; }
        const headers = apiKeyHeaders.reduce<Record<string, string>>((acc, h) => {
          if (h.key.trim()) acc[h.key.trim()] = h.value;
          return acc;
        }, {});
        return { type: 'API_KEY', apiKey: apiKeyValue, endpoint: apiKeyEndpoint || undefined, headers: Object.keys(headers).length > 0 ? headers : undefined, notes: notes || undefined };
      case 'SECURE_NOTE':
        if (!noteContent) { setError('Content is required'); return null; }
        return { type: 'SECURE_NOTE', content: noteContent };
    }
  };

  const handleSubmit = async () => {
    if (!name.trim()) { setError('Name is required'); return; }
    if (scope === 'TEAM' && !teamId) { setError('Please select a team'); return; }

    const payload = buildPayload();
    if (!payload) return;

    const ok = await run(async () => {
      if (isEditMode && secret) {
        await updateSecret(secret.id, {
          name: name.trim(),
          description: description.trim() || null,
          data: payload,
          tags,
          expiresAt: expiresAt ? new Date(expiresAt).toISOString() : null,
        });
      } else {
        await createSecret({
          name: name.trim(),
          description: description.trim() || undefined,
          type,
          scope,
          teamId: scope === 'TEAM' ? teamId : undefined,
          data: payload,
          tags: tags.length > 0 ? tags : undefined,
          expiresAt: expiresAt ? new Date(expiresAt).toISOString() : undefined,
        });
      }
    }, isEditMode ? 'Failed to update secret' : 'Failed to create secret');
    if (ok) onClose();
  };

  const handleAddTag = () => {
    const t = tagInput.trim();
    if (t && !tags.includes(t)) {
      setTags([...tags, t]);
    }
    setTagInput('');
  };

  const handleFileUpload = (target: string) => {
    setFileTarget(target);
    fileInputRef.current?.click();
  };

  const handleFileRead = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      const content = reader.result as string;
      switch (fileTarget) {
        case 'sshPrivateKey': setSshPrivateKey(content); break;
        case 'sshPublicKey': setSshPublicKey(content); break;
        case 'certCertificate': setCertCertificate(content); break;
        case 'certPrivateKey': setCertPrivateKey(content); break;
        case 'certChain': setCertChain(content); break;
      }
    };
    reader.readAsText(file);
    e.target.value = '';
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  const canSelectTeam = user?.tenantId && teams.length > 0;
  const canSelectTenant = user?.tenantId && isAdminOrAbove(user.tenantRole);
  const tenantVaultReady = tenantVaultStatus?.initialized && tenantVaultStatus?.hasAccess;

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) handleClose(); }}>
      <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEditMode ? 'Edit Secret' : 'New Secret'}</DialogTitle>
          <DialogDescription className="sr-only">
            {isEditMode ? 'Edit an existing secret' : 'Create a new secret'}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          {error && <Alert variant="destructive">{error}</Alert>}

          {/* Type selector -- only on create */}
          {!isEditMode && (
            <div className="space-y-1.5">
              <Label>Type</Label>
              <Select value={type} onValueChange={(v) => setType(v as SecretType)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="LOGIN">Login</SelectItem>
                  <SelectItem value="SSH_KEY">SSH Key</SelectItem>
                  <SelectItem value="CERTIFICATE">Certificate</SelectItem>
                  <SelectItem value="API_KEY">API Key</SelectItem>
                  <SelectItem value="SECURE_NOTE">Secure Note</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}

          <div className="space-y-1.5">
            <Label>Name *</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>

          <div className="space-y-1.5">
            <Label>Description</Label>
            <Input value={description} onChange={(e) => setDescription(e.target.value)} />
          </div>

          {/* Scope selector -- only on create */}
          {!isEditMode && (
            <div className="space-y-1.5">
              <Label>Scope</Label>
              <Select value={scope} onValueChange={(v) => setScope(v as SecretScope)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="PERSONAL">Personal</SelectItem>
                  {canSelectTeam && <SelectItem value="TEAM">Team</SelectItem>}
                  {canSelectTenant && (
                    <SelectItem value="TENANT" disabled={!tenantVaultReady}>
                      Organization{!tenantVaultReady ? ' (vault not initialized)' : ''}
                    </SelectItem>
                  )}
                </SelectContent>
              </Select>
            </div>
          )}

          {!isEditMode && scope === 'TEAM' && (
            <div className="space-y-1.5">
              <Label>Team</Label>
              <Select value={teamId} onValueChange={setTeamId}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {teams.map((t) => (
                    <SelectItem key={t.id} value={t.id}>{t.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          {/* Dynamic data section */}
          {type === 'LOGIN' && (
            <>
              <div className="space-y-1.5">
                <Label>Username *</Label>
                <Input value={loginUsername} onChange={(e) => setLoginUsername(e.target.value)} />
              </div>
              <div className="space-y-1.5">
                <Label>Password *</Label>
                <div className="relative">
                  <Input
                    type={showPassword ? 'text' : 'password'}
                    value={loginPassword}
                    onChange={(e) => setLoginPassword(e.target.value)}
                    className="pr-20"
                  />
                  <div className="absolute right-1 top-1/2 -translate-y-1/2 flex gap-0.5">
                    <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setShowPassword(!showPassword)}>
                      {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </Button>
                    <Button variant="ghost" size="icon" className="h-7 w-7" title="Generate password" onClick={() => { setLoginPassword(generatePassword()); setShowPassword(true); }}>
                      <Dices className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </div>
              <div className="space-y-1.5">
                <Label>Domain (optional)</Label>
                <Input value={loginDomain} onChange={(e) => setLoginDomain(e.target.value)} placeholder="e.g. CONTOSO" />
              </div>
              <div className="space-y-1.5">
                <Label>URL</Label>
                <Input value={loginUrl} onChange={(e) => setLoginUrl(e.target.value)} />
              </div>
            </>
          )}

          {type === 'SSH_KEY' && (
            <>
              <div className="space-y-1.5">
                <Label>Username</Label>
                <Input value={sshUsername} onChange={(e) => setSshUsername(e.target.value)} />
              </div>
              <div className="space-y-1.5">
                <div className="flex items-center gap-2">
                  <Label className="text-xs text-muted-foreground">Private Key *</Label>
                  <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => handleFileUpload('sshPrivateKey')} title="Upload file">
                    <Upload className="h-3.5 w-3.5" />
                  </Button>
                </div>
                <Textarea
                  value={sshPrivateKey}
                  onChange={(e) => setSshPrivateKey(e.target.value)}
                  rows={4}
                  placeholder="Paste PEM private key or upload file..."
                />
              </div>
              <div className="space-y-1.5">
                <div className="flex items-center gap-2">
                  <Label className="text-xs text-muted-foreground">Public Key</Label>
                  <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => handleFileUpload('sshPublicKey')} title="Upload file">
                    <Upload className="h-3.5 w-3.5" />
                  </Button>
                </div>
                <Textarea value={sshPublicKey} onChange={(e) => setSshPublicKey(e.target.value)} rows={2} />
              </div>
              <div className="space-y-1.5">
                <Label>Passphrase</Label>
                <Input value={sshPassphrase} onChange={(e) => setSshPassphrase(e.target.value)} type="password" />
              </div>
              <div className="space-y-1.5">
                <Label>Algorithm</Label>
                <Select value={sshAlgorithm || '__unspecified__'} onValueChange={(v) => setSshAlgorithm(v === '__unspecified__' ? '' : v)}>
                  <SelectTrigger><SelectValue placeholder="Not specified" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__unspecified__">Not specified</SelectItem>
                    <SelectItem value="RSA">RSA</SelectItem>
                    <SelectItem value="ED25519">ED25519</SelectItem>
                    <SelectItem value="ECDSA">ECDSA</SelectItem>
                    <SelectItem value="DSA">DSA</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </>
          )}

          {type === 'CERTIFICATE' && (
            <>
              <div className="space-y-1.5">
                <div className="flex items-center gap-2">
                  <Label className="text-xs text-muted-foreground">Certificate *</Label>
                  <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => handleFileUpload('certCertificate')} title="Upload file">
                    <Upload className="h-3.5 w-3.5" />
                  </Button>
                </div>
                <Textarea value={certCertificate} onChange={(e) => setCertCertificate(e.target.value)} rows={4} placeholder="Paste PEM certificate or upload .crt/.pem..." />
              </div>
              <div className="space-y-1.5">
                <div className="flex items-center gap-2">
                  <Label className="text-xs text-muted-foreground">Private Key *</Label>
                  <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => handleFileUpload('certPrivateKey')} title="Upload file">
                    <Upload className="h-3.5 w-3.5" />
                  </Button>
                </div>
                <Textarea value={certPrivateKey} onChange={(e) => setCertPrivateKey(e.target.value)} rows={4} placeholder="Paste PEM private key or upload .key..." />
              </div>
              <div className="space-y-1.5">
                <div className="flex items-center gap-2">
                  <Label className="text-xs text-muted-foreground">CA Chain</Label>
                  <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => handleFileUpload('certChain')} title="Upload file">
                    <Upload className="h-3.5 w-3.5" />
                  </Button>
                </div>
                <Textarea value={certChain} onChange={(e) => setCertChain(e.target.value)} rows={2} />
              </div>
              <div className="space-y-1.5">
                <Label>Passphrase</Label>
                <Input value={certPassphrase} onChange={(e) => setCertPassphrase(e.target.value)} type="password" />
              </div>
              <div className="space-y-1.5">
                <Label>Certificate Expires At</Label>
                <Input type="datetime-local" value={certExpiresAt} onChange={(e) => setCertExpiresAt(e.target.value)} />
              </div>
            </>
          )}

          {type === 'API_KEY' && (
            <>
              <div className="space-y-1.5">
                <Label>API Key *</Label>
                <Input value={apiKeyValue} onChange={(e) => setApiKeyValue(e.target.value)} type="password" />
              </div>
              <div className="space-y-1.5">
                <Label>Endpoint URL</Label>
                <Input value={apiKeyEndpoint} onChange={(e) => setApiKeyEndpoint(e.target.value)} />
              </div>
              <div className="space-y-1.5">
                <div className="flex items-center gap-2 mb-1">
                  <Label className="text-xs text-muted-foreground">Headers</Label>
                  <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => setApiKeyHeaders([...apiKeyHeaders, { key: '', value: '' }])}>
                    <Plus className="h-3.5 w-3.5" />
                  </Button>
                </div>
                {apiKeyHeaders.map((h, i) => (
                  <div key={i} className="flex gap-2 mb-2">
                    <Input
                      placeholder="Key"
                      value={h.key}
                      onChange={(e) => {
                        const updated = [...apiKeyHeaders];
                        updated[i] = { ...updated[i], key: e.target.value };
                        setApiKeyHeaders(updated);
                      }}
                      className="flex-1"
                    />
                    <Input
                      placeholder="Value"
                      value={h.value}
                      onChange={(e) => {
                        const updated = [...apiKeyHeaders];
                        updated[i] = { ...updated[i], value: e.target.value };
                        setApiKeyHeaders(updated);
                      }}
                      className="flex-1"
                    />
                    <Button variant="ghost" size="icon" className="h-10 w-10 shrink-0" onClick={() => setApiKeyHeaders(apiKeyHeaders.filter((_, j) => j !== i))}>
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            </>
          )}

          {type === 'SECURE_NOTE' && (
            <div className="space-y-1.5">
              <Label>Content *</Label>
              <Textarea value={noteContent} onChange={(e) => setNoteContent(e.target.value)} rows={6} />
            </div>
          )}

          {type !== 'SECURE_NOTE' && (
            <div className="space-y-1.5">
              <Label>Notes</Label>
              <Textarea value={notes} onChange={(e) => setNotes(e.target.value)} rows={2} />
            </div>
          )}

          {/* Tags */}
          <div className="space-y-1.5">
            <div className="flex gap-2">
              <Input
                placeholder="Press Enter to add tag"
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); handleAddTag(); } }}
                className="flex-1"
              />
              <Button variant="outline" size="sm" onClick={handleAddTag}>Add</Button>
            </div>
            <div className="flex gap-1 flex-wrap">
              {tags.map((t) => (
                <Badge key={t} variant="secondary" className="gap-1">
                  {t}
                  <button onClick={() => setTags(tags.filter((x) => x !== t))} className="ml-0.5 hover:text-destructive">
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              ))}
            </div>
          </div>

          {/* Expiry */}
          <div className="space-y-1.5">
            <Label>Expires At</Label>
            <Input type="datetime-local" value={expiresAt} onChange={(e) => setExpiresAt(e.target.value)} />
          </div>
        </div>

        {/* Hidden file input */}
        <input
          ref={fileInputRef}
          type="file"
          accept=".pem,.key,.crt,.pub,.txt"
          className="hidden"
          onChange={handleFileRead}
        />

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? (isEditMode ? 'Saving...' : 'Creating...') : (isEditMode ? 'Save' : 'Create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
