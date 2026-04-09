import { useDeferredValue, useMemo, useState } from 'react';
import { ChevronRight, Search, Settings2 } from 'lucide-react';
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarInput,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from '@/components/ui/sidebar';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import { useAuthStore } from '@/store/authStore';
import { useFeatureFlagsStore } from '@/store/featureFlagsStore';
import { useUiPreferencesStore } from '@/store/uiPreferencesStore';
import { isAdminOrAbove } from '@/utils/roles';
import {
  buildSettingsConcerns,
  type SettingsConcern,
  type SettingsSection,
} from '../Dialogs/settingsConcerns';

function sectionMatches(section: SettingsSection, query: string) {
  if (!query) return true;
  const haystack = [section.label, section.description, ...section.keywords]
    .join(' ')
    .toLowerCase();
  return haystack.includes(query);
}

function concernMatches(concern: SettingsConcern, query: string) {
  if (!query) return true;
  const haystack = [concern.label, concern.description, ...concern.keywords]
    .join(' ')
    .toLowerCase();
  return haystack.includes(query);
}

interface SettingsSidePanelProps {
  onOpenSettings: (tab?: string) => void;
}

export default function SettingsSidePanel({ onOpenSettings }: SettingsSidePanelProps) {
  const user = useAuthStore((s) => s.user);
  const connectionsEnabled = useFeatureFlagsStore((s) => s.connectionsEnabled);
  const databaseProxyEnabled = useFeatureFlagsStore((s) => s.databaseProxyEnabled);
  const keychainEnabled = useFeatureFlagsStore((s) => s.keychainEnabled);
  const zeroTrustEnabled = useFeatureFlagsStore((s) => s.zeroTrustEnabled);
  const agenticAIEnabled = useFeatureFlagsStore((s) => s.agenticAIEnabled);
  const enterpriseAuthEnabled = useFeatureFlagsStore((s) => s.enterpriseAuthEnabled);
  const settingsActiveTab = useUiPreferencesStore((s) => s.settingsActiveTab);
  const setPreference = useUiPreferencesStore((s) => s.set);

  const [search, setSearch] = useState('');
  const deferredSearch = useDeferredValue(search);
  const [expandedConcerns, setExpandedConcerns] = useState<Set<string>>(() => new Set([settingsActiveTab]));

  const hasTenant = Boolean(user?.tenantId);
  const isAdmin = isAdminOrAbove(user?.tenantRole);
  const isOwner = user?.tenantRole === 'OWNER';
  const anyConnectionFeature = connectionsEnabled || databaseProxyEnabled;

  const concerns = useMemo(
    () =>
      buildSettingsConcerns({
        hasPassword: true,
        hasTenant,
        isAdmin,
        isOwner,
        anyConnectionFeature,
        connectionsEnabled,
        databaseProxyEnabled,
        keychainEnabled,
        zeroTrustEnabled,
        agenticAIEnabled,
        enterpriseAuthEnabled,
        tenantId: user?.tenantId ?? null,
        onHasPasswordResolved: () => {},
        deleteOrgTrigger: null,
        setDeleteOrgTrigger: () => {},
        navigateToConcern: (target) => {
          setPreference('settingsActiveTab', target);
          onOpenSettings(target);
        },
      }),
    [
      hasTenant, isAdmin, isOwner, anyConnectionFeature, connectionsEnabled,
      databaseProxyEnabled, keychainEnabled, zeroTrustEnabled, agenticAIEnabled,
      enterpriseAuthEnabled, user?.tenantId, setPreference, onOpenSettings,
    ],
  );

  const filteredConcerns = useMemo(() => {
    const query = deferredSearch.trim().toLowerCase();
    if (!query) return concerns;
    return concerns
      .map((concern) => {
        const matchingSections = concern.sections.filter((s) => sectionMatches(s, query));
        return concernMatches(concern, query) || matchingSections.length > 0
          ? { ...concern, sections: matchingSections.length > 0 ? matchingSections : concern.sections }
          : null;
      })
      .filter((c): c is SettingsConcern => c !== null);
  }, [concerns, deferredSearch]);

  const handleConcernClick = (concernId: string) => {
    setExpandedConcerns((prev) => {
      const next = new Set(prev);
      if (next.has(concernId)) next.delete(concernId);
      else next.add(concernId);
      return next;
    });
  };

  const handleSectionClick = (concernId: string, _sectionId: string) => {
    setPreference('settingsActiveTab', concernId);
    onOpenSettings(concernId);
  };

  return (
    <>
      <SidebarGroup>
        <SidebarGroupLabel>
          <Settings2 className="size-4" />
          Settings
        </SidebarGroupLabel>
        <SidebarGroupContent>
          <div className="px-1 pb-2">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
              <SidebarInput
                placeholder="Search settings..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-8"
              />
            </div>
          </div>
        </SidebarGroupContent>
      </SidebarGroup>

      <SidebarGroup>
        <SidebarGroupContent>
          <SidebarMenu>
            {filteredConcerns.map((concern) => (
              <Collapsible
                key={concern.id}
                open={expandedConcerns.has(concern.id) || deferredSearch.trim().length > 0}
                onOpenChange={() => handleConcernClick(concern.id)}
                className="group/collapsible"
              >
                <SidebarMenuItem>
                  <CollapsibleTrigger asChild>
                    <SidebarMenuButton
                      isActive={settingsActiveTab === concern.id}
                      tooltip={concern.description}
                    >
                      {concern.icon}
                      <span>{concern.label}</span>
                      <ChevronRight className="ml-auto size-3.5 transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>
                  <CollapsibleContent>
                    <SidebarMenuSub>
                      {concern.sections.map((section) => (
                        <SidebarMenuSubItem key={section.id}>
                          <SidebarMenuSubButton
                            size="sm"
                            onClick={() => handleSectionClick(concern.id, section.id)}
                          >
                            <span>{section.label}</span>
                          </SidebarMenuSubButton>
                        </SidebarMenuSubItem>
                      ))}
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>
            ))}
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </>
  );
}
