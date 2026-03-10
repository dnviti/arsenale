import { open, Reader, CityResponse } from 'maxmind';
import { config } from '../config';
import { logger } from '../utils/logger';

export interface GeoLookupResult {
  country: string;
  countryCode: string;
  city: string;
  lat: number;
  lng: number;
}

let reader: Reader<CityResponse> | null = null;
let initialized = false;

/**
 * Initialize the MaxMind GeoLite2 database reader.
 * Call once at server startup. If GEOIP_DB_PATH is not configured or the file
 * is missing, geo lookups gracefully return null.
 */
export async function initGeoIp(): Promise<void> {
  if (initialized) return;
  initialized = true;

  if (!config.geoipDbPath) {
    logger.info('[geoip] GEOIP_DB_PATH not configured — geolocation disabled');
    return;
  }

  try {
    reader = await open<CityResponse>(config.geoipDbPath);
    logger.info(`[geoip] GeoLite2 database loaded from ${config.geoipDbPath}`);
  } catch (err) {
    logger.warn(`[geoip] Failed to open GeoLite2 database at ${config.geoipDbPath}: ${err}`);
    reader = null;
  }
}

/**
 * Look up geolocation data for an IP address.
 * Returns null for private/unresolvable IPs, or if the database is not loaded.
 * Sub-millisecond in-memory lookup — safe to call synchronously in hot paths.
 */
export function lookup(ip: string | null | undefined): GeoLookupResult | null {
  if (!reader || !ip) return null;

  try {
    const result = reader.get(ip);
    if (!result) return null;

    const country = result.country?.names?.en ?? result.registered_country?.names?.en;
    const countryCode = result.country?.iso_code ?? result.registered_country?.iso_code;
    const city = result.city?.names?.en;
    const lat = result.location?.latitude;
    const lng = result.location?.longitude;

    if (!country || !countryCode) return null;

    return {
      country,
      countryCode,
      city: city ?? '',
      lat: lat ?? 0,
      lng: lng ?? 0,
    };
  } catch {
    // Private IPs, malformed IPs, etc. — return null silently
    return null;
  }
}

/**
 * Check whether the geo database is loaded and ready for lookups.
 */
export function isReady(): boolean {
  return reader !== null;
}
