import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { SidebarMenuSkeleton } from './sidebar';

function getSkeletonWidths(container: HTMLElement) {
  return Array.from(
    container.querySelectorAll<HTMLElement>('[data-sidebar="menu-skeleton-text"]'),
  ).map((element) => element.style.getPropertyValue('--skeleton-width'));
}

function expectWidthInRange(width: string) {
  const numericWidth = Number.parseInt(width, 10);

  expect(numericWidth).toBeGreaterThanOrEqual(50);
  expect(numericWidth).toBeLessThanOrEqual(89);
}

describe('SidebarMenuSkeleton', () => {
  it('keeps the generated width stable across rerenders', () => {
    const { container, rerender } = render(<SidebarMenuSkeleton />);

    const [initialWidth] = getSkeletonWidths(container);
    expect(initialWidth).toBeTruthy();
    expectWidthInRange(initialWidth);

    rerender(<SidebarMenuSkeleton className="rerendered" />);

    const [rerenderedWidth] = getSkeletonWidths(container);
    expect(rerenderedWidth).toBe(initialWidth);
    expectWidthInRange(rerenderedWidth);
  });

  it('keeps widths varied between sibling skeletons while staying in range', () => {
    const { container } = render(
      <>
        <SidebarMenuSkeleton />
        <SidebarMenuSkeleton />
        <SidebarMenuSkeleton showIcon />
      </>
    );

    const widths = getSkeletonWidths(container);

    expect(widths).toHaveLength(3);
    widths.forEach(expectWidthInRange);
    expect(new Set(widths).size).toBeGreaterThan(1);
  });
});
