import prisma, { Prisma } from '../lib/prisma';
import bcrypt from 'bcrypt';
import {
  generateSalt,
  deriveKeyFromPassword,
  encryptMasterKey,
  decryptMasterKey,
  lockVault,
} from './crypto.service';
import { AppError } from '../middleware/error.middleware';
const BCRYPT_ROUNDS = 12;
const MAX_AVATAR_SIZE = 200 * 1024; // ~200KB base64

export async function getProfile(userId: string) {
  const user = await prisma.user.findUnique({
    where: { id: userId },
    select: { id: true, email: true, username: true, avatarData: true, sshDefaults: true, rdpDefaults: true, createdAt: true },
  });
  if (!user) throw new AppError('User not found', 404);
  return user;
}

export async function updateProfile(
  userId: string,
  data: { username?: string; email?: string }
) {
  if (data.email) {
    const existing = await prisma.user.findUnique({ where: { email: data.email } });
    if (existing && existing.id !== userId) {
      throw new AppError('Email already in use', 409);
    }
  }

  const user = await prisma.user.update({
    where: { id: userId },
    data: {
      ...(data.username !== undefined && { username: data.username || null }),
      ...(data.email !== undefined && { email: data.email }),
    },
    select: { id: true, email: true, username: true, avatarData: true },
  });

  return user;
}

export async function changePassword(
  userId: string,
  oldPassword: string,
  newPassword: string
) {
  const user = await prisma.user.findUnique({ where: { id: userId } });
  if (!user) throw new AppError('User not found', 404);

  // 1. Verify old password
  const valid = await bcrypt.compare(oldPassword, user.passwordHash);
  if (!valid) throw new AppError('Current password is incorrect', 401);

  // 2. Hash new password
  const newPasswordHash = await bcrypt.hash(newPassword, BCRYPT_ROUNDS);

  // 3. Derive old key and decrypt master key
  const oldDerivedKey = await deriveKeyFromPassword(oldPassword, user.vaultSalt);
  const masterKey = decryptMasterKey(
    {
      ciphertext: user.encryptedVaultKey,
      iv: user.vaultKeyIV,
      tag: user.vaultKeyTag,
    },
    oldDerivedKey
  );

  // 4. Generate new salt and derive new key
  const newVaultSalt = generateSalt();
  const newDerivedKey = await deriveKeyFromPassword(newPassword, newVaultSalt);

  // 5. Re-encrypt master key with new derived key
  const newEncryptedVault = encryptMasterKey(masterKey, newDerivedKey);

  // 6. Update DB
  await prisma.user.update({
    where: { id: userId },
    data: {
      passwordHash: newPasswordHash,
      vaultSalt: newVaultSalt,
      encryptedVaultKey: newEncryptedVault.ciphertext,
      vaultKeyIV: newEncryptedVault.iv,
      vaultKeyTag: newEncryptedVault.tag,
    },
  });

  // 7. Zero out sensitive buffers
  masterKey.fill(0);
  oldDerivedKey.fill(0);
  newDerivedKey.fill(0);

  // 8. Lock vault — user must re-unlock with new password
  lockVault(userId);

  // 9. Invalidate all refresh tokens (force re-login on all devices)
  await prisma.refreshToken.deleteMany({ where: { userId } });

  return { success: true };
}

export async function updateSshDefaults(userId: string, sshDefaults: Prisma.InputJsonValue) {
  const user = await prisma.user.update({
    where: { id: userId },
    data: { sshDefaults },
    select: { id: true, sshDefaults: true },
  });
  return user;
}

export async function updateRdpDefaults(userId: string, rdpDefaults: Prisma.InputJsonValue) {
  const user = await prisma.user.update({
    where: { id: userId },
    data: { rdpDefaults },
    select: { id: true, rdpDefaults: true },
  });
  return user;
}

export async function uploadAvatar(userId: string, avatarData: string) {
  if (!avatarData.startsWith('data:image/')) {
    throw new AppError('Invalid image format', 400);
  }
  if (avatarData.length > MAX_AVATAR_SIZE) {
    throw new AppError('Avatar image too large (max 200KB)', 400);
  }

  const user = await prisma.user.update({
    where: { id: userId },
    data: { avatarData },
    select: { id: true, avatarData: true },
  });

  return user;
}
