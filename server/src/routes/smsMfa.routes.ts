import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { smsRateLimiter } from '../middleware/smsRateLimit.middleware';
import { validate } from '../middleware/validate.middleware';
import { asyncHandler } from '../middleware/asyncHandler';
import { setupPhoneSchema, totpCodeSchema } from '../schemas/mfa.schemas';
import * as smsMfaController from '../controllers/smsMfa.controller';

const router = Router();
router.use(authenticate);

router.post('/setup-phone', smsRateLimiter, validate(setupPhoneSchema, 'body', 'Invalid phone number format'), asyncHandler(smsMfaController.setupPhone));
router.post('/verify-phone', validate(totpCodeSchema, 'body', 'Invalid code format'), asyncHandler(smsMfaController.verifyPhone));
router.post('/enable', asyncHandler(smsMfaController.enable));
router.post('/send-disable-code', smsRateLimiter, asyncHandler(smsMfaController.sendDisableCode));
router.post('/disable', validate(totpCodeSchema, 'body', 'Invalid code format'), asyncHandler(smsMfaController.disable));
router.get('/status', asyncHandler(smsMfaController.status));

export default router;
