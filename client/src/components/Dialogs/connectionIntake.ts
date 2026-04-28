import type {
  ConnectionData,
  ConnectionInput,
  ConnectionUpdate,
  DlpPolicy,
  DbProtocol,
  DbSettings,
  TransferRetentionPolicy,
} from '../../api/connections.api';
import type { SshTerminalConfig } from '../../constants/terminalThemes';
import type { RdpSettings } from '../../constants/rdpDefaults';
import type { VncSettings } from '../../constants/vncDefaults';
import { supportsCloudProviderPresets } from '../../utils/dbConnectionSecurity';
import { defaultPortForType, dlpPolicyHasValues, hasValues } from './connectionIntakeHelpers';

export type ConnectionType = ConnectionInput['type'];
export type CredentialMode = 'manual' | 'keychain' | 'external-vault';

export const DEFAULT_CONNECTION_UPLOAD_LIMIT_MB = 100;
export const DEFAULT_CONNECTION_UPLOAD_LIMIT_BYTES = DEFAULT_CONNECTION_UPLOAD_LIMIT_MB * 1048576;

export interface ConnectionIntakeState {
  name: string;
  type: ConnectionType;
  host: string;
  port: string;
  username: string;
  password: string;
  domain: string;
  description: string;
  enableDrive: boolean;
  sshTerminalConfig: Partial<SshTerminalConfig>;
  rdpSettings: Partial<RdpSettings>;
  vncSettings: Partial<VncSettings>;
  dbSettings: Partial<DbSettings>;
  gatewayId: string;
  credentialMode: CredentialMode;
  selectedSecretId: string | null;
  selectedVaultProviderId: string | null;
  vaultSecretPath: string;
  defaultConnectMode: string;
  dlpPolicy: DlpPolicy;
  transferRetentionPolicy: TransferRetentionPolicy;
  fileTransferMaxUploadSizeMb: string;
  targetDbHost: string;
  targetDbPort: string;
  dbType: string;
}

export function supportsPersistedExecutionPlans(protocol?: DbProtocol): boolean {
  return protocol === 'postgresql' || protocol === 'mysql' || protocol === 'oracle' || protocol === 'mssql';
}

export function supportsDatabaseSettings(type: ConnectionType): boolean {
  return type === 'DATABASE' || type === 'DB_TUNNEL';
}

export function supportsTransferRetention(type: ConnectionType): boolean {
  return type === 'SSH' || type === 'RDP';
}

export function normalizeTransferRetentionPolicy(policy?: TransferRetentionPolicy | null): TransferRetentionPolicy {
  return {
    retainSuccessfulUploads: policy?.retainSuccessfulUploads ?? false,
    maxUploadSizeBytes: policy?.maxUploadSizeBytes ?? DEFAULT_CONNECTION_UPLOAD_LIMIT_BYTES,
  };
}

export function inferDbProtocol(value?: string | null): DbProtocol {
  switch ((value ?? '').trim().toLowerCase()) {
    case 'mysql':
    case 'mariadb':
      return 'mysql';
    case 'mongodb':
    case 'mongo':
      return 'mongodb';
    case 'oracle':
      return 'oracle';
    case 'mssql':
    case 'sqlserver':
      return 'mssql';
    case 'db2':
      return 'db2';
    default:
      return 'postgresql';
  }
}

export function seedDbSettings(
  connectionType: ConnectionType,
  settings?: Partial<DbSettings> | null,
  dbType?: string | null,
): Partial<DbSettings> {
  if (!supportsDatabaseSettings(connectionType)) {
    return settings ?? {};
  }

  const nextSettings = { ...(settings ?? {}) };
  if (!nextSettings.protocol) {
    nextSettings.protocol = inferDbProtocol(dbType);
  }
  return nextSettings;
}

export function normalizeDbSettings(settings: Partial<DbSettings>): DbSettings | null {
  const protocol = settings.protocol ?? inferDbProtocol();
  const supportsCloudPresets = supportsCloudProviderPresets(protocol);

  return {
    ...settings,
    protocol,
    cloudProvider: supportsCloudPresets ? settings.cloudProvider : undefined,
    sslMode: settings.sslMode,
    persistExecutionPlan: supportsPersistedExecutionPlans(protocol)
      ? settings.persistExecutionPlan
      : undefined,
  };
}

export function emptyConnectionIntakeState(): ConnectionIntakeState {
  return {
    name: '',
    type: 'SSH',
    host: '',
    port: '22',
    username: '',
    password: '',
    domain: '',
    description: '',
    enableDrive: false,
    sshTerminalConfig: {},
    rdpSettings: {},
    vncSettings: {},
    dbSettings: {},
    gatewayId: '',
    credentialMode: 'manual',
    selectedSecretId: null,
    selectedVaultProviderId: null,
    vaultSecretPath: '',
    defaultConnectMode: '',
    dlpPolicy: {},
    transferRetentionPolicy: normalizeTransferRetentionPolicy(),
    fileTransferMaxUploadSizeMb: String(DEFAULT_CONNECTION_UPLOAD_LIMIT_MB),
    targetDbHost: '',
    targetDbPort: '',
    dbType: '',
  };
}

export function connectionToIntakeState(connection: ConnectionData): ConnectionIntakeState {
  const transferPolicy = normalizeTransferRetentionPolicy(connection.transferRetentionPolicy);
  const base = emptyConnectionIntakeState();
  const credentialState = credentialStateForConnection(connection);

  return {
    ...base,
    ...credentialState,
    name: connection.name,
    type: connection.type,
    host: connection.host,
    port: String(connection.port),
    description: connection.description || '',
    enableDrive: connection.enableDrive ?? false,
    gatewayId: connection.gatewayId || '',
    sshTerminalConfig: (connection.sshTerminalConfig as Partial<SshTerminalConfig>) ?? {},
    rdpSettings: (connection.rdpSettings as Partial<RdpSettings>) ?? {},
    vncSettings: (connection.vncSettings as Partial<VncSettings>) ?? {},
    dbSettings: seedDbSettings(connection.type, connection.dbSettings as Partial<DbSettings>, connection.dbType),
    defaultConnectMode: connection.defaultCredentialMode ?? '',
    dlpPolicy: (connection.dlpPolicy as DlpPolicy) ?? {},
    transferRetentionPolicy: transferPolicy,
    fileTransferMaxUploadSizeMb: String(Math.round(transferPolicy.maxUploadSizeBytes / 1048576)),
    targetDbHost: connection.targetDbHost ?? '',
    targetDbPort: connection.targetDbPort?.toString() ?? '',
    dbType: connection.dbType ?? '',
  };
}

export function applyConnectionTypeChange(state: ConnectionIntakeState, newType: ConnectionType): ConnectionIntakeState {
  const knownPorts = ['22', '3389', '5900', '5432', '3306', '27017', '1521', '1433', '50000'];
  const next = {
    ...state,
    type: newType,
    gatewayId: '',
  };

  if (knownPorts.includes(state.port)) {
    next.port = defaultPortForType(newType);
  }
  if (supportsDatabaseSettings(newType)) {
    next.dbSettings = seedDbSettings(newType, state.dbSettings);
    if (newType === 'DB_TUNNEL' && !state.targetDbPort) {
      next.targetDbPort = '5432';
    }
  }
  return next;
}

export function validateConnectionIntake(state: ConnectionIntakeState, isEditMode: boolean): string | null {
  if (!state.name || !state.host) {
    return 'Name and host are required';
  }
  if (state.type === 'DB_TUNNEL' && (!state.targetDbHost || !state.targetDbPort)) {
    return 'Target database host and port are required for DB Tunnel connections';
  }
  if (state.credentialMode === 'keychain' && !state.selectedSecretId) {
    return 'Please select a secret from the keychain';
  }
  if (state.credentialMode === 'external-vault' && (!state.selectedVaultProviderId || !state.vaultSecretPath)) {
    return 'Please select a vault provider and enter a secret path';
  }
  if (state.credentialMode === 'manual' && !isEditMode && !state.username) {
    return 'Username is required for new connections';
  }
  if (supportsTransferRetention(state.type)) {
    const uploadLimitMb = uploadLimitMbForState(state);
    if (Number.isNaN(uploadLimitMb) || uploadLimitMb < 1 || uploadLimitMb > DEFAULT_CONNECTION_UPLOAD_LIMIT_MB) {
      return `Max upload size must be between 1 and ${DEFAULT_CONNECTION_UPLOAD_LIMIT_MB} MiB`;
    }
  }
  return null;
}

export function buildConnectionUpdate(state: ConnectionIntakeState): ConnectionUpdate {
  const data: ConnectionUpdate = {
    name: state.name,
    type: state.type,
    host: state.host,
    port: parseInt(state.port, 10),
    description: state.description || null,
    enableDrive: state.enableDrive,
    gatewayId: state.gatewayId || null,
    credentialSecretId: state.credentialMode === 'keychain' ? state.selectedSecretId : null,
    externalVaultProviderId: state.credentialMode === 'external-vault' ? state.selectedVaultProviderId : null,
    externalVaultPath: state.credentialMode === 'external-vault' ? state.vaultSecretPath : null,
    defaultCredentialMode: (state.defaultConnectMode as 'saved' | 'domain' | 'prompt') || null,
    ...updateConnectionSpecificSettings(state),
  };

  if (state.credentialMode === 'manual') {
    if (state.username) data.username = state.username;
    if (state.password) data.password = state.password;
    if (state.domain) data.domain = state.domain;
  }
  return data;
}

export function buildConnectionInput(
  state: ConnectionIntakeState,
  folderId?: string | null,
  teamId?: string | null,
): ConnectionInput {
  return {
    name: state.name,
    type: state.type,
    host: state.host,
    port: parseInt(state.port, 10),
    description: state.description || undefined,
    enableDrive: state.enableDrive,
    gatewayId: state.gatewayId || null,
    ...credentialInput(state),
    ...(folderId ? { folderId } : {}),
    ...(teamId ? { teamId } : {}),
    ...(state.defaultConnectMode ? { defaultCredentialMode: state.defaultConnectMode as 'saved' | 'domain' | 'prompt' } : {}),
    ...inputConnectionSpecificSettings(state),
  };
}

function credentialStateForConnection(connection: ConnectionData): Pick<
  ConnectionIntakeState,
  'credentialMode' | 'selectedSecretId' | 'selectedVaultProviderId' | 'vaultSecretPath'
> {
  if (connection.externalVaultProviderId) {
    return {
      credentialMode: 'external-vault',
      selectedVaultProviderId: connection.externalVaultProviderId,
      vaultSecretPath: connection.externalVaultPath ?? '',
      selectedSecretId: null,
    };
  }
  if (connection.credentialSecretId) {
    return {
      credentialMode: 'keychain',
      selectedSecretId: connection.credentialSecretId,
      selectedVaultProviderId: null,
      vaultSecretPath: '',
    };
  }
  return {
    credentialMode: 'manual',
    selectedSecretId: null,
    selectedVaultProviderId: null,
    vaultSecretPath: '',
  };
}

function uploadLimitMbForState(state: ConnectionIntakeState): number {
  return Number.parseInt(
    state.fileTransferMaxUploadSizeMb.trim() || String(DEFAULT_CONNECTION_UPLOAD_LIMIT_MB),
    10,
  );
}

function transferRetentionForState(state: ConnectionIntakeState): TransferRetentionPolicy {
  if (!supportsTransferRetention(state.type)) {
    return state.transferRetentionPolicy;
  }
  return {
    ...state.transferRetentionPolicy,
    maxUploadSizeBytes: uploadLimitMbForState(state) * 1048576,
  };
}

function credentialInput(state: ConnectionIntakeState): Partial<ConnectionInput> {
  if (state.credentialMode === 'keychain' && state.selectedSecretId) {
    return { credentialSecretId: state.selectedSecretId };
  }
  if (state.credentialMode === 'external-vault' && state.selectedVaultProviderId) {
    return {
      externalVaultProviderId: state.selectedVaultProviderId,
      externalVaultPath: state.vaultSecretPath,
    };
  }
  if (state.credentialMode === 'manual') {
    return {
      username: state.username,
      password: state.password,
      ...(state.domain ? { domain: state.domain } : {}),
    };
  }
  return {};
}

function inputConnectionSpecificSettings(state: ConnectionIntakeState): Partial<ConnectionInput> {
  return {
    ...(state.type === 'SSH' && hasValues(state.sshTerminalConfig) && {
      sshTerminalConfig: state.sshTerminalConfig,
    }),
    ...(state.type === 'RDP' && hasValues(state.rdpSettings) && {
      rdpSettings: state.rdpSettings,
    }),
    ...(state.type === 'VNC' && hasValues(state.vncSettings) && {
      vncSettings: state.vncSettings,
    }),
    ...(supportsDatabaseSettings(state.type) && {
      dbSettings: normalizeDbSettings(state.dbSettings) as DbSettings,
    }),
    ...(supportsTransferRetention(state.type) && {
      transferRetentionPolicy: transferRetentionForState(state),
    }),
    ...((state.type === 'RDP' || state.type === 'VNC') && dlpPolicyHasValues(state.dlpPolicy) && {
      dlpPolicy: state.dlpPolicy,
    }),
    ...(state.type === 'DB_TUNNEL' && {
      targetDbHost: state.targetDbHost,
      targetDbPort: parseInt(state.targetDbPort, 10),
      ...(state.dbType ? { dbType: state.dbType } : {}),
    }),
  };
}

function updateConnectionSpecificSettings(state: ConnectionIntakeState): Partial<ConnectionUpdate> {
  return {
    ...(state.type === 'SSH' && {
      sshTerminalConfig: hasValues(state.sshTerminalConfig) ? state.sshTerminalConfig : null,
    }),
    ...(state.type === 'RDP' && {
      rdpSettings: hasValues(state.rdpSettings) ? state.rdpSettings : null,
    }),
    ...(state.type === 'VNC' && {
      vncSettings: hasValues(state.vncSettings) ? state.vncSettings : null,
    }),
    ...(supportsDatabaseSettings(state.type) && {
      dbSettings: normalizeDbSettings(state.dbSettings),
    }),
    ...(supportsTransferRetention(state.type) && {
      transferRetentionPolicy: transferRetentionForState(state),
    }),
    ...((state.type === 'RDP' || state.type === 'VNC') && {
      dlpPolicy: dlpPolicyHasValues(state.dlpPolicy) ? state.dlpPolicy : null,
    }),
    ...(state.type === 'DB_TUNNEL' && {
      targetDbHost: state.targetDbHost || null,
      targetDbPort: state.targetDbPort ? parseInt(state.targetDbPort, 10) : null,
      dbType: state.dbType || null,
    }),
  };
}
