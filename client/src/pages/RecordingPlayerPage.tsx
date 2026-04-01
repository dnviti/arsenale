import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Box, CircularProgress, Typography } from '@mui/material';
import { restoreSessionApi } from '../api/auth.api';
import { getRecording } from '../api/recordings.api';
import type { Recording } from '../api/recordings.api';
import { useAuthStore } from '../store/authStore';
import GuacPlayer from '../components/Recording/GuacPlayer';
import SshPlayer from '../components/Recording/SshPlayer';
import { extractApiError } from '../utils/apiError';

export default function RecordingPlayerPage() {
  const { id } = useParams<{ id: string }>();
  const accessToken = useAuthStore((s) => s.accessToken);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const setAccessToken = useAuthStore((s) => s.setAccessToken);

  const [authReady, setAuthReady] = useState(Boolean(accessToken));
  const [recording, setRecording] = useState<Recording | null>(null);
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

  // Fetch recording data once auth is ready
  useEffect(() => {
    if (!authReady || !id) return;
    getRecording(id)
      .then((data) => {
        setRecording(data);
        document.title = `${data.connection.name} (${data.protocol}) - Recording - Arsenale`;
      })
      .catch((err) => {
        setError(extractApiError(err, 'Failed to load recording'));
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

  if (error || !recording) {
    return (
      <Box sx={{ width: '100vw', height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: '#1a1a2e' }}>
        <Typography color="error">{error || 'Recording not found'}</Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ width: '100vw', height: '100vh', display: 'flex', flexDirection: 'column', overflow: 'hidden', bgcolor: '#1a1a2e' }}>
      {recording.format === 'asciicast' ? (
        <SshPlayer recordingId={recording.id} />
      ) : (
        <GuacPlayer recordingId={recording.id} />
      )}
    </Box>
  );
}
