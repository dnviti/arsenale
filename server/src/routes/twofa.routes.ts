import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { validate } from '../middleware/validate.middleware';
import { asyncHandler } from '../middleware/asyncHandler';
import { totpCodeSchema } from '../schemas/mfa.schemas';
import * as twofaController from '../controllers/twofa.controller';

const router = Router();
router.use(authenticate);

router.post('/setup', asyncHandler(twofaController.setup));
router.post('/verify', validate(totpCodeSchema, 'body', 'Invalid code format'), asyncHandler(twofaController.verify));
router.post('/disable', validate(totpCodeSchema, 'body', 'Invalid code format'), asyncHandler(twofaController.disable));
router.get('/status', asyncHandler(twofaController.status));

export default router;
