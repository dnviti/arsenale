import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { validate } from '../middleware/validate.middleware';
import { vaultUnlockRateLimiter, vaultMfaRateLimiter } from '../middleware/vaultRateLimit.middleware';
import { unlockSchema, codeSchema, credentialSchema, revealSchema, autoLockSchema } from '../schemas/vault.schemas';
import { asyncHandler } from '../middleware/asyncHandler';
import * as vaultController from '../controllers/vault.controller';

const router = Router();

router.use(authenticate);
router.post('/unlock', vaultUnlockRateLimiter, validate(unlockSchema), asyncHandler(vaultController.unlock));
router.post('/lock', vaultController.lock);
router.get('/status', asyncHandler(vaultController.status));
router.post('/reveal-password', validate(revealSchema), asyncHandler(vaultController.revealPassword));

// MFA-based vault unlock
router.post('/unlock-mfa/totp', vaultMfaRateLimiter, validate(codeSchema), asyncHandler(vaultController.unlockWithTotp));
router.post('/unlock-mfa/webauthn-options', vaultMfaRateLimiter, asyncHandler(vaultController.requestWebAuthnOptions));
router.post('/unlock-mfa/webauthn', vaultMfaRateLimiter, validate(credentialSchema), asyncHandler(vaultController.unlockWithWebAuthn));
router.post('/unlock-mfa/request-sms', vaultMfaRateLimiter, asyncHandler(vaultController.requestSmsCode));
router.post('/unlock-mfa/sms', vaultMfaRateLimiter, validate(codeSchema), asyncHandler(vaultController.unlockWithSms));

// Vault auto-lock preference
router.get('/auto-lock', asyncHandler(vaultController.getAutoLock));
router.put('/auto-lock', validate(autoLockSchema), asyncHandler(vaultController.setAutoLock));

export default router;
