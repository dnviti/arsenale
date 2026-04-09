function isValidIpv4(ip: string): boolean {
  const parts = ip.split('.');
  return parts.length === 4 && parts.every((part) => {
    if (!/^\d{1,3}$/.test(part)) {
      return false;
    }
    const value = Number(part);
    return value >= 0 && value <= 255;
  });
}

function expandIpv6(ip: string): string[] | null {
  if (!/^[0-9a-f:]+$/i.test(ip)) {
    return null;
  }

  const sides = ip.split('::');
  if (sides.length > 2) {
    return null;
  }

  const parseGroups = (segment: string) =>
    segment ? segment.split(':').filter(Boolean) : [];

  const leftGroups = parseGroups(sides[0] ?? '');
  const rightGroups = parseGroups(sides[1] ?? '');
  const totalGroups = leftGroups.length + rightGroups.length;

  if (sides.length === 1) {
    if (totalGroups !== 8) {
      return null;
    }
    return leftGroups.every((group) => /^[0-9a-f]{1,4}$/i.test(group)) ? leftGroups : null;
  }

  if (totalGroups >= 8) {
    return null;
  }

  const expanded = [
    ...leftGroups,
    ...Array(8 - totalGroups).fill('0'),
    ...rightGroups,
  ];

  return expanded.every((group) => /^[0-9a-f]{1,4}$/i.test(group)) ? expanded : null;
}

function ipv6ToBigInt(ip: string): bigint | null {
  const groups = expandIpv6(ip);
  if (!groups) {
    return null;
  }

  return groups.reduce((result, group) => {
    return (result << BigInt(16)) | BigInt(Number.parseInt(group, 16));
  }, BigInt(0));
}

function toIpv4Int(ip: string): number {
  return ip.split('.').reduce((result, part) => (result << 8) | Number.parseInt(part, 10), 0) >>> 0;
}

export function isValidNetworkEntry(value: string): boolean {
  const trimmedValue = value.trim();
  if (!trimmedValue) {
    return false;
  }

  const slashIndex = trimmedValue.lastIndexOf('/');
  const address = slashIndex === -1 ? trimmedValue : trimmedValue.slice(0, slashIndex);
  const prefix = slashIndex === -1 ? null : Number.parseInt(trimmedValue.slice(slashIndex + 1), 10);

  if (isValidIpv4(address)) {
    return prefix == null || (Number.isInteger(prefix) && prefix >= 0 && prefix <= 32);
  }

  if (ipv6ToBigInt(address) != null) {
    return prefix == null || (Number.isInteger(prefix) && prefix >= 0 && prefix <= 128);
  }

  return false;
}

export function isIpInCidr(ip: string, cidr: string): boolean {
  const slashIndex = cidr.lastIndexOf('/');
  if (slashIndex === -1) {
    return ip.trim() === cidr.trim();
  }

  const baseAddress = cidr.slice(0, slashIndex).trim();
  const prefix = Number.parseInt(cidr.slice(slashIndex + 1), 10);
  const trimmedIp = ip.trim();

  if (isValidIpv4(baseAddress) && isValidIpv4(trimmedIp)) {
    const mask = prefix === 0 ? 0 : (~0 << (32 - prefix)) >>> 0;
    return (toIpv4Int(trimmedIp) & mask) === (toIpv4Int(baseAddress) & mask);
  }

  const ipValue = ipv6ToBigInt(trimmedIp);
  const baseValue = ipv6ToBigInt(baseAddress);
  if (ipValue == null || baseValue == null) {
    return false;
  }

  if (prefix === 0) {
    return true;
  }

  const bitWidth = BigInt(128);
  const prefixBits = BigInt(prefix);
  const allOnes = (BigInt(1) << bitWidth) - BigInt(1);
  const trailingZeros = (BigInt(1) << (bitWidth - prefixBits)) - BigInt(1);
  const mask = allOnes ^ trailingZeros;

  return (ipValue & mask) === (baseValue & mask);
}
