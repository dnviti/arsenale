import { Loader2, Plus } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { SettingsFieldCard } from './settings-ui';

interface TenantCreateOrganizationFormProps {
  autoFocus?: boolean;
  creating: boolean;
  error?: string;
  id: string;
  name: string;
  onCreate: () => void;
  onNameChange: (value: string) => void;
}

export function TenantCreateOrganizationForm({
  autoFocus = false,
  creating,
  error,
  id,
  name,
  onCreate,
  onNameChange,
}: TenantCreateOrganizationFormProps) {
  return (
    <div className="max-w-lg space-y-4">
      {error ? (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}
      <div className="space-y-2">
        <Label htmlFor={id}>Organization name</Label>
        <Input
          id={id}
          autoFocus={autoFocus}
          value={name}
          maxLength={100}
          onChange={(event) => onNameChange(event.target.value)}
        />
      </div>
      <Button type="button" onClick={onCreate} disabled={creating || !name.trim()}>
        {creating ? <Loader2 className="size-4 animate-spin" /> : <Plus className="size-4" />}
        {creating ? 'Creating...' : 'Create Organization'}
      </Button>
    </div>
  );
}

export function TenantCreateOrganizationCard(props: Omit<TenantCreateOrganizationFormProps, 'id' | 'autoFocus'>) {
  return (
    <SettingsFieldCard
      label="Create another organization"
      description="New organizations are added to your account and become your active workspace."
    >
      <TenantCreateOrganizationForm
        {...props}
        id="tenant-create-additional-name"
      />
    </SettingsFieldCard>
  );
}
