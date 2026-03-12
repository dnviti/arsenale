import { z } from 'zod';

export const createSyncProfileSchema = z.object({
  name: z.string().min(1).max(100),
  provider: z.enum(['NETBOX']),
  url: z.string().url().max(500),
  apiToken: z.string().min(1).max(500),
  filters: z.record(z.string(), z.string()).optional(),
  platformMapping: z.record(z.string(), z.string()).optional(),
  defaultProtocol: z.enum(['SSH', 'RDP', 'VNC']).optional(),
  defaultPort: z.record(z.string(), z.number().int().min(1).max(65535)).optional(),
  conflictStrategy: z.enum(['update', 'skip', 'overwrite']).optional(),
  cronExpression: z.string().max(100).optional(),
  teamId: z.string().uuid().optional(),
});
export type CreateSyncProfileInput = z.infer<typeof createSyncProfileSchema>;

export const updateSyncProfileSchema = z.object({
  name: z.string().min(1).max(100).optional(),
  url: z.string().url().max(500).optional(),
  apiToken: z.string().min(1).max(500).optional(),
  filters: z.record(z.string(), z.string()).optional(),
  platformMapping: z.record(z.string(), z.string()).optional(),
  defaultProtocol: z.enum(['SSH', 'RDP', 'VNC']).optional(),
  defaultPort: z.record(z.string(), z.number().int().min(1).max(65535)).optional(),
  conflictStrategy: z.enum(['update', 'skip', 'overwrite']).optional(),
  cronExpression: z.string().max(100).nullable().optional(),
  enabled: z.boolean().optional(),
  teamId: z.string().uuid().nullable().optional(),
});
export type UpdateSyncProfileInput = z.infer<typeof updateSyncProfileSchema>;

export const triggerSyncSchema = z.object({
  dryRun: z.boolean().default(false),
});
export type TriggerSyncInput = z.infer<typeof triggerSyncSchema>;
