import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { validate, validateUuidParam } from '../middleware/validate.middleware';
import { createConnectionSchema, updateConnectionSchema } from '../schemas/connection.schemas';
import { asyncHandler } from '../middleware/asyncHandler';
import * as connectionsController from '../controllers/connections.controller';

const router = Router();

router.use(authenticate);
router.get('/', asyncHandler(connectionsController.list));
router.post('/', validate(createConnectionSchema), asyncHandler(connectionsController.create));
router.get('/:id', validateUuidParam(), asyncHandler(connectionsController.getOne));
router.put('/:id', validateUuidParam(), validate(updateConnectionSchema), asyncHandler(connectionsController.update));
router.delete('/:id', validateUuidParam(), asyncHandler(connectionsController.remove));
router.patch('/:id/favorite', validateUuidParam(), asyncHandler(connectionsController.toggleFavorite));

export default router;
