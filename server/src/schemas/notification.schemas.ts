import { z } from 'zod';

export const notificationQuerySchema = z.object({
  limit: z.coerce.number().int().min(1).max(100).default(50),
  offset: z.coerce.number().int().min(0).default(0),
});

export type NotificationQueryInput = z.infer<typeof notificationQuerySchema>;

const notificationTypeValues = [
  'CONNECTION_SHARED',
  'SHARE_PERMISSION_UPDATED',
  'SHARE_REVOKED',
  'SECRET_SHARED',
  'SECRET_SHARE_REVOKED',
  'SECRET_EXPIRING',
  'SECRET_EXPIRED',
  'TENANT_INVITATION',
  'RECORDING_READY',
  'IMPOSSIBLE_TRAVEL_DETECTED',
] as const;

export const preferenceUpdateSchema = z.object({
  inApp: z.boolean().optional(),
  email: z.boolean().optional(),
});

export const bulkPreferenceUpdateSchema = z.object({
  preferences: z.array(
    z.object({
      type: z.enum(notificationTypeValues),
      inApp: z.boolean().optional(),
      email: z.boolean().optional(),
    })
  ).min(1).max(50),
});

export type PreferenceUpdateInput = z.infer<typeof preferenceUpdateSchema>;
export type BulkPreferenceUpdateInput = z.infer<typeof bulkPreferenceUpdateSchema>;
