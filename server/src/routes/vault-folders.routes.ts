import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { validate, validateUuidParam } from '../middleware/validate.middleware';
import { createVaultFolderSchema, updateVaultFolderSchema } from '../schemas/vaultFolder.schemas';
import { asyncHandler } from '../middleware/asyncHandler';
import * as vaultFoldersController from '../controllers/vault-folders.controller';

const router = Router();

router.use(authenticate);
router.get('/', asyncHandler(vaultFoldersController.list));
router.post('/', validate(createVaultFolderSchema), asyncHandler(vaultFoldersController.create));
router.put('/:id', validateUuidParam(), validate(updateVaultFolderSchema), asyncHandler(vaultFoldersController.update));
router.delete('/:id', validateUuidParam(), asyncHandler(vaultFoldersController.remove));

export default router;
