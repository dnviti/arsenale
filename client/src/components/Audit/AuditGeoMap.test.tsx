import type { ReactNode } from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import AuditGeoMap from './AuditGeoMap';

const { getTenantGeoSummary } = vi.hoisted(() => ({
  getTenantGeoSummary: vi.fn(),
}));

const fakeMap = {
  fitBounds: vi.fn(),
  getZoom: vi.fn(() => 2),
  invalidateSize: vi.fn(),
  project: vi.fn((latLng: { lat: number; lng: number }) => ({ x: latLng.lat, y: latLng.lng })),
  setView: vi.fn(),
};

vi.mock('../../api/audit.api', async () => {
  const actual = await vi.importActual<typeof import('../../api/audit.api')>('../../api/audit.api');
  return {
    ...actual,
    getTenantGeoSummary,
  };
});

vi.mock('react-leaflet', () => ({
  CircleMarker: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  Popup: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  useMap: () => fakeMap,
  useMapEvents: () => fakeMap,
}));

vi.mock('./WorldMapCanvas', () => ({
  WorldMapCanvas: ({ children }: { children: ReactNode }) => <div data-testid="audit-map">{children}</div>,
}));

describe('AuditGeoMap', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    fakeMap.fitBounds.mockReset();
    fakeMap.getZoom.mockReturnValue(2);
    fakeMap.invalidateSize.mockReset();
    fakeMap.project.mockImplementation((latLng: { lat: number; lng: number }) => ({ x: latLng.lat, y: latLng.lng }));
    fakeMap.setView.mockReset();
  });

  it('keeps the map mounted when no geolocated events match the filters', async () => {
    getTenantGeoSummary.mockResolvedValue([]);

    render(<AuditGeoMap emptyMessage="No points yet" filters={{ search: 'ssh' }} />);

    expect(screen.getByTestId('audit-map')).toBeInTheDocument();

    await waitFor(() => {
      expect(getTenantGeoSummary).toHaveBeenCalledWith({ search: 'ssh' });
    });

    expect(await screen.findByText('No points yet')).toBeInTheDocument();
    expect(screen.getByText('Nearby points aggregate automatically as you zoom out')).toBeInTheDocument();
  });
});
