import { fireEvent } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import SyncPreviewDialog from './SyncPreviewDialog';

describe('SyncPreviewDialog', () => {
  it('renders grouped actions and confirms actionable plans', async () => {
    const onConfirm = vi.fn();

    render(
      <SyncPreviewDialog
        open
        onClose={vi.fn()}
        onConfirm={onConfirm}
        confirming={false}
        plan={{
          toCreate: [
            {
              externalId: 'create-1',
              name: 'web-1',
              host: '10.0.0.10',
              port: 22,
              protocol: 'SSH',
            },
          ],
          toUpdate: [
            {
              connectionId: 'connection-1',
              device: {
                externalId: 'update-1',
                name: 'db-1',
                host: '10.0.0.11',
                port: 5432,
                protocol: 'POSTGRESQL',
              },
              changes: ['host', 'port'],
            },
          ],
          toSkip: [],
          errors: [],
        }}
      />,
    );

    expect(await screen.findByText('Create 1')).toBeInTheDocument();
    expect(screen.getByText('Update 1')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Confirm Import' }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });
});
