import { useState, useEffect, useCallback } from 'react';
import {
  Dialog, AppBar, Toolbar, IconButton, Typography, Box,
  Table, TableHead, TableRow, TableCell, TableBody, Chip,
  Select, MenuItem, FormControl, InputLabel, Button, Tooltip,
  DialogTitle, DialogContent, DialogActions, CircularProgress,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import DeleteIcon from '@mui/icons-material/Delete';
import DownloadIcon from '@mui/icons-material/Download';
import MovieIcon from '@mui/icons-material/Movie';
import { listRecordings, deleteRecording, exportRecordingVideo } from '../../api/recordings.api';
import type { Recording } from '../../api/recordings.api';
import api from '../../api/client';
import RecordingPlayerDialog from './RecordingPlayerDialog';
import { SlideUp } from '../common/SlideUp';

interface RecordingsDialogProps {
  open: boolean;
  onClose: () => void;
}

export default function RecordingsDialog({ open, onClose }: RecordingsDialogProps) {
  const [recordings, setRecordings] = useState<Recording[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [protocolFilter, setProtocolFilter] = useState<string>('');
  const [page, setPage] = useState(0);
  const [playingRecording, setPlayingRecording] = useState<Recording | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Recording | null>(null);
  const [convertingIds, setConvertingIds] = useState<Set<string>>(new Set());
  const limit = 25;

  const fetchRecordings = useCallback(async () => {
    setLoading(true);
    try {
      const result = await listRecordings({
        protocol: protocolFilter || undefined,
        status: 'COMPLETE',
        limit,
        offset: page * limit,
      });
      setRecordings(result.recordings);
      setTotal(result.total);
    } catch {
      // silently handle
    } finally {
      setLoading(false);
    }
  }, [protocolFilter, page]);

  useEffect(() => {
    if (open) fetchRecordings();
  }, [open, fetchRecordings]);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await deleteRecording(deleteTarget.id);
      setDeleteTarget(null);
      fetchRecordings();
    } catch {
      // silently handle
    }
  };

  const formatDuration = (seconds: number | null) => {
    if (seconds === null) return '-';
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}:${s.toString().padStart(2, '0')}`;
  };

  const formatSize = (bytes: number | null) => {
    if (bytes === null) return '-';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  };

  const handleDownload = async (rec: Recording) => {
    const { data } = await api.get(`/recordings/${rec.id}/stream`, { responseType: 'blob' });
    const url = URL.createObjectURL(data);
    const a = document.createElement('a');
    a.href = url;
    a.download = `recording-${rec.id}.${rec.format === 'asciicast' ? 'cast' : rec.format}`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleDownloadVideo = async (rec: Recording) => {
    setConvertingIds((prev) => new Set(prev).add(rec.id));
    try {
      const blob = await exportRecordingVideo(rec.id);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `recording-${rec.id}.m4v`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      // silently handle
    } finally {
      setConvertingIds((prev) => {
        const next = new Set(prev);
        next.delete(rec.id);
        return next;
      });
    }
  };

  const protocolColor = (protocol: string) => {
    switch (protocol) {
      case 'SSH': return 'success';
      case 'RDP': return 'primary';
      case 'VNC': return 'warning';
      default: return 'default';
    }
  };

  return (
    <>
      <Dialog fullScreen open={open} onClose={onClose} TransitionComponent={SlideUp}>
        <AppBar position="static" sx={{ position: 'relative' }}>
          <Toolbar variant="dense">
            <IconButton edge="start" color="inherit" onClick={onClose}>
              <CloseIcon />
            </IconButton>
            <Typography sx={{ ml: 2, flex: 1 }} variant="h6">
              Session Recordings
            </Typography>
          </Toolbar>
        </AppBar>

        <Box sx={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column', p: 2 }}>
          {/* Filters */}
          <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
            <FormControl size="small" sx={{ minWidth: 120 }}>
              <InputLabel>Protocol</InputLabel>
              <Select
                value={protocolFilter}
                onChange={(e) => { setProtocolFilter(e.target.value); setPage(0); }}
                label="Protocol"
              >
                <MenuItem value="">All</MenuItem>
                <MenuItem value="SSH">SSH</MenuItem>
                <MenuItem value="RDP">RDP</MenuItem>
                <MenuItem value="VNC">VNC</MenuItem>
              </Select>
            </FormControl>
            <Typography variant="body2" sx={{ alignSelf: 'center', color: 'text.secondary' }}>
              {total} recording{total !== 1 ? 's' : ''}
            </Typography>
          </Box>

          {/* Recordings table */}
          <Box sx={{ flex: 1, overflow: 'auto' }}>
            {loading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
                <CircularProgress />
              </Box>
            ) : recordings.length === 0 ? (
              <Typography color="text.secondary" sx={{ textAlign: 'center', mt: 4 }}>
                No recordings found. Enable session recording in your environment configuration.
              </Typography>
            ) : (
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    <TableCell>Connection</TableCell>
                    <TableCell>Protocol</TableCell>
                    <TableCell>User</TableCell>
                    <TableCell>Duration</TableCell>
                    <TableCell>Size</TableCell>
                    <TableCell>Date</TableCell>
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {recordings.map((rec) => (
                    <TableRow key={rec.id} hover>
                      <TableCell>{rec.connection.name}</TableCell>
                      <TableCell>
                        <Chip
                          label={rec.protocol}
                          size="small"
                          color={protocolColor(rec.protocol) as 'success' | 'primary' | 'warning' | 'default'}
                        />
                      </TableCell>
                      <TableCell>{rec.user?.username || rec.user?.email || '-'}</TableCell>
                      <TableCell>{formatDuration(rec.duration)}</TableCell>
                      <TableCell>{formatSize(rec.fileSize)}</TableCell>
                      <TableCell>{new Date(rec.createdAt).toLocaleString()}</TableCell>
                      <TableCell align="right">
                        <Tooltip title="Play">
                          <IconButton size="small" onClick={() => setPlayingRecording(rec)}>
                            <PlayArrowIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        <Tooltip title="Download">
                          <IconButton size="small" onClick={() => handleDownload(rec)}>
                            <DownloadIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                        {(rec.format === 'guac' || rec.format === 'asciicast') && (
                          <Tooltip title="Download MP4">
                            <span>
                              <IconButton
                                size="small"
                                onClick={() => handleDownloadVideo(rec)}
                                disabled={convertingIds.has(rec.id)}
                              >
                                {convertingIds.has(rec.id) ? (
                                  <CircularProgress size={18} />
                                ) : (
                                  <MovieIcon fontSize="small" />
                                )}
                              </IconButton>
                            </span>
                          </Tooltip>
                        )}
                        <Tooltip title="Delete">
                          <IconButton size="small" onClick={() => setDeleteTarget(rec)}>
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Box>

          {/* Pagination */}
          {total > limit && (
            <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2, gap: 1 }}>
              <Button size="small" disabled={page === 0} onClick={() => setPage((p) => p - 1)}>
                Previous
              </Button>
              <Typography variant="body2" sx={{ alignSelf: 'center' }}>
                Page {page + 1} of {Math.ceil(total / limit)}
              </Typography>
              <Button size="small" disabled={(page + 1) * limit >= total} onClick={() => setPage((p) => p + 1)}>
                Next
              </Button>
            </Box>
          )}
        </Box>
      </Dialog>

      {/* Player popup dialog */}
      <RecordingPlayerDialog
        open={!!playingRecording}
        onClose={() => setPlayingRecording(null)}
        recording={playingRecording}
      />

      {/* Delete confirmation dialog */}
      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete Recording</DialogTitle>
        <DialogContent>
          <Typography>
            Delete the recording for &quot;{deleteTarget?.connection.name}&quot;? This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button color="error" onClick={handleDelete}>Delete</Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
