import { useState, useEffect, useCallback } from 'react';
import {
  Dialog, DialogContent, DialogTitle, DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import {
  X, Shield, Users, Loader2,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { getUserProfile, UserProfileData } from '../../api/tenant.api';
import { getTenantAuditLogs, TenantAuditLogEntry } from '../../api/audit.api';
import { useAuthStore } from '../../store/authStore';
import { ACTION_LABELS, getActionColor } from '../Audit/auditConstants';
import { useAsyncAction } from '../../hooks/useAsyncAction';

interface UserProfileDialogProps {
  open: boolean;
  onClose: () => void;
  userId: string | null;
}

const ROLE_COLORS: Record<string, string> = {
  OWNER: 'bg-destructive/15 text-destructive border-destructive/30',
  ADMIN: 'bg-yellow-600/15 text-yellow-500 border-yellow-600/30',
  MEMBER: 'bg-primary/15 text-primary border-primary/30',
};

const ACTION_COLOR_MAP: Record<string, string> = {
  default: '',
  primary: 'bg-primary/15 text-primary border-primary/30',
  secondary: 'bg-muted text-muted-foreground',
  error: 'bg-destructive/15 text-destructive border-destructive/30',
  warning: 'bg-yellow-600/15 text-yellow-500 border-yellow-600/30',
  success: 'bg-emerald-600/15 text-emerald-400 border-emerald-600/30',
  info: 'bg-blue-600/15 text-blue-400 border-blue-600/30',
};

export default function UserProfileDialog({ open, onClose, userId }: UserProfileDialogProps) {
  const tenantId = useAuthStore((s) => s.user?.tenantId);

  const [profile, setProfile] = useState<UserProfileData | null>(null);
  const { loading, error, run } = useAsyncAction();

  // Audit log state (admin only)
  const [auditLogs, setAuditLogs] = useState<TenantAuditLogEntry[]>([]);
  const [auditTotal, setAuditTotal] = useState(0);
  const [auditPage, setAuditPage] = useState(0);
  const [auditLoading, setAuditLoading] = useState(false);

  const fetchProfile = useCallback(async () => {
    if (!tenantId || !userId) return;
    await run(async () => {
      const data = await getUserProfile(tenantId, userId);
      setProfile(data);
    }, 'Failed to load profile');
  }, [tenantId, userId, run]);

  const fetchAuditLogs = useCallback(async (page: number) => {
    if (!tenantId || !userId || !profile?.email) return;
    setAuditLoading(true);
    try {
      const res = await getTenantAuditLogs({ userId, page: page + 1, limit: 10 });
      setAuditLogs(res.data);
      setAuditTotal(res.total);
    } catch {
      // Silently fail — audit logs are supplementary
    } finally {
      setAuditLoading(false);
    }
  }, [tenantId, userId, profile?.email]);

  useEffect(() => {
    if (open && userId) {
      setProfile(null);
      setAuditLogs([]);
      setAuditPage(0);
      fetchProfile();
    }
  }, [open, userId, fetchProfile]);

  useEffect(() => {
    if (profile?.email) {
      fetchAuditLogs(auditPage);
    }
  }, [profile?.email, auditPage, fetchAuditLogs]);

  const isAdmin = !!profile?.email;
  const totalAuditPages = Math.ceil(auditTotal / 10);

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) onClose(); }}>
      <DialogContent
        className="h-[100dvh] w-screen max-w-none gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
        showCloseButton={false}
      >
        <DialogTitle className="sr-only">User Profile</DialogTitle>
        <DialogDescription className="sr-only">View user profile details</DialogDescription>

        {/* Header */}
        <div className="flex items-center gap-3 border-b px-4 py-2.5 bg-card">
          <Button variant="ghost" size="icon" onClick={onClose} className="size-8">
            <X className="size-4" />
          </Button>
          <h2 className="flex-1 text-lg font-semibold">User Profile</h2>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-auto p-6">
          {loading && (
            <div className="flex justify-center py-16">
              <Loader2 className="size-8 animate-spin text-muted-foreground" />
            </div>
          )}

          {error && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive mb-4">
              {error}
            </div>
          )}

          {profile && !loading && (
            <div className="max-w-[800px] mx-auto">
              {/* Public Section */}
              <div className="flex items-center gap-4 mb-6">
                <div className="flex items-center justify-center size-20 rounded-full bg-muted text-2xl font-semibold">
                  {profile.avatarData ? (
                    <img src={profile.avatarData} alt="" className="size-20 rounded-full object-cover" />
                  ) : (
                    (profile.username ?? '?')[0]?.toUpperCase()
                  )}
                </div>
                <div>
                  <h3 className="text-xl font-semibold">
                    {profile.username || 'No username'}
                  </h3>
                  <div className="flex items-center gap-2 mt-1">
                    <Badge variant="outline" className={cn('border', ROLE_COLORS[profile.role] || '')}>
                      {profile.role}
                    </Badge>
                    <span className="text-sm text-muted-foreground">
                      Member since {new Date(profile.joinedAt).toLocaleDateString()}
                    </span>
                  </div>
                </div>
              </div>

              {/* Teams */}
              {profile.teams.length > 0 && (
                <div className="mb-6">
                  <div className="flex items-center gap-2 mb-2">
                    <Users className="size-4 text-muted-foreground" />
                    <span className="text-sm font-medium text-muted-foreground">Teams</span>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {profile.teams.map((t) => (
                      <Badge key={t.id} variant="outline">{t.name} ({t.role})</Badge>
                    ))}
                  </div>
                </div>
              )}

              {/* Admin Section */}
              {isAdmin && (
                <>
                  <Separator className="my-6" />

                  <div className="flex items-center gap-2 mb-4">
                    <Shield className="size-4 text-muted-foreground" />
                    <span className="text-base font-semibold">Administration</span>
                  </div>

                  <div className="space-y-3 mb-6">
                    <div className="flex gap-4">
                      <span className="text-sm text-muted-foreground min-w-[120px]">Email</span>
                      <span className="text-sm">{profile.email}</span>
                    </div>

                    <div className="flex gap-4">
                      <span className="text-sm text-muted-foreground min-w-[120px]">MFA Status</span>
                      <div className="flex gap-1.5">
                        {profile.totpEnabled && <Badge className="bg-emerald-600/15 text-emerald-400 border-emerald-600/30" variant="outline">TOTP</Badge>}
                        {profile.smsMfaEnabled && <Badge className="bg-emerald-600/15 text-emerald-400 border-emerald-600/30" variant="outline">SMS</Badge>}
                        {profile.webauthnEnabled && <Badge className="bg-emerald-600/15 text-emerald-400 border-emerald-600/30" variant="outline">WebAuthn</Badge>}
                        {!profile.totpEnabled && !profile.smsMfaEnabled && !profile.webauthnEnabled && (
                          <Badge variant="secondary">None</Badge>
                        )}
                      </div>
                    </div>

                    <div className="flex gap-4">
                      <span className="text-sm text-muted-foreground min-w-[120px]">Last Activity</span>
                      <span className="text-sm">
                        {profile.lastActivity
                          ? new Date(profile.lastActivity).toLocaleString()
                          : 'No activity recorded'}
                      </span>
                    </div>

                    {profile.updatedAt && (
                      <div className="flex gap-4">
                        <span className="text-sm text-muted-foreground min-w-[120px]">Last Updated</span>
                        <span className="text-sm">
                          {new Date(profile.updatedAt).toLocaleString()}
                        </span>
                      </div>
                    )}
                  </div>

                  {/* Embedded Audit Log */}
                  <h4 className="text-sm font-medium text-muted-foreground mb-2">Audit Log</h4>

                  <div className="rounded-lg border overflow-hidden">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b bg-muted/50">
                          <th className="text-left px-3 py-2 font-medium">Date</th>
                          <th className="text-left px-3 py-2 font-medium">Action</th>
                          <th className="text-left px-3 py-2 font-medium">IP Address</th>
                        </tr>
                      </thead>
                      <tbody>
                        {auditLoading && (
                          <tr>
                            <td colSpan={3} className="text-center py-6">
                              <Loader2 className="size-5 animate-spin mx-auto text-muted-foreground" />
                            </td>
                          </tr>
                        )}
                        {!auditLoading && auditLogs.length === 0 && (
                          <tr>
                            <td colSpan={3} className="text-center py-6">
                              <span className="text-sm text-muted-foreground">No audit entries</span>
                            </td>
                          </tr>
                        )}
                        {!auditLoading && auditLogs.map((log) => (
                          <tr key={log.id} className="border-b last:border-0 hover:bg-accent/50">
                            <td className="px-3 py-2 whitespace-nowrap">
                              {new Date(log.createdAt).toLocaleString()}
                            </td>
                            <td className="px-3 py-2">
                              <Badge
                                variant="outline"
                                className={cn('border', ACTION_COLOR_MAP[getActionColor(log.action) as string] || '')}
                              >
                                {ACTION_LABELS[log.action as keyof typeof ACTION_LABELS] ?? log.action}
                              </Badge>
                            </td>
                            <td className="px-3 py-2 font-mono text-xs">
                              {log.ipAddress ?? '\u2014'}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                    {auditTotal > 10 && (
                      <div className="flex items-center justify-between px-3 py-2 border-t text-sm text-muted-foreground">
                        <span>{auditTotal} total entries</span>
                        <div className="flex items-center gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={auditPage === 0}
                            onClick={() => setAuditPage((p) => p - 1)}
                          >
                            Previous
                          </Button>
                          <span>Page {auditPage + 1} of {totalAuditPages}</span>
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={auditPage + 1 >= totalAuditPages}
                            onClick={() => setAuditPage((p) => p + 1)}
                          >
                            Next
                          </Button>
                        </div>
                      </div>
                    )}
                  </div>
                </>
              )}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
