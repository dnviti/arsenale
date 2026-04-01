import { useEffect } from 'react';
import type { VaultStatusStreamSnapshot } from '../api/live.api';
import { connectSSE } from '../api/sse';
import { useAuthStore } from '../store/authStore';
import { useVaultStore } from '../store/vaultStore';

export function useVaultStatusStream() {
  const accessToken = useAuthStore((s) => s.accessToken);
  const applyStatus = useVaultStore((s) => s.applyStatus);

  useEffect(() => {
    if (!accessToken) return undefined;

    return connectSSE({
      url: '/api/vault/status/stream',
      accessToken,
      onEvent: ({ event, data }) => {
        if (event !== 'snapshot') return;
        applyStatus(data as VaultStatusStreamSnapshot);
      },
    });
  }, [accessToken, applyStatus]);
}
