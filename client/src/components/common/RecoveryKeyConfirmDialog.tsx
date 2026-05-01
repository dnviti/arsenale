import { useState } from 'react';
import { Copy, Download } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { useCopyToClipboard } from '../../hooks/useCopyToClipboard';
import { downloadTextFile } from '../../utils/downloadFile';

interface RecoveryKeyConfirmDialogProps {
  open: boolean;
  recoveryKey: string;
  onConfirmed: () => void;
}

export default function RecoveryKeyConfirmDialog({
  open,
  recoveryKey,
  onConfirmed,
}: RecoveryKeyConfirmDialogProps) {
  return (
    <Dialog open={open}>
      {open ? (
        <RecoveryKeyConfirmDialogContent
          recoveryKey={recoveryKey}
          onConfirmed={onConfirmed}
        />
      ) : null}
    </Dialog>
  );
}

function RecoveryKeyConfirmDialogContent({
  recoveryKey,
  onConfirmed,
}: Omit<RecoveryKeyConfirmDialogProps, 'open'>) {
  const [step, setStep] = useState<'display' | 'verify'>('display');
  const [input, setInput] = useState('');
  const [verifyError, setVerifyError] = useState('');
  const { copied, copy } = useCopyToClipboard();

  const handleDownload = () => {
    const now = new Date();
    const pad = (value: number) => value.toString().padStart(2, '0');
    const timestamp = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
    downloadTextFile(recoveryKey, `arsenale-recovery-${timestamp}.key`);
  };

  const handleVerify = () => {
    if (input.trim() !== recoveryKey.trim()) {
      setVerifyError('Key does not match');
      return;
    }

    setInput('');
    setVerifyError('');
    setStep('display');
    onConfirmed();
  };

  return (
    <DialogContent
      showCloseButton={false}
      aria-describedby="recovery-key-description"
      onEscapeKeyDown={(event) => event.preventDefault()}
      onPointerDownOutside={(event) => event.preventDefault()}
    >
      {step === 'display' ? (
        <>
          <DialogHeader>
            <DialogTitle>Save Your Recovery Key</DialogTitle>
            <DialogDescription id="recovery-key-description">
              This key is shown only once. Save it before you continue.
            </DialogDescription>
          </DialogHeader>

          <Alert variant="warning">
            <AlertTitle>Recovery Required</AlertTitle>
            <AlertDescription>
              Your vault recovery key has been regenerated. You will need it if you ever
              lose access to your password.
            </AlertDescription>
          </Alert>

          <div className="rounded-xl border bg-muted/40 p-4">
            <p className="break-all font-mono text-sm leading-6 text-foreground">
              {recoveryKey}
            </p>
          </div>

          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" size="sm" onClick={() => copy(recoveryKey)}>
              <Copy className="size-4" />
              {copied ? 'Copied' : 'Copy'}
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={handleDownload}>
              <Download className="size-4" />
              Download
            </Button>
          </div>

          <DialogFooter>
            <Button type="button" onClick={() => setStep('verify')}>
              Next
            </Button>
          </DialogFooter>
        </>
      ) : (
        <>
          <DialogHeader>
            <DialogTitle>Confirm Your Recovery Key</DialogTitle>
            <DialogDescription id="recovery-key-description">
              Type or paste the key to confirm you have stored it somewhere safe.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3">
            <Textarea
              value={input}
              onChange={(event) => {
                setInput(event.target.value);
                setVerifyError('');
              }}
              rows={3}
              autoFocus
              placeholder="Paste your recovery key here"
            />
            {verifyError && (
              <Alert variant="destructive">
                <AlertDescription>{verifyError}</AlertDescription>
              </Alert>
            )}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setStep('display')}>
              Back
            </Button>
            <Button type="button" onClick={handleVerify} disabled={!input.trim()}>
              Done
            </Button>
          </DialogFooter>
        </>
      )}
    </DialogContent>
  );
}
