import { fireEvent } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import RecoveryKeyConfirmDialog from './RecoveryKeyConfirmDialog';

describe('RecoveryKeyConfirmDialog', () => {
  it('requires the displayed recovery key before confirming', async () => {
    const onConfirmed = vi.fn();

    render(
      <RecoveryKeyConfirmDialog
        open
        recoveryKey="test-recovery-key"
        onConfirmed={onConfirmed}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    fireEvent.change(screen.getByPlaceholderText('Paste your recovery key here'), {
      target: { value: 'wrong-key' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Done' }));

    expect(screen.getByText('Key does not match')).toBeInTheDocument();
    expect(onConfirmed).not.toHaveBeenCalled();

    fireEvent.change(screen.getByPlaceholderText('Paste your recovery key here'), {
      target: { value: 'test-recovery-key' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Done' }));

    expect(onConfirmed).toHaveBeenCalledTimes(1);
  });
});
