import api from './client';

// ---------------------------------------------------------------------------
// AI Query Generation Types (AISQL-2069)
// ---------------------------------------------------------------------------

export interface AiConfig {
  provider: string;
  hasApiKey: boolean;
  modelId: string;
  baseUrl: string | null;
  maxTokensPerRequest: number;
  dailyRequestLimit: number;
  enabled: boolean;
}

export interface AiConfigUpdate {
  provider?: string;
  apiKey?: string;
  modelId?: string;
  baseUrl?: string | null;
  maxTokensPerRequest?: number;
  dailyRequestLimit?: number;
  enabled?: boolean;
}

export interface ObjectRequest {
  name: string;
  schema: string;
  reason: string;
}

export interface AiAnalyzeResult {
  status: 'pending_approval';
  conversationId: string;
  objectRequests: ObjectRequest[];
}

export interface AiGenerateResult {
  status: 'complete';
  sql: string;
  explanation: string;
  firewallWarning?: string;
}

export async function getAiConfig(): Promise<AiConfig> {
  const { data } = await api.get('/ai/config');
  return data;
}

export async function updateAiConfig(update: AiConfigUpdate): Promise<AiConfig> {
  const { data } = await api.put('/ai/config', update);
  return data;
}

export async function analyzeQuery(
  sessionId: string,
  prompt: string,
  dbProtocol?: string,
): Promise<AiAnalyzeResult> {
  const { data } = await api.post('/ai/generate-query', {
    sessionId,
    prompt,
    dbProtocol,
  });
  return data;
}

export async function confirmGeneration(
  conversationId: string,
  approvedObjects: string[],
): Promise<AiGenerateResult> {
  const { data } = await api.post('/ai/generate-query/confirm', {
    conversationId,
    approvedObjects,
  });
  return data;
}

// ---------------------------------------------------------------------------
// AI Query Optimization Types (SQLVIZ-2070)
// ---------------------------------------------------------------------------

export interface DataRequest {
  type: string;
  target: string;
  reason: string;
}

export interface OptimizeQueryParams {
  sql: string;
  executionPlan: unknown;
  sessionId: string;
  dbProtocol: string;
  dbVersion?: string;
  schemaContext?: unknown;
}

export interface OptimizeQueryResult {
  status: 'needs_data' | 'complete';
  conversationId: string;
  dataRequests?: DataRequest[];
  optimizedSql?: string;
  explanation?: string;
  changes?: string[];
}

// ---------------------------------------------------------------------------
// AI Query Optimization API calls
// ---------------------------------------------------------------------------

export async function optimizeQuery(params: OptimizeQueryParams): Promise<OptimizeQueryResult> {
  const { data } = await api.post('/ai/optimize-query', params);
  return data;
}

export async function continueOptimization(
  conversationId: string,
  approvedData: Record<string, unknown>,
): Promise<OptimizeQueryResult> {
  const { data } = await api.post('/ai/optimize-query/continue', { conversationId, approvedData });
  return data;
}
