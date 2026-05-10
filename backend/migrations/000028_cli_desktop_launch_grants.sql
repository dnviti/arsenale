CREATE TABLE IF NOT EXISTS public."DesktopLaunchGrant" (
    id text PRIMARY KEY,
    "secretHash" text NOT NULL,
    "tenantId" text,
    "userId" text NOT NULL,
    "connectionId" text NOT NULL,
    protocol public."SessionProtocol" NOT NULL,
    "expiresAt" timestamp(3) without time zone NOT NULL,
    "consumedAt" timestamp(3) without time zone,
    "consumedSessionId" text,
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "createdIpAddress" text,
    "consumedIpAddress" text,
    "createdUserAgent" text
);

CREATE INDEX IF NOT EXISTS "DesktopLaunchGrant_connection_idx"
ON public."DesktopLaunchGrant" ("connectionId");

CREATE INDEX IF NOT EXISTS "DesktopLaunchGrant_user_idx"
ON public."DesktopLaunchGrant" ("userId", "expiresAt");

CREATE TABLE IF NOT EXISTS public."DesktopViewerControlToken" (
    id text PRIMARY KEY,
    "secretHash" text NOT NULL,
    "tenantId" text,
    "userId" text NOT NULL,
    "sessionId" text NOT NULL REFERENCES public."ActiveSession"(id) ON DELETE CASCADE,
    protocol public."SessionProtocol" NOT NULL,
    "expiresAt" timestamp(3) without time zone NOT NULL,
    "revokedAt" timestamp(3) without time zone,
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS "DesktopViewerControlToken_session_idx"
ON public."DesktopViewerControlToken" ("sessionId");

CREATE TABLE IF NOT EXISTS public."SSHProxyGrant" (
    id text PRIMARY KEY,
    "secretHash" text NOT NULL,
    "tenantId" text,
    "userId" text NOT NULL,
    "connectionId" text NOT NULL,
    "expiresAt" timestamp(3) without time zone NOT NULL,
    "consumedAt" timestamp(3) without time zone,
    "consumedSessionId" text,
    "consumedIpAddress" text,
    "createdAt" timestamp(3) without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "createdIpAddress" text,
    "createdUserAgent" text
);

ALTER TABLE IF EXISTS public."SSHProxyGrant"
ADD COLUMN IF NOT EXISTS "consumedSessionId" text;

ALTER TABLE IF EXISTS public."SSHProxyGrant"
ADD COLUMN IF NOT EXISTS "consumedIpAddress" text;

CREATE INDEX IF NOT EXISTS "SSHProxyGrant_connection_idx"
ON public."SSHProxyGrant" ("connectionId");
