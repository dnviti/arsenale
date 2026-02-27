import passport from 'passport';
import { Strategy as GoogleStrategy } from 'passport-google-oauth20';
import { Strategy as GitHubStrategy } from 'passport-github2';
import { Strategy as MicrosoftStrategy } from 'passport-microsoft';
import { config } from '../config';
import { logger } from '../utils/logger';

export interface OAuthProfile {
  provider: 'GOOGLE' | 'MICROSOFT' | 'GITHUB';
  providerUserId: string;
  email: string;
  displayName: string | null;
  avatarUrl: string | null;
}

export interface OAuthCallbackData {
  oauthProfile: OAuthProfile;
  oauthTokens: { accessToken: string; refreshToken?: string };
}

function makeVerifyCallback(provider: OAuthProfile['provider']) {
  return (
    accessToken: string,
    refreshToken: string,
    profile: { id: string; displayName?: string; emails?: Array<{ value: string }>; photos?: Array<{ value: string }>; _json?: Record<string, unknown> },
    done: (err: Error | null, data?: OAuthCallbackData) => void
  ) => {
    try {
      const email =
        profile.emails?.[0]?.value ||
        (profile._json?.email as string | undefined) ||
        null;

      if (!email) {
        return done(new Error(`No email returned from ${provider}. Ensure the correct scopes are requested.`));
      }

      const oauthProfile: OAuthProfile = {
        provider,
        providerUserId: profile.id,
        email,
        displayName: profile.displayName || null,
        avatarUrl: profile.photos?.[0]?.value || null,
      };

      done(null, {
        oauthProfile,
        oauthTokens: { accessToken, refreshToken },
      });
    } catch (err) {
      done(err instanceof Error ? err : new Error(String(err)));
    }
  };
}

export function initializePassport(): void {
  if (config.oauth.google.enabled) {
    passport.use(
      new GoogleStrategy(
        {
          clientID: config.oauth.google.clientId,
          clientSecret: config.oauth.google.clientSecret,
          callbackURL: config.oauth.google.callbackUrl,
          scope: ['profile', 'email'],
        },
        makeVerifyCallback('GOOGLE') as any
      )
    );
    logger.info('OAuth: Google strategy registered');
  }

  if (config.oauth.microsoft.enabled) {
    passport.use(
      new MicrosoftStrategy(
        {
          clientID: config.oauth.microsoft.clientId,
          clientSecret: config.oauth.microsoft.clientSecret,
          callbackURL: config.oauth.microsoft.callbackUrl,
          scope: ['user.read'],
          tenant: 'common',
        },
        makeVerifyCallback('MICROSOFT') as any
      )
    );
    logger.info('OAuth: Microsoft strategy registered');
  }

  if (config.oauth.github.enabled) {
    passport.use(
      new GitHubStrategy(
        {
          clientID: config.oauth.github.clientId,
          clientSecret: config.oauth.github.clientSecret,
          callbackURL: config.oauth.github.callbackUrl,
          scope: ['user:email'],
        },
        makeVerifyCallback('GITHUB') as any
      )
    );
    logger.info('OAuth: GitHub strategy registered');
  }
}
