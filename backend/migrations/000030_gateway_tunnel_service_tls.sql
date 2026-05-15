ALTER TABLE public."Gateway"
    ADD COLUMN IF NOT EXISTS "tunnelServiceCert" text,
    ADD COLUMN IF NOT EXISTS "tunnelServiceCertExp" timestamp(3) without time zone,
    ADD COLUMN IF NOT EXISTS "tunnelServiceKey" text,
    ADD COLUMN IF NOT EXISTS "tunnelServiceKeyIV" text,
    ADD COLUMN IF NOT EXISTS "tunnelServiceKeyTag" text;
