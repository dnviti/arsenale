import { useState } from 'react';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
} from '@/components/ui/dropdown-menu';
import {
  Copy,
  ClipboardPaste,
  Camera,
  Maximize,
  Minimize,
  FolderOpen,
  Power,
  Keyboard,
} from 'lucide-react';
import type { ResolvedDlpPolicy } from '../../api/connections.api';
import { KEYSYMS } from '../../constants/keysyms';

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
}: SessionContextMenuProps) {
  const [menuOpen, setMenuOpen] = useState(false);

  const isGuac = protocol === 'RDP' || protocol === 'VNC';
  const isVisible = anchorPosition !== null;

  // Sync external open state
  if (isVisible && !menuOpen) {
    // Will be set open by the trigger
  }

  const handleClose = () => {
    setMenuOpen(false);
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
    <DropdownMenu
      open={isVisible}
      onOpenChange={(v) => { if (!v) handleClose(); }}
    >
      <DropdownMenuTrigger asChild>
        <div
          className="fixed w-0 h-0"
          style={anchorPosition ? { top: anchorPosition.top, left: anchorPosition.left } : { top: -9999, left: -9999 }}
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent className="min-w-[200px]" side="bottom" align="start">
        {/* Clipboard operations */}
        {onCopy !== undefined && (
          <DropdownMenuItem
            onClick={() => handleAction(onCopy)}
            disabled={!!dlpPolicy?.disableCopy || !navigator.clipboard?.writeText}
          >
            <Copy className="h-4 w-4 mr-2" />
            Copy
            {protocol === 'SSH' && (
              <span className="ml-auto text-xs text-muted-foreground">Ctrl+Shift+C</span>
            )}
          </DropdownMenuItem>
        )}
        {onPaste !== undefined && (
          <DropdownMenuItem
            onClick={() => handleAction(onPaste)}
            disabled={!!dlpPolicy?.disablePaste || !navigator.clipboard?.readText}
          >
            <ClipboardPaste className="h-4 w-4 mr-2" />
            Paste
            {protocol === 'SSH' && (
              <span className="ml-auto text-xs text-muted-foreground">Ctrl+Shift+V</span>
            )}
          </DropdownMenuItem>
        )}
        {(onCopy !== undefined || onPaste !== undefined) && <DropdownMenuSeparator />}

        {/* Special keys — RDP/VNC only */}
        {isGuac && onSendKeys && (
          <>
            <DropdownMenuItem onClick={() => handleSendKeys(KEYSYMS.CTRL_ALT_DEL)}>
              <Keyboard className="h-4 w-4 mr-2" />
              Ctrl+Alt+Del
            </DropdownMenuItem>
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <Keyboard className="h-4 w-4 mr-2" />
                Send Keys
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuItem onClick={() => handleSendKeys(KEYSYMS.ALT_TAB)}>
                  Alt+Tab
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => handleSendKeys(KEYSYMS.ALT_F4)}>
                  Alt+F4
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => handleSendKeys(KEYSYMS.WINDOWS)}>
                  Windows Key
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => handleSendKeys(KEYSYMS.PRINT_SCREEN)}>
                  PrintScreen
                </DropdownMenuItem>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
            <DropdownMenuSeparator />
          </>
        )}

        {/* Screenshot — RDP/VNC only */}
        {isGuac && onScreenshot && (
          <DropdownMenuItem onClick={() => handleAction(onScreenshot)}>
            <Camera className="h-4 w-4 mr-2" />
            Screenshot
          </DropdownMenuItem>
        )}

        {/* SFTP — SSH only */}
        {protocol === 'SSH' && sftpAvailable && onToggleSftp && (
          <DropdownMenuItem onClick={() => handleAction(onToggleSftp)}>
            <FolderOpen className="h-4 w-4 mr-2" />
            {sftpOpen ? 'Close SFTP Browser' : 'SFTP File Browser'}
          </DropdownMenuItem>
        )}

        {/* Fullscreen */}
        {onFullscreenToggle && (
          <DropdownMenuItem onClick={() => handleAction(onFullscreenToggle)}>
            {isFullscreen ? <Minimize className="h-4 w-4 mr-2" /> : <Maximize className="h-4 w-4 mr-2" />}
            {isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
          </DropdownMenuItem>
        )}

        {/* Shared Drive — RDP only */}
        {protocol === 'RDP' && driveAvailable && onToggleDrive && (
          <DropdownMenuItem onClick={() => handleAction(onToggleDrive)}>
            <FolderOpen className="h-4 w-4 mr-2" />
            {driveOpen ? 'Close Shared Drive' : 'Shared Drive'}
          </DropdownMenuItem>
        )}

        <DropdownMenuSeparator />

        {/* Disconnect */}
        {onDisconnect && (
          <DropdownMenuItem onClick={() => handleAction(onDisconnect)} className="text-red-400 focus:text-red-400">
            <Power className="h-4 w-4 mr-2" />
            Disconnect
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
