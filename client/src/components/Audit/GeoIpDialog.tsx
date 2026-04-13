import { useState, useEffect, useRef } from 'react';
import {
  Dialog,
  DialogContent,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import {
  X,
  Globe,
  MapPin,
  Building2,
  Server,
  Clock,
  Shield,
  Smartphone,
  Lock,
  HardDrive,
  Loader2,
} from 'lucide-react';
import { Marker, Popup } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import api from '../../api/client';
import { countryFlag } from './IpGeoCell';
import { extractApiError } from '../../utils/apiError';
import { ensureLeafletDefaultIcon } from '../../lib/leafletMarkerIcon';
import { WorldMapCanvas } from './WorldMapCanvas';

ensureLeafletDefaultIcon();

interface IpApiData {
  status: 'success' | 'fail';
  message?: string;
  country?: string;
  countryCode?: string;
  regionName?: string;
  city?: string;
  zip?: string;
  lat?: number;
  lon?: number;
  timezone?: string;
  isp?: string;
  org?: string;
  as?: string;
  asname?: string;
  mobile?: boolean;
  proxy?: boolean;
  hosting?: boolean;
  query?: string;
}

interface GeoIpDialogProps {
  open: boolean;
  onClose: () => void;
  ipAddress: string | null;
}

function InfoRow({ icon, label, value }: { icon: React.ReactNode; label: string; value: React.ReactNode }) {
  if (!value) return null;
  return (
    <div className="flex items-center gap-3 py-1.5">
      <div className="text-muted-foreground flex items-center">{icon}</div>
      <span className="text-sm text-muted-foreground min-w-[100px] shrink-0">
        {label}
      </span>
      <span className="text-sm">{value}</span>
    </div>
  );
}

export default function GeoIpDialog({ open, onClose, ipAddress }: GeoIpDialogProps) {
  const [data, setData] = useState<IpApiData | null>(null);
  const [error, setError] = useState('');
  const [fetchKey, setFetchKey] = useState(0);
  const loadingRef = useRef(false);
  const [loading, setLoading] = useState(false);

  const prevIpRef = useRef<string | null>(null);
  if (open && ipAddress && ipAddress !== prevIpRef.current) {
    prevIpRef.current = ipAddress;
    loadingRef.current = true;
    setLoading(true);
    setError('');
    setData(null);
    setFetchKey((k) => k + 1);
  }
  if (!open && prevIpRef.current !== null) {
    prevIpRef.current = null;
  }

  useEffect(() => {
    if (!open || !ipAddress) return;
    let cancelled = false;

    api.get(`/geoip/${encodeURIComponent(ipAddress)}`)
      .then((res) => {
        if (cancelled) return;
        const payload = res.data as IpApiData;
        if (payload?.status === 'fail') {
          setError(payload.message || 'Failed to look up IP');
          setData(null);
          return;
        }
        setData(payload);
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(extractApiError(err, 'Failed to look up IP'));
        }
      })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchKey]);

  const hasCoords = data && typeof data.lat === 'number' && typeof data.lon === 'number' && (data.lat !== 0 || data.lon !== 0);
  const flag = data?.countryCode ? countryFlag(data.countryCode) : '';
  const locationParts = [data?.city, data?.regionName, data?.country].filter(Boolean);

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose(); }}>
      <DialogContent
        showCloseButton={false}
        className="h-[100dvh] w-screen max-w-none gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
      >
        {/* Header */}
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
          <h2 className="ml-1 flex-1 text-lg font-semibold">
            IP Geolocation {ipAddress ? `\u2014 ${ipAddress}` : ''}
          </h2>
        </div>

        <div className="flex-1 overflow-auto flex flex-col">
          {loading && (
            <div className="flex justify-center items-center flex-1">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          )}

          {error && (
            <div className="p-6">
              <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
                {error}
              </div>
            </div>
          )}

          {data && !loading && (
            <div className="flex flex-1 flex-col md:flex-row">
              {/* Info panel */}
              <div className="w-full md:w-[420px] shrink-0 p-6 overflow-auto">
                <div className="p-4 bg-muted/50 rounded-xl mb-4">
                  <h3 className="text-xl flex items-center gap-2 mb-1">
                    {flag && <span className="text-2xl">{flag}</span>}
                    {data.query}
                  </h3>
                  {locationParts.length > 0 && (
                    <p className="text-muted-foreground">
                      {locationParts.join(', ')}
                    </p>
                  )}
                </div>

                <p className="text-sm font-medium text-muted-foreground mb-2 mt-4">
                  Location
                </p>
                <InfoRow icon={<Globe className="h-4 w-4" />} label="Country" value={
                  data.country ? (
                    <span className="flex items-center gap-1">
                      {flag && <span>{flag}</span>}
                      {data.country} ({data.countryCode})
                    </span>
                  ) : null
                } />
                <InfoRow icon={<MapPin className="h-4 w-4" />} label="Region" value={data.regionName} />
                <InfoRow icon={<MapPin className="h-4 w-4" />} label="City" value={data.city} />
                <InfoRow icon={<MapPin className="h-4 w-4" />} label="ZIP" value={data.zip} />
                <InfoRow icon={<MapPin className="h-4 w-4" />} label="Coordinates" value={
                  hasCoords ? `${(data.lat ?? 0).toFixed(4)}, ${(data.lon ?? 0).toFixed(4)}` : undefined
                } />
                <InfoRow icon={<Clock className="h-4 w-4" />} label="Timezone" value={data.timezone} />

                <Separator className="my-4" />

                <p className="text-sm font-medium text-muted-foreground mb-2">
                  Network
                </p>
                <InfoRow icon={<Building2 className="h-4 w-4" />} label="ISP" value={data.isp} />
                <InfoRow icon={<Building2 className="h-4 w-4" />} label="Organization" value={data.org} />
                <InfoRow icon={<Server className="h-4 w-4" />} label="AS" value={data.as} />
                <InfoRow icon={<Server className="h-4 w-4" />} label="AS Name" value={data.asname} />

                <Separator className="my-4" />

                <p className="text-sm font-medium text-muted-foreground mb-2">
                  Flags
                </p>
                <div className="flex gap-2 flex-wrap">
                  {data.proxy && (
                    <Badge className="bg-yellow-500/15 text-yellow-400 border-yellow-500/30 gap-1">
                      <Lock className="h-3 w-3" />
                      Proxy / VPN / Tor
                    </Badge>
                  )}
                  {data.hosting && (
                    <Badge className="bg-blue-500/15 text-blue-400 border-blue-500/30 gap-1">
                      <HardDrive className="h-3 w-3" />
                      Hosting / Datacenter
                    </Badge>
                  )}
                  {data.mobile && (
                    <Badge className="gap-1">
                      <Smartphone className="h-3 w-3" />
                      Mobile / Cellular
                    </Badge>
                  )}
                  {!data.proxy && !data.hosting && !data.mobile && (
                    <Badge variant="outline" className="gap-1 text-green-400 border-green-500/30">
                      <Shield className="h-3 w-3" />
                      Residential
                    </Badge>
                  )}
                </div>
              </div>

              {/* Map */}
              <div className="flex-1 min-h-[300px] md:min-h-0 relative">
                {hasCoords ? (
                  <WorldMapCanvas
                    center={[data.lat ?? 0, data.lon ?? 0]}
                    zoom={10}
                    minHeight={400}
                  >
                    <Marker position={[data.lat ?? 0, data.lon ?? 0]}>
                      <Popup>
                        <strong>{data.query}</strong><br />
                        {locationParts.join(', ')}<br />
                        {data.isp && <>ISP: {data.isp}</>}
                      </Popup>
                    </Marker>
                  </WorldMapCanvas>
                ) : (
                  <div className="flex justify-center items-center h-full">
                    <p className="text-muted-foreground">No coordinates available for this IP</p>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
