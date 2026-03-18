import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { validate, validateUuidParam } from '../middleware/validate.middleware';
import {
  notificationQuerySchema,
  preferenceUpdateSchema,
  bulkPreferenceUpdateSchema,
} from '../schemas/notification.schemas';
import { asyncHandler } from '../middleware/asyncHandler';
import * as notificationController from '../controllers/notification.controller';

const router = Router();

router.use(authenticate);

// Notification preferences (must be registered before /:id routes)
router.get('/preferences', asyncHandler(notificationController.getPreferences));
router.put('/preferences', validate(bulkPreferenceUpdateSchema, 'body'), asyncHandler(notificationController.bulkUpdatePreferences));
router.put('/preferences/:type', validate(preferenceUpdateSchema, 'body'), asyncHandler(notificationController.updatePreference));

// Notification list and actions
router.get('/', validate(notificationQuerySchema, 'query'), asyncHandler(notificationController.list));
router.put('/read-all', asyncHandler(notificationController.markAllRead));
router.put('/:id/read', validateUuidParam(), asyncHandler(notificationController.markRead));
router.delete('/:id', validateUuidParam(), asyncHandler(notificationController.remove));

export default router;
