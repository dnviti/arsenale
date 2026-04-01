import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Box, CircularProgress, Typography } from '@mui/material';
import { getConnection, ConnectionData } from '../api/connections.api';
import { restoreSessionApi } from '../api/auth.api';
import { useAuthStore } from '../store/authStore';
import SshTerminal from '../components/Terminal/SshTerminal';
import RdpViewer from '../components/RDP/RdpViewer';
import { extractApiError } from '../utils/apiError';

export default function ConnectionViewerPage() {
  const { id } = useParams<{ id: string }>();
  const accessToken = useAuthStore((s) => s.accessToken);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const setAccessToken = useAuthStore((s) => s.setAccessToken);

  const [authReady, setAuthReady] = useState(Boolean(accessToken));
  const [connection, setConnection] = useState<ConnectionData | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  // Bootstrap auth: accessToken is not persisted, so refresh it for popup windows
  /* eslint-disable react-hooks/set-state-in-effect -- bootstrap auth state for popup windows */
  useEffect(() => {
    if (accessToken) {
      setAuthReady(true);
      return;
    }
    restoreSessionApi()
      .then((res) => {
        setAccessToken(res.accessToken);
        if (res.csrfToken) useAuthStore.getState().setCsrfToken(res.csrfToken);
        setAuthReady(true);
      })
      .catch(() => {
        if (isAuthenticated) {
          setError('Authentication failed. Please log in again.');
        } else {
          setError('Not authenticated. Please log in.');
        }
        setLoading(false);
      });
  }, [accessToken, isAuthenticated, setAccessToken]);
  /* eslint-enable react-hooks/set-state-in-effect */

  // Fetch connection data once auth is ready
  useEffect(() => {
    if (!authReady || !id) return;
    getConnection(id)
      .then((data) => {
        setConnection(data);
        document.title = `${data.name} - Arsenale`;
      })
      .catch((err) => {
        setError(extractApiError(err, 'Failed to load connection'));
      })
      .finally(() => setLoading(false));
  }, [authReady, id]);

  if (loading || !authReady) {
    return (
      <Box sx={{ width: '100vw', height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: '#1a1a2e' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !connection) {
    return (
      <Box sx={{ width: '100vw', height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: '#1a1a2e' }}>
        <Typography color="error">{error || 'Connection not found'}</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ width: '100vw', height: '100vh', display: 'flex', overflow: 'hidden', bgcolor: '#1a1a2e' }}>
      {connection.type === 'SSH' ? (
        <SshTerminal connectionId={connection.id} tabId={`popup-${connection.id}`} />
      ) : (
        <RdpViewer connectionId={connection.id} tabId={`popup-${connection.id}`} />
      )}
    </Box>
  );
}
