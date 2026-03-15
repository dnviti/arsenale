import { parseExpiry, formatDuration } from './format';

describe('parseExpiry', () => {
  it('parses seconds', () => {
    expect(parseExpiry('10s')).toBe(10_000);
  });

  it('parses minutes', () => {
    expect(parseExpiry('15m')).toBe(900_000);
  });

  it('parses hours', () => {
    expect(parseExpiry('24h')).toBe(86_400_000);
  });

  it('parses days', () => {
    expect(parseExpiry('7d')).toBe(604_800_000);
  });

  it('returns 0 for 0s', () => {
    expect(parseExpiry('0s')).toBe(0);
  });

  it('defaults to 7 days for invalid format', () => {
    const sevenDays = 7 * 24 * 60 * 60 * 1000;
    expect(parseExpiry('')).toBe(sevenDays);
    expect(parseExpiry('abc')).toBe(sevenDays);
    expect(parseExpiry('10x')).toBe(sevenDays);
    expect(parseExpiry('10')).toBe(sevenDays);
    expect(parseExpiry('s10')).toBe(sevenDays);
  });
});

describe('formatDuration', () => {
  it('formats 0ms as 0s', () => {
    expect(formatDuration(0)).toBe('0s');
  });

  it('formats seconds only', () => {
    expect(formatDuration(5_000)).toBe('5s');
  });

  it('formats minutes and seconds', () => {
    expect(formatDuration(90_000)).toBe('1m 30s');
  });

  it('formats hours, minutes, and seconds', () => {
    expect(formatDuration(3_661_000)).toBe('1h 1m 1s');
  });

  it('formats exactly 1 hour', () => {
    expect(formatDuration(3_600_000)).toBe('1h 0m 0s');
  });

  it('formats exactly 1 minute', () => {
    expect(formatDuration(60_000)).toBe('1m 0s');
  });
});
