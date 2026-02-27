import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { requireTenant } from '../middleware/tenant.middleware';
import * as userController from '../controllers/user.controller';

const router = Router();

router.use(authenticate);

router.get('/search', requireTenant, userController.search);
router.get('/profile', userController.getProfile);
router.put('/profile', userController.updateProfile);
router.put('/password', userController.changePassword);
router.put('/ssh-defaults', userController.updateSshDefaults);
router.put('/rdp-defaults', userController.updateRdpDefaults);
router.post('/avatar', userController.uploadAvatar);

export default router;
