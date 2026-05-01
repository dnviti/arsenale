import { describe, expect, it } from 'vitest';

import { extractApiError } from './apiError';

describe('extractApiError', () => {
  it('prefers structured API error text', () => {
    expect(extractApiError({ response: { status: 413, data: { error: 'File exceeds organization limit of 128MB' } } }, 'fallback')).toBe(
      'File exceeds organization limit of 128MB',
    );
  });

  it('normalizes raw 413 string responses', () => {
    expect(extractApiError({ response: { status: 413, data: '413 Request Entity Too Large' } }, 'fallback')).toBe(
      'File exceeds maximum upload size',
    );
  });

  it('normalizes bare 413 responses without a body', () => {
    expect(extractApiError({ response: { status: 413 } }, 'fallback')).toBe('File exceeds maximum upload size');
  });

  it('falls back for unrelated errors', () => {
    expect(extractApiError({ message: 'network down' }, 'fallback')).toBe('fallback');
  });
});
