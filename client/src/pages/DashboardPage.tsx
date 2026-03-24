import { useEffect, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Box } from '@mui/material';
import MainLayout from '../components/Layout/MainLayout';
import { useConnectionsStore } from '../store/connectionsStore';
import { useTabsStore } from '../store/tabsStore';
import { useAuthStore } from '../store/authStore';
import { useFeatureFlagsStore } from '../store/featureFlagsStore';
import type { ConnectionData } from '../api/connections.api';

export default function DashboardPage() {
  const fetchConnections = useConnectionsStore((s) => s.fetchConnections);
  const restoreTabs = useTabsStore((s) => s.restoreTabs);
  const openTab = useTabsStore((s) => s.openTab);
  const fetchDomainProfile = useAuthStore((s) => s.fetchDomainProfile);
  const fetchFeatureFlags = useFeatureFlagsStore((s) => s.fetchFeatureFlags);
  const [searchParams, setSearchParams] = useSearchParams();
  const autoconnectHandled = useRef(false);

  useEffect(() => {
    fetchConnections().then(() => {
      const { ownConnections, sharedConnections, teamConnections } =
        useConnectionsStore.getState();
      const allConnections = [...ownConnections, ...sharedConnections, ...teamConnections];
      restoreTabs(allConnections);

      // Handle autoconnect query parameter (e.g. from browser extension deep link)
      const autoconnectId = searchParams.get('autoconnect');
      if (autoconnectId && !autoconnectHandled.current) {
        autoconnectHandled.current = true;
        const connection: ConnectionData | undefined = allConnections.find(
          (c) => c.id === autoconnectId,
        );
        if (connection) {
          openTab(connection);
        }
        // Remove the autoconnect param from the URL
        searchParams.delete('autoconnect');
        setSearchParams(searchParams, { replace: true });
      }
    });
    fetchDomainProfile();
    fetchFeatureFlags();
  // eslint-disable-next-line react-hooks/exhaustive-deps -- one-time setup on mount
  }, [fetchConnections, restoreTabs, fetchDomainProfile, fetchFeatureFlags]);

  return (
    <Box sx={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <MainLayout />
    </Box>
  );
}
