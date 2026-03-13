-- AlterTable: Add DLP policy columns to Tenant
ALTER TABLE "Tenant" ADD COLUMN "dlpDisableCopy" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE "Tenant" ADD COLUMN "dlpDisablePaste" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE "Tenant" ADD COLUMN "dlpDisableDownload" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE "Tenant" ADD COLUMN "dlpDisableUpload" BOOLEAN NOT NULL DEFAULT false;

-- AlterTable: Add DLP policy JSON to Connection
ALTER TABLE "Connection" ADD COLUMN "dlpPolicy" JSONB;

-- AlterEnum: Add TENANT_DLP_POLICY_UPDATE to AuditAction
ALTER TYPE "AuditAction" ADD VALUE 'TENANT_DLP_POLICY_UPDATE';
