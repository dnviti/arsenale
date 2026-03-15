import { passwordSchema, uuidParam } from './validate';

describe('passwordSchema', () => {
  it('accepts a valid password with 10+ chars, lowercase, uppercase, and digit', () => {
    expect(passwordSchema.safeParse('Abcdefghi1').success).toBe(true);
    expect(passwordSchema.safeParse('MyStr0ngPassword!').success).toBe(true);
  });

  it('rejects a 9-character password', () => {
    const result = passwordSchema.safeParse('Abcdefg1!');
    expect(result.success).toBe(false);
  });

  it('accepts a 10-character password meeting all rules', () => {
    const result = passwordSchema.safeParse('Abcdefghi1');
    expect(result.success).toBe(true);
  });

  it('rejects a password missing a lowercase letter', () => {
    const result = passwordSchema.safeParse('ABCDEFGHI1');
    expect(result.success).toBe(false);
  });

  it('rejects a password missing an uppercase letter', () => {
    const result = passwordSchema.safeParse('abcdefghi1');
    expect(result.success).toBe(false);
  });

  it('rejects a password missing a digit', () => {
    const result = passwordSchema.safeParse('Abcdefghij');
    expect(result.success).toBe(false);
  });
});

describe('uuidParam', () => {
  it('accepts valid UUIDs', () => {
    expect(uuidParam.safeParse('550e8400-e29b-41d4-a716-446655440000').success).toBe(true);
    expect(uuidParam.safeParse('6ba7b810-9dad-11d1-80b4-00c04fd430c8').success).toBe(true);
  });

  it('rejects invalid strings', () => {
    expect(uuidParam.safeParse('not-a-uuid').success).toBe(false);
    expect(uuidParam.safeParse('').success).toBe(false);
    expect(uuidParam.safeParse('550e8400-e29b-41d4-a716').success).toBe(false);
    expect(uuidParam.safeParse('123').success).toBe(false);
  });
});
