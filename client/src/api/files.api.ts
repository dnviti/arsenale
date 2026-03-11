import api from './client';

export interface FileInfo {
  name: string;
  size: number;
  modifiedAt: string;
}

export async function listFiles(): Promise<FileInfo[]> {
  const { data } = await api.get('/files');
  return data;
}

export async function uploadFile(file: File): Promise<FileInfo[]> {
  const formData = new FormData();
  formData.append('file', file);
  const { data } = await api.post('/files', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
  return data;
}

export async function downloadFile(name: string): Promise<void> {
  const { data } = await api.get(`/files/${encodeURIComponent(name)}`, {
    responseType: 'blob',
  });
  const url = window.URL.createObjectURL(new Blob([data]));
  const link = document.createElement('a');
  link.href = url;
  link.setAttribute('download', name);
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
}

export async function deleteFile(name: string): Promise<void> {
  await api.delete(`/files/${encodeURIComponent(name)}`);
}
