import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { AccessPolicyData, AccessPolicyTargetType } from '../../api/accessPolicy.api';
import { PolicyDialogShell } from './databasePolicyUi';
import {
  SettingsButtonRow,
  SettingsFieldCard,
  SettingsFieldGroup,
  SettingsSwitchRow,
} from './settings-ui';
import {
  ACCESS_POLICY_TARGET_LABELS,
  type AccessPolicyFormState,
} from './accessPolicyUtils';

const TARGET_TYPE_OPTIONS: AccessPolicyTargetType[] = ['TENANT', 'TEAM', 'FOLDER'];

export function AccessPolicyFormDialog({
  editingPolicy,
  form,
  formError,
  open,
  saveError,
  saving,
  targetOptions,
  onClose,
  onSave,
  onFormChange,
  onTargetTypeChange,
}: {
  editingPolicy: AccessPolicyData | null;
  form: AccessPolicyFormState;
  formError: string;
  open: boolean;
  saveError: string;
  saving: boolean;
  targetOptions: Array<{ id: string; name: string }>;
  onClose: () => void;
  onSave: () => void;
  onFormChange: (nextForm: AccessPolicyFormState) => void;
  onTargetTypeChange: (targetType: AccessPolicyTargetType) => void;
}) {
  return (
    <PolicyDialogShell
      open={open}
      onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}
      title={editingPolicy ? 'Edit access policy' : 'Create access policy'}
      description="Keep policies explicit and targeted. Most teams only need a few narrow rules instead of broad global restrictions."
      footer={(
        <SettingsButtonRow className="justify-end">
          <Button type="button" variant="outline" onClick={onClose} disabled={saving}>
            Cancel
          </Button>
          <Button type="button" onClick={onSave} disabled={saving}>
            {saving ? 'Saving...' : editingPolicy ? 'Save Policy' : 'Create Policy'}
          </Button>
        </SettingsButtonRow>
      )}
    >
      {(saveError || formError) && (
        <Alert variant="destructive">
          <AlertDescription>{saveError || formError}</AlertDescription>
        </Alert>
      )}

      <SettingsFieldGroup>
        {!editingPolicy && (
          <SettingsFieldCard
            label="Target"
            description="Choose the scope that this rule should govern."
          >
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="access-policy-target-type">Target type</Label>
                <Select value={form.targetType} onValueChange={(value) => onTargetTypeChange(value as AccessPolicyTargetType)}>
                  <SelectTrigger id="access-policy-target-type" aria-label="Target type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {TARGET_TYPE_OPTIONS.map((targetType) => (
                      <SelectItem key={targetType} value={targetType}>
                        {ACCESS_POLICY_TARGET_LABELS[targetType]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="access-policy-target">Target</Label>
                <Select
                  value={form.targetId}
                  disabled={form.targetType === 'TENANT'}
                  onValueChange={(value) => onFormChange({ ...form, targetId: value })}
                >
                  <SelectTrigger id="access-policy-target" aria-label="Target">
                    <SelectValue placeholder="Select a target" />
                  </SelectTrigger>
                  <SelectContent>
                    {targetOptions.map((option) => (
                      <SelectItem key={option.id} value={option.id}>
                        {option.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </SettingsFieldCard>
        )}

        <SettingsFieldCard
          label="Allowed time windows"
          description="Use UTC ranges in HH:MM-HH:MM format. Leave blank to allow any time."
        >
          <div className="space-y-2">
            <Input
              value={form.allowedTimeWindows}
              placeholder="09:00-18:00 or 09:00-12:00,13:00-17:00"
              aria-label="Allowed time windows"
              onChange={(event) => onFormChange({ ...form, allowedTimeWindows: event.target.value })}
            />
            <p className="text-sm text-muted-foreground">
              Use a comma-separated list when the rule has multiple windows.
            </p>
          </div>
        </SettingsFieldCard>

        <SettingsSwitchRow
          title="Require trusted device"
          description="Block access unless the session comes from a previously trusted device."
          checked={form.requireTrustedDevice}
          onCheckedChange={(checked) => onFormChange({ ...form, requireTrustedDevice: checked })}
        />

        <SettingsSwitchRow
          title="Require step-up MFA"
          description="Force a fresh multi-factor challenge when this policy matches."
          checked={form.requireMfaStepUp}
          onCheckedChange={(checked) => onFormChange({ ...form, requireMfaStepUp: checked })}
        />
      </SettingsFieldGroup>
    </PolicyDialogShell>
  );
}

export function AccessPolicyDeleteDialog({
  deleteError,
  open,
  policyName,
  deleting,
  onClose,
  onDelete,
}: {
  deleteError: string;
  deleting: boolean;
  open: boolean;
  policyName: string;
  onClose: () => void;
  onDelete: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Delete access policy?</DialogTitle>
          <DialogDescription>
            Remove the rule for {policyName}.
          </DialogDescription>
        </DialogHeader>

        {deleteError && (
          <Alert variant="destructive">
            <AlertDescription>{deleteError}</AlertDescription>
          </Alert>
        )}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={deleting}>
            Cancel
          </Button>
          <Button type="button" variant="destructive" onClick={onDelete} disabled={deleting}>
            {deleting ? 'Deleting...' : 'Delete Policy'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
