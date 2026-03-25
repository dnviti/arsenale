import { z } from 'zod';

export const createCheckoutSchema = z.object({
  secretId: z.string().uuid().optional(),
  connectionId: z.string().uuid().optional(),
  durationMinutes: z.number().int().min(1).max(1440),
  reason: z.string().max(500).optional(),
}).superRefine((data, ctx) => {
  const hasSecret = !!data.secretId;
  const hasConnection = !!data.connectionId;

  if (hasSecret && hasConnection) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'Provide either secretId or connectionId, not both',
      path: ['secretId'],
    });
  } else if (!hasSecret && !hasConnection) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'Provide one of secretId or connectionId',
      path: ['secretId'],
    });
  }
});

export type CreateCheckoutInput = z.infer<typeof createCheckoutSchema>;

export const listCheckoutSchema = z.object({
  role: z.enum(['requester', 'approver', 'all']).optional().default('all'),
  status: z.enum(['PENDING', 'APPROVED', 'REJECTED', 'EXPIRED', 'CHECKED_IN']).optional(),
  limit: z.coerce.number().int().min(1).max(100).optional().default(50),
  offset: z.coerce.number().int().min(0).optional().default(0),
});

export type ListCheckoutInput = z.infer<typeof listCheckoutSchema>;
