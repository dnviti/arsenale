import { Request, Response, NextFunction } from 'express';

/**
 * Factory that returns Express middleware gating routes behind a feature toggle.
 * The `isEnabled` closure is evaluated on every request so runtime config
 * changes (via the Settings UI) take effect immediately.
 */
export function requireFeature(isEnabled: () => boolean, featureName: string) {
  return (_req: Request, res: Response, next: NextFunction): void => {
    if (!isEnabled()) {
      res.status(403).json({ error: `The ${featureName} feature is currently disabled.` });
      return;
    }
    next();
  };
}
