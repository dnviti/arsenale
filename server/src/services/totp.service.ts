import { generateSecret, generateURI, verifySync } from 'otplib';
import prisma from '../lib/prisma';
import { AppError } from '../middleware/error.middleware';

const APP_NAME = 'Remote Desktop Manager';

export function generateSetup(email: string): { secret: string; otpauthUri: string } {
  const secret = generateSecret();
  const otpauthUri = generateURI({
    issuer: APP_NAME,
    label: email,
    secret,
    algorithm: 'sha1',
    digits: 6,
    period: 30,
  });
  return { secret, otpauthUri };
}

export async function storeSetupSecret(userId: string, secret: string): Promise<void> {
  await prisma.user.update({
    where: { id: userId },
    data: { totpSecret: secret },
  });
}

export async function verifyAndEnable(userId: string, code: string): Promise<void> {
  const user = await prisma.user.findUnique({
    where: { id: userId },
    select: { totpSecret: true, totpEnabled: true },
  });
  if (!user) throw new AppError('User not found', 404);
  if (user.totpEnabled) throw new AppError('2FA is already enabled', 400);
  if (!user.totpSecret) throw new AppError('2FA setup not initiated', 400);

  if (!checkCode(user.totpSecret, code)) {
    throw new AppError('Invalid TOTP code', 400);
  }

  await prisma.user.update({
    where: { id: userId },
    data: { totpEnabled: true },
  });
}

export async function disable(userId: string, code: string): Promise<void> {
  const user = await prisma.user.findUnique({
    where: { id: userId },
    select: { totpSecret: true, totpEnabled: true },
  });
  if (!user) throw new AppError('User not found', 404);
  if (!user.totpEnabled || !user.totpSecret) throw new AppError('2FA is not enabled', 400);

  if (!checkCode(user.totpSecret, code)) {
    throw new AppError('Invalid TOTP code', 400);
  }

  await prisma.user.update({
    where: { id: userId },
    data: { totpEnabled: false, totpSecret: null },
  });
}

function checkCode(secret: string, code: string): boolean {
  const result = verifySync({ secret, token: code });
  return result.valid;
}

export function verifyCode(secret: string, code: string): boolean {
  return checkCode(secret, code);
}
