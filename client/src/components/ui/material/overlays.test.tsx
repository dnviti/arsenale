import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Dialog } from './overlays';

describe('Material Dialog Adapter', () => {
  it('renders full-screen dialogs as flex columns with slide-up motion', () => {
    render(
      <Dialog open fullScreen onClose={() => {}}>
        <div>Fullscreen content</div>
      </Dialog>,
    );

    const dialog = screen.getByRole('dialog');
    expect(dialog.className).toContain('flex');
    expect(dialog.className).toContain('h-screen');
    expect(dialog.className).toContain('p-0');
    expect(dialog.className).toContain('data-[state=open]:animate-in');
  });
});
