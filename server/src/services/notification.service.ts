import prisma, { NotificationType } from '../lib/prisma';
import { logger } from '../utils/logger';
import { shouldDeliver } from './notificationPreference.service';
import { sendEmail } from './email';
import { buildNotificationEmail } from './email/templates/notification';

export { NotificationType };

export interface CreateNotificationInput {
  userId: string;
  type: NotificationType;
  message: string;
  relatedId?: string;
}

export interface NotificationEntry {
  id: string;
  type: NotificationType;
  message: string;
  read: boolean;
  relatedId: string | null;
  createdAt: Date;
}

export interface PaginatedNotifications {
  data: NotificationEntry[];
  total: number;
  unreadCount: number;
}

// Simple in-memory rate limiter: max 10 emails per userId+type per hour.
const emailRateMap = new Map<string, { count: number; windowStart: number }>();
const EMAIL_RATE_LIMIT = 10;
const EMAIL_RATE_WINDOW_MS = 60 * 60 * 1000;

function checkEmailRateLimit(userId: string, type: NotificationType): boolean {
  const key = `${userId}:${type}`;
  const now = Date.now();
  const entry = emailRateMap.get(key);

  if (!entry || now - entry.windowStart > EMAIL_RATE_WINDOW_MS) {
    emailRateMap.set(key, { count: 1, windowStart: now });
    return true;
  }

  if (entry.count >= EMAIL_RATE_LIMIT) return false;

  entry.count++;
  return true;
}

async function dispatchEmail(input: CreateNotificationInput): Promise<void> {
  try {
    const emailEnabled = await shouldDeliver(input.userId, input.type, 'email');
    if (!emailEnabled) return;

    if (!checkEmailRateLimit(input.userId, input.type)) {
      logger.warn(`Email rate limit reached for user=${input.userId} type=${input.type}`);
      return;
    }

    const user = await prisma.user.findUnique({
      where: { id: input.userId },
      select: { email: true },
    });
    if (!user) return;

    const { subject, html, text } = buildNotificationEmail(input.type, input.message);
    await sendEmail({ to: user.email, subject, html, text });
  } catch (err) {
    logger.error('Failed to dispatch notification email:', err);
  }
}

export async function createNotification(
  input: CreateNotificationInput
): Promise<NotificationEntry> {
  const inAppEnabled = await shouldDeliver(input.userId, input.type, 'inApp');
  if (!inAppEnabled) {
    // Still dispatch email even if in-app is disabled for this type
    dispatchEmail(input).catch((err) => logger.error('Email dispatch error:', err));
    // Return a synthetic object so callers don't break
    return {
      id: '',
      type: input.type,
      message: input.message,
      read: false,
      relatedId: input.relatedId ?? null,
      createdAt: new Date(),
    };
  }

  const notification = await prisma.notification.create({
    data: {
      userId: input.userId,
      type: input.type,
      message: input.message,
      relatedId: input.relatedId ?? null,
    },
    select: {
      id: true,
      type: true,
      message: true,
      read: true,
      relatedId: true,
      createdAt: true,
    },
  });

  // Fire-and-forget email dispatch
  dispatchEmail(input).catch((err) => logger.error('Email dispatch error:', err));

  return notification;
}

/**
 * Fire-and-forget variant — used from sharing service where we don't want to block.
 */
export function createNotificationAsync(input: CreateNotificationInput): void {
  createNotification(input).catch((err) => {
    logger.error('Failed to create notification:', err);
  });
}

export async function listNotifications(
  userId: string,
  limit = 50,
  offset = 0
): Promise<PaginatedNotifications> {
  const safeLimit = Math.min(limit, 100);

  const [data, total, unreadCount] = await Promise.all([
    prisma.notification.findMany({
      where: { userId },
      orderBy: { createdAt: 'desc' },
      skip: offset,
      take: safeLimit,
      select: {
        id: true,
        type: true,
        message: true,
        read: true,
        relatedId: true,
        createdAt: true,
      },
    }),
    prisma.notification.count({ where: { userId } }),
    prisma.notification.count({ where: { userId, read: false } }),
  ]);

  return { data, total, unreadCount };
}

export async function markAsRead(notificationId: string, userId: string) {
  return prisma.notification.updateMany({
    where: { id: notificationId, userId },
    data: { read: true },
  });
}

export async function markAllAsRead(userId: string) {
  return prisma.notification.updateMany({
    where: { userId, read: false },
    data: { read: true },
  });
}

export async function deleteNotification(notificationId: string, userId: string) {
  return prisma.notification.deleteMany({
    where: { id: notificationId, userId },
  });
}
