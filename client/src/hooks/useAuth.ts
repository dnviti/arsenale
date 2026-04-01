import axios from 'axios';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../store/authStore';
import { restoreSessionApi } from '../api/auth.api';

export function useAuth() {
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const accessToken = useAuthStore((s) => s.accessToken);
  const setAuth = useAuthStore((s) => s.setAuth);
  const logout = useAuthStore((s) => s.logout);
  const [loading, setLoading] = useState(!accessToken);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (accessToken) {
      setLoading(false);
      return;
    }

    let cancelled = false;
    let retryTimer: number | undefined;

    setLoading(true);
    restoreSessionApi()
      .then((data) => {
        if (cancelled) return;
        setAuth(data.accessToken, data.csrfToken, data.user);
        setLoading(false);
      })
      .catch((error) => {
        if (cancelled) return;
        if (axios.isAxiosError(error)) {
          const status = error.response?.status;
          if (status === 401 || status === 403) {
            logout();
            navigate('/login');
            setLoading(false);
            return;
          }
        }
        retryTimer = window.setTimeout(() => {
          setAttempt((value) => value + 1);
        }, 2000);
      });

    return () => {
      cancelled = true;
      if (retryTimer !== undefined) {
        window.clearTimeout(retryTimer);
      }
    };
  }, [accessToken, attempt, logout, navigate, setAuth]);

  return { isAuthenticated: isAuthenticated || Boolean(accessToken), loading };
}
