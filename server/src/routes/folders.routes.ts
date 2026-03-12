import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { validate, validateUuidParam } from '../middleware/validate.middleware';
import { createFolderSchema, updateFolderSchema } from '../schemas/folder.schemas';
import { asyncHandler } from '../middleware/asyncHandler';
import * as foldersController from '../controllers/folders.controller';

const router = Router();

router.use(authenticate);
router.get('/', asyncHandler(foldersController.list));
router.post('/', validate(createFolderSchema), asyncHandler(foldersController.create));
router.put('/:id', validateUuidParam(), validate(updateFolderSchema), asyncHandler(foldersController.update));
router.delete('/:id', validateUuidParam(), asyncHandler(foldersController.remove));

export default router;
