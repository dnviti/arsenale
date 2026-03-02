import { useState, useEffect, useCallback } from 'react';
import {
  Box, Typography, List, ListItem, ListItemText, Button,
  CircularProgress, Dialog, DialogTitle, DialogContent,
  DialogContentText, DialogActions,
} from '@mui/material';
import { listVersions, restoreVersion } from '../../api/secrets.api';
import type { SecretVersion } from '../../api/secrets.api';

interface SecretVersionHistoryProps {
  secretId: string;
  currentVersion: number;
  onRestore: () => void;
}

export default function SecretVersionHistory({
  secretId,
  currentVersion,
  onRestore,
}: SecretVersionHistoryProps) {
  const [versions, setVersions] = useState<SecretVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [restoreTarget, setRestoreTarget] = useState<number | null>(null);
  const [restoring, setRestoring] = useState(false);

  const loadVersions = useCallback(async () => {
    setLoading(true);
    try {
      const data = await listVersions(secretId);
      setVersions(data);
    } catch {
      // silently fail
    } finally {
      setLoading(false);
    }
  }, [secretId]);

  useEffect(() => {
    loadVersions();
  }, [loadVersions]);

  const handleRestore = async (version: number) => {
    setRestoring(true);
    try {
      await restoreVersion(secretId, version);
      setRestoreTarget(null);
      await loadVersions();
      onRestore();
    } catch {
      // silently fail
    } finally {
      setRestoring(false);
    }
  };

  const formatDate = (iso: string) => {
    const d = new Date(iso);
    return d.toLocaleDateString(undefined, {
      month: 'short', day: 'numeric', year: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
        <CircularProgress size={20} />
      </Box>
    );
  }

  if (versions.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ py: 1 }}>
        No version history available.
      </Typography>
    );
  }

  return (
    <Box>
      <List dense disablePadding>
        {versions.map((v) => (
          <ListItem
            key={v.id}
            secondaryAction={
              v.version !== currentVersion ? (
                <Button size="small" onClick={() => setRestoreTarget(v.version)}>
                  Restore
                </Button>
              ) : undefined
            }
          >
            <ListItemText
              primary={
                <Typography variant="body2">
                  Version {v.version}
                  {v.version === currentVersion && (
                    <Typography component="span" variant="caption" color="primary" sx={{ ml: 1 }}>
                      (current)
                    </Typography>
                  )}
                </Typography>
              }
              secondary={
                <>
                  <Typography variant="caption" color="text.secondary">
                    {v.changer?.username || v.changer?.email || 'Unknown'} — {formatDate(v.createdAt)}
                  </Typography>
                  {v.changeNote && (
                    <Typography variant="caption" display="block" color="text.secondary">
                      {v.changeNote}
                    </Typography>
                  )}
                </>
              }
            />
          </ListItem>
        ))}
      </List>

      <Dialog open={restoreTarget !== null} onClose={() => setRestoreTarget(null)}>
        <DialogTitle>Restore Version</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Restore to version {restoreTarget}? This will create a new version with the restored data.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRestoreTarget(null)}>Cancel</Button>
          <Button
            onClick={() => handleRestore(restoreTarget!)}
            variant="contained"
            disabled={restoring}
          >
            {restoring ? 'Restoring...' : 'Restore'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
