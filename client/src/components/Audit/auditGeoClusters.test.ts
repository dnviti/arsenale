import { describe, expect, it } from 'vitest';
import { clusterGeoSummaryPoints } from './auditGeoClusters';

describe('clusterGeoSummaryPoints', () => {
  it('merges nearby points when zoomed out and splits them as the zoom increases', () => {
    const points = [
      {
        lat: 10,
        lng: 10,
        city: 'New York',
        country: 'United States',
        count: 2,
        lastSeen: '2026-04-12T00:00:00.000Z',
      },
      {
        lat: 60,
        lng: 60,
        city: 'Boston',
        country: 'United States',
        count: 3,
        lastSeen: '2026-04-12T01:00:00.000Z',
      },
      {
        lat: 220,
        lng: 220,
        city: 'Paris',
        country: 'France',
        count: 1,
        lastSeen: '2026-04-11T00:00:00.000Z',
      },
    ];

    const projectPoint = ({ lat, lng }: { lat: number; lng: number }) => ({ x: lat, y: lng });

    const lowZoomClusters = clusterGeoSummaryPoints(points, 2, projectPoint);
    expect(lowZoomClusters).toHaveLength(2);
    expect(lowZoomClusters[0]).toMatchObject({
      count: 5,
      countries: ['United States'],
      cities: ['Boston', 'New York'],
      locationCount: 2,
      lastSeen: '2026-04-12T01:00:00.000Z',
    });

    const highZoomClusters = clusterGeoSummaryPoints(points, 10, projectPoint);
    expect(highZoomClusters).toHaveLength(3);
    expect(highZoomClusters.map((cluster) => cluster.count)).toEqual([3, 2, 1]);
  });
});
