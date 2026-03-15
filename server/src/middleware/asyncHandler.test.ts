import { Request, Response, NextFunction } from 'express';
import { asyncHandler } from './asyncHandler';

function createMocks() {
  const req = {} as Request;
  const res = {
    status: vi.fn().mockReturnThis(),
    json: vi.fn().mockReturnThis(),
  } as unknown as Response;
  const next: NextFunction = vi.fn();

  return { req, res, next };
}

describe('asyncHandler', () => {
  it('calls the wrapped function and does not call next with an error on success', async () => {
    const { req, res, next } = createMocks();
    const handler = asyncHandler(async (_req, _res, _next) => {
      // successful handler, does nothing
    });

    handler(req, res, next);

    // Allow microtask to resolve
    await new Promise((r) => setTimeout(r, 0));

    expect(next).not.toHaveBeenCalled();
  });

  it('calls next with the error when the async function rejects', async () => {
    const { req, res, next } = createMocks();
    const error = new Error('async failure');
    const handler = asyncHandler(async () => {
      throw error;
    });

    handler(req, res, next);

    await new Promise((r) => setTimeout(r, 0));

    expect(next).toHaveBeenCalledWith(error);
  });

  it('calls next with the error when a synchronous throw occurs inside the async handler', async () => {
    const { req, res, next } = createMocks();
    const error = new Error('sync throw inside async');
    const handler = asyncHandler(async () => {
      throw error;
    });

    handler(req, res, next);

    await new Promise((r) => setTimeout(r, 0));

    expect(next).toHaveBeenCalledWith(error);
  });
});
