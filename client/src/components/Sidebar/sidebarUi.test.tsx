import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { SidebarIconButton } from './sidebarUi';

describe('SidebarIconButton', () => {
  it('uses the title as the accessible label when no aria-label is provided', () => {
    render(
      <SidebarIconButton title="New Connection">
        <span aria-hidden="true">+</span>
      </SidebarIconButton>,
    );

    expect(screen.getByRole('button', { name: 'New Connection' })).toBeInTheDocument();
  });
});
