import { Globe } from 'lucide-react';

/**
 * Convert ISO 3166-1 alpha-2 country code to flag emoji.
 * Works by mapping each letter to a Regional Indicator Symbol.
 */
function countryFlag(code: string | null | undefined): string {
  if (!code || code.length !== 2) return '';
  const chars = code
    .toUpperCase()
    .split('')
    .map((c) => String.fromCodePoint(0x1f1e6 + c.charCodeAt(0) - 65));
  return chars.join('');
}

/**
 * Determine the country code from the country name.
 */
function getCountryCode(geoCountry: string | null): string | null {
  if (!geoCountry) return null;
  const map: Record<string, string> = {
    'Afghanistan': 'AF', 'Albania': 'AL', 'Algeria': 'DZ', 'Andorra': 'AD',
    'Angola': 'AO', 'Argentina': 'AR', 'Armenia': 'AM', 'Australia': 'AU',
    'Austria': 'AT', 'Azerbaijan': 'AZ', 'Bahrain': 'BH', 'Bangladesh': 'BD',
    'Belarus': 'BY', 'Belgium': 'BE', 'Bolivia': 'BO', 'Bosnia and Herzegovina': 'BA',
    'Brazil': 'BR', 'Bulgaria': 'BG', 'Cambodia': 'KH', 'Cameroon': 'CM',
    'Canada': 'CA', 'Chile': 'CL', 'China': 'CN', 'Colombia': 'CO',
    'Costa Rica': 'CR', 'Croatia': 'HR', 'Cuba': 'CU', 'Cyprus': 'CY',
    'Czechia': 'CZ', 'Czech Republic': 'CZ', 'Denmark': 'DK',
    'Dominican Republic': 'DO', 'Ecuador': 'EC', 'Egypt': 'EG',
    'El Salvador': 'SV', 'Estonia': 'EE', 'Ethiopia': 'ET',
    'Finland': 'FI', 'France': 'FR', 'Georgia': 'GE', 'Germany': 'DE',
    'Ghana': 'GH', 'Greece': 'GR', 'Guatemala': 'GT', 'Honduras': 'HN',
    'Hong Kong': 'HK', 'Hungary': 'HU', 'Iceland': 'IS', 'India': 'IN',
    'Indonesia': 'ID', 'Iran': 'IR', 'Iraq': 'IQ', 'Ireland': 'IE',
    'Israel': 'IL', 'Italy': 'IT', 'Jamaica': 'JM', 'Japan': 'JP',
    'Jordan': 'JO', 'Kazakhstan': 'KZ', 'Kenya': 'KE', 'Kuwait': 'KW',
    'Kyrgyzstan': 'KG', 'Latvia': 'LV', 'Lebanon': 'LB', 'Libya': 'LY',
    'Lithuania': 'LT', 'Luxembourg': 'LU', 'Macao': 'MO', 'Malaysia': 'MY',
    'Malta': 'MT', 'Mexico': 'MX', 'Moldova': 'MD', 'Mongolia': 'MN',
    'Montenegro': 'ME', 'Morocco': 'MA', 'Mozambique': 'MZ', 'Myanmar': 'MM',
    'Nepal': 'NP', 'Netherlands': 'NL', 'New Zealand': 'NZ', 'Nicaragua': 'NI',
    'Nigeria': 'NG', 'North Korea': 'KP', 'North Macedonia': 'MK',
    'Norway': 'NO', 'Oman': 'OM', 'Pakistan': 'PK', 'Palestine': 'PS',
    'Panama': 'PA', 'Paraguay': 'PY', 'Peru': 'PE', 'Philippines': 'PH',
    'Poland': 'PL', 'Portugal': 'PT', 'Puerto Rico': 'PR', 'Qatar': 'QA',
    'Romania': 'RO', 'Russia': 'RU', 'Rwanda': 'RW', 'Saudi Arabia': 'SA',
    'Senegal': 'SN', 'Serbia': 'RS', 'Singapore': 'SG', 'Slovakia': 'SK',
    'Slovenia': 'SI', 'South Africa': 'ZA', 'South Korea': 'KR', 'Spain': 'ES',
    'Sri Lanka': 'LK', 'Sudan': 'SD', 'Sweden': 'SE', 'Switzerland': 'CH',
    'Syria': 'SY', 'Taiwan': 'TW', 'Tajikistan': 'TJ', 'Tanzania': 'TZ',
    'Thailand': 'TH', 'Tunisia': 'TN', 'Turkey': 'TR', 'Turkmenistan': 'TM',
    'Uganda': 'UG', 'Ukraine': 'UA', 'United Arab Emirates': 'AE',
    'United Kingdom': 'GB', 'United States': 'US', 'Uruguay': 'UY',
    'Uzbekistan': 'UZ', 'Venezuela': 'VE', 'Vietnam': 'VN', 'Yemen': 'YE',
    'Zambia': 'ZM', 'Zimbabwe': 'ZW',
  };
  return map[geoCountry] ?? null;
}

interface IpGeoCellProps {
  ipAddress: string | null;
  geoCountry: string | null;
  geoCity: string | null;
  onGeoIpClick?: (ip: string) => void;
}

/**
 * Renders an IP address cell with geolocation info and a clickable
 * action to inspect the IP in the GeoIpDialog.
 */
export default function IpGeoCell({ ipAddress, geoCountry, geoCity, onGeoIpClick }: IpGeoCellProps) {
  if (!ipAddress) return <>{'\u2014'}</>;

  const code = getCountryCode(geoCountry);
  const flag = countryFlag(code);
  const geoLabel = [geoCity, geoCountry].filter(Boolean).join(', ');

  const publicIp = extractPublicIPv4(ipAddress);
  const displayIp = publicIp ?? ipAddress;
  const isExternal = !!publicIp;

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onGeoIpClick && publicIp) {
      onGeoIpClick(publicIp);
    }
  };

  return (
    <div className="flex items-center gap-1">
      {flag && (
        <span style={{ fontSize: '1rem', lineHeight: 1 }} title={geoLabel || geoCountry || ''}>
          {flag}
        </span>
      )}
      {isExternal && onGeoIpClick ? (
        <button
          onClick={handleClick}
          className="inline-flex items-center gap-0.5 text-primary hover:underline cursor-pointer border-none bg-transparent p-0"
          style={{ font: 'inherit' }}
          title={geoLabel ? `${geoLabel} \u2014 Click to inspect` : 'Click to inspect IP'}
        >
          {displayIp}
          <Globe className="h-3.5 w-3.5 opacity-60" />
        </button>
      ) : isExternal ? (
        <span className="inline-flex items-center gap-0.5" title={geoLabel ? `${geoLabel}` : displayIp}>
          {displayIp}
          <Globe className="h-3.5 w-3.5 opacity-40" />
        </span>
      ) : (
        <span>{displayIp}</span>
      )}
      {geoLabel && (
        <span className="text-xs text-muted-foreground ml-1 whitespace-nowrap">
          {geoLabel}
        </span>
      )}
    </div>
  );
}

/**
 * Check if a single IP address is private/reserved (not externally routable).
 */
function isPrivateIp(ip: string): boolean {
  const clean = ip.startsWith('::ffff:') ? ip.slice(7) : ip;
  if (clean.startsWith('10.') || clean.startsWith('192.168.') || clean === '127.0.0.1' || clean === '::1') {
    return true;
  }
  if (clean.startsWith('172.')) {
    const second = parseInt(clean.split('.')[1], 10);
    if (second >= 16 && second <= 31) return true;
  }
  if (clean === 'localhost' || clean.startsWith('fe80:') || clean.startsWith('fd') || clean.startsWith('fc')) return true;
  return false;
}

/**
 * Check if a string looks like a valid IPv4 address (not IPv6).
 */
function isIPv4(ip: string): boolean {
  return /^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(ip);
}

/**
 * From a raw IP string (which may contain multiple comma-separated IPs),
 * extract the first public IPv4 address suitable for GeoIP lookup.
 * Returns null if no public IPv4 is found.
 */
function extractPublicIPv4(raw: string): string | null {
  const parts = raw.split(',').map((s) => {
    const trimmed = s.trim();
    return trimmed.startsWith('::ffff:') ? trimmed.slice(7) : trimmed;
  });
  return parts.find((p) => isIPv4(p) && !isPrivateIp(p)) ?? null;
}

/**
 * Export the flag helper for CSV export and map usage.
 */
// eslint-disable-next-line react-refresh/only-export-components
export { getCountryCode, countryFlag };
