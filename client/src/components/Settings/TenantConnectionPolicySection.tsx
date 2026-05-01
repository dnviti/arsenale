import { useMemo, useState } from 'react';
import { ShieldCheck } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { useTenantStore } from '../../store/tenantStore';
import { useNotificationStore } from '../../store/notificationStore';
import type { EnforcedConnectionSettings } from '../../api/tenant.api';
import type { SshTerminalConfig } from '../../constants/terminalThemes';
import type { RdpSettings } from '../../constants/rdpDefaults';
import { RDP_DEFAULTS } from '../../constants/rdpDefaults';
import type { VncSettings } from '../../constants/vncDefaults';
import { VNC_DEFAULTS } from '../../constants/vncDefaults';
import { TERMINAL_DEFAULTS } from '../../constants/terminalThemes';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import RdpSettingsSection from './RdpSettingsSection';
import TerminalSettingsSection from './TerminalSettingsSection';
import VncSettingsSection from './VncSettingsSection';
import {
  SettingsButtonRow,
  SettingsPanel,
  SettingsSectionBlock,
  SettingsStatusBadge,
  SettingsSummaryGrid,
  SettingsSummaryItem,
} from './settings-ui';

type PolicyType = 'ssh' | 'rdp' | 'vnc';
type PolicySettingsMap = {
  ssh: Partial<SshTerminalConfig>;
  rdp: Partial<RdpSettings>;
  vnc: Partial<VncSettings>;
};

interface ConnectionPolicyDraft {
  enabled: Record<PolicyType, boolean>;
  expandedPolicies: PolicyType[];
  settings: PolicySettingsMap;
}

const POLICY_TYPES: PolicyType[] = ['ssh', 'rdp', 'vnc'];

function hasPolicySettings(settings: object) {
  return Object.keys(settings).length > 0;
}

function createPolicyDraft(
  enforcedSettings: EnforcedConnectionSettings | null | undefined,
): ConnectionPolicyDraft {
  const settings: PolicySettingsMap = {
    ssh: { ...(enforcedSettings?.ssh ?? {}) },
    rdp: { ...(enforcedSettings?.rdp ?? {}) },
    vnc: { ...(enforcedSettings?.vnc ?? {}) },
  };
  const expandedPolicies = POLICY_TYPES.filter((policy) => hasPolicySettings(settings[policy]));

  return {
    enabled: {
      ssh: expandedPolicies.includes('ssh'),
      rdp: expandedPolicies.includes('rdp'),
      vnc: expandedPolicies.includes('vnc'),
    },
    expandedPolicies,
    settings,
  };
}

interface TenantConnectionPolicyEditorProps {
  error: string;
  initialDraft: ConnectionPolicyDraft;
  loading: boolean;
  onClearPolicy: () => Promise<boolean>;
  onSavePolicy: (draft: ConnectionPolicyDraft) => Promise<boolean>;
}

export default function TenantConnectionPolicySection() {
  const tenant = useTenantStore((state) => state.tenant);
  const updateTenant = useTenantStore((state) => state.updateTenant);
  const notify = useNotificationStore((state) => state.notify);
  const { loading, error, clearError, run } = useAsyncAction();
  const enforcedSettings = tenant?.enforcedConnectionSettings as EnforcedConnectionSettings | null | undefined;
  const initialDraft = useMemo(
    () => createPolicyDraft(enforcedSettings),
    [enforcedSettings],
  );
  const draftKey = useMemo(
    () => JSON.stringify(enforcedSettings ?? null),
    [enforcedSettings],
  );

  if (!tenant) {
    return null;
  }

  const savePolicy = async (draft: ConnectionPolicyDraft) => {
    clearError();

    const nextPayload: EnforcedConnectionSettings = {};
    if (draft.enabled.ssh && hasPolicySettings(draft.settings.ssh)) {
      nextPayload.ssh = draft.settings.ssh;
    }
    if (draft.enabled.rdp && hasPolicySettings(draft.settings.rdp)) {
      nextPayload.rdp = draft.settings.rdp;
    }
    if (draft.enabled.vnc && hasPolicySettings(draft.settings.vnc)) {
      nextPayload.vnc = draft.settings.vnc;
    }

    const hasPolicies = Object.keys(nextPayload).length > 0;
    const isSuccessful = await run(
      () => updateTenant({ enforcedConnectionSettings: hasPolicies ? nextPayload : null }),
      'Failed to save connection policy',
    );

    if (isSuccessful) {
      notify('Connection policy saved.', 'success');
    }

    return isSuccessful;
  };

  const clearPolicy = async () => {
    clearError();

    const isSuccessful = await run(
      () => updateTenant({ enforcedConnectionSettings: null }),
      'Failed to clear connection policy',
    );

    if (isSuccessful) {
      notify('Connection policy cleared.', 'success');
    }

    return isSuccessful;
  };

  return (
    <TenantConnectionPolicyEditor
      key={draftKey}
      error={error}
      initialDraft={initialDraft}
      loading={loading}
      onClearPolicy={clearPolicy}
      onSavePolicy={savePolicy}
    />
  );
}

function TenantConnectionPolicyEditor({
  error,
  initialDraft,
  loading,
  onClearPolicy,
  onSavePolicy,
}: TenantConnectionPolicyEditorProps) {
  const [draft, setDraft] = useState(initialDraft);

  const updateExpandedPolicies = (nextExpandedPolicies: PolicyType[]) => {
    setDraft((currentDraft) => ({
      ...currentDraft,
      expandedPolicies: nextExpandedPolicies,
    }));
  };

  const handleToggle = (policy: PolicyType, enabled: boolean) => {
    setDraft((currentDraft) => ({
      ...currentDraft,
      enabled: {
        ...currentDraft.enabled,
        [policy]: enabled,
      },
      expandedPolicies: enabled
        ? currentDraft.expandedPolicies.includes(policy)
          ? currentDraft.expandedPolicies
          : [...currentDraft.expandedPolicies, policy]
        : currentDraft.expandedPolicies.filter((currentPolicy) => currentPolicy !== policy),
      settings: enabled
        ? currentDraft.settings
        : {
            ...currentDraft.settings,
            [policy]: {},
          },
    }));
  };

  const updatePolicySettings = <T extends PolicyType>(policy: T, settings: PolicySettingsMap[T]) => {
    setDraft((currentDraft) => ({
      ...currentDraft,
      settings: {
        ...currentDraft.settings,
        [policy]: settings,
      },
    }));
  };

  const handleSave = async () => {
    await onSavePolicy(draft);
  };

  const handleClear = async () => {
    const isSuccessful = await onClearPolicy();
    if (isSuccessful) {
      setDraft(createPolicyDraft(null));
    }
  };

  const enabledCount = Object.values(draft.enabled).filter(Boolean).length;

  const renderPolicyHeader = (
    title: string,
    description: string,
  ) => (
    <div className="space-y-1 text-left">
      <div className="text-sm font-semibold text-foreground">{title}</div>
      <p className="text-sm leading-6 text-muted-foreground">{description}</p>
    </div>
  );

  const renderPolicyToggle = (
    policy: PolicyType,
    title: string,
    enabled: boolean,
  ) => (
    <div className="flex shrink-0 items-center gap-3">
      <SettingsStatusBadge tone={enabled ? 'success' : 'neutral'}>
        {enabled ? 'Enforced' : 'Off'}
      </SettingsStatusBadge>
      <Switch
        checked={enabled}
        onCheckedChange={(nextValue) => handleToggle(policy, nextValue)}
        aria-label={`Enforce ${title.toLowerCase()}`}
      />
    </div>
  );

  const renderPolicyAccordionItem = (
    policy: PolicyType,
    title: string,
    description: string,
    enabled: boolean,
    content: React.ReactNode,
    emptyState: string,
  ) => (
    <AccordionItem value={policy} className="rounded-xl border border-border/70 bg-background/70 px-4">
      <div className="flex items-start gap-4 py-4">
        <AccordionTrigger className="flex-1 py-0 hover:no-underline">
          {renderPolicyHeader(title, description)}
        </AccordionTrigger>
        {renderPolicyToggle(policy, title, enabled)}
      </div>
      <AccordionContent className="pb-4">
        {enabled ? content : (
          <p className="text-sm leading-6 text-muted-foreground">{emptyState}</p>
        )}
      </AccordionContent>
    </AccordionItem>
  );

  return (
    <SettingsPanel
      title="Connection Policy"
      description="Enforce tenant-wide SSH, RDP, and VNC defaults that individual users cannot override."
      heading={(
        <SettingsStatusBadge tone={enabledCount > 0 ? 'success' : 'neutral'}>
          <ShieldCheck className="size-3.5" />
          {enabledCount === 0 ? 'No policies enforced' : `${enabledCount} ${enabledCount > 1 ? 'policies' : 'policy'} enforced`}
        </SettingsStatusBadge>
      )}
      contentClassName="space-y-4"
    >
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <SettingsSummaryGrid>
        <SettingsSummaryItem label="SSH" value={draft.enabled.ssh ? 'Enforced' : 'Off'} />
        <SettingsSummaryItem label="RDP" value={draft.enabled.rdp ? 'Enforced' : 'Off'} />
        <SettingsSummaryItem label="VNC" value={draft.enabled.vnc ? 'Enforced' : 'Off'} />
      </SettingsSummaryGrid>

      <SettingsSectionBlock
        title="Policy Editors"
        description="Open a policy to define the exact settings enforced across every matching connection."
      >
        <Accordion
          type="multiple"
          value={draft.expandedPolicies}
          onValueChange={(value) => updateExpandedPolicies(value as PolicyType[])}
          className="space-y-3"
        >
          {renderPolicyAccordionItem(
            'ssh',
            'SSH Terminal Settings',
            'Control terminal appearance and behavior for every SSH session.',
            draft.enabled.ssh,
            (
              <TerminalSettingsSection
                value={draft.settings.ssh}
                onChange={(value) => updatePolicySettings('ssh', value)}
                mode="global"
                resolvedDefaults={TERMINAL_DEFAULTS}
              />
            ),
            'Enable this policy to enforce SSH terminal settings for everyone in the organization.',
          )}

          {renderPolicyAccordionItem(
            'rdp',
            'RDP Settings',
            'Set session quality, display, and audio rules for remote desktops.',
            draft.enabled.rdp,
            (
              <RdpSettingsSection
                value={draft.settings.rdp}
                onChange={(value) => updatePolicySettings('rdp', value)}
                mode="global"
                resolvedDefaults={RDP_DEFAULTS}
              />
            ),
            'Enable this policy to enforce RDP session defaults across the tenant.',
          )}

          {renderPolicyAccordionItem(
            'vnc',
            'VNC Settings',
            'Define cursor, display, clipboard, and read-only behavior for VNC sessions.',
            draft.enabled.vnc,
            (
              <VncSettingsSection
                value={draft.settings.vnc}
                onChange={(value) => updatePolicySettings('vnc', value)}
                mode="global"
                resolvedDefaults={VNC_DEFAULTS}
              />
            ),
            'Enable this policy to enforce VNC session settings for all connections.',
          )}
        </Accordion>
      </SettingsSectionBlock>

      <SettingsButtonRow>
        <Button type="button" onClick={handleSave} disabled={loading}>
          {loading ? 'Saving...' : 'Save Policy'}
        </Button>
        <Button type="button" variant="outline" onClick={handleClear} disabled={loading}>
          Clear All
        </Button>
      </SettingsButtonRow>
    </SettingsPanel>
  );
}
