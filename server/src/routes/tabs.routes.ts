import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { validate } from '../middleware/validate.middleware';
import { syncTabsSchema } from '../schemas/tabs.schemas';
import { asyncHandler } from '../middleware/asyncHandler';
import * as tabsController from '../controllers/tabs.controller';

const router = Router();

router.use(authenticate);
router.get('/', asyncHandler(tabsController.getTabs));
router.put('/', validate(syncTabsSchema), asyncHandler(tabsController.syncTabs));
router.delete('/', asyncHandler(tabsController.clearTabs));

export default router;
