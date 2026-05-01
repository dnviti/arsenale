import { useEffect } from 'react';
import { useAuthStore } from '@/store/authStore';
import { useGatewayStore } from '@/store/gatewayStore';

const SESSION_COUNT_REFRESH_MS = 15000;

export function useSessionCountMonitor() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const fetchSessionCount = useGatewayStore((state) => state.fetchSessionCount);

  useEffect(() => {
    if (!accessToken) {
      return undefined;
    }

    void fetchSessionCount();
    const intervalId = window.setInterval(() => {
      void fetchSessionCount();
    }, SESSION_COUNT_REFRESH_MS);

    return () => window.clearInterval(intervalId);
  }, [accessToken, fetchSessionCount]);
}
