export function downloadFile(content: BlobPart | BlobPart[], filename: string, type: string) {
  const blob = new Blob(Array.isArray(content) ? content : [content], { type });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function downloadTextFile(content: string, filename: string, type = 'text/plain') {
  downloadFile(content, filename, type);
}
