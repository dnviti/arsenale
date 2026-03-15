import { describe, it, expect } from 'vitest';
import { isIpInCidr, isIpAllowed } from './ipAllowlist';

describe('isIpInCidr', () => {
  describe('IPv4', () => {
    it('matches IP within /8 CIDR', () => {
      expect(isIpInCidr('10.0.0.1', '10.0.0.0/8')).toBe(true);
    });

    it('rejects IP outside /8 CIDR', () => {
      expect(isIpInCidr('11.0.0.1', '10.0.0.0/8')).toBe(false);
    });

    it('matches IP within /24 CIDR', () => {
      expect(isIpInCidr('192.168.1.100', '192.168.1.0/24')).toBe(true);
    });

    it('rejects IP outside /24 CIDR', () => {
      expect(isIpInCidr('192.168.2.1', '192.168.1.0/24')).toBe(false);
    });

    it('matches exact bare IP (no /prefix)', () => {
      expect(isIpInCidr('10.0.0.1', '10.0.0.1')).toBe(true);
    });

    it('rejects non-matching bare IP', () => {
      expect(isIpInCidr('10.0.0.2', '10.0.0.1')).toBe(false);
    });

    it('/0 matches any IPv4 address', () => {
      expect(isIpInCidr('1.2.3.4', '0.0.0.0/0')).toBe(true);
    });

    it('/32 matches only the exact IP', () => {
      expect(isIpInCidr('10.0.0.1', '10.0.0.1/32')).toBe(true);
    });
  });

  describe('IPv4-mapped IPv6', () => {
    it('strips ::ffff: prefix and matches IPv4 CIDR', () => {
      expect(isIpInCidr('::ffff:192.168.1.1', '192.168.1.0/24')).toBe(true);
    });
  });

  describe('IPv6', () => {
    it('matches IP within /32 CIDR', () => {
      expect(isIpInCidr('2001:db8::1', '2001:db8::/32')).toBe(true);
    });

    it('matches link-local in /10 CIDR', () => {
      expect(isIpInCidr('fe80::1', 'fe80::/10')).toBe(true);
    });

    it('rejects IP outside CIDR', () => {
      expect(isIpInCidr('2001:db8::1', 'fe80::/10')).toBe(false);
    });

    it('matches bare IPv6 loopback', () => {
      expect(isIpInCidr('::1', '::1')).toBe(true);
    });
  });

  describe('mixed families', () => {
    it('returns false for IPv4 address against IPv6 CIDR', () => {
      expect(isIpInCidr('10.0.0.1', '2001:db8::/32')).toBe(false);
    });
  });

  describe('malformed input', () => {
    it('returns false for malformed CIDR', () => {
      expect(isIpInCidr('10.0.0.1', 'not-a-cidr')).toBe(false);
    });

    it('returns false for NaN prefix length', () => {
      expect(isIpInCidr('10.0.0.1', '10.0.0.0/abc')).toBe(false);
    });
  });
});

describe('isIpAllowed', () => {
  it('allows any IP when entries list is empty', () => {
    expect(isIpAllowed('1.2.3.4', [])).toBe(true);
  });

  it('allows IP that matches one of multiple entries', () => {
    expect(isIpAllowed('192.168.1.50', ['10.0.0.0/8', '192.168.1.0/24'])).toBe(true);
  });

  it('rejects IP that matches none of the entries', () => {
    expect(isIpAllowed('172.16.0.1', ['10.0.0.0/8', '192.168.1.0/24'])).toBe(false);
  });
});
