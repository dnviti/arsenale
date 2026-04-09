import {
  ArrowDownToLine,
  ArrowUpToLine,
  Copy,
  KeyRound,
  Loader2,
  Pencil,
  Play,
  Plus,
  Server,
  ShieldEllipsis,
  Trash2,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import { Textarea } from '@/components/ui/textarea';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import {
  SettingsButtonRow,
  SettingsFieldCard,
  SettingsPanel,
  SettingsStatusBadge,
  SettingsSummaryGrid,
  SettingsSummaryItem,
} from './settings-ui';
import type { GatewayData, SshKeyPairData } from '../../api/gateway.api';
import type { TunnelStatusEvent } from '../../store/gatewayStore';
import ScalingControls from '../orchestration/ScalingControls';
import GatewayInstanceList from '../orchestration/GatewayInstanceList';
import {
  formatGatewayType,
  getGatewayHealthMeta,
  getGatewayModeBadge,
  getGatewayTunnelMeta,
  isGatewayExpandable,
  type GatewayTestState,
} from './gatewaySectionUtils';

function EmptyInventoryState({ onCreateGateway }: { onCreateGateway: () => void }) {
  return (
    <Card className="border-dashed">
      <CardContent className="flex flex-col items-center gap-3 py-12 text-center">
        <Server className="size-10 text-muted-foreground" />
        <div className="space-y-1">
          <div className="text-base font-medium text-foreground">No gateways yet</div>
          <p className="max-w-xl text-sm leading-6 text-muted-foreground">
            Add a gateway to route connections through GUACD, Managed SSH, database proxies,
            or bastion hosts.
          </p>
        </div>
        <Button type="button" onClick={onCreateGateway}>
          <Plus className="size-4" />
          Add Gateway
        </Button>
      </CardContent>
    </Card>
  );
}

function GatewayCard({
  gateway,
  expanded,
  sshKeyReady,
  testState,
  pushState,
  tunnelStatus,
  onDelete,
  onEdit,
  onExpandedChange,
  onPushKey,
  onTest,
}: {
  gateway: GatewayData;
  expanded: boolean;
  pushState?: { loading: boolean; result?: { ok: boolean; error?: string } };
  sshKeyReady: boolean;
  testState?: GatewayTestState;
  tunnelStatus?: TunnelStatusEvent;
  onDelete: (gateway: GatewayData) => void;
  onEdit: (gateway: GatewayData) => void;
  onExpandedChange: (gatewayId: string, expanded: boolean) => void;
  onPushKey: (gateway: GatewayData) => void;
  onTest: (gateway: GatewayData) => void;
}) {
  const health = getGatewayHealthMeta(gateway, testState);
  const tunnel = getGatewayTunnelMeta(gateway, tunnelStatus);
  const expandable = isGatewayExpandable(gateway);
  const endpointValue = gateway.deploymentMode === 'MANAGED_GROUP'
    ? `Managed group · service port ${gateway.port}`
    : `${gateway.host}:${gateway.port}`;

  return (
    <SettingsFieldCard
      label={gateway.name}
      description={gateway.description ?? 'No description provided.'}
      aside={(
        <div className="flex flex-wrap justify-end gap-2">
          <SettingsStatusBadge>{formatGatewayType(gateway.type)}</SettingsStatusBadge>
          <SettingsStatusBadge>{getGatewayModeBadge(gateway)}</SettingsStatusBadge>
          {gateway.isDefault && <SettingsStatusBadge tone="success">Default</SettingsStatusBadge>}
          {gateway.publishPorts && <SettingsStatusBadge tone="warning">Published</SettingsStatusBadge>}
        </div>
      )}
      contentClassName="space-y-4"
    >
      <SettingsSummaryGrid className="xl:grid-cols-3">
        <SettingsSummaryItem label="Endpoint" value={endpointValue} />
        <SettingsSummaryItem label="Health" value={health.label} />
        <SettingsSummaryItem label="Tunnel" value={tunnel.label} />
      </SettingsSummaryGrid>

      <div className="grid gap-1 text-sm text-muted-foreground">
        <p>{health.description}</p>
        <p>{tunnel.description}</p>
      </div>

      {pushState?.result?.error ? (
        <Alert variant="destructive">
          <AlertTitle>SSH key push failed</AlertTitle>
          <AlertDescription>{pushState.result.error}</AlertDescription>
        </Alert>
      ) : pushState?.result?.ok ? (
        <Alert variant="success">
          <AlertTitle>SSH key pushed</AlertTitle>
          <AlertDescription>
            The public key was successfully deployed to this managed SSH gateway.
          </AlertDescription>
        </Alert>
      ) : null}

      <SettingsButtonRow>
        <Button type="button" variant="outline" size="sm" onClick={() => onTest(gateway)}>
          {testState?.loading ? <Loader2 className="size-4 animate-spin" /> : <Play className="size-4" />}
          Test
        </Button>
        {gateway.type === 'MANAGED_SSH' && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={!sshKeyReady || pushState?.loading}
            onClick={() => onPushKey(gateway)}
          >
            {pushState?.loading ? <Loader2 className="size-4 animate-spin" /> : <ShieldEllipsis className="size-4" />}
            Push Key
          </Button>
        )}
        <Button type="button" variant="outline" size="sm" onClick={() => onEdit(gateway)}>
          <Pencil className="size-4" />
          Edit
        </Button>
        <Button type="button" variant="destructive" size="sm" onClick={() => onDelete(gateway)}>
          <Trash2 className="size-4" />
          Delete
        </Button>
      </SettingsButtonRow>

      {expandable && (
        <Accordion
          type="single"
          collapsible
          value={expanded ? 'details' : undefined}
          onValueChange={(value) => onExpandedChange(gateway.id, value === 'details')}
        >
          <AccordionItem value="details" className="border-border/70 bg-background/60">
            <AccordionTrigger>Managed group controls and instances</AccordionTrigger>
            <AccordionContent className="space-y-4">
              <ScalingControls gatewayId={gateway.id} gateway={gateway} />
              {gateway.totalInstances > 0 ? (
                <div className="space-y-2">
                  <div className="text-sm font-medium text-foreground">Instances</div>
                  <GatewayInstanceList gatewayId={gateway.id} />
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No gateway instances are currently registered for this managed group.
                </p>
              )}
            </AccordionContent>
          </AccordionItem>
        </Accordion>
      )}
    </SettingsFieldCard>
  );
}

export function GatewaySshKeyPanel({
  copied,
  keyActionLoading,
  onCopyPublicKey,
  onDownloadPrivateKey,
  onDownloadPublicKey,
  onGenerateKeyPair,
  onRotateKeyPair,
  sshKeyLoading,
  sshKeyPair,
}: {
  copied: boolean;
  keyActionLoading: boolean;
  onCopyPublicKey: () => void;
  onDownloadPrivateKey: () => void;
  onDownloadPublicKey: () => void;
  onGenerateKeyPair: () => void;
  onRotateKeyPair: () => void;
  sshKeyLoading: boolean;
  sshKeyPair: SshKeyPairData | null;
}) {
  return (
    <SettingsPanel
      title="SSH Key Pair"
      description="Manage the tenant-wide key pair used by managed SSH gateways."
      heading={(
        <SettingsStatusBadge tone={sshKeyPair ? 'success' : 'warning'}>
          {sshKeyPair ? 'Ready' : 'Missing'}
        </SettingsStatusBadge>
      )}
      contentClassName="space-y-4"
    >
      {sshKeyLoading ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Loading the current SSH key pair.
        </div>
      ) : !sshKeyPair ? (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
            <KeyRound className="size-10 text-muted-foreground" />
            <div className="space-y-1">
              <div className="text-base font-medium text-foreground">No SSH key pair generated</div>
              <p className="max-w-xl text-sm leading-6 text-muted-foreground">
                Generate a tenant key pair before onboarding Managed SSH gateways so the control
                plane can authenticate cleanly.
              </p>
            </div>
            <Button type="button" onClick={onGenerateKeyPair} disabled={keyActionLoading}>
              {keyActionLoading ? <Loader2 className="size-4 animate-spin" /> : <KeyRound className="size-4" />}
              {keyActionLoading ? 'Generating...' : 'Generate Key Pair'}
            </Button>
          </CardContent>
        </Card>
      ) : (
        <>
          <SettingsSummaryGrid className="xl:grid-cols-3">
            <SettingsSummaryItem label="Algorithm" value={sshKeyPair.algorithm.toUpperCase()} />
            <SettingsSummaryItem label="Fingerprint" value={sshKeyPair.fingerprint} />
            <SettingsSummaryItem
              label="Created"
              value={new Date(sshKeyPair.createdAt).toLocaleDateString()}
            />
          </SettingsSummaryGrid>

          <div className="space-y-2">
            <div className="text-sm font-medium text-foreground">Public Key</div>
            <Textarea
              value={sshKeyPair.publicKey}
              readOnly
              className="min-h-28 font-mono text-xs leading-6"
            />
          </div>

          <SettingsButtonRow>
            <Button type="button" variant="outline" size="sm" onClick={onCopyPublicKey}>
              <Copy className="size-4" />
              {copied ? 'Copied' : 'Copy Public Key'}
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={onDownloadPublicKey}>
              <ArrowDownToLine className="size-4" />
              Download Public Key
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={onDownloadPrivateKey}>
              <ArrowUpToLine className="size-4" />
              Download Private Key
            </Button>
            <Button type="button" variant="outline" size="sm" disabled={keyActionLoading} onClick={onRotateKeyPair}>
              {keyActionLoading ? <Loader2 className="size-4 animate-spin" /> : <ShieldEllipsis className="size-4" />}
              {keyActionLoading ? 'Rotating...' : 'Rotate Key Pair'}
            </Button>
          </SettingsButtonRow>

          <Accordion type="single" collapsible>
            <AccordionItem value="usage" className="border-border/70 bg-background/60">
              <AccordionTrigger>How to use this key</AccordionTrigger>
              <AccordionContent>
                <p className="text-sm leading-6 text-muted-foreground">
                  Use <strong>Push Key</strong> on a managed SSH gateway to deploy the public key over
                  the control channel. You can also place the public key in
                  <code className="mx-1 rounded bg-muted px-1 py-0.5 text-xs">SSH_AUTHORIZED_KEYS</code>
                  or mount it as
                  <code className="mx-1 rounded bg-muted px-1 py-0.5 text-xs">/config/authorized_keys</code>.
                </p>
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </>
      )}
    </SettingsPanel>
  );
}

export function GatewayInventoryPanel({
  expandedGatewayIds,
  gateways,
  loading,
  pushStates,
  sshKeyReady,
  testStates,
  tunnelStatuses,
  onCreateGateway,
  onDeleteGateway,
  onEditGateway,
  onExpandedChange,
  onPushKey,
  onTestGateway,
}: {
  expandedGatewayIds: Set<string>;
  gateways: GatewayData[];
  loading: boolean;
  pushStates: Record<string, { loading: boolean; result?: { ok: boolean; error?: string } }>;
  sshKeyReady: boolean;
  testStates: Record<string, GatewayTestState>;
  tunnelStatuses: Record<string, TunnelStatusEvent>;
  onCreateGateway: () => void;
  onDeleteGateway: (gateway: GatewayData) => void;
  onEditGateway: (gateway: GatewayData) => void;
  onExpandedChange: (gatewayId: string, expanded: boolean) => void;
  onPushKey: (gateway: GatewayData) => void;
  onTestGateway: (gateway: GatewayData) => void;
}) {
  return (
    <SettingsPanel
      title="Gateway Inventory"
      description="Review transport endpoints, health, tunnels, and managed groups without a dense admin table."
      heading={(
        <Button type="button" variant="outline" size="sm" onClick={onCreateGateway}>
          <Plus className="size-4" />
          New Gateway
        </Button>
      )}
      contentClassName="space-y-4"
    >
      {loading ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Loading gateways.
        </div>
      ) : gateways.length === 0 ? (
        <EmptyInventoryState onCreateGateway={onCreateGateway} />
      ) : (
        gateways.map((gateway) => (
          <GatewayCard
            key={gateway.id}
            gateway={gateway}
            expanded={expandedGatewayIds.has(gateway.id)}
            pushState={pushStates[gateway.id]}
            sshKeyReady={sshKeyReady}
            testState={testStates[gateway.id]}
            tunnelStatus={tunnelStatuses[gateway.id]}
            onDelete={onDeleteGateway}
            onEdit={onEditGateway}
            onExpandedChange={onExpandedChange}
            onPushKey={onPushKey}
            onTest={onTestGateway}
          />
        ))
      )}
    </SettingsPanel>
  );
}
