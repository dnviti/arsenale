import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { validate, validateUuidParam } from '../middleware/validate.middleware';
import { notificationQuerySchema } from '../schemas/notification.schemas';
import { asyncHandler } from '../middleware/asyncHandler';
import * as notificationController from '../controllers/notification.controller';

const router = Router();

router.use(authenticate);
router.get('/', validate(notificationQuerySchema, 'query'), asyncHandler(notificationController.list));
router.put('/read-all', asyncHandler(notificationController.markAllRead));
router.put('/:id/read', validateUuidParam(), asyncHandler(notificationController.markRead));
router.delete('/:id', validateUuidParam(), asyncHandler(notificationController.remove));

export default router;
