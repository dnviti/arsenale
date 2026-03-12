import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { validate } from '../middleware/validate.middleware';
import { asyncHandler } from '../middleware/asyncHandler';
import { ipParamSchema } from '../schemas/geoip.schemas';
import * as geoipController from '../controllers/geoip.controller';

const router = Router();

router.use(authenticate);
router.get('/:ip', validate(ipParamSchema, 'params'), asyncHandler(geoipController.lookupIp));

export default router;
