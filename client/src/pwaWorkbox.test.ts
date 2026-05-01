import { describe, expect, it } from 'vitest';
import { isMapAssetsRequest, isStaticAssetRequest, pwaWorkboxConfig } from '../pwaWorkbox';

describe('pwaWorkboxConfig', () => {
  it('routes map-assets requests through a network-only rule', () => {
    const tileUrl = new URL('https://localhost:3000/map-assets/v1/tiles/world/2/1/1');
    const imageRequest = { destination: 'image', mode: 'same-origin' } as Request;
    const mapAssetsRule = pwaWorkboxConfig.runtimeCaching?.[0];

    expect(isMapAssetsRequest(tileUrl)).toBe(true);
    expect(isStaticAssetRequest(imageRequest, tileUrl)).toBe(false);
    expect(mapAssetsRule?.handler).toBe('NetworkOnly');
    expect(typeof mapAssetsRule?.urlPattern).toBe('function');
  });

  it('still allows non-map images through the generic static asset cache rule', () => {
    const assetUrl = new URL('https://localhost:3000/icon-192.png');
    const imageRequest = { destination: 'image', mode: 'same-origin' } as Request;

    expect(isMapAssetsRequest(assetUrl)).toBe(false);
    expect(isStaticAssetRequest(imageRequest, assetUrl)).toBe(true);
  });
});
