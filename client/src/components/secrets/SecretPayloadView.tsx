import type { SecretPayload } from '../../api/secrets.api';

interface SecretPayloadViewProps {
  data: SecretPayload;
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="space-y-1">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="break-all whitespace-pre-wrap text-sm text-foreground">{value}</div>
    </div>
  );
}

function formatCertificateExpiry(iso: string) {
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(new Date(iso));
}

export default function SecretPayloadView({ data }: SecretPayloadViewProps) {
  switch (data.type) {
    case 'LOGIN':
      return (
        <div className="space-y-4">
          <Field label="Username" value={data.username} />
          <Field label="Password" value={data.password} />
          {data.url && <Field label="URL" value={data.url} />}
          {data.domain && <Field label="Domain" value={data.domain} />}
          {data.notes && <Field label="Notes" value={data.notes} />}
        </div>
      );
    case 'SSH_KEY':
      return (
        <div className="space-y-4">
          {data.username && <Field label="Username" value={data.username} />}
          <Field label="Private Key" value={data.privateKey} />
          {data.publicKey && <Field label="Public Key" value={data.publicKey} />}
          {data.passphrase && <Field label="Passphrase" value={data.passphrase} />}
          {data.algorithm && <Field label="Algorithm" value={data.algorithm} />}
          {data.notes && <Field label="Notes" value={data.notes} />}
        </div>
      );
    case 'CERTIFICATE':
      return (
        <div className="space-y-4">
          <Field label="Certificate" value={data.certificate} />
          <Field label="Private Key" value={data.privateKey} />
          {data.chain && <Field label="CA Chain" value={data.chain} />}
          {data.passphrase && <Field label="Passphrase" value={data.passphrase} />}
          {data.expiresAt && <Field label="Certificate Expires" value={formatCertificateExpiry(data.expiresAt)} />}
          {data.notes && <Field label="Notes" value={data.notes} />}
        </div>
      );
    case 'API_KEY':
      return (
        <div className="space-y-4">
          <Field label="API Key" value={data.apiKey} />
          {data.endpoint && <Field label="Endpoint" value={data.endpoint} />}
          {data.headers && Object.entries(data.headers).length > 0 && (
            <Field label="Headers" value={JSON.stringify(data.headers, null, 2)} />
          )}
          {data.notes && <Field label="Notes" value={data.notes} />}
        </div>
      );
    case 'SECURE_NOTE':
      return <Field label="Content" value={data.content} />;
  }
}
