import { useState, useEffect, useMemo } from 'react';
import { Loader2 } from 'lucide-react';
import { MapContainer, CircleMarker, Popup, useMap } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { getTenantGeoSummary, type GeoSummaryPoint } from '../../api/audit.api';
import { ensureLeafletDefaultIcon } from '../../lib/leafletMarkerIcon';
import { WorldBasemap } from './WorldBasemap';

ensureLeafletDefaultIcon();

interface AuditGeoMapProps {
  onSelectCountry?: (country: string) => void;
}

/** Fit map bounds to all markers */
function FitBounds({ points }: { points: GeoSummaryPoint[] }) {
  const map = useMap();
  useEffect(() => {
    if (points.length === 0) return;
    const bounds = L.latLngBounds(points.map((p) => [p.lat, p.lng] as [number, number]));
    map.fitBounds(bounds, { padding: [40, 40], maxZoom: 8 });
  }, [points, map]);
  return null;
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
  if (ratio > 0.6) return '#d32f2f'; // high activity — red
  if (ratio > 0.3) return '#f57c00'; // medium — orange
  return '#1976d2'; // low — blue
}

export default function AuditGeoMap({ onSelectCountry }: AuditGeoMapProps) {
  const [points, setPoints] = useState<GeoSummaryPoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    getTenantGeoSummary(30)
      .then((data) => { if (!cancelled) setPoints(data); })
      .catch(() => { if (!cancelled) setError('Failed to load geo summary'); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, []);

  const maxCount = useMemo(() => Math.max(...points.map((p) => p.count), 1), [points]);

  if (loading) {
    return (
      <div className="flex justify-center items-center py-16">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="m-4 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
        {error}
      </div>
    );
  }

  if (points.length === 0) {
    return (
      <div className="text-center py-16">
        <p className="text-muted-foreground">
          No geolocation data available for the last 30 days
        </p>
      </div>
    );
  }

  return (
    <div className="relative h-[500px]">
      <MapContainer
        center={[20, 0]}
        zoom={2}
        style={{
          width: '100%',
          height: '100%',
          background: 'radial-gradient(circle at top, rgba(15, 23, 42, 0.16), rgba(15, 23, 42, 0.04) 38%, rgba(248, 250, 252, 0.96) 100%)',
        }}
        scrollWheelZoom
      >
        <WorldBasemap />
        <FitBounds points={points} />
        {points.map((point, idx) => {
          const radius = getMarkerRadius(point.count, maxCount);
          const color = getMarkerColor(point.count, maxCount);
          const label = [point.city, point.country].filter(Boolean).join(', ');

          return (
            <CircleMarker
              key={`${point.country}-${point.city}-${idx}`}
              center={[point.lat, point.lng]}
              radius={radius}
              pathOptions={{
                fillColor: color,
                color: color,
                weight: 2,
                opacity: 0.8,
                fillOpacity: 0.4,
              }}
              eventHandlers={{
                click: () => {
                  if (onSelectCountry && point.country) {
                    onSelectCountry(point.country);
                  }
                },
              }}
            >
              <Popup>
                <div className="min-w-[160px]">
                  <p className="text-sm font-semibold">
                    {label}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    {point.count} event{point.count !== 1 ? 's' : ''}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Last: {new Date(point.lastSeen).toLocaleString()}
                  </p>
                  {onSelectCountry && (
                    <p className="text-xs text-primary block mt-1 cursor-pointer">
                      Click to filter by {point.country}
                    </p>
                  )}
                </div>
              </Popup>
            </CircleMarker>
          );
        })}
      </MapContainer>

      {/* Legend */}
      <div className="absolute bottom-4 right-4 z-[1000] rounded-lg border bg-card p-3 shadow-lg min-w-[140px]">
        <span className="text-xs font-semibold mb-1 block">
          Event Density
        </span>
        {[
          { color: '#1976d2', label: 'Low' },
          { color: '#f57c00', label: 'Medium' },
          { color: '#d32f2f', label: 'High' },
        ].map(({ color, label }) => (
          <div key={label} className="flex items-center gap-2">
            <div className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: color }} />
            <span className="text-xs">{label}</span>
          </div>
        ))}
        <span className="text-xs text-muted-foreground mt-1 block">
          Click marker to filter
        </span>
        <span className="text-xs text-muted-foreground mt-1 block">
          Tiles served by the IP geolocation module
        </span>
      </div>
    </div>
  );
}
