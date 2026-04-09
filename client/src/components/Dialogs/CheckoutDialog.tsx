import { useState, useEffect, useCallback } from 'react';
import {
  Dialog, DialogContent, DialogDescription, DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import {
  X, CheckCircle, XCircle, Undo2, RefreshCw, Loader2,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useCheckoutStore } from '../../store/checkoutStore';
import type { CheckoutRequest, CheckoutStatus } from '../../store/checkoutStore';
import { useAuthStore } from '../../store/authStore';
import { useAsyncAction } from '../../hooks/useAsyncAction';

interface CheckoutDialogProps {
  open: boolean;
  onClose: () => void;
}

const STATUS_VARIANT: Record<CheckoutStatus, string> = {
  PENDING: 'bg-yellow-600/15 text-yellow-500 border-yellow-600/30',
  APPROVED: 'bg-emerald-600/15 text-emerald-400 border-emerald-600/30',
  REJECTED: 'bg-destructive/15 text-destructive border-destructive/30',
  EXPIRED: 'bg-muted text-muted-foreground border-border',
  CHECKED_IN: 'bg-blue-600/15 text-blue-400 border-blue-600/30',
};

function formatDuration(minutes: number): string {
  if (minutes < 60) return `${minutes}m`;
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return m > 0 ? `${h}h ${m}m` : `${h}h`;
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString();
}

function useCurrentTime(intervalMs: number): number {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), intervalMs);
    return () => clearInterval(timer);
  }, [intervalMs]);

  return now;
}

function TimeRemaining({ expiresAt }: { expiresAt: string | null }) {
  const now = useCurrentTime(15_000);

  if (!expiresAt) return null;
  const diff = new Date(expiresAt).getTime() - now;
  if (diff <= 0) return <Badge variant="secondary">Expired</Badge>;
  const mins = Math.floor(diff / 60000);
  return <Badge variant="outline" className="border-emerald-600/30 text-emerald-400">{formatDuration(mins)} left</Badge>;
}

export default function CheckoutDialog({ open, onClose }: CheckoutDialogProps) {
  const requests = useCheckoutStore((s) => s.requests);
  const total = useCheckoutStore((s) => s.total);
  const loading = useCheckoutStore((s) => s.loading);
  const fetchRequests = useCheckoutStore((s) => s.fetchRequests);
  const setFilters = useCheckoutStore((s) => s.setFilters);
  const approve = useCheckoutStore((s) => s.approve);
  const reject = useCheckoutStore((s) => s.reject);
  const checkin = useCheckoutStore((s) => s.checkin);
  const currentUserId = useAuthStore((s) => s.user?.id);

  const [tab, setTab] = useState('all');
  const { loading: actionLoading, error: actionError, setError, run } = useAsyncAction();

  const handleTabChange = useCallback((newVal: string) => {
    setTab(newVal);
    const roles: Record<string, 'all' | 'requester' | 'approver'> = { all: 'all', requester: 'requester', approver: 'approver' };
    setFilters({ role: roles[newVal], offset: 0 });
  }, [setFilters]);

  useEffect(() => {
    if (open) {
      fetchRequests();
    }
  }, [open, fetchRequests]);

  // Refetch when filters change
  const filters = useCheckoutStore((s) => s.filters);
  useEffect(() => {
    if (open) {
      fetchRequests();
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters.role, filters.status]);

  const handleApprove = async (id: string) => {
    await run(async () => { await approve(id); }, 'Failed to approve checkout');
  };

  const handleReject = async (id: string) => {
    await run(async () => { await reject(id); }, 'Failed to reject checkout');
  };

  const handleCheckin = async (id: string) => {
    await run(async () => { await checkin(id); }, 'Failed to check in');
  };

  const canApprove = (req: CheckoutRequest): boolean => {
    return req.status === 'PENDING' && req.requesterId !== currentUserId;
  };

  const canCheckin = (req: CheckoutRequest): boolean => {
    return req.status === 'APPROVED' && (req.requesterId === currentUserId || req.approverId === currentUserId);
  };

  const resourceLabel = (req: CheckoutRequest): string => {
    if (req.secretName) return `Secret: ${req.secretName}`;
    if (req.connectionName) return `Connection: ${req.connectionName}`;
    return req.secretId ? `Secret: ${req.secretId.slice(0, 8)}...` : `Connection: ${req.connectionId?.slice(0, 8)}...`;
  };

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) onClose(); }}>
      <DialogContent
        className="h-[100dvh] w-screen max-w-none gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
        showCloseButton={false}
      >
        <DialogTitle className="sr-only">Credential Check-out</DialogTitle>
        <DialogDescription className="sr-only">Manage credential checkout requests</DialogDescription>

        {/* Header */}
        <div className="flex items-center gap-3 border-b px-4 py-2.5 bg-card">
          <Button variant="ghost" size="icon" onClick={onClose} className="size-8">
            <X className="size-4" />
          </Button>
          <h2 className="flex-1 text-lg font-semibold">Credential Check-out</h2>
          <Button variant="ghost" size="icon" onClick={fetchRequests} disabled={loading} className="size-8">
            <RefreshCw className="size-4" />
          </Button>
        </div>

        {/* Body */}
        <div className="flex flex-1 flex-col overflow-hidden p-4">
          {actionError && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive mb-4 flex items-center justify-between">
              {actionError}
              <Button variant="ghost" size="icon" className="size-6" onClick={() => setError('')}>
                <X className="size-3" />
              </Button>
            </div>
          )}

          <Tabs value={tab} onValueChange={handleTabChange}>
            <TabsList className="mb-4">
              <TabsTrigger value="all">All ({total})</TabsTrigger>
              <TabsTrigger value="requester">My Requests</TabsTrigger>
              <TabsTrigger value="approver">Pending Approvals</TabsTrigger>
            </TabsList>

            <TabsContent value={tab} className="flex-1 overflow-auto mt-0">
              {loading && !actionLoading ? (
                <div className="flex justify-center py-8">
                  <Loader2 className="size-8 animate-spin text-muted-foreground" />
                </div>
              ) : requests.length === 0 ? (
                <div className="flex justify-center py-8">
                  <p className="text-sm text-muted-foreground">No checkout requests found</p>
                </div>
              ) : (
                <div className="rounded-lg border overflow-auto flex-1">
                  <table className="w-full text-sm">
                    <thead className="sticky top-0 bg-card">
                      <tr className="border-b">
                        <th className="text-left px-3 py-2 font-medium">Resource</th>
                        <th className="text-left px-3 py-2 font-medium">Requester</th>
                        <th className="text-left px-3 py-2 font-medium">Duration</th>
                        <th className="text-left px-3 py-2 font-medium">Reason</th>
                        <th className="text-left px-3 py-2 font-medium">Status</th>
                        <th className="text-left px-3 py-2 font-medium">Time Left</th>
                        <th className="text-left px-3 py-2 font-medium">Requested</th>
                        <th className="text-right px-3 py-2 font-medium">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {requests.map((req) => (
                        <tr key={req.id} className="border-b hover:bg-accent/50">
                          <td className="px-3 py-2">
                            <span className="truncate max-w-[200px] block text-sm">
                              {resourceLabel(req)}
                            </span>
                          </td>
                          <td className="px-3 py-2">
                            <span className="truncate block text-sm">
                              {req.requester.username || req.requester.email}
                            </span>
                          </td>
                          <td className="px-3 py-2">{formatDuration(req.durationMinutes)}</td>
                          <td className="px-3 py-2">
                            <span className="truncate max-w-[150px] block text-sm">
                              {req.reason || '-'}
                            </span>
                          </td>
                          <td className="px-3 py-2">
                            <Badge variant="outline" className={cn('border', STATUS_VARIANT[req.status])}>
                              {req.status}
                            </Badge>
                          </td>
                          <td className="px-3 py-2">
                            {req.status === 'APPROVED' ? <TimeRemaining expiresAt={req.expiresAt} /> : '-'}
                          </td>
                          <td className="px-3 py-2">
                            <span className="text-xs">{formatDate(req.createdAt)}</span>
                          </td>
                          <td className="px-3 py-2 text-right whitespace-nowrap">
                            {canApprove(req) && (
                              <>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="size-7 text-emerald-400 hover:text-emerald-300"
                                  onClick={() => handleApprove(req.id)}
                                  disabled={actionLoading}
                                  title="Approve"
                                >
                                  <CheckCircle className="size-4" />
                                </Button>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="size-7 text-destructive hover:text-destructive/80"
                                  onClick={() => handleReject(req.id)}
                                  disabled={actionLoading}
                                  title="Reject"
                                >
                                  <XCircle className="size-4" />
                                </Button>
                              </>
                            )}
                            {canCheckin(req) && (
                              <Button
                                variant="ghost"
                                size="icon"
                                className="size-7 text-primary hover:text-primary/80"
                                onClick={() => handleCheckin(req.id)}
                                disabled={actionLoading}
                                title="Check in (return access)"
                              >
                                <Undo2 className="size-4" />
                              </Button>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </TabsContent>
          </Tabs>
        </div>
      </DialogContent>
    </Dialog>
  );
}
