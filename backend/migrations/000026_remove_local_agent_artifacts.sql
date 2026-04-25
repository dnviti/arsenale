WITH local_agent_gateways AS (
  SELECT id
  FROM public."Gateway"
  WHERE type::text IN ('LOCAL_AGENT', 'DESKTOP_GATEWAY')
     OR name ~* '^(Windows|Linux) Agent Test( |$)'
),
local_agent_connections AS (
  SELECT id
  FROM public."Connection"
  WHERE type::text IN ('FSS', 'SUPPORT')
     OR "gatewayId" IN (SELECT id FROM local_agent_gateways)
     OR name ILIKE 'Temporary % support via %'
     OR name ILIKE '% Agent Test %'
     OR name ILIKE 'arsenale-agent-test'
     OR description ILIKE 'Temporary support connection redeemed from %'
)
DELETE FROM public."AuditLog"
WHERE action::text IN ('SUPPORT_CODE_ISSUE', 'SUPPORT_CODE_REDEEM')
   OR "targetId" IN (SELECT id FROM local_agent_connections)
   OR "targetId" IN (SELECT id FROM local_agent_gateways)
   OR "gatewayId" IN (SELECT id FROM local_agent_gateways)
   OR details::text ILIKE '%targetAgentId%'
   OR details::text ILIKE '%supportAgentId%'
   OR details::text ILIKE '%Windows Agent Test%'
   OR details::text ILIKE '%Linux Agent Test%'
   OR details::text ILIKE '%Desktop Gateway%';

DROP TABLE IF EXISTS public."SupportSession" CASCADE;
DROP TABLE IF EXISTS public."FSSSession" CASCADE;
DROP TABLE IF EXISTS public."FSSLease" CASCADE;
DROP TABLE IF EXISTS public."SupportCode" CASCADE;
DROP TABLE IF EXISTS public."SupportAgent" CASCADE;

DELETE FROM public."Connection"
WHERE type::text IN ('FSS', 'SUPPORT')
   OR "gatewayId" IN (
      SELECT id
      FROM public."Gateway"
      WHERE type::text IN ('LOCAL_AGENT', 'DESKTOP_GATEWAY')
         OR name ~* '^(Windows|Linux) Agent Test( |$)'
   )
   OR name ILIKE 'Temporary % support via %'
   OR name ILIKE '% Agent Test %'
   OR name ILIKE 'arsenale-agent-test'
   OR description ILIKE 'Temporary support connection redeemed from %';

DELETE FROM public."Gateway"
WHERE type::text IN ('LOCAL_AGENT', 'DESKTOP_GATEWAY')
   OR name ~* '^(Windows|Linux) Agent Test( |$)';

DELETE FROM public."GatewayTemplate"
WHERE type::text IN ('LOCAL_AGENT', 'DESKTOP_GATEWAY')
   OR name ~* '^(Windows|Linux) Agent Test( |$)';

ALTER TABLE public."Gateway" DROP COLUMN IF EXISTS "tunnelReportedHostname";
ALTER TABLE public."Gateway" DROP COLUMN IF EXISTS "tunnelReportedOs";
ALTER TABLE public."Gateway" DROP COLUMN IF EXISTS "tunnelReportedArch";
ALTER TABLE public."Gateway" DROP COLUMN IF EXISTS "tunnelReportedCapabilities";
ALTER TABLE public."Gateway" DROP COLUMN IF EXISTS "isHidden";

DROP TYPE IF EXISTS public."SupportSessionStatus";
DROP TYPE IF EXISTS public."FSSSessionStatus";
DROP TYPE IF EXISTS public."FSSLeaseStatus";

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM pg_enum e
    JOIN pg_type t ON t.oid = e.enumtypid
    WHERE t.typnamespace = 'public'::regnamespace
      AND t.typname = 'ConnectionType'
      AND e.enumlabel IN ('FSS', 'SUPPORT')
  ) THEN
    ALTER TYPE public."ConnectionType" RENAME TO "ConnectionType_local_agent_old";
    CREATE TYPE public."ConnectionType" AS ENUM ('RDP', 'SSH', 'VNC', 'DATABASE', 'DB_TUNNEL');
    ALTER TABLE public."Connection"
      ALTER COLUMN type TYPE public."ConnectionType"
      USING type::text::public."ConnectionType";
    DROP TYPE public."ConnectionType_local_agent_old";
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM pg_enum e
    JOIN pg_type t ON t.oid = e.enumtypid
    WHERE t.typnamespace = 'public'::regnamespace
      AND t.typname = 'GatewayType'
      AND e.enumlabel IN ('LOCAL_AGENT', 'DESKTOP_GATEWAY')
  ) THEN
    ALTER TYPE public."GatewayType" RENAME TO "GatewayType_local_agent_old";
    CREATE TYPE public."GatewayType" AS ENUM ('GUACD', 'SSH_BASTION', 'MANAGED_SSH', 'DB_PROXY');
    ALTER TABLE public."Gateway"
      ALTER COLUMN type TYPE public."GatewayType"
      USING type::text::public."GatewayType";
    ALTER TABLE public."GatewayTemplate"
      ALTER COLUMN type TYPE public."GatewayType"
      USING type::text::public."GatewayType";
    DROP TYPE public."GatewayType_local_agent_old";
  END IF;
END $$;

DO $$
DECLARE
  audit_action_labels text;
BEGIN
  IF EXISTS (
    SELECT 1
    FROM pg_enum e
    JOIN pg_type t ON t.oid = e.enumtypid
    WHERE t.typnamespace = 'public'::regnamespace
      AND t.typname = 'AuditAction'
      AND e.enumlabel IN ('SUPPORT_CODE_ISSUE', 'SUPPORT_CODE_REDEEM')
  ) THEN
    SELECT string_agg(quote_literal(e.enumlabel), ', ' ORDER BY e.enumsortorder)
    INTO audit_action_labels
    FROM pg_enum e
    JOIN pg_type t ON t.oid = e.enumtypid
    WHERE t.typnamespace = 'public'::regnamespace
      AND t.typname = 'AuditAction'
      AND e.enumlabel NOT IN ('SUPPORT_CODE_ISSUE', 'SUPPORT_CODE_REDEEM');

    ALTER TYPE public."AuditAction" RENAME TO "AuditAction_local_agent_old";
    EXECUTE format('CREATE TYPE public."AuditAction" AS ENUM (%s)', audit_action_labels);
    ALTER TABLE public."AuditLog"
      ALTER COLUMN action TYPE public."AuditAction"
      USING action::text::public."AuditAction";
    DROP TYPE public."AuditAction_local_agent_old";
  END IF;
END $$;

DELETE FROM public.arsenale_schema_migrations
WHERE (version, name) IN (
  (13, '000013_local_agent_gateway.sql'),
  (14, '000014_support_codes.sql'),
  (15, '000015_support_code_connection_retire.sql'),
  (16, '000016_support_sessions.sql'),
  (17, '000017_support_connection_type.sql'),
  (18, '000018_fss_backend_slice.sql'),
  (19, '000019_fss_support_code_backfill.sql'),
  (20, '000020_support_agents.sql'),
  (21, '000021_desktop_gateway.sql'),
  (22, '000022_desktop_gateway_backfill.sql'),
  (23, '000023_support_connection_type.sql'),
  (24, '000024_support_vnc_pivot.sql'),
  (25, '000025_desktop_gateway_capabilities_vnc.sql')
);
