-- AlterTable
ALTER TABLE "AuditLog" ADD COLUMN     "geoCity" TEXT,
ADD COLUMN     "geoCoords" DOUBLE PRECISION[] DEFAULT ARRAY[]::DOUBLE PRECISION[],
ADD COLUMN     "geoCountry" TEXT;

-- CreateIndex
CREATE INDEX "AuditLog_geoCountry_idx" ON "AuditLog"("geoCountry");
