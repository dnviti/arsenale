import { useCallback, useEffect, useMemo, useState } from 'react';
import { Clock3, Plus, ShieldCheck, ShieldAlert } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { useAuthStore } from '../../store/authStore';
import { useAccessPolicyStore } from '../../store/accessPolicyStore';
import { listTeams, type TeamData } from '../../api/team.api';
import { listFolders, type FolderData } from '../../api/folders.api';
import type { AccessPolicyData, AccessPolicyTargetType, CreateAccessPolicyInput } from '../../api/accessPolicy.api';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import { isAdminOrAbove } from '../../utils/roles';
import {
  PolicyEmptyState,
  PolicyMetadataBadge,
  PolicyRecordCard,
} from './databasePolicyUi';
import {
  SettingsLoadingState,
  SettingsPanel,
  SettingsSummaryGrid,
  SettingsSummaryItem,
} from './settings-ui';
import {
  AccessPolicyDeleteDialog,
  AccessPolicyFormDialog,
} from './accessPolicyDialogs';
import {
  ACCESS_POLICY_TARGET_LABELS,
  EMPTY_ACCESS_POLICY_FORM,
  buildAccessPolicyNameMap,
  buildTargetOptions,
  timeWindowBadges,
  validateTimeWindows,
  type AccessPolicyFormState,
} from './accessPolicyUtils';

function policyTargetName(policy: AccessPolicyData, nameMap: Record<string, string>) {
  return nameMap[policy.targetId] ?? policy.targetId;
}

export default function AccessPolicySection() {
  const user = useAuthStore((state) => state.user);
  const tenantId = user?.tenantId;
  const isAdmin = isAdminOrAbove(user?.tenantRole);

  const policies = useAccessPolicyStore((state) => state.policies);
  const loading = useAccessPolicyStore((state) => state.loading);
  const fetchError = useAccessPolicyStore((state) => state.error);
  const fetchPolicies = useAccessPolicyStore((state) => state.fetchPolicies);
  const createPolicyAction = useAccessPolicyStore((state) => state.createPolicy);
  const updatePolicyAction = useAccessPolicyStore((state) => state.updatePolicy);
  const deletePolicyAction = useAccessPolicyStore((state) => state.deletePolicy);

  const [teams, setTeams] = useState<TeamData[]>([]);
  const [folders, setFolders] = useState<FolderData[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingPolicy, setEditingPolicy] = useState<AccessPolicyData | null>(null);
  const [form, setForm] = useState<AccessPolicyFormState>(EMPTY_ACCESS_POLICY_FORM);
  const [formError, setFormError] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<AccessPolicyData | null>(null);

  const saveAction = useAsyncAction();
  const deleteAction = useAsyncAction();

  useEffect(() => {
    if (!tenantId) return;

    fetchPolicies();
    listTeams().then(setTeams).catch(() => {});
    listFolders()
      .then((response) => setFolders([...response.personal, ...response.team]))
      .catch(() => {});
  }, [fetchPolicies, tenantId]);

  const nameMap = useMemo(
    () => buildAccessPolicyNameMap({ tenantId, teams, folders }),
    [folders, teams, tenantId],
  );

  const targetOptions = useMemo(
    () => buildTargetOptions({ targetType: form.targetType, tenantId, teams, folders }),
    [folders, form.targetType, teams, tenantId],
  );

  const summary = useMemo(
    () => ({
      folders: policies.filter((policy) => policy.targetType === 'FOLDER').length,
      mfaStepUp: policies.filter((policy) => policy.requireMfaStepUp).length,
      tenant: policies.filter((policy) => policy.targetType === 'TENANT').length,
      trustedDevice: policies.filter((policy) => policy.requireTrustedDevice).length,
    }),
    [policies],
  );

  const handleCreate = useCallback(() => {
    setEditingPolicy(null);
    setForm({ ...EMPTY_ACCESS_POLICY_FORM, targetId: tenantId ?? '' });
    setFormError('');
    saveAction.clearError();
    setDialogOpen(true);
  }, [saveAction, tenantId]);

  const handleEdit = useCallback((policy: AccessPolicyData) => {
    setEditingPolicy(policy);
    setForm({
      targetType: policy.targetType,
      targetId: policy.targetId,
      allowedTimeWindows: policy.allowedTimeWindows ?? '',
      requireTrustedDevice: policy.requireTrustedDevice,
      requireMfaStepUp: policy.requireMfaStepUp,
    });
    setFormError('');
    saveAction.clearError();
    setDialogOpen(true);
  }, [saveAction]);

  const handleClose = useCallback(() => {
    setDialogOpen(false);
    setEditingPolicy(null);
    setFormError('');
    saveAction.clearError();
  }, [saveAction]);

  const handleTargetTypeChange = (nextTargetType: AccessPolicyTargetType) => {
    setForm((current) => ({
      ...current,
      targetType: nextTargetType,
      targetId: nextTargetType === 'TENANT' ? (tenantId ?? '') : '',
    }));
    setFormError('');
  };

  const handleSave = async () => {
    const timeWindowError = validateTimeWindows(form.allowedTimeWindows);
    if (timeWindowError) {
      setFormError(timeWindowError);
      return;
    }

    if (!editingPolicy && !form.targetId) {
      setFormError('Select a target before saving');
      return;
    }

    const ok = await saveAction.run(async () => {
      if (editingPolicy) {
        await updatePolicyAction(editingPolicy.id, {
          allowedTimeWindows: form.allowedTimeWindows.trim() || null,
          requireTrustedDevice: form.requireTrustedDevice,
          requireMfaStepUp: form.requireMfaStepUp,
        });
        return;
      }

      const payload: CreateAccessPolicyInput = {
        targetType: form.targetType,
        targetId: form.targetId,
        allowedTimeWindows: form.allowedTimeWindows.trim() || null,
        requireTrustedDevice: form.requireTrustedDevice,
        requireMfaStepUp: form.requireMfaStepUp,
      };
      await createPolicyAction(payload);
    }, 'Failed to save policy');

    if (ok) {
      handleClose();
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;

    const ok = await deleteAction.run(async () => {
      await deletePolicyAction(deleteTarget.id);
    }, 'Failed to delete policy');

    if (ok) {
      setDeleteTarget(null);
    }
  };

  if (!tenantId || !isAdmin) return null;

  return (
    <>
      <SettingsPanel
        title="Session access policies"
        description="Define lean rules for when sessions are allowed, when trusted devices are mandatory, and when step-up MFA is required."
        heading={(
          <Button type="button" onClick={handleCreate}>
            <Plus className="size-4" />
            Add Policy
          </Button>
        )}
      >
        <div className="space-y-5">
          <SettingsSummaryGrid className="xl:grid-cols-4">
            <SettingsSummaryItem label="Tenant rules" value={summary.tenant} />
            <SettingsSummaryItem label="Folder rules" value={summary.folders} />
            <SettingsSummaryItem label="Trusted device" value={`${summary.trustedDevice} required`} />
            <SettingsSummaryItem label="Step-up MFA" value={`${summary.mfaStepUp} required`} />
          </SettingsSummaryGrid>

          {fetchError && (
            <Alert variant="destructive">
              <AlertDescription>{fetchError}</AlertDescription>
            </Alert>
          )}

          {loading ? (
            <SettingsLoadingState message="Loading access policies..." />
          ) : policies.length === 0 ? (
            <PolicyEmptyState
              title="No access policies yet"
              description="Sessions currently rely on the broader tenant policy. Add a targeted rule when teams, folders, or the whole organization need stricter access windows."
            />
          ) : (
            <div className="space-y-4">
              {policies.map((policy) => {
                const targetName = policyTargetName(policy, nameMap);
                return (
                  <PolicyRecordCard
                    key={policy.id}
                    title={`${ACCESS_POLICY_TARGET_LABELS[policy.targetType]} policy`}
                    description={`Applies to ${targetName}. Policies stack, so the most restrictive applicable rule wins.`}
                    badges={(
                      <>
                        <PolicyMetadataBadge>{ACCESS_POLICY_TARGET_LABELS[policy.targetType]}</PolicyMetadataBadge>
                        {timeWindowBadges(policy).map((timeWindow) => (
                          <PolicyMetadataBadge key={`${policy.id}-${timeWindow}`} variant="secondary">
                            <Clock3 className="mr-1 size-3.5" />
                            {timeWindow}
                          </PolicyMetadataBadge>
                        ))}
                        <PolicyMetadataBadge variant={policy.requireTrustedDevice ? 'default' : 'outline'}>
                          <ShieldCheck className="mr-1 size-3.5" />
                          {policy.requireTrustedDevice ? 'Trusted device required' : 'Trusted device optional'}
                        </PolicyMetadataBadge>
                        <PolicyMetadataBadge variant={policy.requireMfaStepUp ? 'default' : 'outline'}>
                          <ShieldAlert className="mr-1 size-3.5" />
                          {policy.requireMfaStepUp ? 'Step-up MFA required' : 'No step-up MFA'}
                        </PolicyMetadataBadge>
                      </>
                    )}
                    metadata={(
                      <>
                        <span>Target: {targetName}</span>
                        <span>Updated {new Date(policy.updatedAt).toLocaleDateString()}</span>
                      </>
                    )}
                    onEdit={() => handleEdit(policy)}
                    onDelete={() => setDeleteTarget(policy)}
                  />
                );
              })}
            </div>
          )}
        </div>
      </SettingsPanel>

      <AccessPolicyFormDialog
        open={dialogOpen}
        editingPolicy={editingPolicy}
        form={form}
        formError={formError}
        saveError={saveAction.error}
        saving={saveAction.loading}
        targetOptions={targetOptions}
        onClose={handleClose}
        onSave={handleSave}
        onFormChange={(nextForm) => {
          setForm(nextForm);
          setFormError('');
        }}
        onTargetTypeChange={handleTargetTypeChange}
      />

      <AccessPolicyDeleteDialog
        open={Boolean(deleteTarget)}
        policyName={deleteTarget ? policyTargetName(deleteTarget, nameMap) : 'this target'}
        deleteError={deleteAction.error}
        deleting={deleteAction.loading}
        onClose={() => setDeleteTarget(null)}
        onDelete={handleDelete}
      />
    </>
  );
}
