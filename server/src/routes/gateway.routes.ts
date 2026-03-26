import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { requireTenant, requireTenantRoleAny, requirePermission } from '../middleware/tenant.middleware';
import { validate, validateUuidParam } from '../middleware/validate.middleware';
import {
  createGatewaySchema, updateGatewaySchema, scaleSchema, scalingConfigSchema,
  rotationPolicySchema, createTemplateSchema, updateTemplateSchema,
} from '../schemas/gateway.schemas';
import * as gatewayController from '../controllers/gateway.controller';
import { asyncHandler } from '../middleware/asyncHandler';

const router = Router();

router.use(authenticate);
router.use(requireTenant);

router.get('/', asyncHandler(gatewayController.list));
router.post('/', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validate(createGatewaySchema), asyncHandler(gatewayController.create));

// SSH key pair management (must be before /:id routes)
router.post('/ssh-keypair', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), asyncHandler(gatewayController.generateSshKeyPair));
router.get('/ssh-keypair', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), asyncHandler(gatewayController.getSshPublicKey));
router.get('/ssh-keypair/private', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), asyncHandler(gatewayController.downloadSshPrivateKey));
router.post('/ssh-keypair/rotate', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), asyncHandler(gatewayController.rotateSshKeyPair));
router.patch('/ssh-keypair/rotation', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validate(rotationPolicySchema), asyncHandler(gatewayController.updateRotationPolicy));
router.get('/ssh-keypair/rotation', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), asyncHandler(gatewayController.getRotationStatus));

// Tunnel fleet overview (must be before /:id routes)
router.get('/tunnel-overview', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), asyncHandler(gatewayController.tunnelOverview));

// Gateway templates (must be before /:id routes)
router.get('/templates', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), asyncHandler(gatewayController.listTemplates));
router.post('/templates', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validate(createTemplateSchema), asyncHandler(gatewayController.createTemplate));
router.put('/templates/:templateId', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validate(updateTemplateSchema), asyncHandler(gatewayController.updateTemplate));
router.delete('/templates/:templateId', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), asyncHandler(gatewayController.deleteTemplate));
router.post('/templates/:templateId/deploy', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), asyncHandler(gatewayController.deployFromTemplate));

router.put('/:id', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), validate(updateGatewaySchema), asyncHandler(gatewayController.update));
router.delete('/:id', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.remove));
router.post('/:id/test', validateUuidParam(), asyncHandler(gatewayController.testConnectivity));
router.post('/:id/push-key', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.pushKey));

// Managed gateway lifecycle
router.post('/:id/deploy', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.deploy));
router.delete('/:id/deploy', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.undeploy));
router.post('/:id/scale', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), validate(scaleSchema), asyncHandler(gatewayController.scale));
router.get('/:id/instances', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.listInstances));
router.post('/:id/instances/:instanceId/restart', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.restartInstance));
router.get('/:id/instances/:instanceId/logs', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.getInstanceLogs));

// Auto-scaling configuration
router.get('/:id/scaling', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.getScalingStatus));
router.put('/:id/scaling', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), validate(scalingConfigSchema), asyncHandler(gatewayController.updateScalingConfig));

// Zero-trust tunnel token management
router.post('/:id/tunnel-token', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.generateTunnelToken));
router.delete('/:id/tunnel-token', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.revokeTunnelToken));
router.post('/:id/tunnel-disconnect', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.forceDisconnectTunnel));
router.get('/:id/tunnel-events', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.getTunnelEvents));
router.get('/:id/tunnel-metrics', requireTenantRoleAny('OPERATOR', 'ADMIN', 'OWNER'), requirePermission('canManageGateways'), validateUuidParam(), asyncHandler(gatewayController.getTunnelMetrics));

export default router;
