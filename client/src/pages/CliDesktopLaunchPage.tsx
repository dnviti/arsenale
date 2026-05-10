import { useEffect, useId, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AlertTriangle, Loader2 } from 'lucide-react';
import { redeemCliDesktopLaunch, type CliDesktopLaunchSession } from '../api/cliDesktopLaunch.api';
import RdpViewer from '../components/RDP/RdpViewer';
import VncViewer from '../components/VNC/VncViewer';
import { extractApiError } from '../utils/apiError';

export default function CliDesktopLaunchPage() {
  const [searchParams] = useSearchParams();
  const grant = searchParams.get('grant') ?? '';
  const missingGrant = !grant.trim();
  const [session, setSession] = useState<CliDesktopLaunchSession | null>(null);
  const [error, setError] = useState('');
  const routeId = useId();
  const tabId = useMemo(() => {
    return `cli-desktop-${routeId.replace(/[^a-zA-Z0-9_-]/g, '')}`;
  }, [routeId]);

  useEffect(() => {
    if (missingGrant) {
      return;
    }
    redeemCliDesktopLaunch(grant)
      .then((result) => {
        setSession(result);
        document.title = `${result.protocol} session - Arsenale`;
      })
      .catch((err) => {
        setError(extractApiError(err, 'Desktop launch failed.'));
      });
  }, [grant, missingGrant]);

  const displayError = missingGrant ? 'Launch grant is missing.' : error;

  if (displayError) {
    return (
      <div className="flex h-screen w-screen items-center justify-center bg-[#1a1a2e] p-6 text-foreground">
        <div className="flex max-w-md items-start gap-3 rounded-md border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          <AlertTriangle className="mt-0.5 size-5 shrink-0 text-red-300" aria-hidden="true" />
          <span>{displayError}</span>
        </div>
      </div>
    );
  }

  if (!session) {
    return (
      <div className="flex h-screen w-screen items-center justify-center bg-[#1a1a2e] text-foreground">
        <Loader2 className="mr-2 size-5 animate-spin" aria-hidden="true" />
        <span className="text-sm">Opening desktop session...</span>
      </div>
    );
  }

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[#1a1a2e]">
      {session.protocol === 'VNC' ? (
        <VncViewer connectionId={session.connectionId} tabId={tabId} launchSession={session} />
      ) : (
        <RdpViewer connectionId={session.connectionId} tabId={tabId} launchSession={session} />
      )}
    </div>
  );
}
