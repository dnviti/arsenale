import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { requireTenantRole } from '../middleware/tenant.middleware';
import * as adminController from '../controllers/admin.controller';

const router = Router();

router.use(authenticate);

router.get('/email/status', requireTenantRole('ADMIN'), adminController.emailStatus);
router.post('/email/test', requireTenantRole('ADMIN'), adminController.sendTestEmail);

export default router;
