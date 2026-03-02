import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import * as secretController from '../controllers/secret.controller';

const router = Router();

router.use(authenticate);

// Tenant vault management (before /:id to avoid param collision)
router.post('/tenant-vault/init', secretController.initTenantVault);
router.post('/tenant-vault/distribute', secretController.distributeTenantKey);
router.get('/tenant-vault/status', secretController.tenantVaultStatus);

// CRUD
router.get('/', secretController.list);
router.post('/', secretController.create);
router.get('/:id', secretController.getOne);
router.put('/:id', secretController.update);
router.delete('/:id', secretController.remove);

// Versions
router.get('/:id/versions', secretController.listVersions);
router.post('/:id/versions/:version/restore', secretController.restoreVersion);

// Sharing
router.post('/:id/share', secretController.share);
router.delete('/:id/share/:userId', secretController.unshare);
router.put('/:id/share/:userId', secretController.updateSharePermission);
router.get('/:id/shares', secretController.listShares);

export default router;
