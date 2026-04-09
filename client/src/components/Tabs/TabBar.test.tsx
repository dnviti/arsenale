import { fireEvent } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import TabBar from './TabBar';

const { closeTab, setActiveTab, tabStoreState } = vi.hoisted(() => ({
  closeTab: vi.fn(),
  setActiveTab: vi.fn(),
  tabStoreState: {
    tabs: [] as Array<{
      id: string;
      connection: { id: string; name: string; type: string };
    }>,
    activeTabId: null as string | null,
  },
}));

vi.mock('@/store/tabsStore', () => ({
  useTabsStore: (selector: (state: typeof tabStoreState & {
    closeTab: typeof closeTab;
    setActiveTab: typeof setActiveTab;
  }) => unknown) => selector({
    ...tabStoreState,
    closeTab,
    setActiveTab,
  }),
}));

describe('TabBar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    tabStoreState.tabs = [
      {
        id: 'tab-1',
        connection: {
          id: 'connection-1',
          name: 'Demo Connection',
          type: 'DATABASE',
        },
      },
    ];
    tabStoreState.activeTabId = 'tab-1';
  });

  it('renders separate tab and close actions with accessible labels', () => {
    render(<TabBar />);

    fireEvent.click(screen.getByRole('button', { name: 'Demo Connection' }));
    expect(setActiveTab).toHaveBeenCalledWith('tab-1');

    fireEvent.click(screen.getByRole('button', { name: 'Close Demo Connection' }));
    expect(closeTab).toHaveBeenCalledWith('tab-1');
  });
});
