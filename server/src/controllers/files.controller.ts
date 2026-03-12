import { Response } from 'express';
import { AuthRequest, assertAuthenticated } from '../types';
import * as fileService from '../services/file.service';
import { AppError } from '../middleware/error.middleware';
import type { FileNameInput } from '../schemas/files.schemas';

export async function list(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const files = await fileService.listFiles(req.user.userId);
  res.json(files);
}

export async function download(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const { name } = req.params as FileNameInput;
  const filePath = await fileService.getFilePath(req.user.userId, name);
  res.download(filePath, name);
}

export async function upload(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  if (!req.file) {
    throw new AppError('No file uploaded', 400);
  }
  const files = await fileService.listFiles(req.user.userId);
  res.status(201).json(files);
}

export async function remove(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const { name } = req.params as FileNameInput;
  await fileService.deleteFile(req.user.userId, name);
  res.json({ deleted: true });
}
