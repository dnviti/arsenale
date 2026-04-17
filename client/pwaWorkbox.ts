import type { GenerateSWOptions, RuntimeCaching } from 'workbox-build';

const mapAssetsPathPrefix = '/map-assets/';

type WorkboxRequestDetails = {
  destination: string;
  mode: string;
};

export function isMapAssetsRequest(url: URL): boolean {
  return url.pathname.startsWith(mapAssetsPathPrefix);
}

export function isStaticAssetRequest(request: WorkboxRequestDetails, url: URL): boolean {
  return !isMapAssetsRequest(url)
    && (request.destination === 'script'
      || request.destination === 'style'
      || request.destination === 'image'
      || request.destination === 'font');
}

const runtimeCaching: RuntimeCaching[] = [
  {
    urlPattern: ({ url }) => url.pathname.startsWith(mapAssetsPathPrefix),
    handler: 'NetworkOnly',
  },
  {
    urlPattern: ({ request }) => request.mode === 'navigate',
    handler: 'NetworkFirst',
    options: {
      cacheName: 'pages',
      networkTimeoutSeconds: 3,
    },
  },
  {
    urlPattern: ({ request, url }) => !url.pathname.startsWith(mapAssetsPathPrefix)
      && (request.destination === 'script'
        || request.destination === 'style'
        || request.destination === 'image'
        || request.destination === 'font'),
    handler: 'StaleWhileRevalidate',
    options: {
      cacheName: 'static-assets',
      expiration: {
        maxEntries: 100,
        maxAgeSeconds: 30 * 24 * 60 * 60,
      },
    },
  },
];

export const pwaWorkboxConfig: Partial<GenerateSWOptions> = {
  // Only cache static assets. Tile responses stay owned by the map-assets service.
  maximumFileSizeToCacheInBytes: 3 * 1024 * 1024,
  globIgnores: ['monaco/vs/**/*'],
  navigateFallback: 'index.html',
  navigateFallbackDenylist: [/^\/api\//, /^\/socket\.io\//, /^\/guacamole\//],
  runtimeCaching,
};
