import { Router } from 'express';
import * as authController from '../controllers/auth.controller';
import { smsLoginRateLimiter } from '../middleware/smsRateLimit.middleware';

const router = Router();

router.post('/register', authController.register);
router.get('/verify-email', authController.verifyEmail);
router.post('/resend-verification', authController.resendVerification);
router.post('/login', authController.login);
router.post('/verify-totp', authController.verifyTotp);
router.post('/request-sms-code', smsLoginRateLimiter, authController.requestSmsCode);
router.post('/verify-sms', authController.verifySms);
router.post('/refresh', authController.refresh);
router.post('/logout', authController.logout);

export default router;
