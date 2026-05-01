import type { GenerateSWOptions } from 'workbox-build';
type WorkboxRequestDetails = {
    destination: string;
    mode: string;
};
export declare function isMapAssetsRequest(url: URL): boolean;
export declare function isStaticAssetRequest(request: WorkboxRequestDetails, url: URL): boolean;
export declare const pwaWorkboxConfig: Partial<GenerateSWOptions>;
export {};
//# sourceMappingURL=pwaWorkbox.d.ts.map