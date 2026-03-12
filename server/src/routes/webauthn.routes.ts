import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { validate, validateUuidParam } from '../middleware/validate.middleware';
import { asyncHandler } from '../middleware/asyncHandler';
import { webauthnRegisterSchema, webauthnRenameSchema } from '../schemas/mfa.schemas';
import * as webauthnController from '../controllers/webauthn.controller';

const router = Router();
router.use(authenticate);

router.post('/registration-options', asyncHandler(webauthnController.registrationOptions));
router.post('/register', validate(webauthnRegisterSchema, 'body', 'Invalid registration data'), asyncHandler(webauthnController.register));
router.get('/credentials', asyncHandler(webauthnController.getCredentials));
router.delete('/credentials/:id', validateUuidParam(), asyncHandler(webauthnController.removeCredential));
router.patch('/credentials/:id', validateUuidParam(), validate(webauthnRenameSchema, 'body', 'Invalid name'), asyncHandler(webauthnController.renameCredential));
router.get('/status', asyncHandler(webauthnController.status));

export default router;
