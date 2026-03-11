-- CreateIndex
CREATE INDEX "AuditLog_userId_action_createdAt_idx" ON "AuditLog"("userId", "action", "createdAt");

-- CreateIndex
CREATE INDEX "TenantMember_tenantId_isActive_idx" ON "TenantMember"("tenantId", "isActive");

-- CreateIndex
CREATE INDEX "VaultSecret_expiresAt_userId_idx" ON "VaultSecret"("expiresAt", "userId");
