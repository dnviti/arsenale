import { describe, expect, it } from 'vitest';
import type { ConnectionData } from '../../api/connections.api';
import {
  applyConnectionTypeChange,
  buildConnectionInput,
  buildConnectionUpdate,
  connectionToIntakeState,
  emptyConnectionIntakeState,
  inferDbProtocol,
  validateConnectionIntake,
} from './connectionIntake';

function buildConnection(overrides: Partial<ConnectionData> = {}): ConnectionData {
  return {
    id: 'connection-1',
    name: 'Database tunnel',
    type: 'DB_TUNNEL',
    host: 'bastion.example.com',
    port: 22,
    folderId: null,
    description: null,
    isFavorite: false,
    enableDrive: false,
    defaultCredentialMode: null,
    transferRetentionPolicy: null,
    targetDbHost: 'db.internal',
    targetDbPort: 5432,
    dbType: 'mysql',
    isOwner: true,
    createdAt: '2026-04-15T00:00:00Z',
    updatedAt: '2026-04-15T00:00:00Z',
    ...overrides,
  } as ConnectionData;
}

describe('connectionIntake', () => {
  it('infers supported database protocols from legacy names', () => {
    expect(inferDbProtocol('mariadb')).toBe('mysql');
    expect(inferDbProtocol('sqlserver')).toBe('mssql');
    expect(inferDbProtocol('mongo')).toBe('mongodb');
    expect(inferDbProtocol('unknown')).toBe('postgresql');
  });

  it('applies type changes with protocol defaults and gateway reset', () => {
    const state = {
      ...emptyConnectionIntakeState(),
      type: 'SSH' as const,
      port: '22',
      gatewayId: 'gateway-1',
    };

    const next = applyConnectionTypeChange(state, 'DB_TUNNEL');

    expect(next.port).toBe('22');
    expect(next.gatewayId).toBe('');
    expect(next.targetDbPort).toBe('5432');
    expect(next.dbSettings.protocol).toBe('postgresql');
  });

  it('hydrates edit state from stored connection metadata', () => {
    const state = connectionToIntakeState(buildConnection());

    expect(state.credentialMode).toBe('manual');
    expect(state.dbSettings.protocol).toBe('mysql');
    expect(state.targetDbHost).toBe('db.internal');
    expect(state.targetDbPort).toBe('5432');
  });

  it('validates credential mode and upload limit rules', () => {
    expect(validateConnectionIntake({
      ...emptyConnectionIntakeState(),
      name: 'SSH',
      host: 'ssh.example.com',
      username: '',
    }, false)).toBe('Username is required for new connections');

    expect(validateConnectionIntake({
      ...emptyConnectionIntakeState(),
      name: 'SSH',
      host: 'ssh.example.com',
      username: 'demo',
      fileTransferMaxUploadSizeMb: '101',
    }, false)).toBe('Max upload size must be between 1 and 100 MiB');
  });

  it('builds create and update payloads with mode-specific credentials', () => {
    const state = {
      ...emptyConnectionIntakeState(),
      name: 'SSH',
      host: 'ssh.example.com',
      username: 'demo',
      password: 'secret',
      transferRetentionPolicy: {
        retainSuccessfulUploads: true,
        maxUploadSizeBytes: 100,
      },
    };

    expect(buildConnectionInput(state, 'folder-1', 'team-1')).toMatchObject({
      name: 'SSH',
      host: 'ssh.example.com',
      username: 'demo',
      password: 'secret',
      folderId: 'folder-1',
      teamId: 'team-1',
      transferRetentionPolicy: {
        retainSuccessfulUploads: true,
        maxUploadSizeBytes: 100 * 1048576,
      },
    });
    expect(buildConnectionUpdate({
      ...state,
      username: '',
      password: '',
    })).not.toHaveProperty('username');
  });
});
