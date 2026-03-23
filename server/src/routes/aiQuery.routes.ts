import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { requireTenant, requireTenantRoleAny } from '../middleware/tenant.middleware';
import { asyncHandler } from '../middleware/asyncHandler';
import * as aiQueryController from '../controllers/aiQuery.controller';

const router = Router();

router.use(authenticate);
router.use(requireTenant);

// GET /api/ai/config — Returns tenant AI config (API key redacted). Requires ADMIN or OWNER.
router.get(
  '/config',
  requireTenantRoleAny('ADMIN', 'OWNER'),
  asyncHandler(aiQueryController.getConfig),
);

// PUT /api/ai/config — Updates tenant AI config. Requires OWNER.
router.put(
  '/config',
  requireTenantRoleAny('OWNER'),
  asyncHandler(aiQueryController.updateConfig),
);

// POST /api/ai/generate-query — Analyze prompt and return needed tables for approval.
router.post(
  '/generate-query',
  asyncHandler(aiQueryController.analyzeQuery),
);

// POST /api/ai/generate-query/confirm — Generate SQL with user-approved tables.
router.post(
  '/generate-query/confirm',
  asyncHandler(aiQueryController.confirmGeneration),
);

// POST /api/ai/optimize-query — AI query optimization (initial analysis).
router.post('/optimize-query', asyncHandler(aiQueryController.optimizeQuery));

// POST /api/ai/optimize-query/continue — Continue optimization with approved data.
router.post('/optimize-query/continue', asyncHandler(aiQueryController.continueOptimization));

export default router;
