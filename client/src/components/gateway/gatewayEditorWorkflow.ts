import type {
  GatewayData,
  GatewayDeploymentMode,
  GatewayInput,
  GatewayUpdate,
} from '../../api/gateway.api';

export type EditableGatewayType = GatewayData['type'];

export interface GatewayEditorForm {
  name: string;
  type: EditableGatewayType;
  deploymentMode: GatewayDeploymentMode;
  host: string;
  port: string;
  description: string;
  isDefault: boolean;
  username: string;
  password: string;
  sshPrivateKey: string;
  apiPort: string;
  monitoringEnabled: boolean;
  monitorIntervalMs: string;
  inactivityTimeout: string;
  autoScaleEnabled: boolean;
  minReplicasVal: string;
  maxReplicasVal: string;
  sessPerInstance: string;
  cooldownVal: string;
  publishPorts: boolean;
  lbStrategy: 'ROUND_ROBIN' | 'LEAST_CONNECTIONS';
}

const defaultPorts: Record<EditableGatewayType, string> = {
  GUACD: '4822',
  SSH_BASTION: '22',
  MANAGED_SSH: '2222',
  DB_PROXY: '5432',
};

const autoDefaultPorts = new Set(Object.values(defaultPorts));

export function defaultGatewayEditorForm(): GatewayEditorForm {
  return {
    name: '',
    type: 'GUACD',
    deploymentMode: 'SINGLE_INSTANCE',
    host: '',
    port: '',
    description: '',
    isDefault: false,
    username: '',
    password: '',
    sshPrivateKey: '',
    apiPort: '',
    monitoringEnabled: true,
    monitorIntervalMs: '5000',
    inactivityTimeout: '60',
    autoScaleEnabled: false,
    minReplicasVal: '0',
    maxReplicasVal: '5',
    sessPerInstance: '10',
    cooldownVal: '300',
    publishPorts: false,
    lbStrategy: 'ROUND_ROBIN',
  };
}

export function gatewayToEditorForm(gateway: GatewayData): GatewayEditorForm {
  return {
    ...defaultGatewayEditorForm(),
    name: gateway.name,
    type: gateway.type,
    deploymentMode: gateway.deploymentMode ?? (gateway.isManaged ? 'MANAGED_GROUP' : 'SINGLE_INSTANCE'),
    host: gateway.host,
    port: String(gateway.port),
    description: gateway.description || '',
    isDefault: gateway.isDefault,
    apiPort: gateway.apiPort ? String(gateway.apiPort) : '',
    monitoringEnabled: gateway.monitoringEnabled,
    monitorIntervalMs: String(gateway.monitorIntervalMs),
    inactivityTimeout: String(Math.floor(gateway.inactivityTimeoutSeconds / 60)),
    autoScaleEnabled: gateway.autoScale,
    minReplicasVal: String(gateway.minReplicas),
    maxReplicasVal: String(gateway.maxReplicas),
    sessPerInstance: String(gateway.sessionsPerInstance),
    cooldownVal: String(gateway.scaleDownCooldownSeconds),
    publishPorts: gateway.publishPorts ?? false,
    lbStrategy: gateway.lbStrategy ?? 'ROUND_ROBIN',
  };
}

export function supportsGatewayGroupMode(type: EditableGatewayType): boolean {
  return type === 'MANAGED_SSH' || type === 'GUACD' || type === 'DB_PROXY';
}

export function isGatewayGroupMode(form: GatewayEditorForm): boolean {
  return form.deploymentMode === 'MANAGED_GROUP';
}

export function nextGatewayTypeForm(
  form: GatewayEditorForm,
  type: EditableGatewayType,
): GatewayEditorForm {
  const nextPort = !form.port || autoDefaultPorts.has(form.port)
    ? defaultPorts[type]
    : form.port;

  return {
    ...form,
    type,
    port: nextPort,
    apiPort: type === 'MANAGED_SSH' ? (form.apiPort || '9022') : '',
    deploymentMode: type === 'SSH_BASTION' ? 'SINGLE_INSTANCE' : form.deploymentMode,
  };
}

export function nextPublishPortsForm(form: GatewayEditorForm, publishPorts: boolean): GatewayEditorForm {
  return {
    ...form,
    publishPorts,
    port: publishPorts ? defaultPorts[form.type] : form.port,
  };
}

export function validateGatewayEditorForm(form: GatewayEditorForm): string | null {
  if (!form.name.trim()) return 'Gateway name is required';
  if (!isGatewayGroupMode(form) && !form.host.trim()) return 'Host is required';

  const port = parsePort(form.port);
  if (port == null) return 'Port must be between 1 and 65535';

  if (form.apiPort.trim()) {
    const apiPort = parsePort(form.apiPort);
    if (apiPort == null) return 'gRPC port must be between 1 and 65535';
  }

  return null;
}

export function buildGatewayCreateInput(form: GatewayEditorForm): GatewayInput {
  const type = form.type;
  const supportsGroupMode = supportsGatewayGroupMode(type);
  const apiPortNum = form.apiPort ? parseInt(form.apiPort, 10) : undefined;
  const monitorIntervalMs = parseInt(form.monitorIntervalMs, 10) || 5000;
  const inactivityTimeoutSeconds = (parseInt(form.inactivityTimeout, 10) || 60) * 60;

  return {
    name: form.name.trim(),
    type,
    deploymentMode: form.deploymentMode,
    host: isGatewayGroupMode(form) ? '' : form.host.trim(),
    port: parseInt(form.port, 10),
    description: form.description.trim() || undefined,
    isDefault: form.isDefault || undefined,
    monitoringEnabled: form.monitoringEnabled,
    monitorIntervalMs,
    inactivityTimeoutSeconds,
    ...(type === 'SSH_BASTION' && form.username ? { username: form.username } : {}),
    ...(type === 'SSH_BASTION' && form.password ? { password: form.password } : {}),
    ...(type === 'SSH_BASTION' && form.sshPrivateKey ? { sshPrivateKey: form.sshPrivateKey } : {}),
    ...(type === 'MANAGED_SSH' && apiPortNum ? { apiPort: apiPortNum } : {}),
    ...(supportsGroupMode && form.publishPorts ? { publishPorts: form.publishPorts } : {}),
    ...(supportsGroupMode ? { lbStrategy: form.lbStrategy } : {}),
  };
}

export function buildGatewayUpdate(gateway: GatewayData, form: GatewayEditorForm): GatewayUpdate {
  const data: GatewayUpdate = {};
  const normalizedHost = isGatewayGroupMode(form) ? '' : form.host.trim();
  const existingDeploymentMode = gateway.deploymentMode ?? (gateway.isManaged ? 'MANAGED_GROUP' : 'SINGLE_INSTANCE');
  const portNum = parseInt(form.port, 10);
  const supportsGroupMode = supportsGatewayGroupMode(form.type);

  if (form.name.trim() !== gateway.name) data.name = form.name.trim();
  if (form.deploymentMode !== existingDeploymentMode) data.deploymentMode = form.deploymentMode;
  if (normalizedHost !== gateway.host) data.host = normalizedHost;
  if (portNum !== gateway.port) data.port = portNum;
  if ((form.description.trim() || null) !== gateway.description) data.description = form.description.trim() || null;
  if (form.isDefault !== gateway.isDefault) data.isDefault = form.isDefault;
  if (gateway.type === 'MANAGED_SSH') {
    const newApiPort = form.apiPort ? parseInt(form.apiPort, 10) : null;
    if (newApiPort !== gateway.apiPort) data.apiPort = newApiPort;
  }
  if (form.type === 'SSH_BASTION') {
    if (form.username) data.username = form.username;
    if (form.password) data.password = form.password;
    if (form.sshPrivateKey) data.sshPrivateKey = form.sshPrivateKey;
  }
  if (supportsGroupMode && form.publishPorts !== (gateway.publishPorts ?? false)) data.publishPorts = form.publishPorts;
  if (supportsGroupMode && form.lbStrategy !== (gateway.lbStrategy ?? 'ROUND_ROBIN')) data.lbStrategy = form.lbStrategy;
  if (form.monitoringEnabled !== gateway.monitoringEnabled) data.monitoringEnabled = form.monitoringEnabled;

  const intervalNum = parseInt(form.monitorIntervalMs, 10);
  if (intervalNum && intervalNum !== gateway.monitorIntervalMs) data.monitorIntervalMs = intervalNum;

  const timeoutSec = parseInt(form.inactivityTimeout, 10) * 60;
  if (timeoutSec && timeoutSec !== gateway.inactivityTimeoutSeconds) {
    data.inactivityTimeoutSeconds = timeoutSec;
  }

  return data;
}

function parsePort(raw: string): number | null {
  const port = parseInt(raw, 10);
  if (!raw || Number.isNaN(port) || port < 1 || port > 65535) {
    return null;
  }
  return port;
}
