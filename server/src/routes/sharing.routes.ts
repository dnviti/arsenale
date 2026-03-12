import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { validate, validateUuidParam } from '../middleware/validate.middleware';
import { shareSchema, batchShareSchema, updatePermissionSchema } from '../schemas/sharing.schemas';
import { asyncHandler } from '../middleware/asyncHandler';
import * as sharingController from '../controllers/sharing.controller';

const router = Router();

router.use(authenticate);
router.post('/batch-share', validate(batchShareSchema), asyncHandler(sharingController.batchShare));
router.post('/:id/share', validateUuidParam(), validate(shareSchema), asyncHandler(sharingController.share));
router.delete('/:id/share/:userId', validateUuidParam(), asyncHandler(sharingController.unshare));
router.put('/:id/share/:userId', validateUuidParam(), validate(updatePermissionSchema), asyncHandler(sharingController.updatePermission));
router.get('/:id/shares', validateUuidParam(), asyncHandler(sharingController.listShares));

export default router;
