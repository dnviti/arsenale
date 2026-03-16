import React, { useState } from 'react';
import { sendMessage } from '../../lib/apiClient';
import type { BackgroundResponse, LoginResponse } from '../../types';

interface LoginPageProps {
  /** Called when login needs MFA verification. */
  onMfaRequired: (
    serverUrl: string,
    email: string,
    tempToken: string,
    methods: string[],
    requiresTOTP: boolean,
  ) => void;
  /** Called when MFA setup is required (opens web UI). */
  onMfaSetupRequired: (serverUrl: string) => void;
  /** Called after successful login (no MFA). */
  onSuccess: () => void;
}

export function LoginPage({
  onMfaRequired,
  onMfaSetupRequired,
  onSuccess,
}: LoginPageProps): React.ReactElement {
  const [serverUrl, setServerUrl] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [step, setStep] = useState<'server' | 'credentials'>('server');
  const [serverValid, setServerValid] = useState(false);

  const handleCheckServer = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!serverUrl.trim()) return;
    setLoading(true);
    setError(null);

    try {
      const { healthCheck } = await import('../../lib/apiClient');
      const result = await healthCheck(serverUrl.trim());
      if (result.success) {
        setServerValid(true);
        setStep('credentials');
      } else {
        setError(result.error ?? 'Could not connect to server');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email.trim() || !password) return;
    setLoading(true);
    setError(null);

    try {
      const result: BackgroundResponse<LoginResponse> = await sendMessage<LoginResponse>({
        type: 'LOGIN',
        serverUrl: serverUrl.trim(),
        email: email.trim(),
        password,
      });

      if (!result.success) {
        setError(result.error ?? 'Login failed');
        return;
      }

      const data = result.data;
      if (!data) {
        setError('Unexpected empty response');
        return;
      }

      // MFA setup required — open the web UI for setup
      if ('mfaSetupRequired' in data && data.mfaSetupRequired) {
        onMfaSetupRequired(serverUrl.trim());
        return;
      }

      // MFA verification required — navigate to MFA page
      if ('requiresMFA' in data && data.requiresMFA) {
        onMfaRequired(
          serverUrl.trim(),
          email.trim(),
          data.tempToken,
          data.methods,
          data.requiresTOTP ?? false,
        );
        return;
      }

      // Full success
      onSuccess();
    } finally {
      setLoading(false);
    }
  };

  const handleBack = () => {
    setStep('server');
    setServerValid(false);
    setError(null);
  };

  return (
    <div className="login-page">
      <div className="login-header">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
          <rect x="3" y="11" width="18" height="10" rx="2" />
          <path d="M7 11V7a5 5 0 0110 0v4" />
        </svg>
        <h2>Sign In</h2>
        <p>Connect to your Arsenale server.</p>
      </div>

      {step === 'server' && (
        <form onSubmit={handleCheckServer}>
          <div className="form-group">
            <label htmlFor="login-server">Server URL</label>
            <input
              id="login-server"
              type="text"
              className="input"
              placeholder="https://arsenale.example.com"
              value={serverUrl}
              onChange={(e) => setServerUrl(e.target.value)}
              disabled={loading}
              autoFocus
            />
          </div>
          {error && <p className="form-error">{error}</p>}
          <button
            type="submit"
            className="btn btn-primary btn-full"
            disabled={loading || !serverUrl.trim()}
          >
            {loading ? 'Checking...' : 'Continue'}
          </button>
        </form>
      )}

      {step === 'credentials' && (
        <form onSubmit={handleLogin}>
          <div className="form-server-info">
            <span className="form-server-badge">
              {serverValid ? 'Connected' : 'Checking...'}
            </span>
            <span className="form-server-url">{serverUrl}</span>
            <button type="button" className="btn btn-ghost btn-xs" onClick={handleBack}>
              Change
            </button>
          </div>
          <div className="form-group">
            <label htmlFor="login-email">Email</label>
            <input
              id="login-email"
              type="email"
              className="input"
              placeholder="user@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={loading}
              autoFocus
            />
          </div>
          <div className="form-group">
            <label htmlFor="login-password">Password</label>
            <input
              id="login-password"
              type="password"
              className="input"
              placeholder="Enter your password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={loading}
            />
          </div>
          {error && <p className="form-error">{error}</p>}
          <button
            type="submit"
            className="btn btn-primary btn-full"
            disabled={loading || !email.trim() || !password}
          >
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      )}
    </div>
  );
}
