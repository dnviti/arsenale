import api from "./client";

export interface Recording {
  id: string;
  sessionId: string | null;
  userId: string;
  connectionId: string;
  protocol: "SSH" | "RDP" | "VNC";
  filePath: string;
  fileSize: number | null;
  duration: number | null;
  width: number | null;
  height: number | null;
  format: string;
  status: "RECORDING" | "COMPLETE" | "ERROR";
  createdAt: string;
  completedAt: string | null;
  connection: {
    id: string;
    name: string;
    type: string;
    host: string;
  };
  user?: {
    id: string;
    email: string;
    username: string | null;
  };
}

export interface RecordingsResponse {
  recordings: Recording[];
  total: number;
}

export async function listRecordings(params?: {
  connectionId?: string;
  protocol?: string;
  status?: string;
  limit?: number;
  offset?: number;
}): Promise<RecordingsResponse> {
  const { data } = await api.get("/recordings", { params });
  return data;
}

export async function getRecording(id: string): Promise<Recording> {
  const { data } = await api.get(`/recordings/${id}`);
  return data;
}

export async function deleteRecording(id: string): Promise<void> {
  await api.delete(`/recordings/${id}`);
}

export function getRecordingStreamUrl(id: string): string {
  return `/api/recordings/${id}/stream`;
}

export async function exportRecordingVideo(id: string): Promise<Blob> {
  const { data } = await api.get(`/recordings/${id}/video`, {
    responseType: "blob",
    timeout: 130000,
  });
  return data;
}

export interface RecordingAnalysis {
  fileSize: number;
  truncated: boolean;
  instructions: Record<string, number>;
  syncCount: number;
  displayWidth: number;
  displayHeight: number;
  hasLayer0Image: boolean;
}

export async function analyzeRecording(id: string): Promise<RecordingAnalysis> {
  const { data } = await api.get<RecordingAnalysis>(
    `/recordings/${id}/analyze`,
  );
  return data;
}
