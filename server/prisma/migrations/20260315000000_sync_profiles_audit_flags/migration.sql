-- CreateEnum: SyncProvider
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'SyncProvider') THEN
        CREATE TYPE "SyncProvider" AS ENUM ('NETBOX');
    END IF;
END
$$;

-- CreateEnum: SyncStatus
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'SyncStatus') THEN
        CREATE TYPE "SyncStatus" AS ENUM ('PENDING', 'RUNNING', 'SUCCESS', 'PARTIAL', 'ERROR');
    END IF;
END
$$;

-- AlterEnum: Add LDAP variant to AuthProvider
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'LDAP'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuthProvider')) THEN
        ALTER TYPE "AuthProvider" ADD VALUE 'LDAP';
    END IF;
END
$$;

-- AlterEnum: Add new audit actions
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'LDAP_LOGIN'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'LDAP_LOGIN';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'LDAP_LOGIN_FAILURE'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'LDAP_LOGIN_FAILURE';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'LDAP_SYNC_START'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'LDAP_SYNC_START';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'LDAP_SYNC_COMPLETE'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'LDAP_SYNC_COMPLETE';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'LDAP_SYNC_ERROR'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'LDAP_SYNC_ERROR';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'LDAP_USER_CREATED'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'LDAP_USER_CREATED';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'LDAP_USER_DISABLED'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'LDAP_USER_DISABLED';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'SYNC_PROFILE_CREATE'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'SYNC_PROFILE_CREATE';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'SYNC_PROFILE_UPDATE'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'SYNC_PROFILE_UPDATE';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'SYNC_PROFILE_DELETE'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'SYNC_PROFILE_DELETE';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'SYNC_START'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'SYNC_START';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'SYNC_COMPLETE'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'SYNC_COMPLETE';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'SYNC_ERROR'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'SYNC_ERROR';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'IMPOSSIBLE_TRAVEL_DETECTED'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'IMPOSSIBLE_TRAVEL_DETECTED';
    END IF;
END
$$;

-- AlterEnum: Add new notification types
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'TENANT_INVITATION'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'NotificationType')) THEN
        ALTER TYPE "NotificationType" ADD VALUE 'TENANT_INVITATION';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'RECORDING_READY'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'NotificationType')) THEN
        ALTER TYPE "NotificationType" ADD VALUE 'RECORDING_READY';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'IMPOSSIBLE_TRAVEL_DETECTED'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'NotificationType')) THEN
        ALTER TYPE "NotificationType" ADD VALUE 'IMPOSSIBLE_TRAVEL_DETECTED';
    END IF;
END
$$;

-- AlterTable: Add flags column to AuditLog
ALTER TABLE "AuditLog" ADD COLUMN IF NOT EXISTS "flags" TEXT[] DEFAULT ARRAY[]::TEXT[];

-- CreateTable: SyncProfile
CREATE TABLE IF NOT EXISTS "SyncProfile" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "tenantId" TEXT NOT NULL,
    "provider" "SyncProvider" NOT NULL,
    "config" JSONB NOT NULL,
    "encryptedApiToken" TEXT NOT NULL,
    "apiTokenIV" TEXT NOT NULL,
    "apiTokenTag" TEXT NOT NULL,
    "cronExpression" TEXT,
    "enabled" BOOLEAN NOT NULL DEFAULT true,
    "teamId" TEXT,
    "lastSyncAt" TIMESTAMP(3),
    "lastSyncStatus" "SyncStatus",
    "lastSyncDetails" JSONB,
    "createdById" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "SyncProfile_pkey" PRIMARY KEY ("id")
);

-- CreateTable: SyncLog
CREATE TABLE IF NOT EXISTS "SyncLog" (
    "id" TEXT NOT NULL,
    "syncProfileId" TEXT NOT NULL,
    "status" "SyncStatus" NOT NULL,
    "startedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMP(3),
    "details" JSONB,
    "triggeredBy" TEXT NOT NULL,

    CONSTRAINT "SyncLog_pkey" PRIMARY KEY ("id")
);

-- AlterTable: Add syncProfileId and externalId to Connection
ALTER TABLE "Connection" ADD COLUMN IF NOT EXISTS "syncProfileId" TEXT;
ALTER TABLE "Connection" ADD COLUMN IF NOT EXISTS "externalId" TEXT;

-- CreateIndex: SyncProfile indexes
CREATE INDEX IF NOT EXISTS "SyncProfile_tenantId_idx" ON "SyncProfile"("tenantId");
CREATE INDEX IF NOT EXISTS "SyncProfile_tenantId_provider_idx" ON "SyncProfile"("tenantId", "provider");

-- CreateIndex: SyncLog indexes
CREATE INDEX IF NOT EXISTS "SyncLog_syncProfileId_startedAt_idx" ON "SyncLog"("syncProfileId", "startedAt");

-- CreateIndex: Connection syncProfile index
CREATE INDEX IF NOT EXISTS "Connection_syncProfileId_externalId_idx" ON "Connection"("syncProfileId", "externalId");

-- AddForeignKey: SyncProfile -> Tenant
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'SyncProfile_tenantId_fkey') THEN
        ALTER TABLE "SyncProfile" ADD CONSTRAINT "SyncProfile_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;
    END IF;
END
$$;

-- AddForeignKey: SyncProfile -> User (createdBy)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'SyncProfile_createdById_fkey') THEN
        ALTER TABLE "SyncProfile" ADD CONSTRAINT "SyncProfile_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "User"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
    END IF;
END
$$;

-- AddForeignKey: SyncProfile -> Team
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'SyncProfile_teamId_fkey') THEN
        ALTER TABLE "SyncProfile" ADD CONSTRAINT "SyncProfile_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "Team"("id") ON DELETE SET NULL ON UPDATE CASCADE;
    END IF;
END
$$;

-- AddForeignKey: SyncLog -> SyncProfile
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'SyncLog_syncProfileId_fkey') THEN
        ALTER TABLE "SyncLog" ADD CONSTRAINT "SyncLog_syncProfileId_fkey" FOREIGN KEY ("syncProfileId") REFERENCES "SyncProfile"("id") ON DELETE CASCADE ON UPDATE CASCADE;
    END IF;
END
$$;

-- AddForeignKey: Connection -> SyncProfile
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'Connection_syncProfileId_fkey') THEN
        ALTER TABLE "Connection" ADD CONSTRAINT "Connection_syncProfileId_fkey" FOREIGN KEY ("syncProfileId") REFERENCES "SyncProfile"("id") ON DELETE SET NULL ON UPDATE CASCADE;
    END IF;
END
$$;
