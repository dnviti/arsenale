import type { DlpPolicy } from '../../api/connections.api';
import type { ConnectionType } from './connectionIntake';

export function defaultPortForType(type: ConnectionType): string {
  switch (type) {
    case 'RDP':
      return '3389';
    case 'VNC':
      return '5900';
    case 'DATABASE':
      return '5432';
    case 'DB_TUNNEL':
    case 'SSH':
      return '22';
  }
}

export function hasValues(value: object): boolean {
  return Object.keys(value).length > 0;
}

export function dlpPolicyHasValues(policy: DlpPolicy): boolean {
  return Object.values(policy).some(Boolean);
}
