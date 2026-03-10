import { Command } from 'commander';
import * as secretService from '../../services/secret.service';
import * as auditService from '../../services/audit.service';
import { resolveTenant } from '../helpers/resolve';
import { printJson, printError, printSuccess } from '../helpers/output';
import { unlockUserVault } from '../helpers/vault';
import { AuditAction } from '../../generated/prisma/client';

export function registerSecretCommands(program: Command): void {
  const secret = program
    .command('secret')
    .description('Secret (Password Manager) management commands');

  secret
    .command('create-login')
    .description('Create a new login secret in the tenant vault')
    .requiredOption('--tenant-id <id>', 'Tenant ID or slug')
    .requiredOption('--user-email <email>', 'User email performing the action')
    .requiredOption('--password <password>', 'User password to unlock vault')
    .requiredOption('--name <name>', 'Name of the secret')
    .requiredOption('--login-username <username>', 'Login username')
    .requiredOption('--login-password <loginPassword>', 'Login password')
    .option('--description <desc>', 'Secret description')
    .option('--format <format>', 'Output format (json|table)', 'table')
    .action(async (opts: { tenantId: string; userEmail: string; password: string; name: string; loginUsername: string; loginPassword: string; description?: string; format: string }) => {
      const tenant = await resolveTenant(opts.tenantId);
      if (!tenant) { printError(`Tenant not found: ${opts.tenantId}`); process.exitCode = 1; return; }

      const user = await unlockUserVault(opts.userEmail, opts.password);
      if (!user) { process.exitCode = 1; return; }

      try {
        const result = await secretService.createSecret(
          user.id,
          {
            name: opts.name,
            type: 'LOGIN',
            scope: 'TENANT',
            tenantId: tenant.id,
            description: opts.description,
            data: {
              type: 'LOGIN',
              username: opts.loginUsername,
              password: opts.loginPassword,
            },
          },
          tenant.id
        );

        auditService.log({
          userId: user.id,
          action: AuditAction.SECRET_CREATE,
          targetType: 'SECRET',
          targetId: result.id,
          ipAddress: 'cli',
          details: { name: opts.name, type: 'LOGIN', scope: 'TENANT', tenantId: tenant.id, source: 'cli' },
        });

        if (opts.format === 'json') {
          printJson(result);
        } else {
          printSuccess(`Login secret created: ${result.name} (${result.id})`);
        }
      } catch (err) {
        printError(`Failed to create secret: ${err instanceof Error ? err.message : err}`);
        process.exitCode = 1;
      }
    });
}
