import { useEffect, useState } from 'react';
import { Building2, ChevronsLeftRight, Plus } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useTenantStore } from '@/store/tenantStore';
import { useUiPreferencesStore } from '@/store/uiPreferencesStore';
import { extractApiError } from '@/utils/apiError';

export default function TenantSwitcher() {
  const memberships = useTenantStore((state) => state.memberships);
  const fetchMemberships = useTenantStore((state) => state.fetchMemberships);
  const switchTenant = useTenantStore((state) => state.switchTenant);
  const createTenant = useTenantStore((state) => state.createTenant);
  const [switching, setSwitching] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createName, setCreateName] = useState('');
  const [createError, setCreateError] = useState('');
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    void fetchMemberships();
  }, [fetchMemberships]);

  const hasPending = memberships.some((membership) => membership.pending);
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

  const closeCreateDialog = (force = false) => {
    if (creating && !force) return;
    setCreateOpen(false);
    setCreateName('');
    setCreateError('');
    setCreating(false);
  };

  const handleCreateTenant = async () => {
    if (creating) return;
    const name = createName.trim();
    if (name.length < 2) {
      setCreateError('Organization name must be at least 2 characters.');
      return;
    }

    setCreating(true);
    setCreateError('');
    try {
      const tenant = await createTenant(name);
      useUiPreferencesStore.getState().set('lastActiveTenantId', tenant.id);
      closeCreateDialog(true);
    } catch (err: unknown) {
      setCreateError(extractApiError(err, 'Failed to create organization.'));
      setCreating(false);
    }
  };

  const createDialog = (
    <Dialog open={createOpen} onOpenChange={(next) => { if (next) setCreateOpen(true); else closeCreateDialog(); }}>
      <DialogContent className="sm:max-w-md" showCloseButton={!creating}>
        <DialogHeader>
          <DialogTitle>Create Organization</DialogTitle>
          <DialogDescription className="sr-only">
            Create a new workspace for teams, policies, and infrastructure.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          {createError ? (
            <Alert variant="destructive">
              <AlertDescription>{createError}</AlertDescription>
            </Alert>
          ) : null}
          <div className="space-y-2">
            <Label htmlFor="tenant-switcher-create-name">Organization name</Label>
            <Input
              id="tenant-switcher-create-name"
              value={createName}
              maxLength={100}
              autoFocus
              disabled={creating}
              onChange={(event) => {
                setCreateName(event.target.value);
                setCreateError('');
              }}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && createName.trim()) {
                  event.preventDefault();
                  void handleCreateTenant();
                }
              }}
            />
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" disabled={creating} onClick={() => closeCreateDialog()}>
            Cancel
          </Button>
          <Button type="button" disabled={creating || !createName.trim()} onClick={() => void handleCreateTenant()}>
            {creating ? 'Creating...' : 'Create Organization'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );

  if (memberships.length === 0 && !hasPending) {
    return (
      <>
        <Button type="button" variant="ghost" size="sm" className="gap-2" onClick={() => setCreateOpen(true)}>
          <Plus className="size-4" />
          Create organization
        </Button>
        {createDialog}
      </>
    );
  }

  return (
    <>
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
                  {membership.pending ? `${membership.role} - Invitation pending` : membership.role}
                </span>
              </span>
              {membership.isActive ? (
                <Badge className="shrink-0">Active</Badge>
              ) : membership.pending ? (
                <Badge variant="outline" className="shrink-0">Pending</Badge>
              ) : null}
            </DropdownMenuItem>
          ))}
          <DropdownMenuSeparator />
          <DropdownMenuItem onSelect={() => setCreateOpen(true)}>
            <Plus className="size-4" />
            Create organization
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      {createDialog}
    </>
  );
}
