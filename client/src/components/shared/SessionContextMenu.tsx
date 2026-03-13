import { useState } from 'react';
import { Menu, MenuItem, Divider, ListItemIcon, ListItemText, Typography } from '@mui/material';
import {
  ContentCopy as CopyIcon,
  ContentPaste as PasteIcon,
  CameraAlt as ScreenshotIcon,
  Fullscreen as FullscreenIcon,
  FullscreenExit as FullscreenExitIcon,
  FolderOpen as FolderOpenIcon,
  PowerSettingsNew as DisconnectIcon,
  Keyboard as KeyboardIcon,
  ChevronRight as ChevronRightIcon,
} from '@mui/icons-material';
import type { ResolvedDlpPolicy } from '../../api/connections.api';

// X11 keysym constants for special key combos
const KEYSYMS = {
  CTRL_ALT_DEL: [0xFFE3, 0xFFE9, 0xFFFF],
  ALT_TAB: [0xFFE9, 0xFF09],
  ALT_F4: [0xFFE9, 0xFFC1],
  WINDOWS: [0xFFEB],
  PRINT_SCREEN: [0xFF61],
} as const;

interface SessionContextMenuProps {
  anchorPosition: { top: number; left: number } | null;
  onClose: () => void;
  protocol: 'RDP' | 'VNC' | 'SSH';
  dlpPolicy: ResolvedDlpPolicy | null;
  onCopy?: () => void;
  onPaste?: () => void;
  onScreenshot?: () => void;
  onSendKeys?: (keysyms: readonly number[]) => void;
  onFullscreenToggle?: () => void;
  isFullscreen?: boolean;
  onDisconnect?: () => void;
  onToggleDrive?: () => void;
  driveAvailable?: boolean;
  driveOpen?: boolean;
  onToggleSftp?: () => void;
  sftpAvailable?: boolean;
  sftpOpen?: boolean;
  container?: HTMLElement | null;
}

export default function SessionContextMenu({
  anchorPosition,
  onClose,
  protocol,
  dlpPolicy,
  onCopy,
  onPaste,
  onScreenshot,
  onSendKeys,
  onFullscreenToggle,
  isFullscreen = false,
  onDisconnect,
  onToggleDrive,
  driveAvailable = false,
  driveOpen = false,
  onToggleSftp,
  sftpAvailable = false,
  sftpOpen = false,
  container,
}: SessionContextMenuProps) {
  const [sendKeysAnchor, setSendKeysAnchor] = useState<HTMLElement | null>(null);

  const isGuac = protocol === 'RDP' || protocol === 'VNC';
  const open = anchorPosition !== null;

  const handleClose = () => {
    setSendKeysAnchor(null);
    onClose();
  };

  const handleAction = (action?: () => void) => {
    if (action) action();
    handleClose();
  };

  const handleSendKeys = (keysyms: readonly number[]) => {
    if (onSendKeys) onSendKeys(keysyms);
    handleClose();
  };

  return (
    <>
      <Menu
        open={open}
        onClose={handleClose}
        anchorReference="anchorPosition"
        anchorPosition={anchorPosition ?? undefined}
        slotProps={{
          paper: {
            sx: { minWidth: 200 },
          },
        }}
        {...(container ? { container, disablePortal: true } : {})}
      >
        {/* Clipboard operations */}
        {onCopy !== undefined && (
          <MenuItem
            onClick={() => handleAction(onCopy)}
            disabled={!!dlpPolicy?.disableCopy || !navigator.clipboard?.writeText}
          >
            <ListItemIcon><CopyIcon fontSize="small" /></ListItemIcon>
            <ListItemText>Copy</ListItemText>
            {protocol === 'SSH' && (
              <Typography variant="body2" sx={{ color: 'text.secondary', ml: 2 }}>
                Ctrl+Shift+C
              </Typography>
            )}
          </MenuItem>
        )}
        {onPaste !== undefined && (
          <MenuItem
            onClick={() => handleAction(onPaste)}
            disabled={!!dlpPolicy?.disablePaste || !navigator.clipboard?.readText}
          >
            <ListItemIcon><PasteIcon fontSize="small" /></ListItemIcon>
            <ListItemText>Paste</ListItemText>
            {protocol === 'SSH' && (
              <Typography variant="body2" sx={{ color: 'text.secondary', ml: 2 }}>
                Ctrl+Shift+V
              </Typography>
            )}
          </MenuItem>
        )}
        {(onCopy !== undefined || onPaste !== undefined) && <Divider />}

        {/* Special keys — RDP/VNC only */}
        {isGuac && onSendKeys && (
          <>
            <MenuItem onClick={() => handleSendKeys(KEYSYMS.CTRL_ALT_DEL)}>
              <ListItemIcon><KeyboardIcon fontSize="small" /></ListItemIcon>
              <ListItemText>Ctrl+Alt+Del</ListItemText>
            </MenuItem>
            <MenuItem
              onClick={(e) => setSendKeysAnchor(e.currentTarget)}
            >
              <ListItemIcon><KeyboardIcon fontSize="small" /></ListItemIcon>
              <ListItemText>Send Keys</ListItemText>
              <ChevronRightIcon fontSize="small" sx={{ color: 'text.secondary' }} />
            </MenuItem>
            <Divider />
          </>
        )}

        {/* Screenshot — RDP/VNC only */}
        {isGuac && onScreenshot && (
          <MenuItem onClick={() => handleAction(onScreenshot)}>
            <ListItemIcon><ScreenshotIcon fontSize="small" /></ListItemIcon>
            <ListItemText>Screenshot</ListItemText>
          </MenuItem>
        )}

        {/* SFTP — SSH only */}
        {protocol === 'SSH' && sftpAvailable && onToggleSftp && (
          <MenuItem onClick={() => handleAction(onToggleSftp)}>
            <ListItemIcon><FolderOpenIcon fontSize="small" /></ListItemIcon>
            <ListItemText>{sftpOpen ? 'Close SFTP Browser' : 'SFTP File Browser'}</ListItemText>
          </MenuItem>
        )}

        {/* Fullscreen */}
        {onFullscreenToggle && (
          <MenuItem onClick={() => handleAction(onFullscreenToggle)}>
            <ListItemIcon>
              {isFullscreen ? <FullscreenExitIcon fontSize="small" /> : <FullscreenIcon fontSize="small" />}
            </ListItemIcon>
            <ListItemText>{isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}</ListItemText>
          </MenuItem>
        )}

        {/* Shared Drive — RDP only */}
        {protocol === 'RDP' && driveAvailable && onToggleDrive && (
          <MenuItem onClick={() => handleAction(onToggleDrive)}>
            <ListItemIcon><FolderOpenIcon fontSize="small" /></ListItemIcon>
            <ListItemText>{driveOpen ? 'Close Shared Drive' : 'Shared Drive'}</ListItemText>
          </MenuItem>
        )}

        <Divider />

        {/* Disconnect */}
        {onDisconnect && (
          <MenuItem onClick={() => handleAction(onDisconnect)}>
            <ListItemIcon><DisconnectIcon fontSize="small" sx={{ color: 'error.main' }} /></ListItemIcon>
            <ListItemText sx={{ color: 'error.main' }}>Disconnect</ListItemText>
          </MenuItem>
        )}
      </Menu>

      {/* Send Keys submenu */}
      <Menu
        open={!!sendKeysAnchor}
        anchorEl={sendKeysAnchor}
        onClose={() => setSendKeysAnchor(null)}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'left' }}
        {...(container ? { container, disablePortal: true } : {})}
      >
        <MenuItem onClick={() => handleSendKeys(KEYSYMS.ALT_TAB)}>
          <ListItemText>Alt+Tab</ListItemText>
        </MenuItem>
        <MenuItem onClick={() => handleSendKeys(KEYSYMS.ALT_F4)}>
          <ListItemText>Alt+F4</ListItemText>
        </MenuItem>
        <MenuItem onClick={() => handleSendKeys(KEYSYMS.WINDOWS)}>
          <ListItemText>Windows Key</ListItemText>
        </MenuItem>
        <MenuItem onClick={() => handleSendKeys(KEYSYMS.PRINT_SCREEN)}>
          <ListItemText>PrintScreen</ListItemText>
        </MenuItem>
      </Menu>
    </>
  );
}
