import { fireEvent } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ConnectionData } from '@/api/connections.api';
import { ConnectionItem } from './treeHelpers';

const { openTab } = vi.hoisted(() => ({
  openTab: vi.fn(),
}));

vi.mock('@/store/tabsStore', () => ({
  useTabsStore: (selector: (state: { openTab: typeof openTab }) => unknown) => selector({ openTab }),
}));

vi.mock('@/api/rdGateway.api', () => ({
  downloadRdpFile: vi.fn(),
}));

vi.mock('@/utils/openConnectionWindow', () => ({
  openConnectionWindow: vi.fn(),
}));

vi.mock('@dnd-kit/core', () => ({
  useDraggable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: vi.fn(),
    transform: null,
    isDragging: false,
  }),
  useDroppable: () => ({
    setNodeRef: vi.fn(),
    isOver: false,
  }),
}));

vi.mock('@dnd-kit/utilities', () => ({
  CSS: {
    Translate: {
      toString: () => '',
    },
  },
}));

vi.mock('./sidebarUi', async () => {
  const actual = await vi.importActual<typeof import('./sidebarUi')>('./sidebarUi');
  return {
    ...actual,
    SidebarContextMenu: () => null,
  };
});

function createConnection(overrides: Partial<ConnectionData> = {}): ConnectionData {
  return {
    id: 'connection-1',
    name: 'Demo Connection',
    type: 'DATABASE',
    host: 'db.example.com',
    port: 5432,
    description: '',
    enableDrive: false,
    isFavorite: false,
    isOwner: true,
    defaultCredentialMode: null,
    ...overrides,
  } as ConnectionData;
}

describe('ConnectionItem', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('keeps the quick-connect button separate from the favorite action', () => {
    const connection = createConnection();
    const onToggleFavorite = vi.fn();
    const { container } = render(
      <ConnectionItem
        conn={connection}
        depth={0}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
        onMove={vi.fn()}
        onShare={vi.fn()}
        onConnectAs={vi.fn()}
        onToggleFavorite={onToggleFavorite}
      />,
    );

    expect(container.querySelectorAll('button button')).toHaveLength(0);

    fireEvent.click(screen.getByRole('button', { name: 'Add to favorites' }));
    expect(onToggleFavorite).toHaveBeenCalledWith(connection);
    expect(openTab).not.toHaveBeenCalled();
  });

  it('opens the tab when the main connection button is double-clicked', () => {
    const connection = createConnection();
    render(
      <ConnectionItem
        conn={connection}
        depth={0}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
        onMove={vi.fn()}
        onShare={vi.fn()}
        onConnectAs={vi.fn()}
        onToggleFavorite={vi.fn()}
      />,
    );

    const mainButton = screen.getByText('Demo Connection').closest('button');
    expect(mainButton).not.toBeNull();

    fireEvent.doubleClick(mainButton!);
    expect(openTab).toHaveBeenCalledWith(connection);
  });
});
