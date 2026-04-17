export function extractApiError(err: unknown, fallback: string): string {
  const axiosErr = err as {
    response?: { status?: number; data?: { error?: string; message?: string } | string };
    message?: string;
  };
  const status = axiosErr?.response?.status;
  const data = axiosErr?.response?.data;
  if (data && typeof data === 'object') {
    const objectData = data as { error?: string; message?: string };
    if (objectData.error || objectData.message) {
      return objectData.error || objectData.message || fallback;
    }
  }
  if (typeof data === 'string' && data.trim()) {
    return status === 413 || /request entity too large/i.test(data)
      ? 'File exceeds maximum upload size'
      : data;
  }
  if (status === 413) {
    return 'File exceeds maximum upload size';
  }
  return (
    fallback
  );
}
