import { useEffect, useState } from 'react';
import { AlertCircle, Loader2, ShieldEllipsis } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useCopyToClipboard } from '../../hooks/useCopyToClipboard';
import { testGateway, downloadSshPrivateKey } from '../../api/gateway.api';
import type { GatewayData } from '../../api/gateway.api';
import { useAuthStore } from '../../store/authStore';
import { useGatewayStore } from '../../store/gatewayStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import { extractApiError } from '../../utils/apiError';
import GatewayDialog from '../gateway/GatewayDialog';
import GatewayTemplateSection from '../gateway/GatewayTemplateSection';
import SessionDashboard from '../orchestration/SessionDashboard';
import { GatewayInventoryPanel, GatewaySshKeyPanel } from './gatewaySectionCards';
import { GatewayDeleteDialog, GatewayForceDeleteDialog, GatewayRotateKeyDialog } from './gatewaySectionDialogs';
import { triggerTextDownload, type GatewayTestState } from './gatewaySectionUtils';
import { SettingsPanel, SettingsSummaryGrid, SettingsSummaryItem } from './settings-ui';

interface GatewaySectionProps {
  onNavigateToTab?: (tabId: string) => void;
}

type GatewaySubTab = 'gateways' | 'sessions' | 'templates';

export default function GatewaySection({ onNavigateToTab }: GatewaySectionProps) {
  const user = useAuthStore((state) => state.user);
  const permissionsLoaded = useAuthStore((state) => state.permissionsLoaded);
  const canManageGateways = useAuthStore((state) => state.permissions.canManageGateways);
  const canManageSessions = useAuthStore((state) => state.permissions.canManageSessions);
  const gateways = useGatewayStore((state) => state.gateways);
  const loading = useGatewayStore((state) => state.loading);
  const fetchGateways = useGatewayStore((state) => state.fetchGateways);
  const deleteGatewayAction = useGatewayStore((state) => state.deleteGateway);
  const sshKeyPair = useGatewayStore((state) => state.sshKeyPair);
  const sshKeyLoading = useGatewayStore((state) => state.sshKeyLoading);
  const fetchSshKeyPair = useGatewayStore((state) => state.fetchSshKeyPair);
  const generateSshKeyPairAction = useGatewayStore((state) => state.generateSshKeyPair);
  const rotateSshKeyPairAction = useGatewayStore((state) => state.rotateSshKeyPair);
  const pushKeyToGatewayAction = useGatewayStore((state) => state.pushKeyToGateway);
  const applyHealthUpdate = useGatewayStore((state) => state.applyHealthUpdate);
  const tunnelStatuses = useGatewayStore((state) => state.tunnelStatuses);
  const subTab = useUiPreferencesStore((state) => state.gatewayActiveSubTab);
  const setSubTab = useUiPreferencesStore((state) => state.set);

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingGateway, setEditingGateway] = useState<GatewayData | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<GatewayData | null>(null);
  const [forceDeleteInfo, setForceDeleteInfo] = useState<{ gateway: GatewayData; connectionCount: number } | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState('');
  const [rotatePushInfo, setRotatePushInfo] = useState<string | null>(null);
  const [testStates, setTestStates] = useState<Record<string, GatewayTestState>>({});
  const [pushStates, setPushStates] = useState<Record<string, { loading: boolean; result?: { ok: boolean; error?: string } }>>({});
  const [keyActionLoading, setKeyActionLoading] = useState(false);
  const [rotateConfirmOpen, setRotateConfirmOpen] = useState(false);
  const [expandedGatewayIds, setExpandedGatewayIds] = useState<Set<string>>(new Set());
  const { copied, copy: copyToClipboard } = useCopyToClipboard();

  const hasTenant = Boolean(user?.tenantId);
  const currentTab = subTab as GatewaySubTab;
  const totalGateways = gateways.length;
  const managedGateways = gateways.filter((gateway) => gateway.deploymentMode === 'MANAGED_GROUP').length;
  const tunnelEnabledGateways = gateways.filter((gateway) => gateway.tunnelEnabled).length;
  const defaultGateways = gateways.filter((gateway) => gateway.isDefault).length;

  useEffect(() => {
    if (hasTenant && permissionsLoaded && canManageGateways) {
      fetchGateways();
      fetchSshKeyPair();
    }
  }, [canManageGateways, fetchGateways, fetchSshKeyPair, hasTenant, permissionsLoaded]);

  useEffect(() => {
    const allowedTabs = new Set<GatewaySubTab>(['gateways']);
    if (canManageSessions) allowedTabs.add('sessions');
    if (canManageGateways) allowedTabs.add('templates');
    if (!allowedTabs.has(currentTab)) {
      setSubTab('gatewayActiveSubTab', 'gateways');
    }
  }, [canManageGateways, canManageSessions, currentTab, setSubTab]);

  const handleExpandedChange = (gatewayId: string, expanded: boolean) => {
    setExpandedGatewayIds((previous) => {
      const next = new Set(previous);
      if (expanded) next.add(gatewayId);
      else next.delete(gatewayId);
      return next;
    });
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    setError('');
    try {
      await deleteGatewayAction(deleteTarget.id);
      setDeleteTarget(null);
    } catch (requestError: unknown) {
      const response = (requestError as { response?: { status?: number; data?: { connectionCount?: number } } }).response;
      if (response?.status === 409 && response.data?.connectionCount) {
        setForceDeleteInfo({
          gateway: deleteTarget,
          connectionCount: response.data.connectionCount,
        });
        setDeleteTarget(null);
      } else {
        setError(extractApiError(requestError, 'Failed to delete gateway'));
        setDeleteTarget(null);
      }
    } finally {
      setDeleting(false);
    }
  };

  const handleForceDelete = async () => {
    if (!forceDeleteInfo) return;
    setDeleting(true);
    setError('');
    try {
      await deleteGatewayAction(forceDeleteInfo.gateway.id, true);
      setForceDeleteInfo(null);
    } catch (requestError: unknown) {
      setError(extractApiError(requestError, 'Failed to delete gateway'));
    } finally {
      setDeleting(false);
    }
  };

  const handleTestGateway = async (gateway: GatewayData) => {
    setTestStates((previous) => ({
      ...previous,
      [gateway.id]: { gatewayId: gateway.id, loading: true },
    }));

    try {
      const result = await testGateway(gateway.id);
      setTestStates((previous) => ({
        ...previous,
        [gateway.id]: { gatewayId: gateway.id, loading: false, result },
      }));
      applyHealthUpdate({
        gatewayId: gateway.id,
        status: result.reachable ? 'REACHABLE' : 'UNREACHABLE',
        latencyMs: result.latencyMs,
        error: result.error,
        checkedAt: new Date().toISOString(),
      });
    } catch {
      setTestStates((previous) => ({
        ...previous,
        [gateway.id]: {
          gatewayId: gateway.id,
          loading: false,
          result: { reachable: false, latencyMs: null, error: 'Test request failed' },
        },
      }));
    }
  };

  const handlePushKey = async (gateway: GatewayData) => {
    setPushStates((previous) => ({ ...previous, [gateway.id]: { loading: true } }));
    try {
      const result = await pushKeyToGatewayAction(gateway.id);
      setPushStates((previous) => ({
        ...previous,
        [gateway.id]: { loading: false, result },
      }));
    } catch (requestError: unknown) {
      setPushStates((previous) => ({
        ...previous,
        [gateway.id]: {
          loading: false,
          result: { ok: false, error: extractApiError(requestError, 'Push key request failed') },
        },
      }));
    }
  };

  const handleGenerateKeyPair = async () => {
    setKeyActionLoading(true);
    setError('');
    try {
      await generateSshKeyPairAction();
    } catch (requestError: unknown) {
      setError(extractApiError(requestError, 'Failed to generate SSH key pair'));
    } finally {
      setKeyActionLoading(false);
    }
  };

  const handleRotateKeyPair = async () => {
    setRotateConfirmOpen(false);
    setKeyActionLoading(true);
    setError('');
    setRotatePushInfo(null);
    try {
      const result = await rotateSshKeyPairAction();
      if (result.pushResults?.length) {
        const okCount = result.pushResults.filter((item) => item.ok).length;
        const failed = result.pushResults.filter((item) => !item.ok);
        let message = `Key rotated and pushed to ${okCount}/${result.pushResults.length} gateway(s).`;
        if (failed.length > 0) {
          message += ` Failed: ${failed.map((item) => `${item.name} (${item.error})`).join(', ')}`;
        }
        setRotatePushInfo(message);
      }
    } catch (requestError: unknown) {
      setError(extractApiError(requestError, 'Failed to rotate SSH key pair'));
    } finally {
      setKeyActionLoading(false);
    }
  };

  const handleCopyPublicKey = async () => {
    if (!sshKeyPair) return;
    await copyToClipboard(sshKeyPair.publicKey);
  };

  const handleDownloadPublicKey = () => {
    if (!sshKeyPair) return;
    triggerTextDownload(sshKeyPair.publicKey, 'tenant_ed25519.pub');
  };

  const handleDownloadPrivateKey = async () => {
    setError('');
    try {
      const pem = await downloadSshPrivateKey();
      triggerTextDownload(pem, 'tenant_ed25519');
    } catch (requestError: unknown) {
      setError(extractApiError(requestError, 'Failed to download private key'));
    }
  };

  if (!hasTenant) {
    return (
      <SettingsPanel
        title="Gateway access"
        description="Create or join an organization before managing gateways, sessions, and templates."
        contentClassName="space-y-4"
      >
        <Alert variant="warning">
          <AlertCircle className="size-4" />
          <AlertTitle>No organization yet</AlertTitle>
          <AlertDescription>
            Gateway administration is only available inside an organization workspace.
          </AlertDescription>
        </Alert>
        <Button type="button" onClick={() => onNavigateToTab?.('organization')}>
          Set Up Organization
        </Button>
      </SettingsPanel>
    );
  }

  if (!permissionsLoaded) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        Loading gateway permissions.
      </div>
    );
  }

  if (!canManageGateways) {
    return (
      <Alert variant="warning">
        <AlertCircle className="size-4" />
        <AlertTitle>Gateway access is restricted</AlertTitle>
        <AlertDescription>
          You do not have permission to manage gateways for this organization.
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="space-y-6">
      {error ? (
        <Alert variant="destructive">
          <AlertCircle className="size-4" />
          <AlertTitle>Gateway action failed</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      {rotatePushInfo ? (
        <Alert variant={rotatePushInfo.includes('Failed') ? 'warning' : 'success'}>
          <ShieldEllipsis className="size-4" />
          <AlertTitle>SSH key rotation finished</AlertTitle>
          <AlertDescription>{rotatePushInfo}</AlertDescription>
        </Alert>
      ) : null}

      <SettingsSummaryGrid>
        <SettingsSummaryItem label="Total gateways" value={String(totalGateways)} />
        <SettingsSummaryItem label="Managed groups" value={String(managedGateways)} />
        <SettingsSummaryItem label="Tunnel-enabled" value={String(tunnelEnabledGateways)} />
        <SettingsSummaryItem label="Default routes" value={String(defaultGateways)} />
      </SettingsSummaryGrid>

      <Tabs value={currentTab} onValueChange={(value) => setSubTab('gatewayActiveSubTab', value)}>
        <TabsList>
          <TabsTrigger value="gateways">Gateways</TabsTrigger>
          {canManageSessions && <TabsTrigger value="sessions">Active Sessions</TabsTrigger>}
          {canManageGateways && <TabsTrigger value="templates">Templates</TabsTrigger>}
        </TabsList>

        <TabsContent value="gateways" className="space-y-6">
          <GatewaySshKeyPanel
            copied={copied}
            keyActionLoading={keyActionLoading}
            onCopyPublicKey={handleCopyPublicKey}
            onDownloadPrivateKey={handleDownloadPrivateKey}
            onDownloadPublicKey={handleDownloadPublicKey}
            onGenerateKeyPair={handleGenerateKeyPair}
            onRotateKeyPair={() => setRotateConfirmOpen(true)}
            sshKeyLoading={sshKeyLoading}
            sshKeyPair={sshKeyPair}
          />

          <GatewayInventoryPanel
            expandedGatewayIds={expandedGatewayIds}
            gateways={gateways}
            loading={loading}
            pushStates={pushStates}
            sshKeyReady={Boolean(sshKeyPair)}
            testStates={testStates}
            tunnelStatuses={tunnelStatuses}
            onCreateGateway={() => {
              setEditingGateway(null);
              setDialogOpen(true);
            }}
            onDeleteGateway={setDeleteTarget}
            onEditGateway={(gateway) => {
              setEditingGateway(gateway);
              setDialogOpen(true);
            }}
            onExpandedChange={handleExpandedChange}
            onPushKey={handlePushKey}
            onTestGateway={handleTestGateway}
          />
        </TabsContent>

      {canManageSessions && (
          <TabsContent value="sessions"><SessionDashboard /></TabsContent>
        )}

        {canManageGateways && (
          <TabsContent value="templates"><GatewayTemplateSection /></TabsContent>
        )}
      </Tabs>

      <GatewayDialog
        open={dialogOpen}
        onClose={() => {
          setDialogOpen(false);
          setEditingGateway(null);
        }}
        gateway={editingGateway}
      />

      <GatewayDeleteDialog
        deleting={deleting}
        gateway={deleteTarget}
        onConfirm={handleDelete}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}
      />

      <GatewayForceDeleteDialog
        connectionCount={forceDeleteInfo?.connectionCount ?? 0}
        deleting={deleting}
        gateway={forceDeleteInfo?.gateway ?? null}
        onConfirm={handleForceDelete}
        onOpenChange={(open) => { if (!open) setForceDeleteInfo(null); }}
      />

      <GatewayRotateKeyDialog
        loading={keyActionLoading}
        open={rotateConfirmOpen}
        onConfirm={handleRotateKeyPair}
        onOpenChange={setRotateConfirmOpen}
      />
    </div>
  );
}
