-- AlterEnum: Add SESSION_DENIED_ABAC to AuditAction
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'SESSION_DENIED_ABAC'
                   AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'AuditAction')) THEN
        ALTER TYPE "AuditAction" ADD VALUE 'SESSION_DENIED_ABAC';
    END IF;
END
$$;

-- CreateEnum: AccessPolicyTargetType
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'AccessPolicyTargetType') THEN
        CREATE TYPE "AccessPolicyTargetType" AS ENUM ('TENANT', 'TEAM', 'FOLDER');
    END IF;
END
$$;

-- CreateTable: AccessPolicy
CREATE TABLE IF NOT EXISTS "AccessPolicy" (
    "id" TEXT NOT NULL,
    "targetType" "AccessPolicyTargetType" NOT NULL,
    "targetId" TEXT NOT NULL,
    "allowedTimeWindows" TEXT,
    "requireTrustedDevice" BOOLEAN NOT NULL DEFAULT false,
    "requireMfaStepUp" BOOLEAN NOT NULL DEFAULT false,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "AccessPolicy_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX IF NOT EXISTS "AccessPolicy_targetType_targetId_idx" ON "AccessPolicy"("targetType", "targetId");
