import { extractApiError } from './apiError';

describe('extractApiError', () => {
  it('returns response.data.error when present', () => {
    const err = { response: { data: { error: 'Not found' } } };
    expect(extractApiError(err, 'fallback')).toBe('Not found');
  });

  it('returns response.data.message when error field is absent', () => {
    const err = { response: { data: { message: 'Something went wrong' } } };
    expect(extractApiError(err, 'fallback')).toBe('Something went wrong');
  });

  it('prefers error over message when both are present', () => {
    const err = {
      response: { data: { error: 'Primary error', message: 'Secondary message' } },
    };
    expect(extractApiError(err, 'fallback')).toBe('Primary error');
  });

  it('returns fallback when error has no response', () => {
    const err = { message: 'Network Error' };
    expect(extractApiError(err, 'fallback')).toBe('fallback');
  });

  it('returns fallback for null err', () => {
    expect(extractApiError(null, 'fallback')).toBe('fallback');
  });

  it('returns fallback for undefined err', () => {
    expect(extractApiError(undefined, 'fallback')).toBe('fallback');
  });

  it('returns fallback for plain string err', () => {
    expect(extractApiError('some string', 'fallback')).toBe('fallback');
  });

  it('returns fallback when response.data is empty object', () => {
    const err = { response: { data: {} } };
    expect(extractApiError(err, 'fallback')).toBe('fallback');
  });

  it('returns fallback when response.data.error is an empty string', () => {
    const err = { response: { data: { error: '' } } };
    expect(extractApiError(err, 'fallback')).toBe('fallback');
  });
});
