import { useEffect } from 'react';
import { List, Pause, Play, ScrollText, X } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';
import { useAuthStore } from '../../store/authStore';
import { useFeatureFlagsStore } from '../../store/featureFlagsStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import AuditLogGeneralView from './AuditLogGeneralView';
import AuditLogSqlView from './AuditLogSqlView';

interface AuditLogDialogProps {
  open: boolean;
  onClose: () => void;
  onGeoIpClick?: (ip: string) => void;
  onViewUserProfile?: (userId: string) => void;
}

export default function AuditLogDialog({
  open,
  onClose,
  onGeoIpClick,
  onViewUserProfile,
}: AuditLogDialogProps) {
  const user = useAuthStore((state) => state.user);
  const databaseProxyEnabled = useFeatureFlagsStore((state) => state.databaseProxyEnabled);
  const autoRefreshPaused = useUiPreferencesStore((state) => state.auditLogAutoRefreshPaused);
  const auditLogTab = useUiPreferencesStore((state) => state.auditLogDialogTab);
  const setUiPref = useUiPreferencesStore((state) => state.set);

  const hasTenant = Boolean(user?.tenantId);
  const sqlAuditVisible = hasTenant && databaseProxyEnabled;
  const activeTab = sqlAuditVisible && auditLogTab === 'sql' ? 'sql' : 'general';

  useEffect(() => {
    if (open && auditLogTab === 'sql' && !sqlAuditVisible) {
      setUiPref('auditLogDialogTab', 'general');
    }
  }, [auditLogTab, open, setUiPref, sqlAuditVisible]);

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) onClose(); }}>
      <DialogContent
        className="h-[100dvh] w-screen max-w-none gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
        showCloseButton={false}
      >
        <DialogTitle className="sr-only">Activity Log</DialogTitle>
        <DialogDescription className="sr-only">System audit log</DialogDescription>

        <div className="border-b bg-card">
          <div className="flex items-center gap-3 px-4 py-2.5">
            <Button variant="ghost" size="icon" onClick={onClose} className="size-8">
              <X className="size-4" />
            </Button>
            <h2 className="flex-1 text-lg font-semibold">Activity Log</h2>
            <Button
              variant="ghost"
              size="icon"
              className="size-8"
              onClick={() => setUiPref('auditLogAutoRefreshPaused', !autoRefreshPaused)}
              title={autoRefreshPaused ? 'Resume live updates' : 'Pause live updates'}
            >
              {autoRefreshPaused ? <Play className="size-4" /> : <Pause className="size-4" />}
            </Button>
            <Badge
              variant={autoRefreshPaused ? 'outline' : 'default'}
              className={cn(
                'font-semibold',
                autoRefreshPaused ? '' : 'border-emerald-600/30 bg-emerald-600/15 text-emerald-400',
              )}
            >
              {autoRefreshPaused ? (
                'Paused'
              ) : (
                <span className="inline-flex items-center gap-1.5">
                  <span className="size-1.5 animate-pulse rounded-full bg-current" />
                  Live
                </span>
              )}
            </Badge>
          </div>
          {hasTenant ? (
            <Tabs value={activeTab} onValueChange={(value) => setUiPref('auditLogDialogTab', value)} className="px-4">
              <TabsList className="h-9">
                <TabsTrigger value="general" className="gap-1.5 text-xs">
                  <List className="size-3.5" />
                  General
                </TabsTrigger>
                {sqlAuditVisible ? (
                  <TabsTrigger value="sql" className="gap-1.5 text-xs">
                    <ScrollText className="size-3.5" />
                    SQL Audit
                  </TabsTrigger>
                ) : null}
              </TabsList>
            </Tabs>
          ) : null}
        </div>

        <div className="flex-1 overflow-auto p-4">
          {activeTab === 'general' ? (
            <AuditLogGeneralView
              open={open}
              onGeoIpClick={onGeoIpClick}
              onViewUserProfile={onViewUserProfile}
            />
          ) : null}
          {activeTab === 'sql' ? <AuditLogSqlView open={open} /> : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}
