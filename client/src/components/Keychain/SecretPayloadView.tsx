import type { SecretPayload } from '@/api/secrets.api';

function SecretField({ label, value, preformatted }: { label: string; value?: string; preformatted?: boolean }) {
  if (!value) return null;
  return (
    <div className="space-y-1">
      <div className="text-xs font-medium text-muted-foreground">{label}</div>
      <div className={preformatted ? 'whitespace-pre-wrap break-all font-mono text-sm' : 'break-all text-sm'}>
        {value}
      </div>
    </div>
  );
}

function HeaderFields({ headers }: { headers?: Record<string, string> }) {
  if (!headers || Object.keys(headers).length === 0) return null;
  return (
    <div className="space-y-1">
      <div className="text-xs font-medium text-muted-foreground">Headers</div>
      <div className="space-y-1 font-mono text-xs">
        {Object.entries(headers).map(([key, value]) => (
          <div key={key} className="break-all">
            {key}: {value}
          </div>
        ))}
      </div>
    </div>
  );
}

export default function SecretPayloadView({ data }: { data: SecretPayload }) {
  switch (data.type) {
    case 'LOGIN':
      return (
        <div className="space-y-3">
          <SecretField label="Username" value={data.username} />
          <SecretField label="Password" value={data.password} preformatted />
          <SecretField label="Domain" value={data.domain} />
          <SecretField label="URL" value={data.url} />
          <SecretField label="Notes" value={data.notes} preformatted />
        </div>
      );
    case 'SSH_KEY':
      return (
        <div className="space-y-3">
          <SecretField label="Username" value={data.username} />
          <SecretField label="Private Key" value={data.privateKey} preformatted />
          <SecretField label="Public Key" value={data.publicKey} preformatted />
          <SecretField label="Passphrase" value={data.passphrase} preformatted />
          <SecretField label="Algorithm" value={data.algorithm} />
          <SecretField label="Notes" value={data.notes} preformatted />
        </div>
      );
    case 'CERTIFICATE':
      return (
        <div className="space-y-3">
          <SecretField label="Certificate" value={data.certificate} preformatted />
          <SecretField label="Private Key" value={data.privateKey} preformatted />
          <SecretField label="CA Chain" value={data.chain} preformatted />
          <SecretField label="Passphrase" value={data.passphrase} preformatted />
          <SecretField label="Expires" value={data.expiresAt ? new Date(data.expiresAt).toLocaleDateString() : undefined} />
          <SecretField label="Notes" value={data.notes} preformatted />
        </div>
      );
    case 'API_KEY':
      return (
        <div className="space-y-3">
          <SecretField label="API Key" value={data.apiKey} preformatted />
          <SecretField label="Endpoint" value={data.endpoint} />
          <HeaderFields headers={data.headers} />
          <SecretField label="Notes" value={data.notes} preformatted />
        </div>
      );
    case 'SECURE_NOTE':
      return (
        <div className="space-y-3">
          <SecretField label="Content" value={data.content} preformatted />
        </div>
      );
  }
}
