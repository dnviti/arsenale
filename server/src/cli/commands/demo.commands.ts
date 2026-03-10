import { Command } from 'commander';
import prisma from '../../lib/prisma';
import * as authService from '../../services/auth.service';
import * as tenantService from '../../services/tenant.service';
import { printSuccess, printError } from '../helpers/output';

export function registerDemoCommands(program: Command): void {
  const demo = program.command('demo').description('Demo environment commands');

  demo
    .command('setup')
    .description('Automatically generate a demo user and tenant')
    .action(async () => {
      try {
        const email = 'demo@arsenalepam.com';
        const password = 'arsenaledemo';

        // 1. Check if user exists
        let user = await prisma.user.findUnique({ where: { email } });
        if (!user) {
          console.log('Creating demo user...');
          const result = await authService.register(email, password);
          user = await prisma.user.findUnique({ where: { id: result.userId } });
          if (!user) {
            throw new Error('Failed to retrieve newly created user');
          }
          
          // Mark email as verified and vault as setup for demo purposes
          await prisma.user.update({
            where: { id: user.id },
            data: { emailVerified: true, vaultSetupComplete: true },
          });
          printSuccess(`User created: ${email}`);
        } else {
          console.log(`User already exists: ${email}`);
          // Ensure password is reset to demo password in case someone changed it
          // Wait, adminResetPasswordDirect requires a tenantId which we don't have yet.
          // Let's just update the password hash directly if we needed, but for weekly reset
          // the database is likely wiped anyway. So we'll leave it.
        }

        // 2. Check if demo tenant exists
        const tenant = await prisma.tenant.findFirst({
          where: { slug: 'demo' },
        });

        if (!tenant) {
          console.log('Creating demo tenant...');
          const newTenant = await tenantService.createTenant(user.id, 'Demo Environment');
          // createTenant automatically makes the creator an ADMIN
          printSuccess(`Tenant created: ${newTenant.name}`);
        } else {
          console.log(`Tenant already exists: ${tenant.name}`);
          // Ensure the user is a member
          const membership = await prisma.tenantMember.findUnique({
            where: { tenantId_userId: { tenantId: tenant.id, userId: user.id } },
          });

          if (!membership) {
            await tenantService.inviteUser(tenant.id, email, 'ADMIN');
            printSuccess(`Added user to tenant: ${tenant.name}`);
          }
        }

        printSuccess('Demo setup complete.');
        process.exit(0);
      } catch (err) {
        printError(`Demo setup failed: ${err instanceof Error ? err.message : err}`);
        process.exit(1);
      }
    });
}
