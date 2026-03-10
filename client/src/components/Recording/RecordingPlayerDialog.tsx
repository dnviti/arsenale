import { useState } from 'react';
import {
  Dialog, DialogTitle, DialogContent, Box, IconButton, Typography,
  Chip, Tooltip,
} from '@mui/material';
import {
  Close as CloseIcon,
  Fullscreen as FullscreenIcon,
  FullscreenExit as FullscreenExitIcon,
  OpenInNew as OpenInNewIcon,
} from '@mui/icons-material';
import type { Recording } from '../../api/recordings.api';
import { openRecordingWindow } from '../../utils/openRecordingWindow';
import GuacPlayer from './GuacPlayer';
import SshPlayer from './SshPlayer';

interface RecordingPlayerDialogProps {
  open: boolean;
  onClose: () => void;
  recording: Recording | null;
}

const protocolColor = (protocol: string) => {
  switch (protocol) {
    case 'SSH': return 'success';
    case 'RDP': return 'primary';
    case 'VNC': return 'warning';
    default: return 'default';
  }
};

export default function RecordingPlayerDialog({
  open, onClose, recording,
}: RecordingPlayerDialogProps) {
  const [fullScreen, setFullScreen] = useState(false);

  if (!recording) return null;

  const isSsh = recording.format === 'asciicast';
  // SSH width/height are cols/rows, not pixels — convert with approximate char size
  const contentWidth = isSsh
    ? Math.max((recording.width || 80) * 14, 720)
    : (recording.width || 1024);
  const contentHeight = isSsh
    ? Math.max((recording.height || 24) * 9, 432)
    : (recording.height || 768);

  const handleOpenInNewWindow = () => {
    openRecordingWindow(recording.id, recording.width, recording.height);
    onClose();
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth={false}
      fullScreen={fullScreen}
      slotProps={fullScreen ? undefined : {
        paper: {
          sx: {
            width: Math.min(contentWidth + 48, window.innerWidth - 64),
            height: Math.min(contentHeight + 140, window.innerHeight - 64),
            maxWidth: '100vw',
            maxHeight: '100vh',
            resize: 'both',
            overflow: 'hidden',
          },
        },
      }}
    >
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1, pr: 1, py: 1 }}>
        <Typography variant="subtitle1" component="span" sx={{ flex: 1 }} noWrap>
          {recording.connection.name}
        </Typography>
        <Chip
          label={recording.protocol}
          size="small"
          color={protocolColor(recording.protocol) as 'success' | 'primary' | 'warning' | 'default'}
        />
        <Tooltip title="Open in new window">
          <IconButton onClick={handleOpenInNewWindow} size="small">
            <OpenInNewIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title={fullScreen ? 'Exit full screen' : 'Full screen'}>
          <IconButton onClick={() => setFullScreen((v) => !v)} size="small">
            {fullScreen ? <FullscreenExitIcon fontSize="small" /> : <FullscreenIcon fontSize="small" />}
          </IconButton>
        </Tooltip>
        <IconButton onClick={onClose} size="small" edge="end">
          <CloseIcon fontSize="small" />
        </IconButton>
      </DialogTitle>
      <DialogContent dividers sx={{ p: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <Box
          sx={{
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            minHeight: 0,
          }}
        >
          {isSsh ? (
            <SshPlayer recordingId={recording.id} />
          ) : (
            <GuacPlayer recordingId={recording.id} />
          )}
        </Box>
      </DialogContent>
    </Dialog>
  );
}
