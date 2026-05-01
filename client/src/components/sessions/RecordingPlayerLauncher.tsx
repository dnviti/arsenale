import { useEffect, useState } from 'react';
import { Loader2 } from 'lucide-react';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { getRecording, type Recording } from '@/api/recordings.api';
import { extractApiError } from '@/utils/apiError';
import RecordingPlayerDialog, { type RecordingPlayerInitialPanel } from '@/components/Recording/RecordingPlayerDialog';

interface RecordingPlayerLauncherProps {
  request: { recordingId: string; initialPanel?: RecordingPlayerInitialPanel } | null;
  onClose: () => void;
}

export default function RecordingPlayerLauncher({ request, onClose }: RecordingPlayerLauncherProps) {
  if (!request) {
    return null;
  }

  return (
    <RecordingPlayerLauncherContent
      key={request.recordingId}
      request={request}
      onClose={onClose}
    />
  );
}

interface RecordingPlayerLauncherContentProps {
  request: NonNullable<RecordingPlayerLauncherProps['request']>;
  onClose: () => void;
}

function RecordingPlayerLauncherContent({ request, onClose }: RecordingPlayerLauncherContentProps) {
  const [recording, setRecording] = useState<Recording | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;

    void getRecording(request.recordingId)
      .then((result) => {
        if (!cancelled) {
          setRecording(result);
        }
      })
      .catch((fetchError: unknown) => {
        if (!cancelled) {
          setError(extractApiError(fetchError, 'Failed to load recording details'));
        }
      });

    return () => {
      cancelled = true;
    };
  }, [request.recordingId]);

  const loading = !recording && !error;

  return (
    <>
      {loading || error ? (
        <Dialog open onOpenChange={(open) => { if (!open) onClose(); }}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>Recording viewer</DialogTitle>
              <DialogDescription>
                {loading ? 'Loading the recording details for playback.' : error}
              </DialogDescription>
            </DialogHeader>
            {loading ? (
              <div className="flex items-center justify-center py-6 text-muted-foreground">
                <Loader2 className="size-5 animate-spin" />
              </div>
            ) : null}
          </DialogContent>
        </Dialog>
      ) : null}

      <RecordingPlayerDialog
        open={Boolean(recording)}
        onClose={onClose}
        recording={recording}
        initialPanel={request.initialPanel}
      />
    </>
  );
}
