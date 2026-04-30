import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import SessionsDialog from './SessionsDialog';

const controlledConsoleSpy = vi.fn();

vi.mock('./SessionsConsole', () => ({
  ControlledSessionsConsole: (props: unknown) => {
    controlledConsoleSpy(props);
    return <div>Sessions console body</div>;
  },
}));

describe('SessionsDialog', () => {
  it('renders the in-app overlay and forwards initial presets', () => {
    render(
      <SessionsDialog
        open
        onClose={() => undefined}
        initialState={{ status: ['CLOSED'], recorded: true }}
      />,
    );

    expect(screen.getByText('Sessions')).toBeInTheDocument();
    expect(screen.getByText('Sessions console body')).toBeInTheDocument();
    expect(controlledConsoleSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        layout: 'dialog',
        routeState: expect.objectContaining({
          status: ['CLOSED'],
          recorded: true,
        }),
      }),
    );
  });
});
