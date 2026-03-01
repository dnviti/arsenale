import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { requireTenant, requireTenantRole } from '../middleware/tenant.middleware';
import * as gatewayController from '../controllers/gateway.controller';

const router = Router();

router.use(authenticate);
router.use(requireTenant);

router.get('/', gatewayController.list);
router.post('/', requireTenantRole('ADMIN'), gatewayController.create);
router.put('/:id', requireTenantRole('ADMIN'), gatewayController.update);
router.delete('/:id', requireTenantRole('ADMIN'), gatewayController.remove);
router.post('/:id/test', gatewayController.testConnectivity);

export default router;
