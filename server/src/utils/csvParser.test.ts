import { describe, it, expect } from 'vitest';
import { parseCSV, generateCSV, escapeCSV, unescapeCSV } from './csvParser';

describe('parseCSV', () => {
  it('parses a simple CSV with headers and rows', () => {
    const csv = 'name,age,city\nAlice,30,Rome\nBob,25,Milan';
    const result = parseCSV(csv);
    expect(result.headers).toEqual(['name', 'age', 'city']);
    expect(result.rows).toEqual([
      ['Alice', '30', 'Rome'],
      ['Bob', '25', 'Milan'],
    ]);
  });

  it('handles quoted fields with commas inside', () => {
    const csv = 'name,address\nAlice,"123 Main St, Apt 4"';
    const result = parseCSV(csv);
    expect(result.rows[0]).toEqual(['Alice', '123 Main St, Apt 4']);
  });

  it('handles escaped quotes within quoted fields', () => {
    const csv = 'name,quote\nAlice,"She said ""hello"""';
    const result = parseCSV(csv);
    expect(result.rows[0]).toEqual(['Alice', 'She said "hello"']);
  });

  it('returns empty headers and rows for an empty string', () => {
    expect(parseCSV('')).toEqual({ headers: [], rows: [] });
  });

  it('handles CRLF line endings', () => {
    const csv = 'a,b\r\n1,2\r\n3,4';
    const result = parseCSV(csv);
    expect(result.headers).toEqual(['a', 'b']);
    expect(result.rows).toEqual([
      ['1', '2'],
      ['3', '4'],
    ]);
  });
});

describe('generateCSV', () => {
  it('produces valid CSV from headers and rows', () => {
    const csv = generateCSV(['name', 'age'], [['Alice', '30'], ['Bob', '25']]);
    expect(csv).toBe('name,age\nAlice,30\nBob,25');
  });

  it('roundtrips with parseCSV', () => {
    const headers = ['name', 'note'];
    const rows = [['Alice', 'likes, commas'], ['Bob', 'says "hi"']];
    const csv = generateCSV(headers, rows);
    const parsed = parseCSV(csv);
    expect(parsed.headers).toEqual(headers);
    expect(parsed.rows).toEqual(rows);
  });
});

describe('escapeCSV', () => {
  it('quotes a value that contains a comma', () => {
    expect(escapeCSV('a,b')).toBe('"a,b"');
  });

  it('returns the value unchanged when no special characters are present', () => {
    expect(escapeCSV('hello')).toBe('hello');
  });
});

describe('unescapeCSV', () => {
  it('unescapes a quoted value with escaped quotes', () => {
    expect(unescapeCSV('"She said ""hello"""')).toBe('She said "hello"');
  });
});
