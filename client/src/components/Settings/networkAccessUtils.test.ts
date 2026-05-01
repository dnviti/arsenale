import { describe, expect, it } from 'vitest';
import { isIpInCidr, isValidNetworkEntry } from './networkAccessUtils';

describe('networkAccessUtils', () => {
  it('validates IPv4 and IPv6 network entries', () => {
    expect(isValidNetworkEntry('203.0.113.5')).toBe(true);
    expect(isValidNetworkEntry('203.0.113.0/24')).toBe(true);
    expect(isValidNetworkEntry('2001:db8::/32')).toBe(true);
    expect(isValidNetworkEntry('300.0.113.0/24')).toBe(false);
    expect(isValidNetworkEntry('2001:db8::/129')).toBe(false);
  });

  it('checks whether an address belongs to a CIDR range', () => {
    expect(isIpInCidr('203.0.113.5', '203.0.113.0/24')).toBe(true);
    expect(isIpInCidr('203.0.114.5', '203.0.113.0/24')).toBe(false);
    expect(isIpInCidr('2001:db8::1', '2001:db8::/32')).toBe(true);
    expect(isIpInCidr('2001:db9::1', '2001:db8::/32')).toBe(false);
  });
});
