import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import SessionContextMenu from './SessionContextMenu';

describe('SessionContextMenu - SSH File Browser', () => {
  const defaultProps = {
    anchorPosition: { top: 100, left: 100 },
    onClose: vi.fn(),
    protocol: 'SSH' as const,
    dlpPolicy: null,
    onCopy: vi.fn(),
    onPaste: vi.fn(),
    onDisconnect: vi.fn(),
  };

  it('shows Open File Browser label instead of SFTP when sftpAvailable is true', () => {
    render(
      <SessionContextMenu
        {...defaultProps}
        sftpAvailable={true}
        sftpOpen={false}
        onToggleSftp={vi.fn()}
      />,
    );

    expect(screen.getByText('Open File Browser')).toBeInTheDocument();
    expect(screen.queryByText('SFTP File Browser')).not.toBeInTheDocument();
  });

  it('shows Close File Browser label when sftpOpen is true', () => {
    render(
      <SessionContextMenu
        {...defaultProps}
        sftpAvailable={true}
        sftpOpen={true}
        onToggleSftp={vi.fn()}
      />,
    );

    expect(screen.getByText('Close File Browser')).toBeInTheDocument();
    expect(screen.queryByText('Close SFTP Browser')).not.toBeInTheDocument();
  });

  it('shows Open Shared Drive label for RDP when drive sharing is available', () => {
    render(
      <SessionContextMenu
        {...defaultProps}
        protocol="RDP"
        driveAvailable={true}
        driveOpen={false}
        onToggleDrive={vi.fn()}
      />,
    );

    expect(screen.getByText('Open Shared Drive')).toBeInTheDocument();
  });

  it('does not show file browser menu item when sftpAvailable is false', () => {
    render(
      <SessionContextMenu
        {...defaultProps}
        sftpAvailable={false}
        sftpOpen={false}
        onToggleSftp={vi.fn()}
      />,
    );

    expect(screen.queryByText(/File Browser/)).not.toBeInTheDocument();
    expect(screen.queryByText(/SFTP/)).not.toBeInTheDocument();
  });

  it('does not show file browser menu item for RDP protocol even when sftpAvailable is true', () => {
    render(
      <SessionContextMenu
        {...defaultProps}
        protocol="RDP"
        sftpAvailable={true}
        sftpOpen={false}
        onToggleSftp={vi.fn()}
      />,
    );

    expect(screen.queryByText(/File Browser/)).not.toBeInTheDocument();
    expect(screen.queryByText(/SFTP/)).not.toBeInTheDocument();
  });
});
