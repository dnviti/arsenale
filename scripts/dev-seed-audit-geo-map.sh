#!/usr/bin/env bash
set -euo pipefail

container_runtime="${CONTAINER_RUNTIME:-podman}"
postgres_container="${ARSENALE_POSTGRES_CONTAINER:-arsenale-postgres}"
db_name="${ARSENALE_POSTGRES_DB:-arsenale}"
db_user="${ARSENALE_POSTGRES_USER:-arsenale}"
seed_email="${ARSENALE_AUDIT_GEO_SEED_EMAIL:-admin@example.com}"
seed_count="${1:-${ARSENALE_AUDIT_GEO_SEED_COUNT:-4000}}"
seed_tag="${ARSENALE_AUDIT_GEO_SEED_TAG:-geo-map-demo}"
escaped_seed_email="${seed_email//\'/\'\'}"
escaped_seed_tag="${seed_tag//\'/\'\'}"

if ! [[ "${seed_count}" =~ ^[0-9]+$ ]] || [ "${seed_count}" -lt 1 ]; then
  echo "seed count must be a positive integer" >&2
  exit 1
fi

if ! command -v "${container_runtime}" >/dev/null 2>&1; then
  echo "container runtime '${container_runtime}' is not available" >&2
  exit 1
fi

seed_user_id="$(
  "${container_runtime}" exec -i "${postgres_container}" sh -lc \
    "PGPASSWORD=\$(cat /run/secrets/postgres_password) psql -At -U '${db_user}' -d '${db_name}' -c \"SELECT u.id FROM \\\"User\\\" u JOIN \\\"TenantMember\\\" tm ON tm.\\\"userId\\\" = u.id WHERE u.email = '${escaped_seed_email}' AND tm.status = 'ACCEPTED' ORDER BY tm.\\\"joinedAt\\\" ASC LIMIT 1\""
)"

if [ -z "${seed_user_id}" ]; then
  echo "No accepted tenant member found for ${seed_email}" >&2
  exit 1
fi

escaped_seed_user_id="${seed_user_id//\'/\'\'}"

"${container_runtime}" exec -i "${postgres_container}" sh -lc \
  "PGPASSWORD=\$(cat /run/secrets/postgres_password) psql -v ON_ERROR_STOP=1 -U '${db_user}' -d '${db_name}'" <<SQL
DELETE FROM "AuditLog"
WHERE id LIKE ('${escaped_seed_tag}' || '-%');

WITH locations AS (
  SELECT *
  FROM (VALUES
    (1, 'New York', 'United States', 40.7128, -74.0060, '8.8.8'),
    (2, 'San Francisco', 'United States', 37.7749, -122.4194, '1.1.1'),
    (3, 'Toronto', 'Canada', 43.6532, -79.3832, '99.84.238'),
    (4, 'Sao Paulo', 'Brazil', -23.5505, -46.6333, '54.230.0'),
    (5, 'London', 'United Kingdom', 51.5074, -0.1278, '151.101.1'),
    (6, 'Paris', 'France', 48.8566, 2.3522, '91.198.174'),
    (7, 'Frankfurt', 'Germany', 50.1109, 8.6821, '18.184.0'),
    (8, 'Johannesburg', 'South Africa', -26.2041, 28.0473, '154.127.0'),
    (9, 'Dubai', 'United Arab Emirates', 25.2048, 55.2708, '94.200.0'),
    (10, 'Mumbai', 'India', 19.0760, 72.8777, '49.37.0'),
    (11, 'Singapore', 'Singapore', 1.3521, 103.8198, '13.228.0'),
    (12, 'Tokyo', 'Japan', 35.6762, 139.6503, '210.140.92'),
    (13, 'Seoul', 'South Korea', 37.5665, 126.9780, '211.231.99'),
    (14, 'Sydney', 'Australia', -33.8688, 151.2093, '13.55.0'),
    (15, 'Auckland', 'New Zealand', -36.8509, 174.7645, '202.89.4'),
    (16, 'Mexico City', 'Mexico', 19.4326, -99.1332, '189.203.240'),
    (17, 'Los Angeles', 'United States', 34.0522, -118.2437, '142.250.72'),
    (18, 'Madrid', 'Spain', 40.4168, -3.7038, '80.58.61'),
    (19, 'Rome', 'Italy', 41.9028, 12.4964, '151.6.0'),
    (20, 'Amsterdam', 'Netherlands', 52.3676, 4.9041, '52.85.0')
  ) AS item(idx, city, country, lat, lng, ip_prefix)
),
generated_rows AS (
  SELECT
    format('%s-%s', '${escaped_seed_tag}', lpad(gs::text, 6, '0')) AS id,
    '${escaped_seed_user_id}'::text AS user_id,
    (ARRAY[
      'LOGIN',
      'SESSION_START',
      'SESSION_END',
      'GATEWAY_VIEW_LOGS',
      'TOKEN_HIJACK_ATTEMPT',
      'IMPOSSIBLE_TRAVEL_DETECTED'
    ])[1 + ((gs - 1) % 6)]::"AuditAction" AS action,
    'DemoActivity'::text AS target_type,
    format('demo-target-%s', 1 + ((gs - 1) % 250)) AS target_id,
    jsonb_build_object(
      'seed', '${escaped_seed_tag}',
      'source', 'scripts/dev-seed-audit-geo-map.sh',
      'city', location.city,
      'country', location.country,
      'sequence', gs
    ) AS details,
    format('%s.%s', location.ip_prefix, 1 + ((gs - 1) % 220)) AS ip_address,
    location.city AS geo_city,
    ARRAY[
      location.lat + (((gs % 11) - 5) * 0.08),
      location.lng + ((((gs / 11) % 11) - 5) * 0.08)
    ]::double precision[] AS geo_coords,
    location.country AS geo_country,
    CASE
      WHEN gs % 17 = 0 THEN ARRAY['IMPOSSIBLE_TRAVEL']::text[]
      WHEN gs % 11 = 0 THEN ARRAY['WATCHLIST']::text[]
      ELSE ARRAY[]::text[]
    END AS flags,
    NOW() - make_interval(mins => (gs % 43200)) AS created_at
  FROM generate_series(1, ${seed_count}) AS gs
  JOIN locations location
    ON location.idx = 1 + ((gs - 1) % (SELECT COUNT(*) FROM locations))
)
INSERT INTO "AuditLog" (
  id,
  "userId",
  action,
  "targetType",
  "targetId",
  details,
  "ipAddress",
  "geoCity",
  "geoCoords",
  "geoCountry",
  flags,
  "createdAt"
)
SELECT
  id,
  user_id,
  action,
  target_type,
  target_id,
  details,
  ip_address,
  geo_city,
  geo_coords,
  geo_country,
  flags,
  created_at
FROM generated_rows;

SELECT COUNT(*) AS seeded_rows
FROM "AuditLog"
WHERE id LIKE ('${escaped_seed_tag}' || '-%');
SQL
