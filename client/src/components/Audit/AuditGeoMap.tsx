import { useEffect, useMemo, useState } from 'react';
import { Loader2 } from 'lucide-react';
import { CircleMarker, Popup, useMap, useMapEvents } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import {
  getTenantGeoSummary,
  type GeoSummaryPoint,
  type TenantGeoSummaryParams,
} from '../../api/audit.api';
import { ensureLeafletDefaultIcon } from '../../lib/leafletMarkerIcon';
import { clusterGeoSummaryPoints } from './auditGeoClusters';
import { WorldMapCanvas } from './WorldMapCanvas';

ensureLeafletDefaultIcon();

interface AuditGeoMapProps {
  countLabel?: string;
  emptyMessage?: string;
  filters?: TenantGeoSummaryParams;
  onSelectCountry?: (country: string) => void;
}

function FitBounds({ points }: { points: GeoSummaryPoint[] }) {
  const map = useMap();

  useEffect(() => {
    const frame = window.requestAnimationFrame(() => {
      map.invalidateSize(false);

      if (points.length === 0) {
        map.setView([20, 0], 2);
        return;
      }

      const bounds = L.latLngBounds(points.map((point) => [point.lat, point.lng] as [number, number]));
      map.fitBounds(bounds, { padding: [40, 40], maxZoom: 8 });
    });

    return () => window.cancelAnimationFrame(frame);
  }, [map, points]);

  return null;
}

function ClusteredMarkers({
  countLabel,
  onSelectCountry,
  points,
}: {
  countLabel: string;
  onSelectCountry?: (country: string) => void;
  points: GeoSummaryPoint[];
}) {
  const map = useMap();
  const [zoom, setZoom] = useState(() => map.getZoom());

  useMapEvents({
    zoomend: () => {
      setZoom(map.getZoom());
    },
  });

  const clusters = useMemo(
    () => clusterGeoSummaryPoints(points, zoom, (point, zoomLevel) => {
      const projected = map.project(L.latLng(point.lat, point.lng), zoomLevel);
      return { x: projected.x, y: projected.y };
    }),
    [map, points, zoom],
  );
  const maxCount = Math.max(...clusters.map((point) => point.count), 1);

  return (
    <>
      {clusters.map((cluster, index) => {
        const radius = getMarkerRadius(cluster.count, maxCount);
        const color = getMarkerColor(cluster.count, maxCount);
        const singleCountry = cluster.countries.length === 1 ? cluster.countries[0] : '';
        const label = cluster.locationCount === 1
          ? [cluster.cities[0], singleCountry].filter(Boolean).join(', ') || singleCountry || 'Geolocated activity'
          : singleCountry
            ? `${cluster.locationCount} locations in ${singleCountry}`
            : `${cluster.locationCount} locations across ${cluster.countries.length} countries`;
        const canFilterCountry = Boolean(onSelectCountry && singleCountry);

        return (
          <CircleMarker
            key={`${cluster.lat.toFixed(4)}-${cluster.lng.toFixed(4)}-${index}`}
            center={[cluster.lat, cluster.lng]}
            radius={radius}
            pathOptions={{
              fillColor: color,
              color,
              weight: 2,
              opacity: 0.8,
              fillOpacity: 0.4,
            }}
            eventHandlers={{
              click: () => {
                if (canFilterCountry && onSelectCountry) {
                  onSelectCountry(singleCountry);
                }
              },
            }}
          >
            <Popup>
              <div className="min-w-[200px]">
                <p className="text-sm font-semibold">{label}</p>
                <p className="text-sm text-muted-foreground">
                  {cluster.count} {countLabel}
                </p>
                <p className="text-xs text-muted-foreground">
                  {cluster.locationCount === 1 ? '1 plotted location' : `${cluster.locationCount} grouped locations`}
                </p>
                <p className="text-xs text-muted-foreground">
                  Last: {new Date(cluster.lastSeen).toLocaleString()}
                </p>
                {canFilterCountry ? (
                  <p className="mt-1 text-xs text-primary">
                    Click to filter the audit results by {singleCountry}
                  </p>
                ) : null}
              </div>
            </Popup>
          </CircleMarker>
        );
      })}
    </>
  );
}

function getMarkerRadius(count: number, maxCount: number): number {
  const minRadius = 6;
  const maxRadius = 28;
  if (maxCount <= 1) return minRadius;
  const ratio = Math.log(count + 1) / Math.log(maxCount + 1);
  return minRadius + ratio * (maxRadius - minRadius);
}

function getMarkerColor(count: number, maxCount: number): string {
  const ratio = maxCount > 1 ? count / maxCount : 0;
  if (ratio > 0.6) return '#d32f2f';
  if (ratio > 0.3) return '#f57c00';
  return '#1976d2';
}

export default function AuditGeoMap({
  countLabel = 'audit events',
  emptyMessage = 'No geolocated audit entries matched the current filters.',
  filters,
  onSelectCountry,
}: AuditGeoMapProps) {
  const queryKey = useMemo(() => JSON.stringify(filters ?? {}), [filters]);
  const [result, setResult] = useState<{
    error: string;
    points: GeoSummaryPoint[];
    queryKey: string;
  }>({
    error: '',
    points: [],
    queryKey: '',
  });

  useEffect(() => {
    let cancelled = false;
    const query = JSON.parse(queryKey) as TenantGeoSummaryParams;

    getTenantGeoSummary(query)
      .then((data) => {
        if (!cancelled) {
          setResult({
            error: '',
            points: data,
            queryKey,
          });
        }
      })
      .catch(() => {
        if (!cancelled) {
          setResult({
            error: 'Failed to load geo summary',
            points: [],
            queryKey,
          });
        }
      });

    return () => {
      cancelled = true;
    };
  }, [queryKey]);

  const loading = result.queryKey !== queryKey;
  const error = loading ? '' : result.error;
  const points = loading ? [] : result.points;
  const showEmptyState = !loading && !error && points.length === 0;

  return (
    <div className="relative h-[500px]">
      <WorldMapCanvas
        center={[20, 0]}
        zoom={2}
      >
        <FitBounds points={points} />
        <ClusteredMarkers countLabel={countLabel} onSelectCountry={onSelectCountry} points={points} />
      </WorldMapCanvas>

      {(loading || error || showEmptyState) ? (
        <div className="pointer-events-none absolute inset-x-4 top-4 z-[1000] flex justify-center">
          {loading ? (
            <div className="flex items-center gap-2 rounded-lg border bg-card/95 px-3 py-2 text-sm text-muted-foreground shadow-sm backdrop-blur">
              <Loader2 className="size-4 animate-spin" />
              Loading geo activity...
            </div>
          ) : error ? (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400 shadow-sm">
              {error}
            </div>
          ) : (
            <div className="rounded-lg border bg-card/95 px-3 py-2 text-sm text-muted-foreground shadow-sm backdrop-blur">
              {emptyMessage}
            </div>
          )}
        </div>
      ) : null}

      <div className="absolute bottom-4 right-4 z-[1000] min-w-[140px] rounded-lg border bg-card p-3 shadow-lg">
        <span className="mb-1 block text-xs font-semibold">Event Density</span>
        {[
          { color: '#1976d2', label: 'Low' },
          { color: '#f57c00', label: 'Medium' },
          { color: '#d32f2f', label: 'High' },
        ].map(({ color, label }) => (
          <div key={label} className="flex items-center gap-2">
            <div className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: color }} />
            <span className="text-xs">{label}</span>
          </div>
        ))}
        {onSelectCountry ? (
          <span className="mt-1 block text-xs text-muted-foreground">Click a point to filter by country</span>
        ) : null}
        <span className="mt-1 block text-xs text-muted-foreground">
          Nearby points aggregate automatically as you zoom out
        </span>
      </div>
    </div>
  );
}
