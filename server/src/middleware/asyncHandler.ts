import { Request, Response, NextFunction } from 'express';

type AsyncHandler<Req extends Request = Request> = (
  req: Req,
  res: Response,
  next: NextFunction,
) => Promise<unknown>;

export function asyncHandler<Req extends Request = Request>(fn: AsyncHandler<Req>) {
  return (req: Req, res: Response, next: NextFunction) => {
    Promise.resolve(fn(req, res, next)).catch(next);
  };
}
