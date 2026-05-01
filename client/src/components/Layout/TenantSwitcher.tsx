import { useEffect, useState } from 'react';
import { Building2, ChevronsLeftRight, Plus } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useFeatureFlagsStore } from '@/store/featureFlagsStore';
import { useTenantStore } from '@/store/tenantStore';
import { useUiPreferencesStore } from '@/store/uiPreferencesStore';

interface TenantSwitcherProps {
  onCreateOrg?: () => void;
}

export default function TenantSwitcher({ onCreateOrg }: TenantSwitcherProps) {
  const memberships = useTenantStore((state) => state.memberships);
  const fetchMemberships = useTenantStore((state) => state.fetchMemberships);
  const switchTenant = useTenantStore((state) => state.switchTenant);
  const multiTenancyEnabled = useFeatureFlagsStore((state) => state.multiTenancyEnabled);
  const [switching, setSwitching] = useState(false);

  useEffect(() => {
    if (!multiTenancyEnabled) {
      return;
    }
    void fetchMemberships();
  }, [fetchMemberships, multiTenancyEnabled]);

  if (!multiTenancyEnabled) {
    return null;
  }

  const hasPending = memberships.some((membership) => membership.pending);
  if (memberships.length <= 1 && !hasPending) {
    return null;
  }

  const activeMembership = memberships.find((membership) => membership.isActive);

  const handleSwitch = async (tenantId: string) => {
    if (tenantId === activeMembership?.tenantId) {
      return;
    }
    setSwitching(true);
    try {
      await switchTenant(tenantId);
      useUiPreferencesStore.getState().set('lastActiveTenantId', tenantId);
    } finally {
      setSwitching(false);
    }
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant="ghost" size="sm" disabled={switching} className="gap-2">
          <ChevronsLeftRight className="size-4" />
          <span className="max-w-40 truncate">
            {activeMembership?.name ?? (hasPending ? 'Invitations' : 'Select organization')}
          </span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-72">
        <DropdownMenuLabel>Switch organization</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {memberships.map((membership) => (
          <DropdownMenuItem
            key={membership.tenantId}
            className="items-start gap-3 py-2"
            disabled={switching}
            onSelect={() => { void handleSwitch(membership.tenantId); }}
          >
            <span className="mt-0.5 inline-flex size-8 items-center justify-center rounded-full bg-muted text-muted-foreground">
              <Building2 className="size-4" />
            </span>
            <span className="min-w-0 flex-1 space-y-1">
              <span className="block truncate text-sm font-medium">{membership.name}</span>
              <span className="block text-xs text-muted-foreground">
                {membership.pending ? `${membership.role} · Invitation pending` : membership.role}
              </span>
            </span>
            {membership.isActive ? (
              <Badge className="shrink-0">Active</Badge>
            ) : membership.pending ? (
              <Badge variant="outline" className="shrink-0">Pending</Badge>
            ) : null}
          </DropdownMenuItem>
        ))}
        {onCreateOrg ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={onCreateOrg}>
              <Plus className="size-4" />
              Create organization
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
