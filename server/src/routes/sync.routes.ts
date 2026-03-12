import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { requireTenant, requireTenantRole } from '../middleware/tenant.middleware';
import { validate, validateUuidParam } from '../middleware/validate.middleware';
import { createSyncProfileSchema, updateSyncProfileSchema, triggerSyncSchema } from '../schemas/sync.schemas';
import * as syncController from '../controllers/sync.controller';
import { asyncHandler } from '../middleware/asyncHandler';

const router = Router();

router.use(authenticate);
router.use(requireTenant);
router.use(requireTenantRole('ADMIN'));

router.post('/', validate(createSyncProfileSchema), asyncHandler(syncController.create));
router.get('/', asyncHandler(syncController.list));

router.get('/:id', validateUuidParam(), asyncHandler(syncController.get));
router.put('/:id', validateUuidParam(), validate(updateSyncProfileSchema), asyncHandler(syncController.update));
router.delete('/:id', validateUuidParam(), asyncHandler(syncController.remove));
router.post('/:id/test', validateUuidParam(), asyncHandler(syncController.testConnection));
router.post('/:id/sync', validateUuidParam(), validate(triggerSyncSchema), asyncHandler(syncController.triggerSync));
router.get('/:id/logs', validateUuidParam(), asyncHandler(syncController.getLogs));

export default router;
