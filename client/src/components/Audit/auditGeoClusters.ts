import type { GeoSummaryPoint } from '../../api/audit.api';

export interface GeoCluster {
  lat: number;
  lng: number;
  count: number;
  lastSeen: string;
  countries: string[];
  cities: string[];
  locationCount: number;
}

interface ProjectedPoint {
  x: number;
  y: number;
}

type ProjectPoint = (point: Pick<GeoSummaryPoint, 'lat' | 'lng'>, zoom: number) => ProjectedPoint;

function getClusterCellSize(zoom: number): number {
  if (zoom >= 10) return 28;
  if (zoom >= 8) return 36;
  if (zoom >= 6) return 48;
  if (zoom >= 4) return 64;
  return 82;
}

export function clusterGeoSummaryPoints(
  points: GeoSummaryPoint[],
  zoom: number,
  projectPoint: ProjectPoint,
): GeoCluster[] {
  const cellSize = getClusterCellSize(zoom);
  const buckets = new Map<string, {
    count: number;
    countries: Set<string>;
    cities: Set<string>;
    lastSeen: string;
    locationCount: number;
    weightedLat: number;
    weightedLng: number;
  }>();

  for (const point of points) {
    if (!Number.isFinite(point.lat) || !Number.isFinite(point.lng)) {
      continue;
    }

    const projected = projectPoint(point, zoom);
    const bucketX = Math.floor(projected.x / cellSize);
    const bucketY = Math.floor(projected.y / cellSize);
    const bucketKey = `${bucketX}:${bucketY}`;
    const weight = Math.max(point.count, 1);
    const entry = buckets.get(bucketKey) ?? {
      count: 0,
      countries: new Set<string>(),
      cities: new Set<string>(),
      lastSeen: point.lastSeen,
      locationCount: 0,
      weightedLat: 0,
      weightedLng: 0,
    };

    entry.count += weight;
    entry.locationCount += 1;
    entry.weightedLat += point.lat * weight;
    entry.weightedLng += point.lng * weight;
    if (Date.parse(point.lastSeen) > Date.parse(entry.lastSeen)) {
      entry.lastSeen = point.lastSeen;
    }
    if (point.country.trim()) {
      entry.countries.add(point.country.trim());
    }
    if (point.city.trim()) {
      entry.cities.add(point.city.trim());
    }

    buckets.set(bucketKey, entry);
  }

  return Array.from(buckets.values())
    .map((entry) => ({
      lat: entry.count > 0 ? entry.weightedLat / entry.count : 0,
      lng: entry.count > 0 ? entry.weightedLng / entry.count : 0,
      count: entry.count,
      lastSeen: entry.lastSeen,
      countries: Array.from(entry.countries).sort(),
      cities: Array.from(entry.cities).sort(),
      locationCount: entry.locationCount,
    }))
    .sort((left, right) => right.count - left.count);
}
