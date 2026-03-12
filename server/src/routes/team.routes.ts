import { Router } from 'express';
import { authenticate } from '../middleware/auth.middleware';
import { requireTenant } from '../middleware/tenant.middleware';
import { requireTeamMember, requireTeamRole } from '../middleware/team.middleware';
import { validate, validateUuidParam } from '../middleware/validate.middleware';
import { createTeamSchema, updateTeamSchema, addMemberSchema, updateMemberRoleSchema } from '../schemas/team.schemas';
import * as teamController from '../controllers/team.controller';
import { asyncHandler } from '../middleware/asyncHandler';

const router = Router();

router.use(authenticate);
router.use(requireTenant);

// Team CRUD
router.post('/', validate(createTeamSchema), asyncHandler(teamController.createTeam));
router.get('/', asyncHandler(teamController.listTeams));
router.get('/:id', validateUuidParam(), requireTeamMember, asyncHandler(teamController.getTeam));
router.put('/:id', validateUuidParam(), requireTeamMember, requireTeamRole('TEAM_ADMIN'), validate(updateTeamSchema), asyncHandler(teamController.updateTeam));
router.delete('/:id', validateUuidParam(), requireTeamMember, requireTeamRole('TEAM_ADMIN', { allowTenantAdmin: true }), asyncHandler(teamController.deleteTeam));

// Member management
router.get('/:id/members', validateUuidParam(), requireTeamMember, asyncHandler(teamController.listMembers));
router.post('/:id/members', validateUuidParam(), requireTeamMember, requireTeamRole('TEAM_ADMIN'), validate(addMemberSchema), asyncHandler(teamController.addMember));
router.put('/:id/members/:userId', validateUuidParam(), requireTeamMember, requireTeamRole('TEAM_ADMIN'), validateUuidParam('userId'), validate(updateMemberRoleSchema), asyncHandler(teamController.updateMemberRole));
router.delete('/:id/members/:userId', validateUuidParam(), requireTeamMember, requireTeamRole('TEAM_ADMIN', { allowTenantAdmin: true }), validateUuidParam('userId'), asyncHandler(teamController.removeMember));

export default router;
