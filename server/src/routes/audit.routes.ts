import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import * as auditController from '../controllers/audit.controller';

const router = Router();

router.use(authenticate);
router.get('/', auditController.list);

export default router;
