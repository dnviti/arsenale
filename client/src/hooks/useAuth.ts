import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../store/authStore';
import { refreshApi } from '../api/auth.api';

export function useAuth() {
  const navigate = useNavigate();
  const { isAuthenticated, accessToken, setAccessToken, logout } = useAuthStore();

  useEffect(() => {
    if (isAuthenticated && !accessToken) {
      refreshApi()
        .then((data) => {
          setAccessToken(data.accessToken);
          if (data.csrfToken) useAuthStore.getState().setCsrfToken(data.csrfToken);
          if (data.user) useAuthStore.getState().updateUser(data.user);
        })
        .catch(() => {
          logout();
          navigate('/login');
        });
    }
  }, []);

  return { isAuthenticated };
}
