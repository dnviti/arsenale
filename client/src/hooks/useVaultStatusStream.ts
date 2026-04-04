import { useEffect } from 'react';
import type { VaultStatusStreamSnapshot } from '../api/live.api';
import { connectSSE } from '../api/sse';
import { useAuthStore } from '../store/authStore';
import { useFeatureFlagsStore } from '../store/featureFlagsStore';
import { useVaultStore } from '../store/vaultStore';

export function useVaultStatusStream() {
  const accessToken = useAuthStore((s) => s.accessToken);
  const featureFlagsLoaded = useFeatureFlagsStore((s) => s.loaded);
  const keychainEnabled = useFeatureFlagsStore((s) => s.keychainEnabled);
  const applyStatus = useVaultStore((s) => s.applyStatus);

  useEffect(() => {
    if (!accessToken || !featureFlagsLoaded || !keychainEnabled) return undefined;

    return connectSSE({
      url: '/api/vault/status/stream',
      accessToken,
      onEvent: ({ event, data }) => {
        if (event !== 'snapshot') return;
        applyStatus(data as VaultStatusStreamSnapshot);
      },
    });
  }, [accessToken, applyStatus, featureFlagsLoaded, keychainEnabled]);
}
