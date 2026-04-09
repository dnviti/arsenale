import { useState, useEffect, useRef, useMemo } from 'react';
import {
  Pencil, Share2, Trash2, Star, Copy, Eye, EyeOff, ChevronDown,
  KeyRound, Key, ShieldCheck, Webhook, StickyNote,
  ExternalLink, Link, ShieldAlert, Shield, Loader2,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert } from '@/components/ui/alert';
import { Separator } from '@/components/ui/separator';
import {
  Accordion, AccordionContent, AccordionItem, AccordionTrigger,
} from '@/components/ui/accordion';
import { cn } from '@/lib/utils';
import type { SecretDetail, SecretPayload, SecretType, SecretScope } from '../../api/secrets.api';
import SecretVersionHistory from './SecretVersionHistory';
import PasswordRotationPanel from './PasswordRotationPanel';
import { useCopyToClipboard } from '../../hooks/useCopyToClipboard';

const TYPE_ICONS: Record<SecretType, React.ReactNode> = {
  LOGIN: <KeyRound className="h-5 w-5" />,
  SSH_KEY: <Key className="h-5 w-5" />,
  CERTIFICATE: <ShieldCheck className="h-5 w-5" />,
  API_KEY: <Webhook className="h-5 w-5" />,
  SECURE_NOTE: <StickyNote className="h-5 w-5" />,
};

const TYPE_LABELS: Record<SecretType, string> = {
  LOGIN: 'Login',
  SSH_KEY: 'SSH Key',
  CERTIFICATE: 'Certificate',
  API_KEY: 'API Key',
  SECURE_NOTE: 'Secure Note',
};

const SCOPE_LABELS: Record<SecretScope, string> = {
  PERSONAL: 'Personal',
  TEAM: 'Team',
  TENANT: 'Organization',
};

const AUTO_HIDE_MS = 30_000;

interface SecretDetailViewProps {
  secret: SecretDetail;
  onEdit: () => void;
  onShare: () => void;
  onExternalShare?: () => void;
  onDelete: () => void;
  onToggleFavorite: () => void;
  onRestore: () => void;
  onCheckBreach?: (secretId: string) => Promise<number>;
}

function SensitiveField({ label, value }: { label: string; value: string }) {
  const [revealed, setRevealed] = useState(false);
  const { copied, copy: handleCopy } = useCopyToClipboard();
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => { if (timerRef.current) clearTimeout(timerRef.current); };
  }, []);

  const handleReveal = () => {
    setRevealed(true);
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => setRevealed(false), AUTO_HIDE_MS);
  };

  const handleHide = () => {
    setRevealed(false);
    if (timerRef.current) clearTimeout(timerRef.current);
  };

  return (
    <div className="mb-3">
      <span className="text-xs text-muted-foreground">{label}</span>
      <div className="flex items-center gap-1">
        <p
          className={cn(
            'flex-1 text-sm break-all',
            revealed && 'font-mono whitespace-pre-wrap',
          )}
        >
          {revealed ? value : '\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022'}
        </p>
        <Button
          variant="ghost" size="icon" className="h-7 w-7"
          onClick={revealed ? handleHide : handleReveal}
          title={revealed ? 'Hide' : 'Reveal'}
        >
          {revealed ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
        </Button>
        <Button
          variant="ghost" size="icon" className="h-7 w-7"
          onClick={() => handleCopy(value)}
          title={copied ? 'Copied!' : 'Copy'}
        >
          <Copy className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

function PlainField({ label, value, copyable, isLink }: { label: string; value: string; copyable?: boolean; isLink?: boolean }) {
  const { copied, copy: handleCopy } = useCopyToClipboard();

  return (
    <div className="mb-3">
      <span className="text-xs text-muted-foreground">{label}</span>
      <div className="flex items-center gap-1">
        <p className="flex-1 text-sm break-all whitespace-pre-wrap">
          {value}
        </p>
        {isLink && value && (
          <Button
            variant="ghost" size="icon" className="h-7 w-7"
            onClick={() => window.open(value.startsWith('http') ? value : `https://${value}`, '_blank')}
            title="Open in browser"
          >
            <ExternalLink className="h-4 w-4" />
          </Button>
        )}
        {copyable && (
          <Button
            variant="ghost" size="icon" className="h-7 w-7"
            onClick={() => handleCopy(value)}
            title={copied ? 'Copied!' : 'Copy'}
          >
            <Copy className="h-4 w-4" />
          </Button>
        )}
      </div>
    </div>
  );
}

function renderSecretFields(data: SecretPayload) {
  switch (data.type) {
    case 'LOGIN':
      return (
        <>
          <PlainField label="Username" value={data.username} copyable />
          <SensitiveField label="Password" value={data.password} />
          {data.domain && <PlainField label="Domain" value={data.domain} copyable />}
          {data.url && <PlainField label="URL" value={data.url} copyable isLink />}
          {data.notes && <PlainField label="Notes" value={data.notes} />}
        </>
      );
    case 'SSH_KEY':
      return (
        <>
          {data.username && <PlainField label="Username" value={data.username} copyable />}
          <SensitiveField label="Private Key" value={data.privateKey} />
          {data.publicKey && <PlainField label="Public Key" value={data.publicKey} copyable />}
          {data.passphrase && <SensitiveField label="Passphrase" value={data.passphrase} />}
          {data.algorithm && <PlainField label="Algorithm" value={data.algorithm} />}
          {data.notes && <PlainField label="Notes" value={data.notes} />}
        </>
      );
    case 'CERTIFICATE':
      return (
        <>
          <SensitiveField label="Certificate" value={data.certificate} />
          <SensitiveField label="Private Key" value={data.privateKey} />
          {data.chain && <SensitiveField label="CA Chain" value={data.chain} />}
          {data.passphrase && <SensitiveField label="Passphrase" value={data.passphrase} />}
          {data.expiresAt && <PlainField label="Certificate Expires" value={new Date(data.expiresAt).toLocaleDateString()} />}
          {data.notes && <PlainField label="Notes" value={data.notes} />}
        </>
      );
    case 'API_KEY':
      return (
        <>
          <SensitiveField label="API Key" value={data.apiKey} />
          {data.endpoint && <PlainField label="Endpoint" value={data.endpoint} copyable isLink />}
          {data.headers && Object.entries(data.headers).length > 0 && (
            <div className="mb-3">
              <span className="text-xs text-muted-foreground">Headers</span>
              {Object.entries(data.headers).map(([k, v]) => (
                <p key={k} className="font-mono text-xs">
                  {k}: {v}
                </p>
              ))}
            </div>
          )}
          {data.notes && <PlainField label="Notes" value={data.notes} />}
        </>
      );
    case 'SECURE_NOTE':
      return <PlainField label="Content" value={data.content} copyable />;
  }
}

export default function SecretDetailView({
  secret,
  onEdit,
  onShare,
  onExternalShare,
  onDelete,
  onToggleFavorite,
  onRestore,
  onCheckBreach,
}: SecretDetailViewProps) {
  const [breachChecking, setBreachChecking] = useState(false);

  const formatDate = (iso: string) =>
    new Date(iso).toLocaleDateString(undefined, {
      month: 'short', day: 'numeric', year: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });

  const daysUntilExpiry = useMemo(() => {
    if (!secret.expiresAt) return null;
    const now = new Date();
    return Math.ceil((new Date(secret.expiresAt).getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
  }, [secret.expiresAt]);

  const isReadOnly = secret.shared && secret.permission === 'READ_ONLY';

  const hasCheckablePassword = ['LOGIN', 'SSH_KEY', 'CERTIFICATE'].includes(secret.type);

  const handleCheckBreach = async () => {
    if (!onCheckBreach) return;
    setBreachChecking(true);
    try {
      await onCheckBreach(secret.id);
    } finally {
      setBreachChecking(false);
    }
  };

  return (
    <div className="p-4 overflow-auto h-full">
      {/* Header */}
      <div className="flex items-center gap-2 mb-4">
        {TYPE_ICONS[secret.type]}
        <h3 className="text-lg font-semibold flex-1">{secret.name}</h3>
        <Badge variant="outline">{TYPE_LABELS[secret.type]}</Badge>
        <Badge variant={secret.scope === 'PERSONAL' ? 'secondary' : 'default'}>
          {SCOPE_LABELS[secret.scope]}
        </Badge>
        {secret.shared && (
          <Badge variant="outline">
            {secret.permission === 'READ_ONLY' ? 'Read Only' : 'Full Access'}
          </Badge>
        )}
      </div>

      {/* Action buttons */}
      <div className="flex gap-1 mb-4">
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onToggleFavorite} title="Toggle favorite">
          <Star className={cn('h-4 w-4', secret.isFavorite && 'fill-yellow-500 text-yellow-500')} />
        </Button>
        {!isReadOnly && (
          <>
            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onEdit} title="Edit">
              <Pencil className="h-4 w-4" />
            </Button>
            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onShare} title="Share">
              <Share2 className="h-4 w-4" />
            </Button>
            {onExternalShare && (
              <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onExternalShare} title="External share link">
                <Link className="h-4 w-4" />
              </Button>
            )}
            <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive" onClick={onDelete} title="Delete">
              <Trash2 className="h-4 w-4" />
            </Button>
          </>
        )}
      </div>

      {secret.pwnedCount > 0 && (
        <Alert variant="destructive" className="mb-4 flex items-start gap-2">
          <ShieldAlert className="h-4 w-4 mt-0.5 shrink-0" />
          <div className="flex-1">
            Password found in {secret.pwnedCount.toLocaleString()} data breach(es). You should change this password immediately.
          </div>
          {!isReadOnly && (
            <Button variant="destructive" size="sm" onClick={onEdit}>Rotate</Button>
          )}
        </Alert>
      )}

      {daysUntilExpiry !== null && daysUntilExpiry <= 30 && (
        <Alert
          variant={daysUntilExpiry <= 0 ? 'destructive' : daysUntilExpiry <= 7 ? 'warning' : 'info'}
          className="mb-4"
        >
          {daysUntilExpiry <= 0
            ? 'This secret has expired. Update the credentials or the expiry date.'
            : `This secret expires in ${daysUntilExpiry} day(s). Consider rotating credentials.`}
        </Alert>
      )}

      {hasCheckablePassword && secret.pwnedCount === 0 && onCheckBreach && (
        <div className="mb-4">
          <Button
            variant="outline"
            size="sm"
            onClick={handleCheckBreach}
            disabled={breachChecking}
          >
            {breachChecking ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Shield className="h-4 w-4 mr-2" />}
            {breachChecking ? 'Checking...' : 'Check for breaches'}
          </Button>
        </div>
      )}

      {secret.description && (
        <p className="text-sm text-muted-foreground mb-4">
          {secret.description}
        </p>
      )}

      <Separator className="mb-4" />

      {/* Type-specific fields */}
      {renderSecretFields(secret.data)}

      <Separator className="my-4" />

      {/* Metadata */}
      <div className="flex flex-wrap gap-1 mb-2">
        {secret.tags.map((t) => (
          <Badge key={t} variant="outline">{t}</Badge>
        ))}
      </div>

      <span className="text-xs text-muted-foreground block">
        Created: {formatDate(secret.createdAt)}
      </span>
      <span className="text-xs text-muted-foreground block">
        Updated: {formatDate(secret.updatedAt)}
      </span>
      <span className="text-xs text-muted-foreground block">
        Version: {secret.currentVersion}
      </span>

      {daysUntilExpiry !== null && (
        <Badge
          variant={daysUntilExpiry <= 7 ? 'destructive' : daysUntilExpiry <= 30 ? 'secondary' : 'outline'}
          className="mt-2"
        >
          {daysUntilExpiry <= 0 ? 'Expired' : `Expires in ${daysUntilExpiry} day(s)`}
        </Badge>
      )}

      {/* Password Rotation (LOGIN secrets only) */}
      {secret.type === 'LOGIN' && (
        <Accordion type="single" collapsible className="mt-4">
          <AccordionItem value="rotation">
            <AccordionTrigger>
              <span className="text-sm font-medium">Password Rotation</span>
            </AccordionTrigger>
            <AccordionContent>
              <PasswordRotationPanel secretId={secret.id} isReadOnly={isReadOnly} />
            </AccordionContent>
          </AccordionItem>
        </Accordion>
      )}

      {/* Version history */}
      <Accordion type="single" collapsible className="mt-4">
        <AccordionItem value="history">
          <AccordionTrigger>
            <span className="text-sm font-medium">
              Version History
              {secret.currentVersion > 1 && (
                <Badge variant="secondary" className="ml-2 text-[0.65rem] px-1.5 py-0">
                  {secret.currentVersion} versions
                </Badge>
              )}
            </span>
          </AccordionTrigger>
          <AccordionContent>
            <SecretVersionHistory
              secretId={secret.id}
              currentVersion={secret.currentVersion}
              currentData={secret.data}
              onRestore={onRestore}
            />
          </AccordionContent>
        </AccordionItem>
      </Accordion>
    </div>
  );
}
