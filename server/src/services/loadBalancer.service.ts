import prisma, { LoadBalancingStrategy, ManagedInstanceStatus } from '../lib/prisma';
import { logger } from '../utils/logger';

const LOG_PREFIX = '[load-balancer]';

export interface SelectedInstance {
  id: string;
  host: string;
  port: number;
}

/**
 * Selects the best RUNNING + healthy instance for a managed gateway.
 * Returns null if no eligible instances exist (caller falls back to gateway host:port).
 */
export async function selectInstance(
  gatewayId: string,
  strategy: LoadBalancingStrategy,
): Promise<SelectedInstance | null> {
  const instances = await prisma.managedGatewayInstance.findMany({
    where: {
      gatewayId,
      status: ManagedInstanceStatus.RUNNING,
      healthStatus: 'healthy',
    },
    select: {
      id: true,
      host: true,
      port: true,
      _count: {
        select: {
          sessions: {
            where: { status: { not: 'CLOSED' } },
          },
        },
      },
    },
    orderBy: { createdAt: 'asc' },
  });

  if (instances.length === 0) {
    logger.warn(`${LOG_PREFIX} No healthy RUNNING instances for gateway ${gatewayId}`);
    return null;
  }

  let selected: (typeof instances)[number];

  if (strategy === 'LEAST_CONNECTIONS') {
    selected = instances.reduce((best, curr) =>
      curr._count.sessions < best._count.sessions ? curr : best,
    );
  } else {
    // ROUND_ROBIN: pick instance with fewest sessions, random tiebreak
    const minSessions = Math.min(...instances.map((i) => i._count.sessions));
    const candidates = instances.filter((i) => i._count.sessions === minSessions);
    selected = candidates[Math.floor(Math.random() * candidates.length)];
  }

  logger.debug(
    `${LOG_PREFIX} Selected instance ${selected.id} (${selected.host}:${selected.port}) ` +
      `for gateway ${gatewayId} using ${strategy} (${selected._count.sessions} active sessions)`,
  );

  return { id: selected.id, host: selected.host, port: selected.port };
}
