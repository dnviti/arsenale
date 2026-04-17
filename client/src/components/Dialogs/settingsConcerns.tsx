import type { ReactNode } from 'react';
import {
  Blend,
  Building2,
  CloudDownload,
  CloudUpload,
  Link2,
  LockKeyhole,
  Network,
  ServerCog,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import ProfileSection from '../Settings/ProfileSection';
import ChangePasswordSection from '../Settings/ChangePasswordSection';
import ConnectionDefaultsSection from '../Settings/ConnectionDefaultsSection';
import TwoFactorSection from '../Settings/TwoFactorSection';
import SmsMfaSection from '../Settings/SmsMfaSection';
import WebAuthnSection from '../Settings/WebAuthnSection';
import LinkedAccountsSection from '../Settings/LinkedAccountsSection';
import VaultAutoLockSection from '../Settings/VaultAutoLockSection';
import DomainProfileSection from '../Settings/DomainProfileSection';
import TenantSection from '../Settings/TenantSection';
import TeamSection from '../Settings/TeamSection';
import GatewaySection from '../Settings/GatewaySection';
import EmailProviderSection from '../Settings/EmailProviderSection';
import SelfSignupSection from '../Settings/SelfSignupSection';
import SystemSettingsSection from '../Settings/SystemSettingsSection';
import IpAllowlistSection from '../Settings/IpAllowlistSection';
import LdapConfigSection from '../Settings/LdapConfigSection';
import SyncProfileSection from '../Settings/SyncProfileSection';
import TenantConnectionPolicySection from '../Settings/TenantConnectionPolicySection';
import TunnelConfigSection from '../Settings/TunnelConfigSection';
import SamlConfigSection from '../Settings/SamlConfigSection';
import OAuthProvidersAdminSection from '../Settings/OAuthProvidersAdminSection';
import VaultProvidersSection from '../Settings/VaultProvidersSection';
import AccessPolicySection from '../Settings/AccessPolicySection';
import NativeSshSection from '../Settings/NativeSshSection';
import RdGatewayConfigSection from '../Settings/RdGatewayConfigSection';
import AiQueryConfigSection from '../Settings/AiQueryConfigSection';
import AppearanceSection from '../Settings/AppearanceSection';
import SqlEditorSection from '../Settings/SqlEditorSection';
import NotificationPreferencesSection from '../Settings/NotificationPreferencesSection';
import NotificationsSection from '../Settings/NotificationsSection';
import DbFirewallSection from '../Settings/DbFirewallSection';
import DbMaskingSection from '../Settings/DbMaskingSection';
import DbRateLimitSection from '../Settings/DbRateLimitSection';
import type { SessionsRouteState } from '@/components/sessions/sessionConsoleRoute';

export interface SettingsConcern {
  id: string;
  label: string;
  description: string;
  icon: ReactNode;
  keywords: string[];
  sections: SettingsSection[];
}

export interface SettingsSection {
  id: string;
  label: string;
  description: string;
  keywords: string[];
  content: ReactNode;
}

export interface SettingsConcernContext {
  hasPassword: boolean;
  hasTenant: boolean;
  isAdmin: boolean;
  isOwner: boolean;
  anyConnectionFeature: boolean;
  connectionsEnabled: boolean;
  databaseProxyEnabled: boolean;
  keychainEnabled: boolean;
  zeroTrustEnabled: boolean;
  agenticAIEnabled: boolean;
  enterpriseAuthEnabled: boolean;
  linkedProvider?: string | null;
  tenantId?: string | null;
  onHasPasswordResolved: (hasPassword: boolean) => void;
  onViewUserProfile?: (userId: string) => void;
  onImport?: () => void;
  onExport?: () => void;
  onOpenSessions?: (initialState?: Partial<SessionsRouteState>) => void;
  deleteOrgTrigger: (() => void) | null;
  setDeleteOrgTrigger: (trigger: (() => void) | null) => void;
  navigateToConcern: (concernId: string) => void;
}

function ImportExportCard({ onImport, onExport }: Pick<SettingsConcernContext, 'onImport' | 'onExport'>) {
  if (!onImport && !onExport) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Import & Export</CardTitle>
        <CardDescription>
          Bring connection data in or move it out without digging through separate menus.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-wrap gap-2">
        {onImport && (
          <Button type="button" variant="outline" onClick={onImport}>
            <CloudUpload className="size-4" />
            Import
          </Button>
        )}
        {onExport && (
          <Button type="button" variant="outline" onClick={onExport}>
            <CloudDownload className="size-4" />
            Export
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

function OrganizationDangerZone({
  deleteOrgTrigger,
}: Pick<SettingsConcernContext, 'deleteOrgTrigger'>) {
  if (!deleteOrgTrigger) return null;

  return (
    <Card className="border-destructive/30 bg-destructive/5">
      <CardHeader>
        <CardTitle className="text-lg text-destructive">Danger Zone</CardTitle>
        <CardDescription>
          Permanently delete this organization, all teams, and all memberships.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Button type="button" variant="destructive" onClick={deleteOrgTrigger}>
          Delete Organization
        </Button>
      </CardContent>
    </Card>
  );
}

export function buildSettingsConcerns(context: SettingsConcernContext): SettingsConcern[] {
  const concerns: SettingsConcern[] = [
    {
      id: 'personal',
      label: 'Personal',
      description: 'Your profile, interface, notifications, and personal defaults.',
      icon: <Blend className="size-4" />,
      keywords: ['profile', 'appearance', 'notifications', 'defaults', 'import', 'export'],
      sections: [
        {
          id: 'profile',
          label: 'Profile',
          description: 'Identity, recovery details, and linked provider state.',
          keywords: ['profile', 'account', 'identity'],
          content: (
            <ProfileSection
              onHasPasswordResolved={context.onHasPasswordResolved}
              linkedProvider={context.linkedProvider}
            />
          ),
        },
        {
          id: 'password',
          label: 'Password',
          description: 'Change your password without leaving the dialog.',
          keywords: ['password', 'credentials'],
          content: <ChangePasswordSection hasPassword={context.hasPassword} />,
        },
        {
          id: 'appearance',
          label: 'Appearance',
          description: 'Theme, mode, and SQL editor presentation.',
          keywords: ['theme', 'appearance', 'color', 'sql editor'],
          content: (
            <>
              <AppearanceSection />
              {context.databaseProxyEnabled && <SqlEditorSection />}
            </>
          ),
        },
        {
          id: 'notifications',
          label: 'Notifications',
          description: 'Delivery channels and quiet-hour preferences.',
          keywords: ['notifications', 'alerts', 'quiet hours'],
          content: (
            <>
              <NotificationsSection />
              <NotificationPreferencesSection />
            </>
          ),
        },
        {
          id: 'data-movement',
          label: 'Import & Export',
          description: 'Move connection data in and out.',
          keywords: ['import', 'export', 'backup'],
          content: context.onImport || context.onExport
            ? <ImportExportCard onImport={context.onImport} onExport={context.onExport} />
            : null,
        },
        {
          id: 'connection-defaults',
          label: 'Connection Defaults',
          description: 'Baseline preferences for new connections.',
          keywords: ['connection defaults', 'defaults', 'connection'],
          content: context.anyConnectionFeature ? <ConnectionDefaultsSection /> : null,
        },
      ].filter((section) => section.content !== null) as SettingsSection[],
    },
    {
      id: 'security',
      label: 'Security',
      description: 'Authentication factors, vault behavior, and identity protection.',
      icon: <LockKeyhole className="size-4" />,
      keywords: ['security', 'mfa', 'webauthn', 'vault', 'domain', 'linked accounts'],
      sections: [
        { id: 'two-factor', label: 'Two-Factor', description: 'TOTP and email-based protection.', keywords: ['2fa', 'totp', 'mfa'], content: <TwoFactorSection /> },
        { id: 'sms-mfa', label: 'SMS MFA', description: 'SMS codes and fallback second factor settings.', keywords: ['sms', 'mfa'], content: <SmsMfaSection /> },
        { id: 'webauthn', label: 'Passkeys & WebAuthn', description: 'Hardware-backed sign-in methods.', keywords: ['passkey', 'webauthn'], content: <WebAuthnSection /> },
        {
          id: 'vault',
          label: 'Vault Auto-Lock',
          description: 'Session lock behavior for keychain access.',
          keywords: ['vault', 'lock', 'keychain'],
          content: context.keychainEnabled ? <VaultAutoLockSection /> : null,
        },
        {
          id: 'trusted-domains',
          label: 'Trusted Domains',
          description: 'Domain trust and browser-side identity controls.',
          keywords: ['domain', 'trusted domains'],
          content: <DomainProfileSection />,
        },
        {
          id: 'linked-accounts',
          label: 'Linked Accounts',
          description: 'Connected enterprise identity providers.',
          keywords: ['oauth', 'identity', 'linked accounts'],
          content: context.enterpriseAuthEnabled ? <LinkedAccountsSection hasPassword={context.hasPassword} /> : null,
        },
      ].filter((section) => section.content !== null) as SettingsSection[],
    },
  ];

  if (context.hasTenant) {
    concerns.push({
      id: 'organization',
      label: 'Organization',
      description: 'People, collaboration, and tenant-wide workspace policy.',
      icon: <Building2 className="size-4" />,
      keywords: ['organization', 'tenant', 'teams', 'members', 'policy'],
      sections: [
        {
          id: 'organization-profile',
          label: 'Organization',
          description: 'Members, roles, and tenant controls.',
          keywords: ['organization', 'tenant', 'members'],
          content: (
            <TenantSection
              onViewUserProfile={context.onViewUserProfile}
              onDeleteRequest={(trigger) => context.setDeleteOrgTrigger(trigger)}
            />
          ),
        },
        {
          id: 'teams',
          label: 'Teams',
          description: 'Team membership and collaboration boundaries.',
          keywords: ['teams', 'members', 'roles'],
          content: <TeamSection onNavigateToTab={context.navigateToConcern} />,
        },
        {
          id: 'connection-policy',
          label: 'Connection Policy',
          description: 'Tenant-wide connection behavior and safety defaults.',
          keywords: ['policy', 'connection policy'],
          content: context.isAdmin && context.anyConnectionFeature ? <TenantConnectionPolicySection /> : null,
        },
        {
          id: 'danger-zone',
          label: 'Danger Zone',
          description: 'Irreversible organization-level actions.',
          keywords: ['delete', 'danger zone'],
          content: context.isOwner ? <OrganizationDangerZone deleteOrgTrigger={context.deleteOrgTrigger} /> : null,
        },
      ].filter((section) => section.content !== null) as SettingsSection[],
    });
  }

  concerns.push({
    id: 'infrastructure',
    label: 'Infrastructure',
    description: 'Gateways, tunnels, and transport-level connectivity settings.',
    icon: <Network className="size-4" />,
    keywords: ['gateways', 'tunnel', 'rd gateway', 'ssh', 'infrastructure'],
    sections: [
      {
        id: 'gateways',
        label: 'Gateways',
        description: 'Gateway inventory, sessions, orchestration, and templates.',
        keywords: ['gateways', 'orchestration', 'sessions'],
          content: context.hasTenant && context.anyConnectionFeature
            ? <GatewaySection onNavigateToTab={context.navigateToConcern} onOpenSessions={context.onOpenSessions} />
            : null,
      },
      {
        id: 'tunnel',
        label: 'Zero-Trust Tunnel',
        description: 'Tunnel brokers and outbound-only access controls.',
        keywords: ['zero trust', 'tunnel'],
        content: context.isAdmin && context.zeroTrustEnabled ? <TunnelConfigSection /> : null,
      },
      {
        id: 'native-ssh',
        label: 'Native SSH',
        description: 'System-level SSH bridge controls.',
        keywords: ['ssh', 'native ssh'],
        content: context.isAdmin && context.connectionsEnabled ? <NativeSshSection /> : null,
      },
      {
        id: 'rd-gateway',
        label: 'RD Gateway',
        description: 'Remote Desktop gateway integration options.',
        keywords: ['rd gateway', 'rdp'],
        content: context.isAdmin && context.connectionsEnabled ? <RdGatewayConfigSection /> : null,
      },
    ].filter((section) => section.content !== null) as SettingsSection[],
  });

  concerns.push({
    id: 'integrations',
    label: 'Integrations',
    description: 'Identity, sync, delivery, and external service configuration.',
    icon: <Link2 className="size-4" />,
    keywords: ['sync', 'email', 'ldap', 'saml', 'oauth', 'ai'],
    sections: [
      { id: 'sync', label: 'Sync Profiles', description: 'Directory and source-of-truth synchronization.', keywords: ['sync', 'profiles'], content: <SyncProfileSection /> },
      {
        id: 'vault-providers',
        label: 'External Vault Providers',
        description: 'Bring external secret managers into the workspace without burying them under organization controls.',
        keywords: ['vault', 'secrets', 'keychain'],
        content: context.isAdmin && context.tenantId ? <VaultProvidersSection tenantId={context.tenantId} /> : null,
      },
      {
        id: 'ai-query',
        label: 'AI Query',
        description: 'Model-backed database assistance and query generation.',
        keywords: ['ai', 'query', 'database'],
        content: context.isOwner && context.databaseProxyEnabled && context.agenticAIEnabled ? <AiQueryConfigSection /> : null,
      },
      {
        id: 'email',
        label: 'Email Provider',
        description: 'Transactional delivery and account messaging.',
        keywords: ['email', 'smtp'],
        content: context.isAdmin ? <EmailProviderSection /> : null,
      },
      {
        id: 'signup',
        label: 'Self Signup',
        description: 'Public registration gates and onboarding policy.',
        keywords: ['signup', 'registration'],
        content: context.isAdmin ? <SelfSignupSection /> : null,
      },
      {
        id: 'oauth-providers',
        label: 'OAuth Providers',
        description: 'Managed OAuth/OIDC provider configuration.',
        keywords: ['oauth', 'oidc'],
        content: context.isAdmin && context.enterpriseAuthEnabled ? <OAuthProvidersAdminSection /> : null,
      },
      {
        id: 'ldap',
        label: 'LDAP',
        description: 'Directory integration and authentication mapping.',
        keywords: ['ldap', 'directory'],
        content: context.isAdmin && context.enterpriseAuthEnabled ? <LdapConfigSection /> : null,
      },
      {
        id: 'saml',
        label: 'SAML',
        description: 'Enterprise SSO trust and metadata.',
        keywords: ['saml', 'sso'],
        content: context.isAdmin && context.enterpriseAuthEnabled ? <SamlConfigSection /> : null,
      },
    ].filter((section) => section.content !== null) as SettingsSection[],
  });

  concerns.push({
    id: 'governance',
    label: 'Governance',
    description: 'Platform-wide policy and operational safeguards.',
    icon: <ServerCog className="size-4" />,
    keywords: ['system', 'policy', 'governance', 'sql firewall', 'masking', 'rate limits'],
    sections: [
      { id: 'system', label: 'System Settings', description: 'Global runtime defaults and control plane behavior.', keywords: ['system'], content: context.isAdmin ? <SystemSettingsSection /> : null },
      { id: 'ip-allowlist', label: 'IP Allowlist', description: 'Restrict who can reach the platform.', keywords: ['ip', 'allowlist'], content: context.isAdmin ? <IpAllowlistSection /> : null },
      { id: 'access-policy', label: 'Access Policy', description: 'Connection authorization and decision rules.', keywords: ['access policy', 'authorization'], content: context.isAdmin && context.connectionsEnabled ? <AccessPolicySection /> : null },
      {
        id: 'db-firewall',
        label: 'SQL Firewall',
        description: 'Tenant-wide SQL firewall rules and enforcement.',
        keywords: ['sql firewall', 'database firewall', 'db firewall'],
        content: context.isAdmin && context.databaseProxyEnabled ? <DbFirewallSection /> : null,
      },
      {
        id: 'db-masking',
        label: 'SQL Masking',
        description: 'Masking policies for sensitive database fields.',
        keywords: ['masking', 'sql masking', 'database masking'],
        content: context.isAdmin && context.databaseProxyEnabled ? <DbMaskingSection /> : null,
      },
      {
        id: 'db-rate-limit',
        label: 'SQL Rate Limits',
        description: 'Query throttling and burst protection policies.',
        keywords: ['rate limits', 'sql rate limit', 'database rate limit'],
        content: context.isAdmin && context.databaseProxyEnabled ? <DbRateLimitSection /> : null,
      },
    ].filter((section) => section.content !== null) as SettingsSection[],
  });

  return concerns.filter((concern) => concern.sections.length > 0);
}
