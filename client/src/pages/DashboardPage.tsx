import { useEffect, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import WorkspaceShell from '@/components/Workspace/WorkspaceShell';
import type { ConnectionData } from '@/api/connections.api';
import { useAuthStore } from '@/store/authStore';
import { useConnectionsStore } from '@/store/connectionsStore';
import { useFeatureFlagsStore } from '@/store/featureFlagsStore';
import { useTabsStore } from '@/store/tabsStore';

export default function DashboardPage() {
  const fetchConnections = useConnectionsStore((state) => state.fetchConnections);
  const restoreTabs = useTabsStore((state) => state.restoreTabs);
  const openTab = useTabsStore((state) => state.openTab);
  const fetchDomainProfile = useAuthStore((state) => state.fetchDomainProfile);
  const fetchFeatureFlags = useFeatureFlagsStore((state) => state.fetchFeatureFlags);
  const [searchParams, setSearchParams] = useSearchParams();
  const autoconnectHandled = useRef(false);

  useEffect(() => {
    fetchConnections().then(() => {
      const { ownConnections, sharedConnections, teamConnections } = useConnectionsStore.getState();
      const allConnections = [...ownConnections, ...sharedConnections, ...teamConnections];
      void restoreTabs(allConnections);

      const autoconnectId = searchParams.get('autoconnect');
      if (autoconnectId && !autoconnectHandled.current) {
        autoconnectHandled.current = true;
        const connection: ConnectionData | undefined = allConnections.find(
          (candidate) => candidate.id === autoconnectId,
        );
        if (connection) {
          openTab(connection);
        }
        const nextParams = new URLSearchParams(searchParams);
        nextParams.delete('autoconnect');
        setSearchParams(nextParams, { replace: true });
      }
    });

    void fetchDomainProfile();
    void fetchFeatureFlags();
  }, [fetchConnections, restoreTabs, openTab, fetchDomainProfile, fetchFeatureFlags, searchParams, setSearchParams]);

  return <WorkspaceShell />;
}
