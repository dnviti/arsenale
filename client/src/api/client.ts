import axios from 'axios';
import { useAuthStore } from '../store/authStore';

const api = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json' },
});

// Request interceptor: attach JWT
api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Refresh lock: when multiple requests get 401 simultaneously,
// only the first one triggers a refresh; the rest wait for it.
let refreshPromise: Promise<string> | null = null;

// Response interceptor: handle 401 and refresh
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      const refreshToken = useAuthStore.getState().refreshToken;
      if (refreshToken) {
        try {
          if (!refreshPromise) {
            refreshPromise = axios
              .post('/api/auth/refresh', { refreshToken })
              .then((res) => {
                const { accessToken, user } = res.data;
                useAuthStore.getState().setAccessToken(accessToken);
                if (user) useAuthStore.getState().updateUser(user);
                return accessToken as string;
              })
              .finally(() => {
                refreshPromise = null;
              });
          }

          const accessToken = await refreshPromise;
          originalRequest.headers.Authorization = `Bearer ${accessToken}`;
          return api(originalRequest);
        } catch {
          useAuthStore.getState().logout();
        }
      }
    }

    return Promise.reject(error);
  }
);

export default api;
