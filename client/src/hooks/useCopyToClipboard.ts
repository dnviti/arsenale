import { useState, useCallback, useRef } from 'react';

const COPIED_TIMEOUT_MS = 2000;

export function useCopyToClipboard() {
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const copy = useCallback(async (text: string) => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => setCopied(false), COPIED_TIMEOUT_MS);
  }, []);

  return { copied, copy };
}
