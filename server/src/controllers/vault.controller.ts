import { Response } from 'express';
import { AuthRequest, assertAuthenticated } from '../types';
import * as vaultService from '../services/vault.service';
import * as auditService from '../services/audit.service';
import { getClientIp } from '../utils/ip';
import type { UnlockInput, CodeInput, CredentialInput, RevealInput, AutoLockInput, RecoverWithKeyInput, ExplicitResetInput } from '../schemas/vault.schemas';

export async function unlock(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const { password } = req.body as UnlockInput;
  const result = await vaultService.unlockVault(req.user.userId, password);
  auditService.log({ userId: req.user.userId, action: 'VAULT_UNLOCK', ipAddress: getClientIp(req) });
  res.json(result);
}

export function lock(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const result = vaultService.lockVault(req.user.userId);
  auditService.log({ userId: req.user.userId, action: 'VAULT_LOCK', ipAddress: getClientIp(req) });
  res.json(result);
}

export async function status(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const result = await vaultService.getVaultStatus(req.user.userId);
  res.json(result);
}

export async function unlockWithTotp(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const { code } = req.body as CodeInput;
  const result = await vaultService.unlockVaultWithTotp(req.user.userId, code);
  auditService.log({ userId: req.user.userId, action: 'VAULT_UNLOCK', ipAddress: getClientIp(req), details: { method: 'totp' } });
  res.json(result);
}

export async function requestWebAuthnOptions(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const options = await vaultService.requestVaultWebAuthnOptions(req.user.userId);
  res.json(options);
}

export async function unlockWithWebAuthn(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const { credential } = req.body as CredentialInput;
  const result = await vaultService.unlockVaultWithWebAuthn(req.user.userId, credential);
  auditService.log({ userId: req.user.userId, action: 'VAULT_UNLOCK', ipAddress: getClientIp(req), details: { method: 'webauthn' } });
  res.json(result);
}

export async function requestSmsCode(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  await vaultService.requestVaultSmsCode(req.user.userId);
  res.json({ sent: true });
}

export async function unlockWithSms(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const { code } = req.body as CodeInput;
  const result = await vaultService.unlockVaultWithSms(req.user.userId, code);
  auditService.log({ userId: req.user.userId, action: 'VAULT_UNLOCK', ipAddress: getClientIp(req), details: { method: 'sms' } });
  res.json(result);
}

export async function getAutoLock(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const result = await vaultService.getAutoLockPreference(req.user.userId);
  res.json(result);
}

export async function setAutoLock(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const { autoLockMinutes } = req.body as AutoLockInput;
  const result = await vaultService.setAutoLockPreference(req.user.userId, autoLockMinutes);
  res.json(result);
}

export async function revealPassword(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const { connectionId, password } = req.body as RevealInput;
  const result = await vaultService.revealPassword(
    req.user.userId,
    connectionId,
    password || ''
  );
  auditService.log({
    userId: req.user.userId, action: 'PASSWORD_REVEAL',
    targetType: 'Connection', targetId: connectionId,
    ipAddress: getClientIp(req),
  });
  res.json(result);
}

// Vault recovery (after password reset)

export async function recoveryStatus(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const result = await vaultService.getVaultRecoveryStatus(req.user.userId);
  res.json(result);
}

export async function recoverWithKey(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const { recoveryKey, password } = req.body as RecoverWithKeyInput;
  const result = await vaultService.recoverVaultWithKey(req.user.userId, recoveryKey, password);
  auditService.log({
    userId: req.user.userId,
    action: 'VAULT_RECOVERED',
    ipAddress: getClientIp(req),
    details: { method: 'recovery_key' },
  });
  res.json(result);
}

export async function explicitReset(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const { password } = req.body as ExplicitResetInput;
  const result = await vaultService.explicitVaultReset(req.user.userId, password);
  auditService.log({
    userId: req.user.userId,
    action: 'VAULT_EXPLICIT_RESET',
    ipAddress: getClientIp(req),
    details: { reason: 'user_requested' },
  });
  res.json(result);
}
