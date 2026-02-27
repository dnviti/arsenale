import { Router } from 'express';
import * as authController from '../controllers/auth.controller';

const router = Router();

router.post('/register', authController.register);
router.get('/verify-email', authController.verifyEmail);
router.post('/resend-verification', authController.resendVerification);
router.post('/login', authController.login);
router.post('/verify-totp', authController.verifyTotp);
router.post('/refresh', authController.refresh);
router.post('/logout', authController.logout);

export default router;
