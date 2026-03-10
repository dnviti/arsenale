import { Command } from 'commander';
import * as gatewayService from '../../services/gateway.service';
import * as auditService from '../../services/audit.service';
import { resolveTenant, resolveUser } from '../helpers/resolve';
import { printJson, printTable, printError, printSuccess } from '../helpers/output';
import { AuditAction, GatewayType } from '../../generated/prisma/client';

export function registerGatewayCommands(program: Command): void {
  const gateway = program
    .command('gateway')
    .description('Gateway management commands');

  gateway
    .command('list')
    .description('List gateways for a tenant')
    .requiredOption('--tenant-id <id>', 'Tenant ID or slug')
    .option('--format <format>', 'Output format (json|table)', 'table')
    .action(async (opts: { tenantId: string; format: string }) => {
      const tenant = await resolveTenant(opts.tenantId);
      if (!tenant) { printError(`Tenant not found: ${opts.tenantId}`); process.exitCode = 1; return; }

      const gateways = await gatewayService.listGateways(tenant.id);

      if (opts.format === 'json') {
        printJson(gateways);
      } else {
        printTable(
          gateways.map((g) => ({
            id: g.id,
            name: g.name,
            type: g.type,
            host: g.host ?? '',
            port: g.port ?? '',
            health: g.lastHealthStatus ?? 'UNKNOWN',
            latency: g.lastLatencyMs != null ? `${g.lastLatencyMs}ms` : '',
            default: g.isDefault ? 'yes' : 'no',
          })),
          [
            { key: 'id', header: 'ID', width: 36 },
            { key: 'name', header: 'NAME' },
            { key: 'type', header: 'TYPE', width: 12 },
            { key: 'host', header: 'HOST' },
            { key: 'port', header: 'PORT', width: 5 },
            { key: 'health', header: 'HEALTH', width: 10 },
            { key: 'latency', header: 'LATENCY', width: 9 },
            { key: 'default', header: 'DEFAULT', width: 7 },
          ],
        );
        console.log(`\nTotal: ${gateways.length}`);
      }
    });

  gateway
    .command('status')
    .description('Show gateway status with health details')
    .requiredOption('--tenant-id <id>', 'Tenant ID or slug')
    .option('--format <format>', 'Output format (json|table)', 'table')
    .action(async (opts: { tenantId: string; format: string }) => {
      const tenant = await resolveTenant(opts.tenantId);
      if (!tenant) { printError(`Tenant not found: ${opts.tenantId}`); process.exitCode = 1; return; }

      const gateways = await gatewayService.listGateways(tenant.id);

      if (opts.format === 'json') {
        printJson(gateways.map((g) => ({
          id: g.id,
          name: g.name,
          type: g.type,
          healthStatus: g.lastHealthStatus,
          latencyMs: g.lastLatencyMs,
          lastCheckedAt: g.lastCheckedAt,
          lastError: g.lastError,
          monitoringEnabled: g.monitoringEnabled,
          isManaged: g.isManaged,
          totalInstances: g.totalInstances,
          runningInstances: g.runningInstances,
        })));
      } else {
        for (const g of gateways) {
          console.log(`--- ${g.name} (${g.id}) ---`);
          console.log(`  Type:        ${g.type}`);
          console.log(`  Health:      ${g.lastHealthStatus ?? 'UNKNOWN'}`);
          console.log(`  Latency:     ${g.lastLatencyMs != null ? `${g.lastLatencyMs}ms` : 'N/A'}`);
          console.log(`  Last check:  ${g.lastCheckedAt ?? 'never'}`);
          if (g.lastError) console.log(`  Last error:  ${g.lastError}`);
          if (g.isManaged) {
            console.log(`  Instances:   ${g.runningInstances}/${g.totalInstances} running`);
          }
          console.log();
        }
      }
    });

  gateway
    .command('health-check')
    .description('Test connectivity to a specific gateway')
    .argument('<gateway-id>', 'Gateway UUID')
    .requiredOption('--tenant-id <id>', 'Tenant ID or slug')
    .option('--format <format>', 'Output format (json|table)', 'table')
    .action(async (gatewayId: string, opts: { tenantId: string; format: string }) => {
      const tenant = await resolveTenant(opts.tenantId);
      if (!tenant) { printError(`Tenant not found: ${opts.tenantId}`); process.exitCode = 1; return; }

      const result = await gatewayService.testGatewayConnectivity(tenant.id, gatewayId);

      if (opts.format === 'json') {
        printJson(result);
      } else {
        const status = result.reachable ? 'REACHABLE' : 'UNREACHABLE';
        console.log(`Status:   ${status}`);
        console.log(`Latency:  ${result.latencyMs}ms`);
        if (result.error) console.log(`Error:    ${result.error}`);
      }

      if (!result.reachable) process.exitCode = 1;
    });

  gateway
    .command('create')
    .description('Create a new non-managed gateway')
    .requiredOption('--tenant-id <id>', 'Tenant ID or slug')
    .requiredOption('--user-email <email>', 'User email performing the action')
    .requiredOption('--name <name>', 'Name of the gateway')
    .requiredOption('--type <type>', 'Type of gateway (GUACD|SSH_BASTION)')
    .requiredOption('--host <host>', 'Gateway hostname or IP')
    .requiredOption('--port <port>', 'Gateway port')
    .option('--description <desc>', 'Gateway description')
    .option('--is-default', 'Set as default gateway')
    .option('--format <format>', 'Output format (json|table)', 'table')
    .action(async (opts: { tenantId: string; userEmail: string; name: string; type: string; host: string; port: string; description?: string; isDefault?: boolean; format: string }) => {
      const tenant = await resolveTenant(opts.tenantId);
      if (!tenant) { printError(`Tenant not found: ${opts.tenantId}`); process.exitCode = 1; return; }

      const user = await resolveUser(opts.userEmail);
      if (!user) { printError(`User not found: ${opts.userEmail}`); process.exitCode = 1; return; }

      const portNum = parseInt(opts.port, 10);
      if (isNaN(portNum)) { printError(`Invalid port: ${opts.port}`); process.exitCode = 1; return; }

      try {
        const result = await gatewayService.createGateway(user.id, tenant.id, {
          name: opts.name,
          type: opts.type as GatewayType,
          host: opts.host,
          port: portNum,
          description: opts.description,
          isDefault: opts.isDefault,
        });

        auditService.log({
          userId: user.id,
          action: AuditAction.GATEWAY_CREATE,
          targetType: 'GATEWAY',
          targetId: result.id,
          ipAddress: 'cli',
          details: { name: opts.name, type: opts.type, host: opts.host, port: portNum, tenantId: tenant.id, source: 'cli' },
        });

        if (opts.format === 'json') {
          printJson(result);
        } else {
          printSuccess(`Gateway created: ${result.name} (${result.id})`);
        }
      } catch (err) {
        printError(`Failed to create gateway: ${err instanceof Error ? err.message : err}`);
        process.exitCode = 1;
      }
    });
}
