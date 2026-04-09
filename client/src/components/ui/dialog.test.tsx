import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Dialog, DialogContent } from './dialog';

describe('DialogContent', () => {
  it('uses shadcn-style zoom animations by default', () => {
    render(
      <Dialog open>
        <DialogContent>Dialog body</DialogContent>
      </Dialog>,
    );

    const dialog = screen.getByRole('dialog');
    expect(dialog.className).toContain('p-6');
    expect(dialog.className).toContain('data-[state=open]:zoom-in-95');
    expect(dialog.className).toContain('data-[state=closed]:zoom-out-95');
  });
});
